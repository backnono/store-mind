package customerqa

import (
	"context"
	"strings"
	"time"

	domain "store-mind/domain/customerqa"
)

type ChatRequest struct {
	RequestID string
	StoreID   int64
	SessionID int64
	UserID    *int64
	Channel   string
	Message   string
}

type ChatResponse struct {
	SessionID       int64            `json:"session_id"`
	MessageID       int64            `json:"message_id"`
	Intent          string           `json:"intent"`
	Answer          string           `json:"answer"`
	Cards           []ChatCard       `json:"cards"`
	GuidanceChips   []GuidanceChip   `json:"guidance_chips"`
	HandoffRequired bool             `json:"handoff_required"`
	Meta            ChatResponseMeta `json:"meta,omitempty"`
}

type ChatResponseMeta struct {
	Route         string  `json:"route,omitempty"`
	Confidence    float64 `json:"confidence,omitempty"`
	RewriteQuery  string  `json:"rewrite_query,omitempty"`
	FallbackUsed  bool    `json:"fallback_used,omitempty"`
	EvidenceCount int     `json:"evidence_count,omitempty"`
}

type ChatCard struct {
	Type     string `json:"type"`
	SKUID    int64  `json:"sku_id,omitempty"`
	Name     string `json:"name,omitempty"`
	Location string `json:"location,omitempty"`
	Quantity int    `json:"quantity,omitempty"`
	Title    string `json:"title,omitempty"`
	Content  string `json:"content,omitempty"`
	Validity string `json:"validity,omitempty"`
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

type SessionListRequest struct {
	RequestID string
	StoreID   int64
	Limit     int
}

type ToolCallListRequest struct {
	RequestID string
	SessionID int64
	Limit     int
}

type Service interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	SearchFAQ(ctx context.Context, req FAQSearchRequest) ([]domain.FAQ, error)
	SearchProducts(ctx context.Context, req ProductSearchRequest) ([]domain.Product, error)
	GetProductLocation(ctx context.Context, req ProductLocationRequest) (*domain.ProductLocation, error)
	GetInventory(ctx context.Context, req InventoryRequest) (*domain.Inventory, error)
	ListActivePromotions(ctx context.Context, req PromotionListRequest) ([]domain.Promotion, error)
	ListSessions(ctx context.Context, req SessionListRequest) ([]domain.Session, error)
	ListToolCalls(ctx context.Context, req ToolCallListRequest) ([]domain.ToolCall, error)
	SaveFeedback(ctx context.Context, messageID, sessionID int64, feedbackValue int8) error
}

type service struct {
	repo         domain.Repository
	log          Logger
	orchestrator Orchestrator
}

func NewService(repo domain.Repository, log Logger) Service {
	return NewServiceWithOrchestrator(repo, log, nil)
}

func NewServiceWithOrchestrator(repo domain.Repository, log Logger, orchestrator Orchestrator) Service {
	if log == nil {
		log = nopLogger{}
	}
	if orchestrator == nil {
		orchestrator = newDefaultOrchestrator(repo, log)
	}
	return &service{repo: repo, log: log, orchestrator: orchestrator}
}

