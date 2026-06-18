package ai

import (
	"context"
	"testing"

	app "store-mind/application/customerqa"
)

func TestFakeAnalyzerAnalyzeIntent(t *testing.T) {
	analyzer := FakeIntentAnalyzer{
		Decisions: map[string]app.Decision{
			"可乐库存": {Intent: "inventory", Route: app.RouteTool, RewrittenQuery: "可口可乐库存", Confidence: 0.93},
		},
	}

	decision, err := analyzer.AnalyzeIntent(context.Background(), app.IntentRequest{Message: "可乐库存"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decision.Intent != "inventory" || decision.RewrittenQuery != "可口可乐库存" {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestFakeComposerComposeAnswer(t *testing.T) {
	composer := FakeAnswerComposer{}
	result, err := composer.ComposeAnswer(context.Background(), app.AnswerRequest{
		Message: "怎么付款",
		Evidence: []app.Evidence{{
			Kind:    "faq",
			Title:   "支付方式",
			Content: "支持微信和支付宝扫码支付",
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || result.Answer == "" {
		t.Fatalf("expected non-empty answer")
	}
}
