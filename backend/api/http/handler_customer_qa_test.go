package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	app "store-mind/application/customerqa"
	domain "store-mind/domain/customerqa"

	"go.uber.org/zap"
)

type fakeService struct{}

func (f fakeService) Chat(_ context.Context, req app.ChatRequest) (*app.ChatResponse, error) {
	return &app.ChatResponse{SessionID: 1, MessageID: 2, Intent: "product_location", Answer: "可口可乐在饮料区 B-02 货架"}, nil
}

func (f fakeService) SearchFAQ(_ context.Context, req app.FAQSearchRequest) ([]domain.FAQ, error) {
	return []domain.FAQ{{ID: 1, StoreID: req.StoreID, Question: req.Query, Answer: "A", Category: "payment"}}, nil
}

func (f fakeService) SearchProducts(_ context.Context, req app.ProductSearchRequest) ([]domain.Product, error) {
	return []domain.Product{{ID: 101, Name: "可口可乐", Category: "饮料", Aliases: []string{"可乐"}}}, nil
}

func (f fakeService) GetProductLocation(_ context.Context, req app.ProductLocationRequest) (*domain.ProductLocation, error) {
	return &domain.ProductLocation{ProductID: req.ProductID, ZoneName: "饮料区", ShelfCode: "B-02", LayerNo: 2, PositionDesc: "进门后左手边"}, nil
}

func (f fakeService) GetInventory(_ context.Context, req app.InventoryRequest) (*domain.Inventory, error) {
	return &domain.Inventory{StoreID: req.StoreID, SKUID: req.SKUID, Quantity: 12, SafetyStock: 3}, nil
}

func (f fakeService) ListActivePromotions(_ context.Context, req app.PromotionListRequest) ([]domain.Promotion, error) {
	return []domain.Promotion{{ID: 1, StoreID: req.StoreID, Title: "饮料第二件半价", Status: "active", StartAt: time.Now(), EndAt: time.Now().Add(time.Hour)}}, nil
}

func TestCustomerQAChat(t *testing.T) {
	r := NewRouter(zap.NewNop(), NewCustomerQAHandler(fakeService{}, zap.NewNop()))
	body, _ := json.Marshal(map[string]any{"store_id": 1, "channel": "miniapp", "message": "可乐在哪里"})
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
	if resp["intent"] != "product_location" {
		t.Fatalf("expected intent in response, got %+v", resp)
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

func TestCustomerQASearchProducts(t *testing.T) {
	r := NewRouter(zap.NewNop(), NewCustomerQAHandler(fakeService{}, zap.NewNop()))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customer-qa/products/search?store_id=1&q=可乐", nil)
	req.Header.Set("X-Request-Id", "rid-product")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	items, ok := resp["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one item, got %+v", resp["items"])
	}
}

func TestCustomerQAGetProductLocation(t *testing.T) {
	r := NewRouter(zap.NewNop(), NewCustomerQAHandler(fakeService{}, zap.NewNop()))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customer-qa/products/101/location?store_id=1", nil)
	req.Header.Set("X-Request-Id", "rid-location")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["zone_name"] != "饮料区" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestCustomerQAGetInventory(t *testing.T) {
	r := NewRouter(zap.NewNop(), NewCustomerQAHandler(fakeService{}, zap.NewNop()))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customer-qa/skus/1001/inventory?store_id=1", nil)
	req.Header.Set("X-Request-Id", "rid-inventory")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["quantity"] != float64(12) {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestCustomerQAListActivePromotions(t *testing.T) {
	r := NewRouter(zap.NewNop(), NewCustomerQAHandler(fakeService{}, zap.NewNop()))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customer-qa/promotions/active?store_id=1", nil)
	req.Header.Set("X-Request-Id", "rid-promotion")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	items, ok := resp["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected one item, got %+v", resp["items"])
	}
}
