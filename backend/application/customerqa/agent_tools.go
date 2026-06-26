package customerqa

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domain "store-mind/domain/customerqa"
)

// —— Tool 接口 ——

// Tool 是 LLM 可调用的工具接口。
// Description 会注入 system prompt，告诉 LLM 何时使用此工具。
type Tool interface {
	Name() string
	Description() string
	Run(ctx context.Context, storeID, sessionID, messageID int64, args json.RawMessage) (string, error)
}

// —— 工具依赖 ——

// ToolDeps 聚合所有 Tool 需要的依赖。
type ToolDeps struct {
	Repo domain.Repository
	Log  Logger
}

// —— 工具注册表 ——

// AllAgentTools 返回 Agent 循环可用的全部工具列表。
func AllAgentTools(deps ToolDeps) []Tool {
	if deps.Log == nil {
		deps.Log = nopLogger{}
	}
	return []Tool{
		&SearchProductsTool{deps: deps},
		&GetProductLocationTool{deps: deps},
		&GetInventoryTool{deps: deps},
		&ListPromotionsTool{deps: deps},
		&SearchFAQTool{deps: deps},
		&GetPriceTool{deps: deps},
	}
}

// findTool 按名称查找工具。
func findTool(name string, tools []Tool) Tool {
	for _, t := range tools {
		if t.Name() == name {
			return t
		}
	}
	return nil
}

// toolDefinitions 生成 OpenAI 兼容的 tool_definitions JSON。
func toolDefinitions(tools []Tool) []map[string]any {
	defs := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		defs = append(defs, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name(),
				"description": t.Description(),
			},
		})
	}
	return defs
}

// —— 工具调用日志 ——

// recordToolCall 记录一次工具调用日志到 ToolCall 表。
func recordToolCall(ctx context.Context, deps ToolDeps, sessionID, messageID int64, toolName string, args json.RawMessage, output string, latency time.Duration, callErr error) {
	inputJSON := string(args)
	toolCall := &domain.ToolCall{
		SessionID:  sessionID,
		MessageID:  messageID,
		ToolName:   toolName,
		InputJSON:  inputJSON,
		OutputJSON: output,
		LatencyMS:  int(latency.Milliseconds()),
		Success:    callErr == nil,
	}
	if callErr != nil {
		toolCall.ErrorMessage = callErr.Error()
	}
	if _, err := deps.Repo.CreateToolCall(ctx, toolCall); err != nil {
		deps.Log.Error("agent_tool_call_log_failed", "tool_name", toolName, "error", err)
	}
}

// —— Tool 1: search_products ——

// SearchProductsTool 按关键词搜索门店商品。
type SearchProductsTool struct{ deps ToolDeps }

func (t *SearchProductsTool) Name() string { return "search_products" }
func (t *SearchProductsTool) Description() string {
	return `按关键词搜索门店商品。当用户询问商品是否存在、在哪里能买到时使用。
输入: {"query": "薯片"}
输出: [{"product_id":106, "name":"乐事原味薯片", "brand":"乐事", "category":"零食"}]`
}

func (t *SearchProductsTool) Run(ctx context.Context, storeID, sessionID, messageID int64, args json.RawMessage) (string, error) {
	start := time.Now()
	var input struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return `{"error": "invalid args: query is required"}`, err
	}
	if strings.TrimSpace(input.Query) == "" {
		return `{"error": "query is required"}`, nil
	}
	if input.Limit <= 0 {
		input.Limit = 5
	}

	products, err := t.deps.Repo.SearchProducts(ctx, storeID, input.Query, input.Limit)
	if err != nil {
		output := fmt.Sprintf(`{"error": "%s"}`, err.Error())
		recordToolCall(ctx, t.deps, sessionID, messageID, "search_products", args, output, time.Since(start), err)
		return output, err
	}

	type result struct {
		ProductID int64  `json:"product_id"`
		Name      string `json:"name"`
		Brand     string `json:"brand"`
		Category  string `json:"category"`
	}
	results := make([]result, len(products))
	for i, p := range products {
		results[i] = result{p.ID, p.Name, p.Brand, p.Category}
	}
	b, _ := json.Marshal(results)
	output := string(b)
	recordToolCall(ctx, t.deps, sessionID, messageID, "search_products", args, output, time.Since(start), nil)
	return output, nil
}

// —— Tool 2: get_product_location ——

// GetProductLocationTool 查询商品在门店的具体位置。
type GetProductLocationTool struct{ deps ToolDeps }

