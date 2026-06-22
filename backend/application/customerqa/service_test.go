package customerqa

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	domain "store-mind/domain/customerqa"
)

type fakeRepo struct {
	sid       int64
	mid       int64
	toolCalls []domain.ToolCall
	messages  []domain.Message
	sessions  []domain.Session
}

type fakeLogger struct{}

func (fakeLogger) Info(string, ...any)  {}
func (fakeLogger) Warn(string, ...any)  {}
func (fakeLogger) Error(string, ...any) {}

func (f *fakeRepo) CreateSession(_ context.Context, s *domain.Session) (*domain.Session, error) {
	f.sid++
	s.ID = f.sid
	f.sessions = append(f.sessions, *s)
	return s, nil
}

func (f *fakeRepo) GetSession(_ context.Context, sessionID int64) (*domain.Session, error) {
	for _, session := range f.sessions {
		if session.ID == sessionID {
			copy := session
			return &copy, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeRepo) CreateMessage(_ context.Context, m *domain.Message) (*domain.Message, error) {
	f.mid++
	m.ID = f.mid
	f.messages = append(f.messages, *m)
	return m, nil
}

func (f *fakeRepo) CreateToolCall(_ context.Context, tc *domain.ToolCall) (*domain.ToolCall, error) {
	tc.ID = int64(len(f.toolCalls) + 1)
	f.toolCalls = append(f.toolCalls, *tc)
	return tc, nil
}

func (f *fakeRepo) ListSessions(_ context.Context, storeID int64, limit int) ([]domain.Session, error) {
	return f.sessions, nil
}

func (f *fakeRepo) ListToolCalls(_ context.Context, sessionID int64, limit int) ([]domain.ToolCall, error) {
	return f.toolCalls, nil
}

func (f *fakeRepo) SearchFAQ(_ context.Context, storeID int64, query string, limit int) ([]domain.FAQ, error) {
	return []domain.FAQ{{ID: 1, StoreID: storeID, Question: query, Answer: "你可以使用小程序扫码结算，支持微信和支付宝。", Category: "payment"}}, nil
}

func (f *fakeRepo) SearchProducts(_ context.Context, storeID int64, query string, limit int) ([]domain.Product, error) {
	if strings.Contains(query, "可乐") || strings.Contains(query, "可口") {
		return []domain.Product{{ID: 101, Name: "可口可乐", Brand: "可口可乐", Category: "饮料", Aliases: []string{"可乐"}}}, nil
	}
	return nil, nil
}

func (f *fakeRepo) ListProductsByLocation(_ context.Context, storeID int64, zoneID, shelfID *int64, limit int) ([]domain.Product, error) {
	return nil, nil
}

func (f *fakeRepo) GetProductLocation(_ context.Context, storeID, productID int64) (*domain.ProductLocation, error) {
	if productID != 101 && productID != 1 {
		return nil, domain.ErrNotFound
	}
	skuID := int64(1001)
	return &domain.ProductLocation{ProductID: productID, SKUID: &skuID, ZoneName: "饮料区", ShelfCode: "B-02", LayerNo: 2, PositionDesc: "进门后左手边"}, nil
}

func (f *fakeRepo) GetInventory(_ context.Context, storeID, skuID int64) (*domain.Inventory, error) {
	if skuID != 1001 {
		return nil, domain.ErrNotFound
	}
	return &domain.Inventory{
		StoreID: storeID, SKUID: skuID, ProductID: 101,
		ProductName: "可口可乐", Spec: "500ml", Price: 3.50,
		Quantity: 12, SafetyStock: 3,
	}, nil
}

func (f *fakeRepo) ListActivePromotions(_ context.Context, storeID int64, now time.Time, limit int) ([]domain.Promotion, error) {
	return []domain.Promotion{{ID: 1, StoreID: storeID, Title: "饮料第二件半价", Status: "active", StartAt: now.Add(-time.Hour), EndAt: now.Add(time.Hour)}}, nil
}

func (f *fakeRepo) ListRecentMessages(_ context.Context, sessionID int64, limit int) ([]domain.Message, error) {
	return f.messages, nil
}

func TestServiceChat(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeLogger{})
	resp, err := svc.Chat(context.Background(), ChatRequest{RequestID: "r1", StoreID: 1, Channel: "miniapp", Message: "可乐在哪里"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SessionID == 0 || resp.MessageID == 0 {
		t.Fatalf("expected persisted ids, got %+v", resp)
	}
	if resp.Answer == "" {
		t.Fatalf("expected non-empty answer, got %+v", resp)
	}
	if resp.Meta.Route != RouteFallback {
		t.Fatalf("expected fallback route metadata, got %+v", resp.Meta)
	}
	if !resp.Meta.FallbackUsed {
		t.Fatalf("expected fallback_used true, got %+v", resp.Meta)
	}
	if resp.Meta.Confidence != 0.95 {
		t.Fatalf("expected confidence 0.95, got %+v", resp.Meta)
	}
	if resp.Intent != "product_location" {
		t.Fatalf("expected product_location intent, got %s", resp.Intent)
	}
	if len(resp.Cards) != 1 || resp.Cards[0].Type != "product" {
		t.Fatalf("expected product card, got %+v", resp.Cards)
	}
	if resp.HandoffRequired {
		t.Fatalf("expected handoff_required false")
	}
	if !strings.Contains(resp.Answer, "饮料区") {
		t.Fatalf("unexpected answer: %s", resp.Answer)
	}
	if len(repo.toolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(repo.toolCalls))
	}
	if len(repo.messages) != 2 || repo.messages[1].Role != "assistant" {
		t.Fatalf("expected assistant message to be persisted, got %+v", repo.messages)
	}
}

func TestServiceChatFallbackBehaviorCoverage(t *testing.T) {
	tests := []struct {
		name           string
		repo           domain.Repository
		message        string
		wantIntent     string
		wantAnswerPart string
		wantCards      int
		wantHandoff    bool
	}{
		{
			name:           "inventory fallback",
			repo:           &fakeRepoWithToolFailure{},
			message:        "可乐还有吗",
			wantIntent:     "inventory",
			wantAnswerPart: "暂时无法查询库存信息",
		},
		{
			name:           "promotion fallback",
			repo:           &fakeRepoWithPromotionFailure{},
			message:        "今天有什么优惠",
			wantIntent:     "promotion",
			wantAnswerPart: "暂时无法查询活动信息",
		},
		{
			name:           "faq fallback",
			repo:           &fakeRepoWithFAQFailure{},
			message:        "怎么付款",
			wantIntent:     "faq",
			wantAnswerPart: "暂时无法查询门店规则",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo, fakeLogger{})
			resp, err := svc.Chat(context.Background(), ChatRequest{RequestID: "r-fallback", StoreID: 1, Channel: "miniapp", Message: tt.message})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.Intent != tt.wantIntent {
				t.Fatalf("expected intent %s, got %s", tt.wantIntent, resp.Intent)
			}
			if !strings.Contains(resp.Answer, tt.wantAnswerPart) {
				t.Fatalf("expected answer containing %q, got %q", tt.wantAnswerPart, resp.Answer)
			}
			if len(resp.Cards) != tt.wantCards {
				t.Fatalf("expected %d cards, got %+v", tt.wantCards, resp.Cards)
			}
			if resp.HandoffRequired != tt.wantHandoff {
				t.Fatalf("expected handoff %v, got %v", tt.wantHandoff, resp.HandoffRequired)
			}
			if resp.SessionID == 0 || resp.MessageID == 0 {
				t.Fatalf("expected persisted ids on fallback, got %+v", resp)
			}
		})
	}
}

