package customerqa

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domain "store-mind/domain/customerqa"
)

// —— 降级编排器 ——

// fallbackOrchestrator 是关键词匹配降级编排器。
// 当 LLM sidecar 不可用（analyzer/composer/retriever 任一为 nil）时自动启用。
// 通过硬编码关键词规则 → 调用 repo 的 tool 方法 → 拼装自然语言回答，不依赖任何 AI/LLM 组件。
type fallbackOrchestrator struct {
	repo domain.Repository
	log  Logger
}

// newFallbackOrchestrator 创建降级编排器，作为主编排器的兜底。
func newFallbackOrchestrator(repo domain.Repository, log Logger) Orchestrator {
	if log == nil {
		log = nopLogger{}
	}
	return &fallbackOrchestrator{repo: repo, log: log}
}

// routeIntent 基于关键词的意图识别（离线规则引擎）。
// 优先级从高到低：人工 → 活动 → FAQ → 库存 → 位置 → 不支持。
// 注意：关键词是硬编码的，新增规则需改代码，但在 LLM 可用时不会被调用。
func routeIntent(message string) string {
	switch {
	case strings.Contains(message, "人工") || strings.Contains(message, "客服"):
		return "handoff"
	case strings.Contains(message, "优惠") || strings.Contains(message, "活动"):
		return "promotion"
	case strings.Contains(message, "付款") || strings.Contains(message, "支付") || strings.Contains(message, "退款") || strings.Contains(message, "营业"):
		return "faq"
	case strings.Contains(message, "还有吗") || strings.Contains(message, "库存") || strings.Contains(message, "还有没有"):
		return "inventory"
	case strings.Contains(message, "在哪") || strings.Contains(message, "哪里") || strings.Contains(message, "位置"):
		return "product_location"
	default:
		return "unsupported"
	}
}

// Run 降级编排入口：关键词路由 → 调用数据库 tool → 返回自然语言回答。
// 每个步骤出错时打印 warn 日志但不中断，返回通用的兜底话术。
func (o *fallbackOrchestrator) Run(ctx context.Context, req OrchestratorRequest) (OrchestratorResult, error) {
	intent := routeIntent(req.Message)
	answer, cards, handoffRequired, toolErr := o.answerChat(ctx, req.SessionID, req.MessageID, req.StoreID, req.Message, intent)
	if toolErr != nil {
		o.log.Warn("app_chat_fallback_answer", "request_id", req.RequestID, "session_id", req.SessionID, "error", toolErr)
	}
	return OrchestratorResult{
		Decision: Decision{
			Intent:       intent,
			Route:        RouteFallback,
			NeedsHandoff: handoffRequired,
			Confidence:   0.95, // 规则引擎默认高置信度（规则命中率高）
			FallbackUsed: true, // 标记为降级，便于可观测
		},
		Answer: answer,
		Cards:  cards,
	}, nil
}

// answerChat 按意图分发到具体的回答方法。
func (o *fallbackOrchestrator) answerChat(ctx context.Context, sessionID, messageID, storeID int64, message, intent string) (string, []ChatCard, bool, error) {
	switch intent {
	case "product_location":
		return o.answerProductLocation(ctx, sessionID, messageID, storeID, message)
	case "inventory":
		return o.answerInventory(ctx, sessionID, messageID, storeID, message)
	case "promotion":
		return o.answerPromotion(ctx, sessionID, messageID, storeID)
	case "faq":
		return o.answerFAQ(ctx, sessionID, messageID, storeID, message)
	case "handoff":
		return "你可以在小程序右上角点击「联系客服」进入人工服务。", nil, true, nil
	default:
		// unsupported 意图：返回能力范围说明
		return "我主要负责回答本门店的商品、库存、优惠和购物流程问题。你可以问我商品在哪里，或者今天有什么活动。", nil, false, nil
	}
}

// answerProductLocation 处理商品位置查询：
//
//	搜索商品 → 查位置 → 拼装自然语言回答 + product 卡片。
func (o *fallbackOrchestrator) answerProductLocation(ctx context.Context, sessionID, messageID, storeID int64, message string) (string, []ChatCard, bool, error) {
	query := extractProductQuery(message)
	products, err := o.searchProductsTool(ctx, sessionID, messageID, storeID, query)
	if err != nil {
		return "暂时无法查询商品位置，你可以稍后再试或联系人工客服。", nil, false, err
	}
	if len(products) == 0 {
		return fmt.Sprintf("我没有找到「%s」的在售信息。你可以换个叫法再问我，或联系人工客服确认。", query), nil, false, nil
	}

	location, err := o.getProductLocationTool(ctx, sessionID, messageID, storeID, products[0].ID)
	if err != nil {
		return "暂时无法查询商品位置，你可以稍后再试或联系人工客服。", nil, false, err
	}
	card := ChatCard{
		Type:     "product",
		Name:     buildProductDisplayName(products[0].Name, location.SKUID),
		Location: fmt.Sprintf("%s %s 货架第%d层", location.ZoneName, location.ShelfCode, location.LayerNo),
	}
	if location.SKUID != nil {
		card.SKUID = *location.SKUID
	}
	return fmt.Sprintf("找到了。%s在%s %s 货架第%d层，%s。", products[0].Name, location.ZoneName, location.ShelfCode, location.LayerNo, location.PositionDesc), []ChatCard{card}, false, nil
}

