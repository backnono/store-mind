package retrieval

import (
	"context"
	"testing"

	app "store-mind/application/customerqa"
	domain "store-mind/domain/customerqa"
)

type fakeKnowledgeRepo struct {
	items []domain.KnowledgeChunk
}

func (f fakeKnowledgeRepo) SearchKnowledge(_ context.Context, storeID int64, query string, knowledgeTypes []string, limit int) ([]domain.KnowledgeChunk, error) {
	if query == "missing" {
		return nil, nil
	}
	if limit > len(f.items) {
		limit = len(f.items)
	}
	return f.items[:limit], nil
}

func TestMySQLRetrieverRetrieveFAQ(t *testing.T) {
	retriever := NewMySQLRetriever(fakeKnowledgeRepo{
		items: []domain.KnowledgeChunk{{
			ID:            1,
			StoreID:       1,
			KnowledgeType: "faq",
			Title:         "支付方式",
			Content:       "支持微信和支付宝扫码支付",
		}},
	})

	items, err := retriever.Retrieve(context.Background(), app.RetrievalRequest{
		StoreID: 1,
		Query:   "付款",
		Intent:  "faq",
		Limit:   3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 || items[0].Kind != "faq" {
		t.Fatalf("unexpected retrieval result: %+v", items)
	}
}

func TestMySQLRetrieverRetrieveStorePolicy(t *testing.T) {
	retriever := NewMySQLRetriever(fakeKnowledgeRepo{
		items: []domain.KnowledgeChunk{{
			ID:            2,
			StoreID:       1,
			KnowledgeType: "store_policy",
			Title:         "退款规则",
			Content:       "如商品存在质量问题，可在订单详情提交退款申请。",
		}},
	})

	items, err := retriever.Retrieve(context.Background(), app.RetrievalRequest{
		StoreID: 1,
		Query:   "退款",
		Intent:  "faq",
		Limit:   3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 || items[0].Kind != "store_policy" {
		t.Fatalf("unexpected retrieval result: %+v", items)
	}
}

func TestMySQLRetrieverRetrieveEmpty(t *testing.T) {
	retriever := NewMySQLRetriever(fakeKnowledgeRepo{})

	items, err := retriever.Retrieve(context.Background(), app.RetrievalRequest{
		StoreID: 1,
		Query:   "missing",
		Intent:  "faq",
		Limit:   3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty retrieval result, got %+v", items)
	}
}

func TestMySQLRetrieverRetrieveRankingPreserved(t *testing.T) {
	retriever := NewMySQLRetriever(fakeKnowledgeRepo{
		items: []domain.KnowledgeChunk{
			{ID: 3, StoreID: 1, KnowledgeType: "faq", Title: "支付方式", Content: "支持微信和支付宝扫码支付"},
			{ID: 4, StoreID: 1, KnowledgeType: "faq", Title: "营业时间", Content: "每天 08:00-23:00"},
		},
	})

	items, err := retriever.Retrieve(context.Background(), app.RetrievalRequest{
		StoreID: 1,
		Query:   "支付",
		Intent:  "faq",
		Limit:   2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 || items[0].RecordID != 3 {
		t.Fatalf("expected ranking preserved, got %+v", items)
	}
}
