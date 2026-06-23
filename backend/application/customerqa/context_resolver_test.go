package customerqa

import (
	"context"
	"testing"

	domain "store-mind/domain/customerqa"
)

// ── 关键场景：省略追问 L1 ─────────────────────────

func TestResolve_OmissionFollowup_L1(t *testing.T) {
	// state=product_focus, focus=[101], message="还有几瓶？"
	// 预期：L1 命中，NeedsClarify=false，ResolvedEntities 包含 product_id=101
	resolver := NewContextResolver(nil, nopLogger{})

	focus := &domain.FocusEntityIDs{ProductIDs: []int64{101}}
	req := ResolveRequest{
		Message:       "还有几瓶？",
		SessionState:  StateProductFocus,
		FocusEntities: focus,
		ContextStack:  nil,
	}

	result, err := resolver.Resolve(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Layer != "L1" {
		t.Errorf("expected L1, got %s", result.Layer)
	}
	if result.NeedsClarify {
		t.Errorf("expected NeedsClarify=false")
	}
	if len(result.ResolvedEntities) == 0 {
		t.Fatal("expected ResolvedEntities to be non-empty")
	}
	found := false
	for _, e := range result.ResolvedEntities {
		if e.Type == "product" && e.ProductID != nil && *e.ProductID == 101 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected resolved product_id=101, got %+v", result.ResolvedEntities)
	}
}

func TestResolve_PriceFollowup_L1(t *testing.T) {
	// state=product_focus, focus=[101], message="多少钱？"
	// 预期：L1 命中
	resolver := NewContextResolver(nil, nopLogger{})

	focus := &domain.FocusEntityIDs{ProductIDs: []int64{101}}
	req := ResolveRequest{
		Message:       "多少钱？",
		SessionState:  StateProductFocus,
		FocusEntities: focus,
	}

	result, err := resolver.Resolve(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Layer != "L1" {
		t.Errorf("expected L1, got %s", result.Layer)
	}
	if result.NeedsClarify {
		t.Errorf("expected NeedsClarify=false")
	}
}

func TestResolve_ExplicitAnaphora_L1(t *testing.T) {
	// state=product_focus, focus=[101], message="那个多少钱？"
	// 预期：L1 命中（显式指代）
	resolver := NewContextResolver(nil, nopLogger{})

	focus := &domain.FocusEntityIDs{ProductIDs: []int64{101}}
	req := ResolveRequest{
		Message:       "那个多少钱？",
		SessionState:  StateProductFocus,
		FocusEntities: focus,
	}

	result, err := resolver.Resolve(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Layer != "L1" {
		t.Errorf("expected L1, got %s", result.Layer)
	}
}

func TestResolve_IdleState_L3(t *testing.T) {
	// state=idle, message="还有吗？" → 预期 L3 澄清
	resolver := NewContextResolver(nil, nopLogger{})

	req := ResolveRequest{
		Message:      "还有吗？",
		SessionState: StateIdle,
	}

	result, err := resolver.Resolve(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Layer != "L3" {
		t.Errorf("expected L3, got %s", result.Layer)
	}
	if !result.NeedsClarify {
		t.Errorf("expected NeedsClarify=true when idle")
	}
}

func TestResolve_NoFocus_L3(t *testing.T) {
	// state=product_focus 但没有 focus entities → L3
	resolver := NewContextResolver(nil, nopLogger{})

	req := ResolveRequest{
		Message:       "还有几瓶？",
		SessionState:  StateProductFocus,
		FocusEntities: nil,
	}

	result, err := resolver.Resolve(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Without focus entities + no llmClient → L3
	if result.Layer != "L3" {
		t.Errorf("expected L3 when no focus, got %s", result.Layer)
	}
}

// ── L2 low confidence → L3 ─────────────────────────

func TestResolve_L2_LowConfidence_FallsBackTo_L3(t *testing.T) {
	// L2 返回 confidence=0.59 → 应该进入 L3
	lowConfClient := &stubAnaphoraClient{confidence: 0.59, entities: []domain.ResolvedEntity{
		{Type: "product", ProductID: p64(101)},
	}}
	resolver := NewContextResolver(lowConfClient, nopLogger{})

	focus := &domain.FocusEntityIDs{ProductIDs: []int64{101}}
	req := ResolveRequest{
		Message:       "那雪碧呢？",
		SessionState:  StateProductFocus,
		FocusEntities: focus,
		ContextStack: []domain.ContextStackItem{
			{Turn: 1, Intent: "product_location", SystemSummary: "可乐在 B-02"},
		},
	}

	result, err := resolver.Resolve(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Layer != "L3" {
		t.Errorf("expected L3 for low confidence, got %s", result.Layer)
	}
}

func TestResolve_L2_LowConfidence_ExplicitNewQueryContinuesToOrchestrator(t *testing.T) {
	lowConfClient := &stubAnaphoraClient{confidence: 0.3, entities: nil}
	resolver := NewContextResolver(lowConfClient, nopLogger{})

	focus := &domain.FocusEntityIDs{ProductIDs: []int64{101}}
	req := ResolveRequest{
		Message:       "薯片在哪里？",
		SessionState:  StateProductFocus,
		FocusEntities: focus,
		ContextStack: []domain.ContextStackItem{
			{Turn: 1, Intent: "product_location", SystemSummary: "可乐在 B-02"},
		},
	}

	result, err := resolver.Resolve(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.NeedsClarify {
		t.Fatalf("expected explicit new product query to continue, got clarify: %+v", result)
	}
	if result.Layer != "L2" {
		t.Fatalf("expected low-confidence L2 passthrough, got %s", result.Layer)
	}
}

func TestResolve_L2_HighConfidence(t *testing.T) {
	// L2 返回 confidence=0.85 → L2 命中
	highConfClient := &stubAnaphoraClient{confidence: 0.85, entities: []domain.ResolvedEntity{
		{Type: "product", Name: "雪碧", ProductID: p64(102)},
	}}
	resolver := NewContextResolver(highConfClient, nopLogger{})

	focus := &domain.FocusEntityIDs{ProductIDs: []int64{101}}
	req := ResolveRequest{
		Message:       "那雪碧呢？",
		SessionState:  StateProductFocus,
		FocusEntities: focus,
		ContextStack: []domain.ContextStackItem{
			{Turn: 1, Intent: "product_location", SystemSummary: "可乐在 B-02"},
		},
	}

	result, err := resolver.Resolve(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Layer != "L2" {
		t.Errorf("expected L2, got %s", result.Layer)
	}
	if len(result.ResolvedEntities) == 0 {
		t.Errorf("expected resolved entities from L2")
	}
}

// ── L1 不触发：长消息包含新实体 → 走 L2 ──────────

func TestResolve_LongMessage_Not_L1(t *testing.T) {
	// 长消息（>15 字）且包含追问词 → 但消息中有潜在新实体 → 交给 L2
	resolver := NewContextResolver(nil, nopLogger{})

	focus := &domain.FocusEntityIDs{ProductIDs: []int64{101}}
	req := ResolveRequest{
		Message:       "可乐的价格是多少钱，我想对比一下雪碧",
		SessionState:  StateProductFocus,
		FocusEntities: focus,
	}

	result, err := resolver.Resolve(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// L1 不触发（长消息 + 可能新实体）→ 无 llmClient → L3
	if result.Layer != "L3" {
		t.Errorf("expected L3 for long message without llmClient, got %s", result.Layer)
	}
}

// ── Helpers ──

type stubAnaphoraClient struct {
	confidence float64
	entities   []domain.ResolvedEntity
}

func (s *stubAnaphoraClient) ResolveAnaphora(_ context.Context, _ string, _ []domain.ContextStackItem, _ *domain.FocusEntityIDs) (*AnaphoraLLMResult, error) {
	return &AnaphoraLLMResult{
		ResolvedEntities: s.entities,
		Confidence:       s.confidence,
	}, nil
}

var _ AnaphoraClient = (*stubAnaphoraClient)(nil)
var _ ContextResolver = NewContextResolver(nil, nil)
