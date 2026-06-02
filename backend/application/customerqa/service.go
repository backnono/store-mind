package customerqa

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	domain "store-mind/domain/customerqa"
)

type ChatRequest struct {
	RequestID string
	StoreID   int64
	UserID    *int64
	Channel   string
	Message   string
}

type ChatResponse struct {
	SessionID int64
	MessageID int64
	Intent    string
	Answer    string
}

type FAQSearchRequest struct {
	RequestID string
	StoreID   int64
	Query     string
	Limit     int
}

type ProductSearchRequest struct {
	RequestID string
	StoreID   int64
	Query     string
	Limit     int
}

type ProductLocationRequest struct {
	RequestID string
	StoreID   int64
	ProductID int64
}

type InventoryRequest struct {
	RequestID string
	StoreID   int64
	SKUID     int64
}

type PromotionListRequest struct {
	RequestID string
	StoreID   int64
	Limit     int
	Now       time.Time
}

type Service interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	SearchFAQ(ctx context.Context, req FAQSearchRequest) ([]domain.FAQ, error)
	SearchProducts(ctx context.Context, req ProductSearchRequest) ([]domain.Product, error)
	GetProductLocation(ctx context.Context, req ProductLocationRequest) (*domain.ProductLocation, error)
	GetInventory(ctx context.Context, req InventoryRequest) (*domain.Inventory, error)
	ListActivePromotions(ctx context.Context, req PromotionListRequest) ([]domain.Promotion, error)
}

type service struct {
	repo domain.Repository
	log  Logger
}

func NewService(repo domain.Repository, log Logger) Service {
	if log == nil {
		log = nopLogger{}
	}
	return &service{repo: repo, log: log}
}

func (s *service) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if req.StoreID <= 0 || strings.TrimSpace(req.Message) == "" {
		s.log.Warn("app_chat_invalid_argument", "request_id", req.RequestID, "store_id", req.StoreID)
		return nil, domain.ErrInvalidArgument
	}
	if strings.TrimSpace(req.Channel) == "" {
		req.Channel = "miniapp"
	}
	req.Message = strings.TrimSpace(req.Message)

	s.log.Info("app_chat_start", "request_id", req.RequestID, "store_id", req.StoreID, "channel", req.Channel)
	session, err := s.repo.CreateSession(ctx, &domain.Session{StoreID: req.StoreID, UserID: req.UserID, Channel: req.Channel})
	if err != nil {
		s.log.Error("app_chat_create_session_failed", "request_id", req.RequestID, "error", err)
		return nil, err
	}
	intent := routeIntent(req.Message)
	confidence := 0.95
	msg, err := s.repo.CreateMessage(ctx, &domain.Message{SessionID: session.ID, Role: "user", Content: req.Message, Intent: intent, Confidence: &confidence})
	if err != nil {
		s.log.Error("app_chat_create_message_failed", "request_id", req.RequestID, "session_id", session.ID, "error", err)
		return nil, err
	}

	answer, toolErr := s.answerChat(ctx, session.ID, msg.ID, req.StoreID, req.Message, intent)
	assistantMsg, err := s.repo.CreateMessage(ctx, &domain.Message{SessionID: session.ID, Role: "assistant", Content: answer, Intent: intent})
	if err != nil {
		s.log.Error("app_chat_create_assistant_message_failed", "request_id", req.RequestID, "session_id", session.ID, "error", err)
		return nil, err
	}

	if toolErr != nil {
		s.log.Warn("app_chat_fallback_answer", "request_id", req.RequestID, "session_id", session.ID, "error", toolErr)
	}
	s.log.Info("app_chat_success", "request_id", req.RequestID, "session_id", session.ID, "message_id", assistantMsg.ID, "intent", intent)
	return &ChatResponse{SessionID: session.ID, MessageID: assistantMsg.ID, Intent: intent, Answer: answer}, nil
}