// answerInventory 处理库存查询：
//
//	搜索商品 → 查位置（获取 SKU）→ 查库存 → 拼装 answer + inventory 卡片。
func (o *fallbackOrchestrator) answerInventory(ctx context.Context, sessionID, messageID, storeID int64, message string) (string, []ChatCard, bool, error) {
	query := extractProductQuery(message)
	products, err := o.searchProductsTool(ctx, sessionID, messageID, storeID, query)
	if err != nil {
		return "暂时无法查询库存信息，你可以稍后再试或联系人工客服。", nil, false, err
	}
	if len(products) == 0 {
		return fmt.Sprintf("我没有找到「%s」的在售信息。你可以换个叫法再问我，或联系人工客服确认。", query), nil, false, nil
	}

	location, err := o.getProductLocationTool(ctx, sessionID, messageID, storeID, products[0].ID)
	if err != nil {
		return "暂时无法查询库存信息，你可以稍后再试或联系人工客服。", nil, false, err
	}
	if location.SKUID == nil {
		return "暂时无法定位到对应 SKU 的库存记录，你可以联系人工客服确认。", nil, false, nil
	}

	inventory, err := o.getInventoryTool(ctx, sessionID, messageID, storeID, *location.SKUID)
	if err != nil {
		return "暂时无法查询库存信息，你可以稍后再试或联系人工客服。", nil, false, err
	}
	card := ChatCard{
		Type:     "inventory",
		SKUID:    inventory.SKUID,
		Name:     products[0].Name,
		Location: fmt.Sprintf("%s %s 货架", location.ZoneName, location.ShelfCode),
		Quantity: inventory.Quantity,
	}
	return fmt.Sprintf("系统显示%s还有 %d 件，在%s %s 货架。", products[0].Name, inventory.Quantity, location.ZoneName, location.ShelfCode), []ChatCard{card}, false, nil
}

// answerPromotion 处理促销活动查询：
//
//	列出当前有效活动 → 拼装 answer + promotion 卡片。
func (o *fallbackOrchestrator) answerPromotion(ctx context.Context, sessionID, messageID, storeID int64) (string, []ChatCard, bool, error) {
	items, err := o.listPromotionsTool(ctx, sessionID, messageID, storeID)
	if err != nil {
		return "暂时无法查询活动信息，你可以稍后再试或联系人工客服。", nil, false, err
	}
	if len(items) == 0 {
		return "当前没有查询到有效活动。你也可以问我某个商品有没有优惠。", nil, false, nil
	}
	card := ChatCard{
		Type:     "promotion",
		Title:    items[0].Title,
		Content:  items[0].Description,
		Validity: items[0].EndAt.Format("01-02 15:04"),
	}
	return fmt.Sprintf("当前有活动：%s，有效期到 %s。", items[0].Title, items[0].EndAt.Format("01-02 15:04")), []ChatCard{card}, false, nil
}

// answerFAQ 处理 FAQ 查询：
//
//	搜索 FAQ 知识库 → 取第一条配 → 拼装 answer + faq 卡片。
func (o *fallbackOrchestrator) answerFAQ(ctx context.Context, sessionID, messageID, storeID int64, message string) (string, []ChatCard, bool, error) {
	items, err := o.searchFAQTool(ctx, sessionID, messageID, storeID, message)
	if err != nil {
		return "暂时无法查询门店规则，你可以稍后再试或联系人工客服。", nil, false, err
	}
	if len(items) == 0 {
		return "我暂时没有找到对应的门店规则，你可以换个问法，或点击联系客服。", nil, false, nil
	}
	card := ChatCard{
		Type:    "faq",
		Title:   items[0].Question,
		Content: items[0].Answer,
	}
	return items[0].Answer, []ChatCard{card}, false, nil
}

// —— 工具调用包装方法 ——
// 每个 tool 方法负责：调用 repo → 计时 → 记录 tool_call 日志。
// 这些方法同时被 fallback 编排器和主编排器（collectToolEvidence）内部的路径使用。

