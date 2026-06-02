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
}

type fakeLogger struct{}

func (fakeLogger) Info(string, ...any)  {}
func (fakeLogger) Warn(string, ...any)  {}
func (fakeLogger) Error(string, ...any) {}

func (f *fakeRepo) CreateSession(_ context.Context, s *domain.Session) (*domain.Session, error) {
	f.sid++
	s.ID = f.sid
	return s, nil
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

func (f *fakeRepo) SearchFAQ(_ context.Context, storeID int64, query string, limit int) ([]domain.FAQ, error) {
	return []domain.FAQ{{ID: 1, StoreID: storeID, Question: query, Answer: "你可以使用小程序扫码结算，支持微信和支付宝。", Category: "payment"}}, nil
}

func (f *fakeRepo) SearchProducts(_ context.Context, storeID int64, query string, limit int) ([]domain.Product, error) {
	if strings.Contains(query, "可乐") {
		return []domain.Product{{ID: 101, Name: "可口可乐", Brand: "可口可乐", Category: "饮料", Aliases: []string{"可乐"}}}, nil
	}
	return nil, nil
}

func (f *fakeRepo) GetProductLocation(_ context.Context, storeID, productID int64) (*domain.ProductLocation, error) {
	if productID != 101 {
		return nil, domain.ErrNotFound
	}
	skuID := int64(1001)
	return &domain.ProductLocation{ProductID: productID, SKUID: &skuID, ZoneName: "饮料区", ShelfCode: "B-02", LayerNo: 2, PositionDesc: "进门后左手边"}, nil
}

func (f *fakeRepo) GetInventory(_ context.Context, storeID, skuID int64) (*domain.Inventory, error) {
	if skuID != 1001 {
		return nil, domain.ErrNotFound
	}
	return &domain.Inventory{StoreID: storeID, SKUID: skuID, Quantity: 12, SafetyStock: 3}, nil
}

func (f *fakeRepo) ListActivePromotions(_ context.Context, storeID int64, now time.Time, limit int) ([]domain.Promotion, error) {
	return []domain.Promotion{{ID: 1, StoreID: storeID, Title: "饮料第二件半价", Status: "active", StartAt: now.Add(-time.Hour), EndAt: now.Add(time.Hour)}}, nil
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
	if resp.Intent != "product_location" {
		t.Fatalf("expected product_location intent, got %s", resp.Intent)
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
	if !strings.Contains(resp.Answer, "暂时") {
		t.Fatalf("unexpected answer: %s", resp.Answer)
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

type fakeRepoWithToolFailure struct {
	fakeRepo
}

func (f *fakeRepoWithToolFailure) GetProductLocation(_ context.Context, storeID, productID int64) (*domain.ProductLocation, error) {
	return nil, errors.New("db timeout")
}
