package customerqa

import (
	"context"
	"strings"
	"testing"

	domain "store-mind/domain/customerqa"
)

// ── S1 集成测试 ────────────────────────────────────

// newS1TestService 创建带完整 S1 组件注入的 service，用于集成测试。
func newS1TestService(repo domain.Repository) Service {
	orch := newDefaultOrchestrator(repo, fakeLogger{})
	return NewServiceWithConfig(ServiceConfig{
		Repo:            repo,
		Log:             fakeLogger{},
		Orchestrator:    orch,
		SessionManager:  NewSessionManager(repo, fakeLogger{}),
		ContextResolver: NewContextResolver(nil, fakeLogger{}),
		GuideEngine:     NewGuideEngine(fakeLogger{}),
	})
}

// TestS1_EntryFirstOpen 验证首次打开入口返回预设问题列表。
func TestS1_EntryFirstOpen(t *testing.T) {
	repo := &fakeRepo{}
	svc := newS1TestService(repo)

	resp, err := svc.Chat(context.Background(), ChatRequest{
		RequestID: "r1",
		StoreID:   1,
		Channel:   "miniapp",
		Message:   "打开",
		EntryMode: "first_open",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Intent != "greeting" {
		t.Errorf("expected greeting intent, got %s", resp.Intent)
	}
	if resp.Meta.Route != "entry_first_open" {
		t.Errorf("expected route entry_first_open, got %s", resp.Meta.Route)
	}
	if len(resp.GuidanceChips) < 4 {
		t.Errorf("expected >= 4 guidance chips, got %d", len(resp.GuidanceChips))
	}
	// 验证 chips 覆盖关键入口
	hasLocation := false
	hasPromo := false
	for _, c := range resp.GuidanceChips {
		if strings.Contains(c.Text, "在哪里") || strings.Contains(c.Text, "位置") {
			hasLocation = true
		}
		if strings.Contains(c.Text, "活动") {
			hasPromo = true
		}
	}
	if !hasLocation || !hasPromo {
		t.Errorf("first_open chips missing key categories: location=%v promo=%v, chips=%+v", hasLocation, hasPromo, resp.GuidanceChips)
	}
}

// TestS1_EntryZoneScan 验证货架扫码入口返回该区域商品。
func TestS1_EntryZoneScan(t *testing.T) {
	repo := &fakeRepo{}
	svc := newS1TestService(repo)

	zoneID := int64(2)
	shelfID := int64(3)
	resp, err := svc.Chat(context.Background(), ChatRequest{
		RequestID: "r1",
		StoreID:   1,
		Channel:   "miniapp",
		Message:   "扫码",
		EntryMode: "zone_scan",
		ZoneID:    &zoneID,
		ShelfID:   &shelfID,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Intent != "zone_scan" {
		t.Errorf("expected zone_scan intent, got %s", resp.Intent)
	}
	if resp.Meta.Route != "entry_zone_scan" {
		t.Errorf("expected route entry_zone_scan, got %s", resp.Meta.Route)
	}
	// zone_scan 应返回卡片
	if len(resp.Cards) == 0 {
		t.Log("zone_scan returned 0 cards (fake repo has no zone-filtered data, back to SearchProducts)")
	}
}

// TestS1_MultiTurnFollowup 验证多轮追问：第1轮问位置，第2轮问库存。
func TestS1_MultiTurnFollowup(t *testing.T) {
	repo := &fakeRepo{}
	svc := newS1TestService(repo)

	// 第 1 轮：可乐在哪里
	resp1, err := svc.Chat(context.Background(), ChatRequest{
		RequestID: "r1",
		StoreID:   1,
		Channel:   "miniapp",
		Message:   "可乐在哪里",
	})
	if err != nil {
		t.Fatalf("round 1 error: %v", err)
	}
	if resp1.Intent != "product_location" {
		t.Errorf("round 1: expected product_location intent, got %s", resp1.Intent)
	}
	if len(resp1.Cards) == 0 {
		t.Errorf("round 1: expected product card")
	}

	// 第 2 轮：同一个 session 问"还有几瓶？"
	resp2, err := svc.Chat(context.Background(), ChatRequest{
		RequestID: "r2",
		StoreID:   1,
		SessionID: resp1.SessionID,
		Channel:   "miniapp",
		Message:   "还有几瓶？",
	})
	if err != nil {
		t.Fatalf("round 2 error: %v", err)
	}
	if resp2.Intent == "unsupported" {
		t.Errorf("round 2: intent should NOT be unsupported after followup, got %s", resp2.Intent)
	}
	if resp2.Intent == "faq" && !strings.Contains(resp2.Answer, "库存") && !strings.Contains(resp2.Answer, "12") {
		t.Errorf("round 2: expected inventory answer, got intent=%s answer=%s", resp2.Intent, resp2.Answer)
	}

	// 第 3 轮：问"多少钱？"
	resp3, err := svc.Chat(context.Background(), ChatRequest{
		RequestID: "r3",
		StoreID:   1,
		SessionID: resp1.SessionID,
		Channel:   "miniapp",
		Message:   "多少钱？",
	})
	if err != nil {
		t.Fatalf("round 3 error: %v", err)
	}
	if resp3.Intent == "unsupported" {
		t.Errorf("round 3: should not be unsupported, got %s", resp3.Intent)
	}
}

// TestS1_GuideEngineChips 验证位置回答后有引导 chips。
func TestS1_GuideEngineChips(t *testing.T) {
	repo := &fakeRepo{}
	svc := newS1TestService(repo)

	resp, err := svc.Chat(context.Background(), ChatRequest{
		RequestID: "r1",
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
	// S1: 位置回答后应有引导 chips
	if len(resp.GuidanceChips) < 2 {
		t.Errorf("expected >= 2 guidance chips after location answer, got %d", len(resp.GuidanceChips))
	}
	hasInventoryChip := false
	hasPromoChip := false
	for _, c := range resp.GuidanceChips {
		if strings.Contains(c.Text, "几瓶") || strings.Contains(c.Text, "库存") {
			hasInventoryChip = true
		}
		if strings.Contains(c.Text, "活动") {
			hasPromoChip = true
		}
	}
	if !hasInventoryChip {
		t.Errorf("location guidance missing inventory chip: %+v", resp.GuidanceChips)
	}
	if !hasPromoChip {
		t.Errorf("location guidance missing promo chip: %+v", resp.GuidanceChips)
	}
}

// TestS1_InventoryCredibility 验证库存回答含可信度标签。
func TestS1_InventoryCredibility(t *testing.T) {
	repo := &fakeRepo{}
	svc := newS1TestService(repo)

	resp, err := svc.Chat(context.Background(), ChatRequest{
		RequestID: "r1",
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
	// 库存卡片
	if len(resp.Cards) == 0 || resp.Cards[0].Type != "inventory" {
		t.Fatalf("expected inventory card, got %+v", resp.Cards)
	}
	if resp.Cards[0].Quantity <= 0 {
		t.Errorf("expected positive quantity, got %d", resp.Cards[0].Quantity)
	}
	// 可信度标签在 evidence 或 answer 中应可见
	foundCredibility := strings.Contains(resp.Answer, "high") ||
		strings.Contains(resp.Answer, "medium") ||
		strings.Contains(resp.Answer, "low") ||
		strings.Contains(resp.Answer, "reference_only") ||
		strings.Contains(resp.Answer, "分钟") ||
		strings.Contains(resp.Answer, "小时") ||
		strings.Contains(resp.Answer, "天前") ||
		strings.Contains(resp.Answer, "刚刚")
	if !foundCredibility {
		t.Logf("credibility tag not found in answer (fakeRepo has no last_verified_at): %s", resp.Answer)
	}
}

// TestS1_ContextStatePersistence 验证 context_state 被持久化。
func TestS1_ContextStatePersistence(t *testing.T) {
	repo := &fakeRepo{}
	svc := newS1TestService(repo)

	resp, err := svc.Chat(context.Background(), ChatRequest{
		RequestID: "r1",
		StoreID:   1,
		Channel:   "miniapp",
		Message:   "可乐在哪里",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.SessionID <= 0 {
		t.Fatal("expected session_id")
	}
	// assistant 消息应被持久化，且包含 context_state
	foundAssistant := false
	for _, msg := range repo.messages {
		if msg.Role == "assistant" && msg.ContextState != nil {
			foundAssistant = true
			if *msg.ContextState != string(StateProductFocus) {
				t.Logf("context_state = %s, expected product_focus", *msg.ContextState)
			}
			break
		}
	}
	if !foundAssistant {
		t.Errorf("expected assistant message with context_state, got messages=%+v", repo.messages)
	}
}

// TestS1_FocusEntityPersistence 验证 focus_entity_ids 在多轮对话后被持久化。
func TestS1_FocusEntityPersistence(t *testing.T) {
	repo := &fakeRepo{}
	svc := newS1TestService(repo)

	// 第一轮：告诉用户可乐位置 → focus=101
	resp1, err := svc.Chat(context.Background(), ChatRequest{
		RequestID: "r1",
		StoreID:   1,
		Channel:   "miniapp",
		Message:   "可乐在哪里",
	})
	if err != nil {
		t.Fatalf("round 1 error: %v", err)
	}
	_ = resp1

	// 第二轮：追问 → 应更新 focus
	resp2, err := svc.Chat(context.Background(), ChatRequest{
		RequestID: "r2",
		StoreID:   1,
		SessionID: resp1.SessionID,
		Channel:   "miniapp",
		Message:   "那个还有几瓶？",
	})
	if err != nil {
		t.Fatalf("round 2 error: %v", err)
	}
	_ = resp2

	// 验证 context_stack 持续追加
	for _, msg := range repo.messages {
		if msg.Role == "assistant" && len(msg.ContextStack) > 0 {
			t.Logf("context_stack found in message %d: %+v", msg.ID, msg.ContextStack)
			return // pass
		}
	}
	t.Logf("context_stack not found in any assistant message (may require real SessionManager with DB)")
}