func (t *GetProductLocationTool) Name() string { return "get_product_location" }
func (t *GetProductLocationTool) Description() string {
	return `查询商品在门店的具体位置（区域、货架、层）。需要先通过 search_products 获取 product_id。
输入: {"product_id": 106}
输出: {"product_id":106, "zone_name":"零食区", "shelf_code":"A-03", "layer_no":2, "position_desc":"中间位置", "card_type":"product"}`
}

func (t *GetProductLocationTool) Run(ctx context.Context, storeID, sessionID, messageID int64, args json.RawMessage) (string, error) {
	start := time.Now()
	var input struct {
		ProductID int64 `json:"product_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil || input.ProductID <= 0 {
		output := `{"error": "invalid args: product_id is required"}`
		recordToolCall(ctx, t.deps, sessionID, messageID, "get_product_location", args, output, time.Since(start), nil)
		return output, nil
	}

	loc, err := t.deps.Repo.GetProductLocation(ctx, storeID, input.ProductID)
	if err != nil {
		output := fmt.Sprintf(`{"error": "%s"}`, err.Error())
		recordToolCall(ctx, t.deps, sessionID, messageID, "get_product_location", args, output, time.Since(start), err)
		return output, err
	}

	result := map[string]any{
		"product_id":    loc.ProductID,
		"zone_name":     loc.ZoneName,
		"shelf_code":    loc.ShelfCode,
		"layer_no":      loc.LayerNo,
		"position_desc": loc.PositionDesc,
		"location":      fmt.Sprintf("%s %s 货架第%d层", loc.ZoneName, loc.ShelfCode, loc.LayerNo),
		"card_type":     "product",
	}
	if loc.SKUID != nil {
		result["sku_id"] = *loc.SKUID
	}
	b, _ := json.Marshal(result)
	output := string(b)
	recordToolCall(ctx, t.deps, sessionID, messageID, "get_product_location", args, output, time.Since(start), nil)
	return output, nil
}

// —— Tool 3: get_inventory ——

// GetInventoryTool 查询商品库存信息。
type GetInventoryTool struct{ deps ToolDeps }

func (t *GetInventoryTool) Name() string { return "get_inventory" }
func (t *GetInventoryTool) Description() string {
	return `查询商品库存数量。通常需要先通过 get_product_location 获取 sku_id。
输入: {"sku_id": 2001}
输出: {"product_name":"乐事原味薯片", "quantity":15, "price":8.50, "spec":"75g", "credibility":"🟢 高可信 · 10分钟前更新", "card_type":"inventory"}`
}

func (t *GetInventoryTool) Run(ctx context.Context, storeID, sessionID, messageID int64, args json.RawMessage) (string, error) {
	start := time.Now()
	var input struct {
		SKUID int64 `json:"sku_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil || input.SKUID <= 0 {
		output := `{"error": "invalid args: sku_id is required"}`
		recordToolCall(ctx, t.deps, sessionID, messageID, "get_inventory", args, output, time.Since(start), nil)
		return output, nil
	}

	inv, err := t.deps.Repo.GetInventory(ctx, storeID, input.SKUID)
	if err != nil {
		output := fmt.Sprintf(`{"error": "%s"}`, err.Error())
		recordToolCall(ctx, t.deps, sessionID, messageID, "get_inventory", args, output, time.Since(start), err)
		return output, err
	}

	credTag := CredibilityTag(inv)
	result := map[string]any{
		"sku_id":       inv.SKUID,
		"product_id":   inv.ProductID,
		"product_name": inv.ProductName,
		"quantity":     inv.Quantity,
		"price":        inv.Price,
		"spec":         inv.Spec,
		"credibility":  credTag,
		"card_type":    "inventory",
	}
	b, _ := json.Marshal(result)
	output := string(b)
	recordToolCall(ctx, t.deps, sessionID, messageID, "get_inventory", args, output, time.Since(start), nil)
	return output, nil
}

// —— Tool 4: list_promotions ——

// ListPromotionsTool 列出当前有效的促销活动。
type ListPromotionsTool struct{ deps ToolDeps }

func (t *ListPromotionsTool) Name() string { return "list_promotions" }
func (t *ListPromotionsTool) Description() string {
	return `列出当前有效的门店促销活动。当用户询问有什么活动、优惠时使用。
输入: {}
输出: [{"promotion_id":1, "title":"夏日特惠", "description":"全场饮料8折", "end_at":"2026-07-15"}]`
}