func (s *service) SearchFAQ(ctx context.Context, req FAQSearchRequest) ([]domain.FAQ, error) {
	if req.StoreID <= 0 || strings.TrimSpace(req.Query) == "" {
		s.log.Warn("app_faq_invalid_argument", "request_id", req.RequestID, "store_id", req.StoreID)
		return nil, domain.ErrInvalidArgument
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 50 {
		req.Limit = 50
	}

	items, err := s.repo.SearchFAQ(ctx, req.StoreID, req.Query, req.Limit)
	if err != nil {
		s.log.Error("app_faq_search_failed", "request_id", req.RequestID, "store_id", req.StoreID, "error", err)
		return nil, err
	}
	s.log.Info("app_faq_search_success", "request_id", req.RequestID, "store_id", req.StoreID, "count", len(items))
	return items, nil
}

func (s *service) SearchProducts(ctx context.Context, req ProductSearchRequest) ([]domain.Product, error) {
	if req.StoreID <= 0 || strings.TrimSpace(req.Query) == "" {
		s.log.Warn("app_product_search_invalid_argument", "request_id", req.RequestID, "store_id", req.StoreID)
		return nil, domain.ErrInvalidArgument
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 50 {
		req.Limit = 50
	}

	items, err := s.repo.SearchProducts(ctx, req.StoreID, req.Query, req.Limit)
	if err != nil {
		s.log.Error("app_product_search_failed", "request_id", req.RequestID, "store_id", req.StoreID, "error", err)
		return nil, err
	}
	s.log.Info("app_product_search_success", "request_id", req.RequestID, "store_id", req.StoreID, "count", len(items))
	return items, nil
}

func (s *service) GetProductLocation(ctx context.Context, req ProductLocationRequest) (*domain.ProductLocation, error) {
	if req.StoreID <= 0 || req.ProductID <= 0 {
		s.log.Warn("app_product_location_invalid_argument", "request_id", req.RequestID, "store_id", req.StoreID, "product_id", req.ProductID)
		return nil, domain.ErrInvalidArgument
	}

	item, err := s.repo.GetProductLocation(ctx, req.StoreID, req.ProductID)
	if err != nil {
		s.log.Error("app_product_location_failed", "request_id", req.RequestID, "store_id", req.StoreID, "product_id", req.ProductID, "error", err)
		return nil, err
	}
	s.log.Info("app_product_location_success", "request_id", req.RequestID, "store_id", req.StoreID, "product_id", req.ProductID)
	return item, nil
}

func (s *service) GetInventory(ctx context.Context, req InventoryRequest) (*domain.Inventory, error) {
	if req.StoreID <= 0 || req.SKUID <= 0 {
		s.log.Warn("app_inventory_invalid_argument", "request_id", req.RequestID, "store_id", req.StoreID, "sku_id", req.SKUID)
		return nil, domain.ErrInvalidArgument
	}

	item, err := s.repo.GetInventory(ctx, req.StoreID, req.SKUID)
	if err != nil {
		s.log.Error("app_inventory_failed", "request_id", req.RequestID, "store_id", req.StoreID, "sku_id", req.SKUID, "error", err)
		return nil, err
	}
	s.log.Info("app_inventory_success", "request_id", req.RequestID, "store_id", req.StoreID, "sku_id", req.SKUID)
	return item, nil
}

func (s *service) ListActivePromotions(ctx context.Context, req PromotionListRequest) ([]domain.Promotion, error) {
	if req.StoreID <= 0 {
		s.log.Warn("app_promotion_list_invalid_argument", "request_id", req.RequestID, "store_id", req.StoreID)
		return nil, domain.ErrInvalidArgument
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 50 {
		req.Limit = 50
	}
	if req.Now.IsZero() {
		req.Now = time.Now()
	}

	items, err := s.repo.ListActivePromotions(ctx, req.StoreID, req.Now, req.Limit)
	if err != nil {
		s.log.Error("app_promotion_list_failed", "request_id", req.RequestID, "store_id", req.StoreID, "error", err)
		return nil, err
	}
	s.log.Info("app_promotion_list_success", "request_id", req.RequestID, "store_id", req.StoreID, "count", len(items))
	return items, nil
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

func (s *service) answerChat(ctx context.Context, sessionID, messageID, storeID int64, message, intent string) (string, error) {
	switch intent {
	case "product_location":
		return s.answerProductLocation(ctx, sessionID, messageID, storeID, message)
	case "inventory":
		return s.answerInventory(ctx, sessionID, messageID, storeID, message)
	case "promotion":
		return s.answerPromotion(ctx, sessionID, messageID, storeID)
	case "faq":
		return s.answerFAQ(ctx, sessionID, messageID, storeID, message)
	case "handoff":
		return "你可以在小程序右上角点击“联系客服”进入人工服务。", nil
	default:
		return "我主要负责回答本门店的商品、库存、优惠和购物流程问题。你可以问我商品在哪里，或者今天有什么活动。", nil
	}
}

func (s *service) answerProductLocation(ctx context.Context, sessionID, messageID, storeID int64, message string) (string, error) {
	query := extractProductQuery(message)
	products, err := s.searchProductsTool(ctx, sessionID, messageID, storeID, query)
	if err != nil {
		return "暂时无法查询商品位置，你可以稍后再试或联系人工客服。", err
	}
	if len(products) == 0 {
		return fmt.Sprintf("我没有找到“%s”的在售信息。你可以换个叫法再问我，或联系人工客服确认。", query), nil
	}

	location, err := s.getProductLocationTool(ctx, sessionID, messageID, storeID, products[0].ID)
	if err != nil {
		return "暂时无法查询商品位置，你可以稍后再试或联系人工客服。", err
	}
	return fmt.Sprintf("找到了。%s在%s %s 货架第%d层，%s。", products[0].Name, location.ZoneName, location.ShelfCode, location.LayerNo, location.PositionDesc), nil
}

func (s *service) answerInventory(ctx context.Context, sessionID, messageID, storeID int64, message string) (string, error) {
	query := extractProductQuery(message)
	products, err := s.searchProductsTool(ctx, sessionID, messageID, storeID, query)
	if err != nil {
		return "暂时无法查询库存信息，你可以稍后再试或联系人工客服。", err
	}
	if len(products) == 0 {
		return fmt.Sprintf("我没有找到“%s”的在售信息。你可以换个叫法再问我，或联系人工客服确认。", query), nil
	}

	location, err := s.getProductLocationTool(ctx, sessionID, messageID, storeID, products[0].ID)
	if err != nil {
		return "暂时无法查询库存信息，你可以稍后再试或联系人工客服。", err
	}
	if location.SKUID == nil {
		return "暂时无法定位到对应 SKU 的库存记录，你可以联系人工客服确认。", nil
	}

	inventory, err := s.getInventoryTool(ctx, sessionID, messageID, storeID, *location.SKUID)
	if err != nil {
		return "暂时无法查询库存信息，你可以稍后再试或联系人工客服。", err
	}
	return fmt.Sprintf("系统显示%s还有 %d 件，在%s %s 货架。", products[0].Name, inventory.Quantity, location.ZoneName, location.ShelfCode), nil
}

func (s *service) answerPromotion(ctx context.Context, sessionID, messageID, storeID int64) (string, error) {
	items, err := s.listPromotionsTool(ctx, sessionID, messageID, storeID)
	if err != nil {
		return "暂时无法查询活动信息，你可以稍后再试或联系人工客服。", err
	}
	if len(items) == 0 {
		return "当前没有查询到有效活动。你也可以问我某个商品有没有优惠。", nil
	}
	return fmt.Sprintf("当前有活动：%s，有效期到 %s。", items[0].Title, items[0].EndAt.Format("01-02 15:04")), nil
}

func (s *service) answerFAQ(ctx context.Context, sessionID, messageID, storeID int64, message string) (string, error) {
	items, err := s.searchFAQTool(ctx, sessionID, messageID, storeID, message)
	if err != nil {
		return "暂时无法查询门店规则，你可以稍后再试或联系人工客服。", err
	}
	if len(items) == 0 {
		return "我暂时没有找到对应的门店规则，你可以换个问法，或点击联系客服。", nil
	}
	return items[0].Answer, nil
}

func (s *service) searchProductsTool(ctx context.Context, sessionID, messageID, storeID int64, query string) ([]domain.Product, error) {
	start := time.Now()
	input := map[string]any{"store_id": storeID, "query": query, "limit": 5}
	items, err := s.repo.SearchProducts(ctx, storeID, query, 5)
	s.recordToolCall(ctx, sessionID, messageID, "search_products", input, items, time.Since(start), err)
	return items, err
}

func (s *service) getProductLocationTool(ctx context.Context, sessionID, messageID, storeID, productID int64) (*domain.ProductLocation, error) {
	start := time.Now()
	input := map[string]any{"store_id": storeID, "product_id": productID}
	item, err := s.repo.GetProductLocation(ctx, storeID, productID)
	s.recordToolCall(ctx, sessionID, messageID, "get_product_location", input, item, time.Since(start), err)
	return item, err
}

func (s *service) getInventoryTool(ctx context.Context, sessionID, messageID, storeID, skuID int64) (*domain.Inventory, error) {
	start := time.Now()
	input := map[string]any{"store_id": storeID, "sku_id": skuID}
	item, err := s.repo.GetInventory(ctx, storeID, skuID)
	s.recordToolCall(ctx, sessionID, messageID, "get_inventory", input, item, time.Since(start), err)
	return item, err
}

func (s *service) listPromotionsTool(ctx context.Context, sessionID, messageID, storeID int64) ([]domain.Promotion, error) {
	start := time.Now()
	now := time.Now()
	input := map[string]any{"store_id": storeID, "limit": 5, "now": now}
	items, err := s.repo.ListActivePromotions(ctx, storeID, now, 5)
	s.recordToolCall(ctx, sessionID, messageID, "search_promotions", input, items, time.Since(start), err)
	return items, err
}

func (s *service) searchFAQTool(ctx context.Context, sessionID, messageID, storeID int64, query string) ([]domain.FAQ, error) {
	start := time.Now()
	input := map[string]any{"store_id": storeID, "query": query, "limit": 5}
	items, err := s.repo.SearchFAQ(ctx, storeID, query, 5)
	s.recordToolCall(ctx, sessionID, messageID, "search_faq", input, items, time.Since(start), err)
	return items, err
}

func (s *service) recordToolCall(ctx context.Context, sessionID, messageID int64, toolName string, input any, output any, latency time.Duration, callErr error) {
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
	if _, err := s.repo.CreateToolCall(ctx, toolCall); err != nil {
		s.log.Error("app_tool_call_log_failed", "tool_name", toolName, "error", err)
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
