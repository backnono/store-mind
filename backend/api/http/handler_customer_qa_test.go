package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	app "store-mind/application/customerqa"
	domain "store-mind/domain/customerqa"

	"go.uber.org/zap"
)

type fakeService struct{}

func (f fakeService) Chat(_ context.Context, req app.ChatRequest) (*app.ChatResponse, error) {
	return &app.ChatResponse{SessionID: 1, MessageID: 2, Intent: "customer_qa", Answer: "ok"}, nil
}

func (f fakeService) SearchFAQ(_ context.Context, req app.FAQSearchRequest) ([]domain.FAQ, error) {
	return []domain.FAQ{{ID: 1, StoreID: req.StoreID, Question: req.Query, Answer: "A", Category: "payment"}}, nil
}

func TestCustomerQAChat(t *testing.T) {
	r := NewRouter(zap.NewNop(), NewCustomerQAHandler(fakeService{}, zap.NewNop()))
	body, _ := json.Marshal(map[string]any{"store_id": 1, "channel": "miniapp", "message": "hi"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/customer-qa/chat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-Id", "rid-chat")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	meta, ok := resp["meta"].(map[string]any)
	if !ok {
		t.Fatalf("meta not found in response")
	}
	if meta["request_id"] != "rid-chat" {
		t.Fatalf("expected request_id rid-chat, got %v", meta["request_id"])
	}
}

func TestCustomerQASearchFAQ(t *testing.T) {
	r := NewRouter(zap.NewNop(), NewCustomerQAHandler(fakeService{}, zap.NewNop()))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customer-qa/faqs/search?store_id=1&q=付款", nil)
	req.Header.Set("X-Request-Id", "rid-faq")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	meta, ok := resp["meta"].(map[string]any)
	if !ok {
		t.Fatalf("meta not found in response")
	}
	if meta["request_id"] != "rid-faq" {
		t.Fatalf("expected request_id rid-faq, got %v", meta["request_id"])
	}
}
