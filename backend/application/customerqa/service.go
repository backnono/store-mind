package customerqa

import (
	"context"
	"strings"

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

type Service interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	SearchFAQ(ctx context.Context, req FAQSearchRequest) ([]domain.FAQ, error)
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

	s.log.Info("app_chat_start", "request_id", req.RequestID, "store_id", req.StoreID, "channel", req.Channel)
	session, err := s.repo.CreateSession(ctx, &domain.Session{StoreID: req.StoreID, UserID: req.UserID, Channel: req.Channel})
	if err != nil {
		s.log.Error("app_chat_create_session_failed", "request_id", req.RequestID, "error", err)
		return nil, err
	}
	msg, err := s.repo.CreateMessage(ctx, &domain.Message{SessionID: session.ID, Role: "user", Content: req.Message, Intent: "customer_qa"})
	if err != nil {
		s.log.Error("app_chat_create_message_failed", "request_id", req.RequestID, "session_id", session.ID, "error", err)
		return nil, err
	}

	s.log.Info("app_chat_success", "request_id", req.RequestID, "session_id", session.ID, "message_id", msg.ID)
	return &ChatResponse{SessionID: session.ID, MessageID: msg.ID, Intent: "customer_qa", Answer: "已收到你的问题，我们会基于门店数据为你解答。"}, nil
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
