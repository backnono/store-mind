package customerqa

import (
	"context"
	"errors"
	"testing"

	domain "store-mind/domain/customerqa"
)

func TestDefaultOrchestratorNormalizeRoute(t *testing.T) {
	tests := []struct {
		name     string
		decision Decision
		want     string
	}{
		{
			name:     "inventory defaults to tool",
			decision: Decision{Intent: "inventory", Route: "rag"},
			want:     RouteTool,
		},
		{
			name:     "faq defaults to rag",
			decision: Decision{Intent: "faq", Route: "tool"},
			want:     RouteRAG,
		},
		{
			name:     "mixed question uses hybrid",
			decision: Decision{Intent: "product_policy", Route: "hybrid"},
			want:     RouteHybrid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orch := &defaultOrchestrator{}
			got := orch.normalizeRoute(tt.decision)
			if got != tt.want {
				t.Fatalf("expected route %s, got %s", tt.want, got)
			}
		})
	}
}

func TestDefaultOrchestratorFallbackWhenAnalyzerFails(t *testing.T) {
	fallback := &stubOrchestratorResult{
		result: OrchestratorResult{
			Decision: Decision{Intent: "faq", Route: RouteFallback, FallbackUsed: true},
			Answer:   "fallback answer",
		},
	}
	orch := &defaultOrchestrator{
		analyzer: stubIntentAnalyzer{err: errors.New("timeout")},
		fallback: fallback,
		repo:     &fakeRepo{},
		log:      fakeLogger{},
	}

	result, err := orch.Run(context.Background(), OrchestratorRequest{StoreID: 1, Message: "怎么付款"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decision.Route != RouteFallback {
		t.Fatalf("expected fallback route, got %+v", result.Decision)
	}
	if !result.Decision.FallbackUsed {
		t.Fatalf("expected fallback used, got %+v", result.Decision)
	}
	if fallback.calls != 1 {
		t.Fatalf("expected fallback to be called once, got %d", fallback.calls)
	}
}

func TestFallbackOrchestratorRunUsesRuleRouting(t *testing.T) {
	orch := &fallbackOrchestrator{
		repo: &fakeRepo{},
		log:  fakeLogger{},
	}

	result, err := orch.Run(context.Background(), OrchestratorRequest{StoreID: 1, Message: "可乐在哪里"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decision.Intent != "product_location" {
		t.Fatalf("expected product_location intent, got %+v", result.Decision)
	}
	if result.Decision.Route != RouteFallback {
		t.Fatalf("expected fallback route, got %+v", result.Decision)
	}
	if !result.Decision.FallbackUsed {
		t.Fatalf("expected fallback used, got %+v", result.Decision)
	}
	if len(result.Cards) != 1 || result.Cards[0].Type != "product" {
		t.Fatalf("expected product card, got %+v", result.Cards)
	}
}

func TestFallbackOrchestratorRunUnsupportedQuestion(t *testing.T) {
	orch := &fallbackOrchestrator{
		repo: &fakeRepo{},
		log:  fakeLogger{},
	}

	result, err := orch.Run(context.Background(), OrchestratorRequest{StoreID: 1, Message: "帮我写论文"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decision.Intent != "unsupported" {
		t.Fatalf("expected unsupported intent, got %+v", result.Decision)
	}
	if result.Decision.Route != RouteFallback || !result.Decision.FallbackUsed {
		t.Fatalf("expected fallback metadata, got %+v", result.Decision)
	}
	if result.Answer == "" {
		t.Fatalf("expected safe default answer, got %+v", result)
	}
}

func TestServiceChatUsesFallbackOrchestratorByDefault(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo, fakeLogger{})

	resp, err := svc.Chat(context.Background(), ChatRequest{RequestID: "r-default", StoreID: 1, Channel: "miniapp", Message: "可乐在哪里"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Intent != "product_location" {
		t.Fatalf("expected product_location intent, got %+v", resp)
	}
	if len(resp.Cards) != 1 || resp.Cards[0].Type != "product" {
		t.Fatalf("expected product card, got %+v", resp.Cards)
	}
}

func TestOrchestratorResultSupportsKnowledgeEvidence(t *testing.T) {
	chunk := domain.KnowledgeChunk{
		ID:            11,
		StoreID:       1,
		KnowledgeType: "faq",
		Title:         "支付方式",
		Content:       "支持微信和支付宝扫码支付",
	}

	result := OrchestratorResult{
		Decision: Decision{Intent: "faq", Route: RouteRAG},
		Evidence: []Evidence{{
			Source:   "retriever",
			Kind:     chunk.KnowledgeType,
			RecordID: chunk.ID,
			Title:    chunk.Title,
			Content:  chunk.Content,
		}},
	}

	if len(result.Evidence) != 1 {
		t.Fatalf("expected one evidence item, got %+v", result.Evidence)
	}
	if result.Evidence[0].Kind != "faq" {
		t.Fatalf("expected faq evidence kind, got %+v", result.Evidence[0])
	}
}

func TestDefaultOrchestratorRunToolRouteWithComposer(t *testing.T) {
	orch := &defaultOrchestrator{
		analyzer: stubIntentAnalyzer{decision: Decision{
			Intent:         "inventory",
			Route:          RouteTool,
			RewrittenQuery: "可口可乐库存",
			Confidence:     0.92,
		}},
		composer: stubAnswerComposer{answer: "系统显示可口可乐还有 12 件"},
		fallback: &stubOrchestratorResult{},
		repo:     &fakeRepo{},
		log:      fakeLogger{},
	}

	result, err := orch.Run(context.Background(), OrchestratorRequest{StoreID: 1, Message: "可乐还有吗"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decision.Route != RouteTool {
		t.Fatalf("expected tool route, got %+v", result.Decision)
	}
	if result.Answer != "系统显示可口可乐还有 12 件" {
		t.Fatalf("unexpected answer: %+v", result)
	}
	if len(result.Evidence) == 0 {
		t.Fatalf("expected tool evidence, got %+v", result)
	}
}

func TestDefaultOrchestratorRunRAGRouteWithRetriever(t *testing.T) {
	orch := &defaultOrchestrator{
		analyzer: stubIntentAnalyzer{decision: Decision{
			Intent:         "faq",
			Route:          RouteRAG,
			RewrittenQuery: "支付方式",
			Confidence:     0.91,
		}},
		retriever: stubRetriever{items: []Evidence{{
			Source:   "retriever",
			Kind:     "faq",
			RecordID: 1,
			Title:    "支付方式",
			Content:  "支持微信和支付宝扫码支付",
		}}},
		composer: stubAnswerComposer{answer: "支持微信和支付宝扫码支付"},
		fallback: &stubOrchestratorResult{},
		repo:     &fakeRepo{},
		log:      fakeLogger{},
	}

	result, err := orch.Run(context.Background(), OrchestratorRequest{StoreID: 1, Message: "怎么付款"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decision.Route != RouteRAG {
		t.Fatalf("expected rag route, got %+v", result.Decision)
	}
	if len(result.Evidence) != 1 || result.Evidence[0].Kind != "faq" {
		t.Fatalf("unexpected evidence: %+v", result.Evidence)
	}
}

func TestDefaultOrchestratorRunHybridRoute(t *testing.T) {
	orch := &defaultOrchestrator{
		analyzer: stubIntentAnalyzer{decision: Decision{
			Intent:         "product_policy",
			Route:          RouteHybrid,
			RewrittenQuery: "可乐退款规则",
			Confidence:     0.89,
		}},
		retriever: stubRetriever{items: []Evidence{{
			Source:   "retriever",
			Kind:     "store_policy",
			RecordID: 2,
			Title:    "退款规则",
			Content:  "如商品存在质量问题，可提交退款申请",
		}}},
		composer: stubAnswerComposer{answer: "可乐有货，如商品存在质量问题，可提交退款申请"},
		fallback: &stubOrchestratorResult{},
		repo:     &fakeRepo{},
		log:      fakeLogger{},
	}

	result, err := orch.Run(context.Background(), OrchestratorRequest{StoreID: 1, Message: "可乐有货吗，能退款吗"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decision.Route != RouteHybrid {
		t.Fatalf("expected hybrid route, got %+v", result.Decision)
	}
	if len(result.Evidence) < 2 {
		t.Fatalf("expected hybrid evidence, got %+v", result.Evidence)
	}
}

func TestDefaultOrchestratorRunFallsBackOnComposerFailure(t *testing.T) {
	fallback := &stubOrchestratorResult{
		result: OrchestratorResult{
			Decision: Decision{Intent: "faq", Route: RouteFallback, FallbackUsed: true},
			Answer:   "fallback answer",
		},
	}
	orch := &defaultOrchestrator{
		analyzer: stubIntentAnalyzer{decision: Decision{
			Intent:         "faq",
			Route:          RouteRAG,
			RewrittenQuery: "支付方式",
			Confidence:     0.91,
		}},
		retriever: stubRetriever{items: []Evidence{{Kind: "faq", Title: "支付方式", Content: "支持微信和支付宝扫码支付"}}},
		composer:  stubAnswerComposer{err: errors.New("llm down")},
		fallback:  fallback,
		repo:      &fakeRepo{},
		log:       fakeLogger{},
	}

	result, err := orch.Run(context.Background(), OrchestratorRequest{StoreID: 1, Message: "怎么付款"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decision.Route != RouteFallback || !result.Decision.FallbackUsed {
		t.Fatalf("expected fallback result, got %+v", result.Decision)
	}
}

type stubIntentAnalyzer struct {
	decision Decision
	err      error
}

func (s stubIntentAnalyzer) AnalyzeIntent(context.Context, IntentRequest) (Decision, error) {
	return s.decision, s.err
}

type stubAnswerComposer struct {
	answer string
	err    error
}

func (s stubAnswerComposer) ComposeAnswer(context.Context, AnswerRequest) (string, error) {
	return s.answer, s.err
}

type stubRetriever struct {
	items []Evidence
	err   error
}

func (s stubRetriever) Retrieve(context.Context, RetrievalRequest) ([]Evidence, error) {
	return s.items, s.err
}

type stubOrchestratorResult struct {
	result OrchestratorResult
	err    error
	calls  int
}

func (s *stubOrchestratorResult) Run(context.Context, OrchestratorRequest) (OrchestratorResult, error) {
	s.calls++
	return s.result, s.err
}