// searchProductsTool 搜索商品并记录 tool_call。
func (o *fallbackOrchestrator) searchProductsTool(ctx context.Context, sessionID, messageID, storeID int64, query string) ([]domain.Product, error) {
	start := time.Now()
	input := map[string]any{"store_id": storeID, "query": query, "limit": 5}
	items, err := o.repo.SearchProducts(ctx, storeID, query, 5)
	o.recordToolCall(ctx, sessionID, messageID, "search_products", input, items, time.Since(start), err)
	return items, err
}

// getProductLocationTool 查询商品位置并记录 tool_call。
func (o *fallbackOrchestrator) getProductLocationTool(ctx context.Context, sessionID, messageID, storeID, productID int64) (*domain.ProductLocation, error) {
	start := time.Now()
	input := map[string]any{"store_id": storeID, "product_id": productID}
	item, err := o.repo.GetProductLocation(ctx, storeID, productID)
	o.recordToolCall(ctx, sessionID, messageID, "get_product_location", input, item, time.Since(start), err)
	return item, err
}

// getInventoryTool 查询库存并记录 tool_call。
func (o *fallbackOrchestrator) getInventoryTool(ctx context.Context, sessionID, messageID, storeID, skuID int64) (*domain.Inventory, error) {
	start := time.Now()
	input := map[string]any{"store_id": storeID, "sku_id": skuID}
	item, err := o.repo.GetInventory(ctx, storeID, skuID)
	o.recordToolCall(ctx, sessionID, messageID, "get_inventory", input, item, time.Since(start), err)
	return item, err
}

// listPromotionsTool 列出活动并记录 tool_call。
func (o *fallbackOrchestrator) listPromotionsTool(ctx context.Context, sessionID, messageID, storeID int64) ([]domain.Promotion, error) {
	start := time.Now()
	now := time.Now()
	input := map[string]any{"store_id": storeID, "limit": 5, "now": now}
	items, err := o.repo.ListActivePromotions(ctx, storeID, now, 5)
	o.recordToolCall(ctx, sessionID, messageID, "search_promotions", input, items, time.Since(start), err)
	return items, err
}

// searchFAQTool 搜索 FAQ 并记录 tool_call。
func (o *fallbackOrchestrator) searchFAQTool(ctx context.Context, sessionID, messageID, storeID int64, query string) ([]domain.FAQ, error) {
	start := time.Now()
	input := map[string]any{"store_id": storeID, "query": query, "limit": 5}
	items, err := o.repo.SearchFAQ(ctx, storeID, query, 5)
	o.recordToolCall(ctx, sessionID, messageID, "search_faq", input, items, time.Since(start), err)
	return items, err
}

// recordToolCall 将一次工具调用的入参、输出、耗时和错误序列化为 JSON 存入 ToolCall 表。
// 写入失败仅记录日志不中断业务，因为 tool_call 日志属于可观测性数据。
func (o *fallbackOrchestrator) recordToolCall(ctx context.Context, sessionID, messageID int64, toolName string, input any, output any, latency time.Duration, callErr error) {
	inputJSON, _ := json.Marshal(input)
	outputJSON, _ := json.Marshal(output)
	toolCall := &domain.ToolCall{
		SessionID:  sessionID,
		MessageID:  messageID,
		ToolName:   toolName,
		InputJSON:  string(inputJSON),
		OutputJSON: string(outputJSON),
		LatencyMS:  int(latency.Milliseconds()),
		Success:    callErr == nil,
	}
	if callErr != nil {
		toolCall.ErrorMessage = callErr.Error()
	}
	if _, err := o.repo.CreateToolCall(ctx, toolCall); err != nil {
		o.log.Error("app_tool_call_log_failed", "tool_name", toolName, "error", err)
	}
}

// —— 辅助函数 ——

// extractProductQuery 从用户消息中剥离问句噪声词，提取核心商品关键词。
// 例如："请问牛奶在哪里？" → "牛奶"
// 注意：当前使用简单的字符串替换，未来可由 LLM NER 替代以获得更精确的实体抽取。
func extractProductQuery(message string) string {
	query := strings.TrimSpace(message)
	replacements := []string{
		"请问", "",
		"我想找", "",
		"帮我找", "",
		"在哪里", "",
		"在哪", "",
		"位置", "",
		"还有没有", "",
		"还有吗", "",
		"库存", "",
		"有吗", "",
		"呢", "",
		"？", "",
		"?", "",
	}
	// 两两一组：偶数索引为待替换词，奇数索引为替换内容（均为空）
	for i := 0; i < len(replacements); i += 2 {
		query = strings.ReplaceAll(query, replacements[i], replacements[i+1])
	}
	return strings.TrimSpace(query)
}

// buildProductDisplayName 构造商品的展示名称。
// 当前直接返回 name，预留了 SKU 维度拼接的扩展点（如「牛奶(250ml)」）。
func buildProductDisplayName(name string, skuID *int64) string {
	if skuID == nil {
		return name
	}
	return name
}
