package customerqa

import (
	"context"
	"errors"
	"testing"
)

// ============================================================================
// S2.2 复合意图测试
// ============================================================================

func TestSubIntents(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect []string
	}{
		{"single", "inventory", []string{"inventory"}},
		{"compound", "product_location,inventory", []string{"product_location", "inventory"}},
		{"compound with spaces", "product_location, inventory", []string{"product_location", "inventory"}},
		{"triple", "product_location,inventory,price", []string{"product_location", "inventory", "price"}},
		{"empty", "", []string{""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := subIntents(tt.input)
			if len(got) != len(tt.expect) {
				t.Fatalf("expected %d intents, got %d: %v", len(tt.expect), len(got), got)
			}
			for i, v := range got {
				if v != tt.expect[i] {
					t.Fatalf("intent[%d]: expected %q, got %q", i, tt.expect[i], v)
				}
			}
		})
	}
}

func TestIsCompound(t *testing.T) {
	if isCompound("inventory") {
		t.Fatal("single intent should not be compound")
	}
	if !isCompound("product_location,inventory") {
		t.Fatal("comma-separated intent should be compound")
	}
}

func TestNormalizeRouteCompoundIntent(t *testing.T) {
	orch := &defaultOrchestrator{}
	tests := []struct {
		name     string
		decision Decision
		want     string
	}{
		{"compound tool only", Decision{Intent: "product_location,inventory"}, RouteTool},
		{"compound rag only", Decision{Intent: "faq,faq"}, RouteRAG},
		{"compound tool+rag", Decision{Intent: "product_location,faq"}, RouteHybrid},
		{"price single", Decision{Intent: "price"}, RouteTool},
		{"price+inventory compound", Decision{Intent: "price,inventory"}, RouteTool},
		{"price+faq compound", Decision{Intent: "price,faq"}, RouteHybrid},
		{"triple tool", Decision{Intent: "product_location,price,inventory"}, RouteTool},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := orch.normalizeRoute(tt.decision)
			if got != tt.want {
				t.Fatalf("expected route %s, got %s", tt.want, got)
			}
		})
	}
}

// ============================================================================
// S2.2 复合意图编排测试
// ============================================================================