func UsesPrimaryOrchestrator(svc Service) bool {
	impl, ok := svc.(*service)
	if !ok || impl.orchestrator == nil {
		return false
	}
	orch, ok := impl.orchestrator.(*defaultOrchestrator)
	if !ok {
		return false
	}
	return orch.analyzer != nil && orch.composer != nil && orch.retriever != nil
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
	session, err := s.resolveSession(ctx, req)
	if err != nil {
		s.log.Error("app_chat_resolve_session_failed", "request_id", req.RequestID, "session_id", req.SessionID, "error", err)
		return nil, err
	}
	// 用户消息先不设 intent，orchestrator 执行后 intent 通过 ChatDecisionLog（message_id 关联）持久化
	msg, err := s.repo.CreateMessage(ctx, &domain.Message{SessionID: session.ID, Role: "user", Content: req.Message, Intent: ""})
	if err != nil {
		s.log.Error("app_chat_create_message_failed", "request_id", req.RequestID, "session_id", session.ID, "error", err)
		return nil, err
	}

	result, err := s.orchestrator.Run(ctx, OrchestratorRequest{
		RequestID: req.RequestID,
		StoreID:   req.StoreID,
		SessionID: session.ID,
		MessageID: msg.ID,
		UserID:    req.UserID,
		Channel:   req.Channel,
		Message:   req.Message,
	})
	if err != nil {
		s.log.Error("app_chat_orchestrator_failed", "request_id", req.RequestID, "session_id", session.ID, "error", err)
		return nil, err
	}

	assistantMsg, err := s.repo.CreateMessage(ctx, &domain.Message{SessionID: session.ID, Role: "assistant", Content: result.Answer, Intent: result.Decision.Intent})
	if err != nil {
		s.log.Error("app_chat_create_assistant_message_failed", "request_id", req.RequestID, "session_id", session.ID, "error", err)
		return nil, err
	}

	s.persistDecisionLog(ctx, session.ID, msg.ID, result.Decision)

	s.log.Info("app_chat_success", "request_id", req.RequestID, "session_id", session.ID, "message_id", assistantMsg.ID, "intent", result.Decision.Intent)
	return &ChatResponse{
		SessionID:       session.ID,
		MessageID:       assistantMsg.ID,
		Intent:          result.Decision.Intent,
		Answer:          result.Answer,
		Cards:           result.Cards,
		GuidanceChips:   result.GuidanceChips,
		HandoffRequired: result.Decision.NeedsHandoff,
		Meta: ChatResponseMeta{
			Route:         result.Decision.Route,
			Confidence:    result.Decision.Confidence,
			RewriteQuery:  result.Decision.RewrittenQuery,
			FallbackUsed:  result.Decision.FallbackUsed,
			EvidenceCount: len(result.Evidence),
		},
	}, nil
}

func (s *service) persistDecisionLog(ctx context.Context, sessionID, messageID int64, decision Decision) {
	repo, ok := s.repo.(domain.DecisionLogRepository)
	if !ok {
		return
	}
	_, err := repo.CreateDecisionLog(ctx, &domain.ChatDecisionLog{
		SessionID:       sessionID,
		MessageID:       messageID,
		Intent:          decision.Intent,
		Route:           decision.Route,
		RewriteQuery:    decision.RewrittenQuery,
		Confidence:      decision.Confidence,
		FallbackUsed:    decision.FallbackUsed,
		HandoffRequired: decision.NeedsHandoff,
	})
	if err != nil {
		s.log.Error("app_chat_decision_log_failed", "session_id", sessionID, "message_id", messageID, "error", err)
	}
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

func (s *service) ListSessions(ctx context.Context, req SessionListRequest) ([]domain.Session, error) {
	if req.StoreID <= 0 {
		return nil, domain.ErrInvalidArgument
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	return s.repo.ListSessions(ctx, req.StoreID, req.Limit)
}

func (s *service) ListToolCalls(ctx context.Context, req ToolCallListRequest) ([]domain.ToolCall, error) {
	if req.SessionID <= 0 {
		return nil, domain.ErrInvalidArgument
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	return s.repo.ListToolCalls(ctx, req.SessionID, req.Limit)
}

// SaveFeedback 保存用户对回答的反馈 (👍/👎)
func (s *service) SaveFeedback(ctx context.Context, messageID, sessionID int64, feedbackValue int8) error {
	if messageID <= 0 || sessionID <= 0 {
		return domain.ErrInvalidArgument
	}
	if feedbackValue != 0 && feedbackValue != 1 {
		return domain.ErrInvalidArgument
	}
	feedbackRepo, ok := s.repo.(domain.FeedbackRepository)
	if !ok {
		s.log.Error("app_feedback_repo_not_available")
		return domain.ErrInvalidArgument
	}
	_, err := feedbackRepo.CreateFeedback(ctx, &domain.Feedback{
		MessageID:     messageID,
		SessionID:     sessionID,
		FeedbackValue: feedbackValue,
	})
	if err != nil {
		s.log.Error("app_feedback_create_failed", "message_id", messageID, "session_id", sessionID, "error", err)
		return err
	}
	s.log.Info("app_feedback_saved", "message_id", messageID, "session_id", sessionID, "value", feedbackValue)
	return nil
}

func (s *service) resolveSession(ctx context.Context, req ChatRequest) (*domain.Session, error) {
	if req.SessionID > 0 {
		session, err := s.repo.GetSession(ctx, req.SessionID)
		if err != nil {
			return nil, err
		}
		if session.StoreID != req.StoreID {
			return nil, domain.ErrInvalidArgument
		}
		return session, nil
	}
	return s.repo.CreateSession(ctx, &domain.Session{StoreID: req.StoreID, UserID: req.UserID, Channel: req.Channel})
}
