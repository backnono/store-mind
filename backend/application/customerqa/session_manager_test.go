package customerqa

import (
	"testing"
	"time"

	domain "store-mind/domain/customerqa"
)

func TestStateTransition(t *testing.T) {
	tests := []struct {
		name    string
		current SessionState
		intent  string
		want    SessionState
	}{
		{"idle→product_focus via product_location", StateIdle, "product_location", StateProductFocus},
		{"idle→product_focus via inventory", StateIdle, "inventory", StateProductFocus},
		{"idle→product_focus via product_policy", StateIdle, "product_policy", StateProductFocus},
		{"product_focus stays on inventory", StateProductFocus, "inventory", StateProductFocus},
		{"product_focus stays on product_location", StateProductFocus, "product_location", StateProductFocus},
		{"idle→list_browse via promotion", StateIdle, "promotion", StateListBrowse},
		{"idle→list_browse via faq", StateIdle, "faq", StateListBrowse},
		{"product_focus→list_browse stays if promotion", StateProductFocus, "promotion", StateProductFocus},
		{"idle→transaction via checkout", StateIdle, "checkout", StateTransaction},
		{"idle→transaction via refund", StateIdle, "refund", StateTransaction},
		{"any→handoff", StateProductFocus, "handoff", StateHandoff},
		{"handoff stays on unsupported", StateHandoff, "unsupported", StateHandoff},
		{"unsupported→idle", StateProductFocus, "unsupported", StateIdle},
		{"list_browse stays on faq", StateListBrowse, "faq", StateListBrowse},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StateTransition(tt.current, tt.intent, "")
			if got != tt.want {
				t.Errorf("StateTransition(%s, %s) = %s, want %s", tt.current, tt.intent, got, tt.want)
			}
		})
	}
}

func TestAppendContextStack(t *testing.T) {
	stack := []domain.ContextStackItem{}
	for i := 1; i <= 6; i++ {
		item := domain.ContextStackItem{Turn: i, Intent: "test", SystemSummary: "round"}
		stack = AppendContextStack(stack, item, 5)
	}
	if len(stack) != 5 {
		t.Errorf("after 6 appends, expected 5 items, got %d", len(stack))
	}
	if stack[0].Turn != 2 || stack[4].Turn != 6 {
		t.Errorf("expected turns 2-6, got %d-%d", stack[0].Turn, stack[4].Turn)
	}
}

func TestMarshalContextStack(t *testing.T) {
	stack := []domain.ContextStackItem{
		{Turn: 1, Intent: "product_location", SystemSummary: "found product"},
	}
	json, err := MarshalContextStack(stack)
	if err != nil {
		t.Fatalf("MarshalContextStack failed: %v", err)
	}
	if json == "" || json == "[]" {
		t.Errorf("expected non-empty JSON, got %s", json)
	}

	// empty stack
	empty, err := MarshalContextStack(nil)
	if err != nil || empty != "[]" {
		t.Errorf("expected '[]' for nil stack, got %s (err=%v)", empty, err)
	}
}

func TestMarshalFocusEntityIDs(t *testing.T) {
	focus := &domain.FocusEntityIDs{ProductIDs: []int64{101, 102}}
	json, err := MarshalFocusEntityIDs(focus)
	if err != nil {
		t.Fatalf("MarshalFocusEntityIDs failed: %v", err)
	}
	if json == "" || json == "null" {
		t.Errorf("expected non-null JSON, got %s", json)
	}

	// nil focus
	nullJSON, err := MarshalFocusEntityIDs(nil)
	if err != nil || nullJSON != "null" {
		t.Errorf("expected 'null' for nil focus, got %s (err=%v)", nullJSON, err)
	}
}

func TestBuildTurnSummary(t *testing.T) {
	stack := []domain.ContextStackItem{{Turn: 1}}
	summary := BuildTurnSummary(stack, "inventory", []domain.ResolvedEntity{
		{Type: "product", Name: "可乐", ProductID: p64(101)},
	}, "collect_evidence", "查到了可乐库存 12 瓶")
	if summary.Turn != 2 {
		t.Errorf("expected turn 2, got %d", summary.Turn)
	}
	if summary.Intent != "inventory" {
		t.Errorf("expected inventory intent, got %s", summary.Intent)
	}
	if len(summary.ResolvedEntities) != 1 {
		t.Errorf("expected 1 resolved entity, got %d", len(summary.ResolvedEntities))
	}
}

func TestDecayCheck(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		elapsed time.Duration
		want    DecayAction
	}{
		{"10s ago → none", 10 * time.Second, DecayNone},
		{"29s ago → none", 29 * time.Second, DecayNone},
		{"30s ago → wait", 30 * time.Second, DecayWait},
		{"80s ago → wait", 80 * time.Second, DecayWait},
		{"90s ago → light_summary", 90 * time.Second, DecayLightSummary},
		{"4min ago → light_summary", 4 * time.Minute, DecayLightSummary},
		{"5min ago → suspend", 5 * time.Minute, DecaySuspend},
		{"20min ago → suspend", 20 * time.Minute, DecaySuspend},
		{"30min ago → confirm_resume", 30 * time.Minute, DecayConfirmResume},
		{"1h ago → confirm_resume", time.Hour, DecayConfirmResume},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lastActive := now.Add(-tt.elapsed)
			mgr := &defaultSessionManager{repo: nil, log: nopLogger{}}
			got := mgr.decayCheck(lastActive)
			if got != tt.want {
				t.Errorf("decayCheck(%v ago) = %s, want %s", tt.elapsed, got, tt.want)
			}
		})
	}
}

// ── Helpers ──

func p64(v int64) *int64 { return &v }

// ensure interface satisfaction
var _ SessionManager = (*defaultSessionManager)(nil)