func TestServiceChatPersistsDecisionLogWhenSupported(t *testing.T) {
	repo := &fakeRepoWithDecisionLog{}
	svc := NewService(repo, fakeLogger{})

	resp, err := svc.Chat(context.Background(), ChatRequest{RequestID: "r-log", StoreID: 1, Channel: "miniapp", Message: "怎么付款"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Intent != "faq" {
		t.Fatalf("expected faq intent, got %+v", resp)
	}
	if len(repo.decisionLogs) != 1 {
		t.Fatalf("expected one decision log, got %+v", repo.decisionLogs)
	}
	if repo.decisionLogs[0].Intent != "faq" || repo.decisionLogs[0].Route != RouteFallback {
		t.Fatalf("unexpected decision log: %+v", repo.decisionLogs[0])
	}
}

func TestServiceChatReuseSession(t *testing.T) {
	repo := &fakeRepo{
		sid:      9,
		sessions: []domain.Session{{ID: 7, StoreID: 1, Channel: "miniapp"}},
	}
	svc := NewService(repo, fakeLogger{})
	resp, err := svc.Chat(context.Background(), ChatRequest{RequestID: "r2", StoreID: 1, SessionID: 7, Channel: "miniapp", Message: "怎么付款"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SessionID != 7 {
		t.Fatalf("expected reused session 7, got %d", resp.SessionID)
	}
	if len(repo.sessions) != 1 {
		t.Fatalf("expected no new session, got %+v", repo.sessions)
	}
}

func TestServiceChatInventory(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeLogger{})
	resp, err := svc.Chat(context.Background(), ChatRequest{RequestID: "r1", StoreID: 1, Channel: "miniapp", Message: "可乐还有吗"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Intent != "inventory" {
		t.Fatalf("expected inventory intent, got %s", resp.Intent)
	}
	if len(resp.Cards) != 1 || resp.Cards[0].Type != "inventory" {
		t.Fatalf("expected inventory card, got %+v", resp.Cards)
	}
	if !strings.Contains(resp.Answer, "系统显示") {
		t.Fatalf("unexpected answer: %s", resp.Answer)
	}
	if len(repo.toolCalls) != 3 {
		t.Fatalf("expected 3 tool calls, got %d", len(repo.toolCalls))
	}
}

func TestServiceChatPromotion(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeLogger{})
	resp, err := svc.Chat(context.Background(), ChatRequest{RequestID: "r1", StoreID: 1, Channel: "miniapp", Message: "今天有什么优惠"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Intent != "promotion" {
		t.Fatalf("expected promotion intent, got %s", resp.Intent)
	}
	if len(resp.Cards) != 1 || resp.Cards[0].Type != "promotion" {
		t.Fatalf("expected promotion card, got %+v", resp.Cards)
	}
	if !strings.Contains(resp.Answer, "饮料第二件半价") {
		t.Fatalf("unexpected answer: %s", resp.Answer)
	}
	if len(repo.toolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(repo.toolCalls))
	}
}

func TestServiceChatFAQ(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeLogger{})
	resp, err := svc.Chat(context.Background(), ChatRequest{RequestID: "r1", StoreID: 1, Channel: "miniapp", Message: "怎么付款"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Intent != "faq" {
		t.Fatalf("expected faq intent, got %s", resp.Intent)
	}
	if len(resp.Cards) != 1 || resp.Cards[0].Type != "faq" {
		t.Fatalf("expected faq card, got %+v", resp.Cards)
	}
	if !strings.Contains(resp.Answer, "微信") && !strings.Contains(resp.Answer, "支付宝") {
		t.Fatalf("unexpected answer: %s", resp.Answer)
	}
	if len(repo.toolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(repo.toolCalls))
	}
}

func TestServiceChatUnsupported(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeLogger{})
	resp, err := svc.Chat(context.Background(), ChatRequest{RequestID: "r1", StoreID: 1, Channel: "miniapp", Message: "帮我写论文"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Intent != "unsupported" {
		t.Fatalf("expected unsupported intent, got %s", resp.Intent)
	}
	if len(resp.Cards) != 0 {
		t.Fatalf("expected no cards, got %+v", resp.Cards)
	}
	if !strings.Contains(resp.Answer, "商品") {
		t.Fatalf("unexpected answer: %s", resp.Answer)
	}
	if len(repo.toolCalls) != 0 {
		t.Fatalf("expected 0 tool calls, got %d", len(repo.toolCalls))
	}
}

func TestServiceChatToolFailureFallback(t *testing.T) {
	repo := &fakeRepoWithToolFailure{}
	svc := NewService(repo, fakeLogger{})
	resp, err := svc.Chat(context.Background(), ChatRequest{RequestID: "r1", StoreID: 1, Channel: "miniapp", Message: "可乐在哪里"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Intent != "product_location" {
		t.Fatalf("expected product_location intent, got %s", resp.Intent)
	}
	if len(resp.Cards) != 0 {
		t.Fatalf("expected no cards on fallback, got %+v", resp.Cards)
	}
	if !strings.Contains(resp.Answer, "暂时") {
		t.Fatalf("unexpected answer: %s", resp.Answer)
	}
}

func TestServiceChatHandoff(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeLogger{})
	resp, err := svc.Chat(context.Background(), ChatRequest{RequestID: "r1", StoreID: 1, Channel: "miniapp", Message: "我要找人工"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Intent != "handoff" {
		t.Fatalf("expected handoff intent, got %s", resp.Intent)
	}
	if !resp.HandoffRequired {
		t.Fatalf("expected handoff_required true")
	}
	if len(resp.Cards) != 0 {
		t.Fatalf("expected no cards, got %+v", resp.Cards)
	}
}

func TestServiceSearchFAQ(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeLogger{})
	resp, err := svc.SearchFAQ(context.Background(), FAQSearchRequest{RequestID: "r1", StoreID: 1, Query: "付款"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp) != 1 {
		t.Fatalf("expected one faq, got %d", len(resp))
	}
}

func TestServiceSearchProducts(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeLogger{})
	resp, err := svc.SearchProducts(context.Background(), ProductSearchRequest{RequestID: "r1", StoreID: 1, Query: "可乐"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp) != 1 || resp[0].Name != "可口可乐" {
		t.Fatalf("unexpected products: %+v", resp)
	}
}

func TestServiceGetProductLocation(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeLogger{})
	resp, err := svc.GetProductLocation(context.Background(), ProductLocationRequest{RequestID: "r1", StoreID: 1, ProductID: 101})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ZoneName != "饮料区" || resp.ShelfCode != "B-02" {
		t.Fatalf("unexpected location: %+v", resp)
	}
}

func TestServiceGetInventory(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeLogger{})
	resp, err := svc.GetInventory(context.Background(), InventoryRequest{RequestID: "r1", StoreID: 1, SKUID: 1001})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Quantity != 12 {
		t.Fatalf("unexpected inventory: %+v", resp)
	}
}

func TestServiceListActivePromotions(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeLogger{})
	resp, err := svc.ListActivePromotions(context.Background(), PromotionListRequest{RequestID: "r1", StoreID: 1, Limit: 5, Now: time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp) != 1 || resp[0].Title == "" {
		t.Fatalf("unexpected promotions: %+v", resp)
	}
}

func TestServiceListSessions(t *testing.T) {
	repo := &fakeRepo{sessions: []domain.Session{{ID: 7, StoreID: 1, Channel: "miniapp"}}}
	svc := NewService(repo, fakeLogger{})
	resp, err := svc.ListSessions(context.Background(), SessionListRequest{RequestID: "r1", StoreID: 1, Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp) != 1 || resp[0].ID != 7 {
		t.Fatalf("unexpected sessions: %+v", resp)
	}
}

func TestServiceListToolCalls(t *testing.T) {
	repo := &fakeRepo{toolCalls: []domain.ToolCall{{ID: 1, SessionID: 7, ToolName: "search_faq", Success: true}}}
	svc := NewService(repo, fakeLogger{})
	resp, err := svc.ListToolCalls(context.Background(), ToolCallListRequest{RequestID: "r1", SessionID: 7, Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp) != 1 || resp[0].ToolName != "search_faq" {
		t.Fatalf("unexpected tool calls: %+v", resp)
	}
}

type fakeRepoWithToolFailure struct {
	fakeRepo
}

func (f *fakeRepoWithToolFailure) GetProductLocation(_ context.Context, storeID, productID int64) (*domain.ProductLocation, error) {
	return nil, errors.New("db timeout")
}

type fakeRepoWithPromotionFailure struct {
	fakeRepo
}

func (f *fakeRepoWithPromotionFailure) ListActivePromotions(_ context.Context, storeID int64, now time.Time, limit int) ([]domain.Promotion, error) {
	return nil, errors.New("db timeout")
}

type fakeRepoWithFAQFailure struct {
	fakeRepo
}

func (f *fakeRepoWithFAQFailure) SearchFAQ(_ context.Context, storeID int64, query string, limit int) ([]domain.FAQ, error) {
	return nil, errors.New("db timeout")
}

type fakeRepoWithDecisionLog struct {
	fakeRepo
	decisionLogs []domain.ChatDecisionLog
}

func (f *fakeRepoWithDecisionLog) CreateDecisionLog(_ context.Context, log *domain.ChatDecisionLog) (*domain.ChatDecisionLog, error) {
	log.ID = int64(len(f.decisionLogs) + 1)
	f.decisionLogs = append(f.decisionLogs, *log)
	return log, nil
}
