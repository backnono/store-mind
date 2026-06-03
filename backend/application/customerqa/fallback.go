package customerqa

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domain "store-mind/domain/customerqa"
)

type fallbackOrchestrator struct {
	repo domain.Repository
	log  Logger
}

func newFallbackOrchestrator(repo domain.Repository, log Logger) Orchestrator {
	if log == nil {
		log = nopLogger{}
	}
	return &fallbackOrchestrator{repo: repo, log: log}
}

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
			Confidence:   0.95,
			FallbackUsed: true,
		},
		Answer: answer,
		Cards:  cards,
	}, nil
}

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
		return "你可以在小程序右上角点击“联系客服”进入人工服务。", nil, true, nil
	default:
		return "我主要负责回答本门店的商品、库存、优惠和购物流程问题。你可以问我商品在哪里，或者今天有什么活动。", nil, false, nil
	}
}

func (o *fallbackOrchestrator) answerProductLocation(ctx context.Context, sessionID, messageID, storeID int64, message string) (string, []ChatCard, bool, error) {
	query := extractProductQuery(message)
	products, err := o.searchProductsTool(ctx, sessionID, messageID, storeID, query)
	if err != nil {
		return "暂时无法查询商品位置，你可以稍后再试或联系人工客服。", nil, false, err
	}
	if len(products) == 0 {
		return fmt.Sprintf("我没有找到“%s”的在售信息。你可以换个叫法再问我，或联系人工客服确认。", query), nil, false, nil
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

func (o *fallbackOrchestrator) answerInventory(ctx context.Context, sessionID, messageID, storeID int64, message string) (string, []ChatCard, bool, error) {
	query := extractProductQuery(message)
	products, err := o.searchProductsTool(ctx, sessionID, messageID, storeID, query)
	if err != nil {
		return "暂时无法查询库存信息，你可以稍后再试或联系人工客服。", nil, false, err
	}
	if len(products) == 0 {
		return fmt.Sprintf("我没有找到“%s”的在售信息。你可以换个叫法再问我，或联系人工客服确认。", query), nil, false, nil
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

func (o *fallbackOrchestrator) searchProductsTool(ctx context.Context, sessionID, messageID, storeID int64, query string) ([]domain.Product, error) {
	start := time.Now()
	input := map[string]any{"store_id": storeID, "query": query, "limit": 5}
	items, err := o.repo.SearchProducts(ctx, storeID, query, 5)
	o.recordToolCall(ctx, sessionID, messageID, "search_products", input, items, time.Since(start), err)
	return items, err
}

func (o *fallbackOrchestrator) getProductLocationTool(ctx context.Context, sessionID, messageID, storeID, productID int64) (*domain.ProductLocation, error) {
	start := time.Now()
	input := map[string]any{"store_id": storeID, "product_id": productID}
	item, err := o.repo.GetProductLocation(ctx, storeID, productID)
	o.recordToolCall(ctx, sessionID, messageID, "get_product_location", input, item, time.Since(start), err)
	return item, err
}

func (o *fallbackOrchestrator) getInventoryTool(ctx context.Context, sessionID, messageID, storeID, skuID int64) (*domain.Inventory, error) {
	start := time.Now()
	input := map[string]any{"store_id": storeID, "sku_id": skuID}
	item, err := o.repo.GetInventory(ctx, storeID, skuID)
	o.recordToolCall(ctx, sessionID, messageID, "get_inventory", input, item, time.Since(start), err)
	return item, err
}

func (o *fallbackOrchestrator) listPromotionsTool(ctx context.Context, sessionID, messageID, storeID int64) ([]domain.Promotion, error) {
	start := time.Now()
	now := time.Now()
	input := map[string]any{"store_id": storeID, "limit": 5, "now": now}
	items, err := o.repo.ListActivePromotions(ctx, storeID, now, 5)
	o.recordToolCall(ctx, sessionID, messageID, "search_promotions", input, items, time.Since(start), err)
	return items, err
}

func (o *fallbackOrchestrator) searchFAQTool(ctx context.Context, sessionID, messageID, storeID int64, query string) ([]domain.FAQ, error) {
	start := time.Now()
	input := map[string]any{"store_id": storeID, "query": query, "limit": 5}
	items, err := o.repo.SearchFAQ(ctx, storeID, query, 5)
	o.recordToolCall(ctx, sessionID, messageID, "search_faq", input, items, time.Since(start), err)
	return items, err
}

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
	for i := 0; i < len(replacements); i += 2 {
		query = strings.ReplaceAll(query, replacements[i], replacements[i+1])
	}
	return strings.TrimSpace(query)
}

func buildProductDisplayName(name string, skuID *int64) string {
	if skuID == nil {
		return name
	}
	return name
}
