package mysql

import (
	"testing"

	domain "store-mind/domain/customerqa"
)

func TestRankKnowledgeChunks(t *testing.T) {
	items := []domain.KnowledgeChunk{
		{ID: 1, KnowledgeType: "faq", Title: "支付方式", Content: "支持微信和支付宝扫码支付"},
		{ID: 2, KnowledgeType: "store_policy", Title: "退款规则", Content: "如商品存在质量问题，可提交退款申请"},
	}

	ranked := rankKnowledgeChunks("支付", items)
	if len(ranked) != 2 {
		t.Fatalf("expected 2 chunks, got %+v", ranked)
	}
	if ranked[0].ID != 1 {
		t.Fatalf("expected payment chunk first, got %+v", ranked)
	}
}