func (t *ListPromotionsTool) Run(ctx context.Context, storeID, sessionID, messageID int64, args json.RawMessage) (string, error) {
	start := time.Now()
	now := time.Now()
	items, err := t.deps.Repo.ListActivePromotions(ctx, storeID, now, 5)
	if err != nil {
		output := fmt.Sprintf(`{"error": "%s"}`, err.Error())
		recordToolCall(ctx, t.deps, sessionID, messageID, "list_promotions", args, output, time.Since(start), err)
		return output, err
	}

	type promoResult struct {
		PromotionID int64  `json:"promotion_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		EndAt       string `json:"end_at"`
		CardType    string `json:"card_type"`
	}
	results := make([]promoResult, len(items))
	for i, p := range items {
		results[i] = promoResult{p.ID, p.Title, p.Description, p.EndAt.Format("01-02 15:04"), "promotion"}
	}
	b, _ := json.Marshal(results)
	output := string(b)
	recordToolCall(ctx, t.deps, sessionID, messageID, "list_promotions", args, output, time.Since(start), nil)
	return output, nil
}

// —— Tool 5: search_faq ——

// SearchFAQTool 搜索 FAQ 知识库。
type SearchFAQTool struct{ deps ToolDeps }

func (t *SearchFAQTool) Name() string { return "search_faq" }
func (t *SearchFAQTool) Description() string {
	return `搜索门店 FAQ 知识库。当用户询问门店规则、营业时间、退换货政策、支付方式等非商品类问题时使用。
输入: {"query": "怎么退款"}
输出: [{"faq_id":10, "question":"如何退款？", "answer":"...", "card_type":"faq"}]`
}

func (t *SearchFAQTool) Run(ctx context.Context, storeID, sessionID, messageID int64, args json.RawMessage) (string, error) {
	start := time.Now()
	var input struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return `{"error": "invalid args: query is required"}`, err
	}
	if strings.TrimSpace(input.Query) == "" {
		return `{"error": "query is required"}`, nil
	}
	if input.Limit <= 0 {
		input.Limit = 5
	}

	items, err := t.deps.Repo.SearchFAQ(ctx, storeID, input.Query, input.Limit)
	if err != nil {
		output := fmt.Sprintf(`{"error": "%s"}`, err.Error())
		recordToolCall(ctx, t.deps, sessionID, messageID, "search_faq", args, output, time.Since(start), err)
		return output, err
	}

	type faqResult struct {
		FAQID    int64  `json:"faq_id"`
		Question string `json:"question"`
		Answer   string `json:"answer"`
		CardType string `json:"card_type"`
	}
	results := make([]faqResult, len(items))
	for i, f := range items {
		results[i] = faqResult{f.ID, f.Question, f.Answer, "faq"}
	}
	b, _ := json.Marshal(results)
	output := string(b)
	recordToolCall(ctx, t.deps, sessionID, messageID, "search_faq", args, output, time.Since(start), nil)
	return output, nil
}

// —— Tool 6: get_price ——

// GetPriceTool 查询商品价格。
type GetPriceTool struct{ deps ToolDeps }

func (t *GetPriceTool) Name() string { return "get_price" }
func (t *GetPriceTool) Description() string {
	return `查询商品价格。需要先通过 get_product_location 获取 sku_id。
输入: {"sku_id": 2001}
输出: {"product_name":"乐事原味薯片", "price":8.50, "spec":"75g", "card_type":"price"}`
}

func (t *GetPriceTool) Run(ctx context.Context, storeID, sessionID, messageID int64, args json.RawMessage) (string, error) {
	start := time.Now()
	var input struct {
		SKUID int64 `json:"sku_id"`
	}
	if err := json.Unmarshal(args, &input); err != nil || input.SKUID <= 0 {
		output := `{"error": "invalid args: sku_id is required"}`
		recordToolCall(ctx, t.deps, sessionID, messageID, "get_price", args, output, time.Since(start), nil)
		return output, nil
	}

	inv, err := t.deps.Repo.GetInventory(ctx, storeID, input.SKUID)
	if err != nil {
		output := fmt.Sprintf(`{"error": "%s"}`, err.Error())
		recordToolCall(ctx, t.deps, sessionID, messageID, "get_price", args, output, time.Since(start), err)
		return output, err
	}

	result := map[string]any{
		"sku_id":       inv.SKUID,
		"product_id":   inv.ProductID,
		"product_name": inv.ProductName,
		"price":        inv.Price,
		"spec":         inv.Spec,
		"card_type":    "price",
	}
	b, _ := json.Marshal(result)
	output := string(b)
	recordToolCall(ctx, t.deps, sessionID, messageID, "get_price", args, output, time.Since(start), nil)
	return output, nil
}