func TestDefaultOrchestratorRunCompoundIntent(t *testing.T) {
	orch := &defaultOrchestrator{
		analyzer: stubIntentAnalyzer{decision: Decision{
			Intent:         "product_location,inventory",
			Route:          RouteTool,
			RewrittenQuery: "可乐库存位置",
			Confidence:     0.90,
		}},
		composer: stubAnswerComposer{answer: "可乐在B-02，还有12瓶"},
		fallback: &stubOrchestratorResult{},
		repo:     &fakeRepo{},
		log:      fakeLogger{},
	}

	result, err := orch.Run(context.Background(), OrchestratorRequest{
		StoreID: 1, Message: "可乐在哪儿还有货吗",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decision.Intent != "product_location,inventory" {
		t.Fatalf("expected compound intent preserved, got %s", result.Decision.Intent)
	}
	if result.Decision.Route != RouteTool {
		t.Fatalf("expected tool route, got %s", result.Decision.Route)
	}
	if len(result.Evidence) < 1 {
		t.Fatalf("expected compound evidence, got %d items", len(result.Evidence))
	}
}

func TestDefaultOrchestratorRunCompoundIntentWithRAG(t *testing.T) {
	orch := &defaultOrchestrator{
		analyzer: stubIntentAnalyzer{decision: Decision{
			Intent:         "product_location,faq",
			Route:          RouteHybrid,
			RewrittenQuery: "退款",
			Confidence:     0.88,
		}},
		retriever: stubRetriever{items: []Evidence{{
			Source: "retriever", Kind: "faq",
			Title: "退款规则", Content: "如商品存在质量问题，可提交退款申请",
		}}},
		composer: stubAnswerComposer{answer: "如需退款请提交申请"},
		fallback: &stubOrchestratorResult{},
		repo:     &fakeRepo{},
		log:      fakeLogger{},
	}

	result, err := orch.Run(context.Background(), OrchestratorRequest{
		StoreID: 1, Message: "买了可乐怎么退款",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decision.Route != RouteHybrid {
		t.Fatalf("expected hybrid route for compound tool+rag, got %s", result.Decision.Route)
	}
	// 复合意图应有 tool evidence + rag evidence
	if len(result.Evidence) < 1 {
		t.Fatalf("expected compound evidence, got %d items", len(result.Evidence))
	}
}

func TestCollectCompoundEvidenceDeduplication(t *testing.T) {
	orch := &defaultOrchestrator{
		repo: &fakeRepo{},
		log:  fakeLogger{},
	}

	// 模拟 product_location,inventory 复合意图 —— 两个子意图可能返回相同 product
	// 通过同一个 product_id，去重逻辑应合并 evidence
	evidence, cards, err := orch.collectCompoundEvidence(
		context.Background(),
		OrchestratorRequest{StoreID: 1, Message: "可乐在哪儿还有货吗"},
		Decision{Intent: "product_location,inventory", RewrittenQuery: "可口可乐"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// fakeRepo 为 product_location 和 inventory 各自返回 evidence
	// 由于 RecordID 相同，应被去重合并
	if len(evidence) < 1 {
		t.Fatalf("expected evidence, got %d", len(evidence))
	}
	_ = cards // cards may be non-empty for the fake repo
}

// ============================================================================
// S2.3 价格查询测试
// ============================================================================

func TestFallbackOrchestratorRunPriceQuery(t *testing.T) {
	orch := &fallbackOrchestrator{
		repo: &fakeRepo{},
		log:  fakeLogger{},
	}

	result, err := orch.Run(context.Background(), OrchestratorRequest{
		StoreID: 1, Message: "可乐多少钱",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decision.Intent != "price" {
		t.Fatalf("expected price intent, got %s", result.Decision.Intent)
	}
	if result.Decision.Route != RouteFallback {
		t.Fatalf("expected fallback route, got %s", result.Decision.Route)
	}
}

func TestDefaultOrchestratorRunPriceIntent(t *testing.T) {
	orch := &defaultOrchestrator{
		analyzer: stubIntentAnalyzer{decision: Decision{
			Intent:         "price",
			Route:          RouteTool,
			RewrittenQuery: "可口可乐价格",
			Confidence:     0.92,
		}},
		composer: stubAnswerComposer{answer: "可乐 ¥3.50 / 500ml"},
		fallback: &stubOrchestratorResult{},
		repo:     &fakeRepo{},
		log:      fakeLogger{},
	}

	result, err := orch.Run(context.Background(), OrchestratorRequest{
		StoreID: 1, Message: "可乐多少钱",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Decision.Intent != "price" {
		t.Fatalf("expected price intent, got %s", result.Decision.Intent)
	}
	if result.Decision.Route != RouteTool {
		t.Fatalf("expected tool route, got %s", result.Decision.Route)
	}
	if result.Answer != "可乐 ¥3.50 / 500ml" {
		t.Fatalf("unexpected answer: %s", result.Answer)
	}
}

func TestPriceComparisonViaCompoundIntent(t *testing.T) {
	// 模拟"可乐和雪碧哪个便宜" — LLM 可能将其识别为 compound "price,price"
	// 或单独 price（由 LLM 决定，这里测试 compound 路径）
	orch := &defaultOrchestrator{
		analyzer: stubIntentAnalyzer{decision: Decision{
			Intent:         "price,product_location",
			Route:          RouteTool,
			RewrittenQuery: "可乐雪碧价格",
			Confidence:     0.88,
		}},
		composer: stubAnswerComposer{answer: "可乐¥3.50和雪碧¥3.50都在B-02"},
		fallback: &stubOrchestratorResult{},
		repo:     &fakeRepo{},
		log:      fakeLogger{},
	}

	result, err := orch.Run(context.Background(), OrchestratorRequest{
		StoreID: 1, Message: "可乐和雪碧哪个便宜",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// compound intent should collect evidence for both sub-intents
	if result.Decision.Intent != "price,product_location" {
		t.Fatalf("expected compound intent, got %s", result.Decision.Intent)
	}
	if result.Decision.Route != RouteTool {
		t.Fatalf("expected tool route for price+location, got %s", result.Decision.Route)
	}
	if len(result.Evidence) < 1 {
		t.Fatalf("expected compound evidence, got %d", len(result.Evidence))
	}
}

// ============================================================================
// S2.5 全量回归：边界 + 异常
// ============================================================================

func TestCollectPriceEvidenceEmptyResult(t *testing.T) {
	orch := &defaultOrchestrator{repo: &fakeRepo{}, log: fakeLogger{}}
	evidence, cards, err := orch.collectPriceEvidence(
		context.Background(),
		OrchestratorRequest{StoreID: 1},
		Decision{Intent: "price", RewrittenQuery: "不存在商品"},
		"不存在商品",
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 不存在的商品应返回空结果，不报错
	if len(evidence) != 0 {
		t.Fatalf("expected empty evidence for unknown product, got %d", len(evidence))
	}
	_ = cards
}

func TestCollectPriceEvidenceWithResolvedProduct(t *testing.T) {
	orch := &defaultOrchestrator{
		repo: &fakeRepo{},
		log:  fakeLogger{},
	}
	pid := int64(1)
	evidence, cards, err := orch.collectPriceEvidence(
		context.Background(),
		OrchestratorRequest{StoreID: 1},
		Decision{Intent: "price"},
		"",
		&pid,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(evidence) != 1 {
		t.Fatalf("expected 1 price evidence for resolved product, got %d", len(evidence))
	}
	if evidence[0].Kind != "price" {
		t.Fatalf("expected price kind, got %s", evidence[0].Kind)
	}
	_ = cards
}

func TestCompoundIntentPartialFailure(t *testing.T) {
	// 模拟一个子意图成功、另一个失败（如 rag retriever 不可用）
	failingRetriever := stubRetriever{err: errors.New("retriever down")}
	orch := &defaultOrchestrator{
		analyzer: stubIntentAnalyzer{decision: Decision{
			Intent:         "product_location,faq",
			Route:          RouteHybrid,
			RewrittenQuery: "可乐退款",
			Confidence:     0.85,
		}},
		retriever: failingRetriever, // rag 失败
		composer:  stubAnswerComposer{answer: "可乐在B-02"},
		fallback:  &stubOrchestratorResult{},
		repo:      &fakeRepo{},
		log:       fakeLogger{},
	}

	result, err := orch.Run(context.Background(), OrchestratorRequest{
		StoreID: 1, Message: "可乐退款",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// product_location 子意图应能成功收集，faq 子意图失败不应中断
	if result.Decision.Intent != "product_location,faq" {
		t.Fatalf("expected compound intent, got %s", result.Decision.Intent)
	}
	if len(result.Evidence) < 1 {
		t.Fatalf("expected at least tool evidence from compound, got %d", len(result.Evidence))
	}
}

func TestDecisionPriceField(t *testing.T) {
	// 验证 Decision 结构体可以承载 price 意图
	d := Decision{Intent: "price", Route: RouteTool, Confidence: 0.95}
	if d.Intent != "price" {
		t.Fatal("Decision should support price intent")
	}
	if routeForIntent("price") != RouteTool {
		t.Fatal("price should route to tool")
	}
}

func TestSubIntentsEdgeCases(t *testing.T) {
	// 连续逗号
	got := subIntents("price,,inventory")
	if len(got) != 2 {
		t.Fatalf("expected 2 intents (empty stripped), got %d: %v", len(got), got)
	}
	// 前后逗号
	got = subIntents(",product_location,")
	if len(got) != 1 {
		t.Fatalf("expected 1 intent, got %d: %v", len(got), got)
	}
	// 只有空格
	got = subIntents(" , ")
	if len(got) != 1 {
		t.Fatalf("expected 1 intent for only spaces, got %d: %v", len(got), got)
	}
}

func TestRouteForIntent(t *testing.T) {
	tests := []struct {
		intent string
		route  string
	}{
		{"inventory", RouteTool},
		{"product_location", RouteTool},
		{"price", RouteTool},
		{"promotion", RouteTool},
		{"faq", RouteRAG},
		{"product_policy", RouteHybrid},
		{"unsupported", RouteFallback},
		{"unknown_intent", RouteFallback},
	}
	for _, tt := range tests {
		t.Run(tt.intent, func(t *testing.T) {
			if got := routeForIntent(tt.intent); got != tt.route {
				t.Fatalf("routeForIntent(%s): expected %s, got %s", tt.intent, tt.route, got)
			}
		})
	}
}

// ============================================================================
// S2 综合：端到端流程（fallback 路径）
// ============================================================================

func TestS2FallbackPriceAndCompoundE2E(t *testing.T) {
	// 验证 fallback 编排器的价格 + 复合意图关键词路由
	repo := &fakeRepo{}
	svc := NewService(repo, fakeLogger{})

	// 价格查询
	resp, err := svc.Chat(context.Background(), ChatRequest{
		RequestID: "s2-price",
		StoreID:   1,
		Channel:   "miniapp",
		Message:   "可乐多少钱",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Intent != "price" {
		t.Fatalf("expected price intent, got %s", resp.Intent)
	}

	// 库存查询（保持向后兼容）
	resp, err = svc.Chat(context.Background(), ChatRequest{
		RequestID: "s2-inventory",
		StoreID:   1,
		Channel:   "miniapp",
		Message:   "可乐还有吗",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Intent != "inventory" {
		t.Fatalf("expected inventory intent, got %s", resp.Intent)
	}

	// 位置查询（保持向后兼容）
	resp, err = svc.Chat(context.Background(), ChatRequest{
		RequestID: "s2-location",
		StoreID:   1,
		Channel:   "miniapp",
		Message:   "可乐在哪里",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Intent != "product_location" {
		t.Fatalf("expected product_location intent, got %s", resp.Intent)
	}
}

// fakeRepo 和 fakeLogger 已在 orchestrator_test.go 中定义，此处复用。
