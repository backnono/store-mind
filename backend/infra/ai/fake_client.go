package ai

import (
	"context"
	"strings"

	app "store-mind/application/customerqa"
)

type FakeIntentAnalyzer struct {
	Decisions map[string]app.Decision
}

func (f FakeIntentAnalyzer) AnalyzeIntent(_ context.Context, req app.IntentRequest) (app.Decision, error) {
	if decision, ok := f.Decisions[req.Message]; ok {
		return decision, nil
	}
	message := strings.TrimSpace(req.Message)
	switch {
	case strings.Contains(message, "库存") || strings.Contains(message, "还有吗"):
		return app.Decision{Intent: "inventory", Route: app.RouteTool, RewrittenQuery: message, Confidence: 0.9}, nil
	case strings.Contains(message, "退款") && (strings.Contains(message, "可乐") || strings.Contains(message, "商品")):
		return app.Decision{Intent: "product_policy", Route: app.RouteHybrid, RewrittenQuery: message, Confidence: 0.88}, nil
	default:
		return app.Decision{Intent: "faq", Route: app.RouteRAG, RewrittenQuery: message, Confidence: 0.86}, nil
	}
}

type FakeAnswerComposer struct{}

func (FakeAnswerComposer) ComposeAnswer(_ context.Context, req app.AnswerRequest) (*app.AnswerResult, error) {
	answer := "暂时没有足够证据回答这个问题。"
	if len(req.Evidence) > 0 {
		answer = req.Evidence[0].Content
	}
	return &app.AnswerResult{
		Answer:        answer,
		GuidanceChips: nil,
	}, nil
}
