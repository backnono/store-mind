package customerqa

import (
	"context"
	"testing"

	domain "store-mind/domain/customerqa"
)

type fakeRepo struct {
	sid int64
	mid int64
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
	return m, nil
}

func (f *fakeRepo) SearchFAQ(_ context.Context, storeID int64, query string, limit int) ([]domain.FAQ, error) {
	return []domain.FAQ{{ID: 1, StoreID: storeID, Question: query, Answer: "A", Category: "payment"}}, nil
}

func TestServiceChat(t *testing.T) {
	svc := NewService(&fakeRepo{}, fakeLogger{})
	resp, err := svc.Chat(context.Background(), ChatRequest{RequestID: "r1", StoreID: 1, Channel: "miniapp", Message: "商品在哪里"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SessionID == 0 || resp.MessageID == 0 {
		t.Fatalf("expected persisted ids, got %+v", resp)
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
