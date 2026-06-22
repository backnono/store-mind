package customerqa

import (
	"testing"

	domain "store-mind/domain/customerqa"
)

func TestGuideEngine_LocationGuidance(t *testing.T) {
	engine := NewGuideEngine(nopLogger{})
	chips := engine.Evaluate(GuideContext{
		Intent:       "product_location",
		SessionState: StateProductFocus,
		Products:     []domain.Product{{ID: 101, Name: "可乐"}},
	})
	if len(chips) < 2 {
		t.Errorf("location guidance: expected >= 2 chips, got %d", len(chips))
	}
	// check chips are actionable
	for _, c := range chips {
		if c.Text == "" || c.Prompt == "" {
			t.Errorf("guidance chip has empty text/prompt: %+v", c)
		}
	}
}

func TestGuideEngine_NoProducts_NoChips(t *testing.T) {
	engine := NewGuideEngine(nopLogger{})
	chips := engine.Evaluate(GuideContext{
		Intent:       "product_location",
		SessionState: StateProductFocus,
		Products:     nil,
	})
	// product_location requires len(Products) > 0 → no chips without products
	if len(chips) > 0 {
		t.Errorf("expected 0 chips without products list, got %d", len(chips))
	}
}

func TestGuideEngine_InventoryEmptyEvidence(t *testing.T) {
	engine := NewGuideEngine(nopLogger{})
	chips := engine.Evaluate(GuideContext{
		Intent:       "inventory",
		SessionState: StateProductFocus,
		Evidence:     nil, // no evidence → out of stock
	})
	if len(chips) < 2 {
		t.Errorf("out of stock guidance: expected >= 2 chips, got %d", len(chips))
	}
}

func TestGuideEngine_ListBrowseRefine(t *testing.T) {
	engine := NewGuideEngine(nopLogger{})
	products := make([]domain.Product, 6)
	for i := range products {
		products[i] = domain.Product{ID: int64(i + 1), Name: "test"}
	}
	chips := engine.Evaluate(GuideContext{
		Intent:       "product_location",
		SessionState: StateListBrowse,
		Products:     products,
	})
	if len(chips) < 2 {
		t.Errorf("list refine guidance: expected >= 2 chips, got %d", len(chips))
	}
}

func TestGuideEngine_TransactionGuidance(t *testing.T) {
	engine := NewGuideEngine(nopLogger{})
	chips := engine.Evaluate(GuideContext{
		Intent:       "faq",
		SessionState: StateTransaction,
		Message:      "怎么退款？",
	})
	if len(chips) < 2 {
		t.Errorf("transaction guidance: expected >= 2 chips, got %d", len(chips))
	}
}

func TestGuideEngine_NonTransactionFaq_NoChips(t *testing.T) {
	engine := NewGuideEngine(nopLogger{})
	chips := engine.Evaluate(GuideContext{
		Intent:       "faq",
		SessionState: StateListBrowse,
		Message:      "营业时间是什么？",
	})
	// Not a transaction-related message → no chips
	if len(chips) > 0 {
		t.Errorf("non-transaction faq: expected 0 chips, got %d", len(chips))
	}
}

func TestIsTransactionRelated(t *testing.T) {
	tests := []struct {
		msg  string
		want bool
	}{
		{"怎么退款？", true},
		{"我要退款", true},
		{"支付失败怎么办", true},
		{"付款方式", true},
		{"结算", true},
		{"退货流程", true},
		{"退款流程是什么？", true},
		{"怎么退", true},
		{"营业时间是什么？", false},
		{"超市几点关门", false},
		{"可乐多少钱", false},
	}
	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			got := isTransactionRelated(tt.msg)
			if got != tt.want {
				t.Errorf("isTransactionRelated(%q) = %v, want %v", tt.msg, got, tt.want)
			}
		})
	}
}

var _ GuideEngine = NewGuideEngine(nil)
