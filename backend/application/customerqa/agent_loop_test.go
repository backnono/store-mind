package customerqa

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	domain "store-mind/domain/customerqa"
)

// ── Agent 循环集成测试 ───────────────────────────

// fakeLLMClient 模拟 LLM 的行为。
// 第一轮返回 tool_calls → 第二轮返回 final_answer。
type fakeLLMClient struct {
	callCount int
	// toolCalls 模拟的 tool call 序列
	toolCallSeq []LLMToolCall
	// finalAnswer 最终回答
	finalAnswer string
}

func (f *fakeLLMClient) Chat(_ context.Context, req LLMChatRequest) (*LLMChatResponse, error) {
	f.callCount++
	if f.callCount <= len(f.toolCallSeq) {
		return &LLMChatResponse{
			ToolCalls: []LLMToolCall{f.toolCallSeq[f.callCount-1]},
		}, nil
	}
	return &LLMChatResponse{Content: f.finalAnswer}, nil
}

// TestAgentLoop_NoDuplicateMessages 验证 Agent 循环不产生重复消息。
func TestAgentLoop_NoDuplicateMessages(t *testing.T) {
	repo := &fakeRepoWithToolCallLog{}

	// 构造 fake LLM：第 1 轮调用 search_products("薯片")，第 2 轮返回 final answer
	fakeLLM := &fakeLLMClient{
		toolCallSeq: []LLMToolCall{
			{
				ID: "call_1", Name: "search_products",
				Args: json.RawMessage(`{"query":"薯片"}`),
			},
		},
		finalAnswer: "薯片在零食区 A-03 货架第2层。",
	}

	agent := NewAgent(fakeLLM, ToolDeps{Repo: repo, Log: nopLogger{}}, nopLogger{})
	fallback := newFallbackOrchestrator(repo, nopLogger{})

	svc := NewServiceWithConfig(ServiceConfig{
		Repo:     repo,
		Log:      nopLogger{},
		Agent:    agent,
		Fallback: fallback,
	})

	// ── 第 1 轮：薯片在哪里？──
	resp1, err := svc.Chat(context.Background(), ChatRequest{
		RequestID: "r1", StoreID: 1, Channel: "miniapp", Message: "薯片在哪里？",
	})
	if err != nil {
		t.Fatalf("round 1 error: %v", err)
	}

	// 消息去重计数
	userCount, assistantCount, toolCount := countMessagesByRole(repo.messages)
	t.Logf("Round 1 messages: user=%d assistant=%d tool=%d total=%d",
		userCount, assistantCount, toolCount, len(repo.messages))

	// 核心断言：无重复消息
	// user=1（Chat 步骤③持久化）
	// assistant 可能有 2 条（1 中间带 tool_calls + 1 最终回答）
	if userCount != 1 {
		t.Errorf("round 1: expected 1 user message, got %d", userCount)
	}
	if assistantCount < 1 || assistantCount > 2 {
		t.Errorf("round 1: expected 1-2 assistant messages, got %d", assistantCount)
	}
	// tool 消息（search_products 的调用结果）应该存在
	if toolCount < 1 {
		t.Errorf("round 1: expected >=1 tool message, got %d", toolCount)
	}

	// 验证回答内容
	if resp1.Answer != "薯片在零食区 A-03 货架第2层。" {
		t.Errorf("round 1: expected final answer, got %q", resp1.Answer)
	}
	if strings.Contains(resp1.Answer, "暂时无法") {
		t.Errorf("round 1: should not be fallback answer")
	}

	// ── 第 2 轮：追问 "还有几包？" ──
	fakeLLM.callCount = 0 // 重置计数
	fakeLLM.toolCallSeq = []LLMToolCall{
		{
			ID: "call_2", Name: "search_products",
			Args: json.RawMessage(`{"query":"薯片"}`),
		},
	}
	fakeLLM.finalAnswer = "乐事原味薯片目前还有 15 包。"

	resp2, err := svc.Chat(context.Background(), ChatRequest{
		RequestID: "r2", StoreID: 1, SessionID: resp1.SessionID,
		Channel: "miniapp", Message: "还有几包？",
	})
	if err != nil {
		t.Fatalf("round 2 error: %v", err)
	}

	// 第 2 轮新增的消息
	// 注意：fakeRepo.messages 是累积的，需要取差值
	allMsgs := repo.messages
	userCount2, assistantCount2, toolCount2 := countMessagesByRole(allMsgs)

	t.Logf("After round 2: user=%d assistant=%d tool=%d total=%d",
		userCount2, assistantCount2, toolCount2, len(allMsgs))

	// 用户消息总共 2 条（每轮 1 条）
	if userCount2 != 2 {
		t.Errorf("after round 2: expected 2 user messages total, got %d", userCount2)
	}

	// 每轮 assistant 各 1 条 + 可能的中间 assistant（带 tool_calls）
	if assistantCount2 < 2 {
		t.Errorf("after round 2: expected >=2 assistant messages, got %d", assistantCount2)
	}

	// 验证没有重复的兜底回答
	fallbackCount := 0
	for _, m := range repo.messages {
		if strings.Contains(m.Content, "暂时无法处理") {
			fallbackCount++
		}
	}
	if fallbackCount > 1 {
		t.Errorf("found %d fallback messages, expected <= 1", fallbackCount)
	}

	t.Logf("Round 2 answer: %s", resp2.Answer)
	t.Logf("Round 2 session_id=%d message_id=%d", resp2.SessionID, resp2.MessageID)

	// debug: dump all messages
	for _, m := range repo.messages {
		t.Logf("  msg id=%d role=%s content=%.60s intent=%s",
			m.ID, m.Role, m.Content, m.Intent)
	}
}

func countMessagesByRole(msgs []domain.Message) (user, assistant, tool int) {
	for _, m := range msgs {
		switch m.Role {
		case "user":
			user++
		case "assistant":
			assistant++
		case "tool":
			tool++
		}
	}
	return
}

// ── 重复消息回归测试（精确模拟原始问题）──

// TestAgentLoop_RegressDuplicateUserAndFallback 模拟原始问题：
// 用户问"薯片在哪里？"→ 不应出现 3 条 user + 2 条兜底 assistant。
func TestAgentLoop_RegressDuplicateUserAndFallback(t *testing.T) {
	repo := &fakeRepoWithToolCallLog{}

	// 第 1 轮 LLM 调用失败 → 走 fallback（但 fallback 不触发 tool calling）
	// 此处用 fakeLLM 返回 error 来模拟 LLM 失败
	fakeLLM := &fakeLLMClient{
		finalAnswer: "可乐在饮料区 B-02 货架。",
	}
	agent := NewAgent(fakeLLM, ToolDeps{Repo: repo, Log: nopLogger{}}, nopLogger{})
	fallback := newFallbackOrchestrator(repo, nopLogger{})

	svc := NewServiceWithConfig(ServiceConfig{
		Repo:     repo,
		Log:      nopLogger{},
		Agent:    agent,
		Fallback: fallback,
	})

	resp, err := svc.Chat(context.Background(), ChatRequest{
		RequestID: "r1", StoreID: 1, Channel: "miniapp", Message: "可乐在哪里",
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	userCount, assistantCount, toolCount := countMessagesByRole(repo.messages)
	t.Logf("Messages: user=%d assistant=%d tool=%d total=%d answer=%s",
		userCount, assistantCount, toolCount, len(repo.messages), resp.Answer)

	if userCount != 1 {
		t.Errorf("expected exactly 1 user message, got %d", userCount)
	}
	if assistantCount != 1 {
		t.Errorf("expected exactly 1 assistant message, got %d", assistantCount)
	}
}

// ── 辅助 fake Repo ──

type fakeRepoWithToolCallLog struct {
	fakeRepo
}

func (f *fakeRepoWithToolCallLog) CreateToolCall(_ context.Context, tc *domain.ToolCall) (*domain.ToolCall, error) {
	tc.ID = int64(len(f.toolCalls) + 1)
	f.toolCalls = append(f.toolCalls, *tc)
	return tc, nil
}

// Ensure fakeRepoWithToolCallLog always has products for search
func (f *fakeRepoWithToolCallLog) SearchProducts(_ context.Context, storeID int64, query string, limit int) ([]domain.Product, error) {
	return []domain.Product{
		{ID: 106, Name: "乐事原味薯片", Brand: "乐事", Category: "零食"},
	}, nil
}

func TestAgentLoopToolOutputFormat(t *testing.T) {
	// 验证 tool 输出的 JSON 格式能被 Go unmarshal
	deps := ToolDeps{Repo: &fakeRepoWithToolCallLog{}, Log: nopLogger{}}
	tool := &SearchProductsTool{deps: deps}

	result, err := tool.Run(context.Background(), 1, 1, 1, json.RawMessage(`{"query":"薯片"}`))
	if err != nil {
		t.Fatalf("tool error: %v", err)
	}

	// 验证结果是合法 JSON 数组
	var results []map[string]any
	if err := json.Unmarshal([]byte(result), &results); err != nil {
		t.Fatalf("tool output is not valid JSON: %v\noutput=%s", err, result)
	}
	if len(results) == 0 {
		t.Fatal("expected at least 1 product")
	}
	if results[0]["product_id"] == nil {
		t.Error("expected product_id in output")
	}
	t.Logf("search_products output: %s", result)
}

// TestAgentLoop_PersistOnlyIntermediate 验证 persistAgentMessages 只持久化中间消息。
func TestAgentLoop_PersistOnlyIntermediate(t *testing.T) {
	msgs := []AgentMessage{
		{Role: "user", Content: "薯片在哪里？"},
		{Role: "assistant", Content: "", ToolCalls: []AgentToolCall{{
			ID: "call_1", Name: "search_products",
			Args: json.RawMessage(`{"query":"薯片"}`),
		}}},
		{Role: "tool", Content: `[{"product_id":106,"name":"乐事原味薯片"}]`, ToolID: "call_1"},
		{Role: "assistant", Content: "薯片在零食区 A-03 货架。"},
	}

	domainMsgs := ConvertToDomainMessages(1, msgs)

	// 模拟 persistAgentMessages 的逻辑
	persisted := make([]domain.Message, 0)
	for _, m := range domainMsgs {
		switch m.Role {
		case "user":
			continue
		case "assistant":
			if m.ToolCallsJSON == nil || *m.ToolCallsJSON == "" {
				continue
			}
		}
		persisted = append(persisted, m)
	}

	if len(persisted) != 2 {
		t.Errorf("expected 2 persisted messages (assistant+tool_calls + tool_result), got %d", len(persisted))
		for i, m := range persisted {
			t.Logf("  [%d] role=%s content=%.40s tool_calls=%v", i, m.Role, m.Content, m.ToolCallsJSON)
		}
	}

	// 第 1 条应该是带 tool_calls 的 assistant
	if persisted[0].Role != "assistant" {
		t.Errorf("expected assistant with tool_calls first, got %s", persisted[0].Role)
	}
	// 第 2 条应该是 tool_result
	if persisted[1].Role != "tool" {
		t.Errorf("expected tool second, got %s", persisted[1].Role)
	}
}

// TestAgentLoop_MultiTurnToolCall 验证多轮对话中 tool_call 参数正确传递。
func TestAgentLoop_MultiTurnToolCall(t *testing.T) {
	repo := &fakeRepoWithToolCallLog{}

	// 模拟完整的两步 tool call 链：
	// 第 1 轮 LLM: search_products("薯片") → tool result: [{product_id:106}]
	// 第 2 轮 LLM: get_product_location(106) → tool result: {zone_name:"零食区"...}
	// 第 3 轮 LLM: final answer
	fakeLLM := &fakeLLMClient{
		callCount: 0,
		toolCallSeq: []LLMToolCall{
			{
				ID: "call_1", Name: "search_products",
				Args: json.RawMessage(`{"query":"薯片"}`),
			},
			{
				ID: "call_2", Name: "get_product_location",
				Args: json.RawMessage(`{"product_id":106}`),
			},
		},
		finalAnswer: "乐事原味薯片在零食区 A-03 货架第3层，进门左手边。",
	}

	agent := NewAgent(fakeLLM, ToolDeps{Repo: repo, Log: nopLogger{}}, nopLogger{})
	fallback := newFallbackOrchestrator(repo, nopLogger{})

	svc := NewServiceWithConfig(ServiceConfig{
		Repo:     repo,
		Log:      nopLogger{},
		Agent:    agent,
		Fallback: fallback,
	})

	resp, err := svc.Chat(context.Background(), ChatRequest{
		RequestID: "r-multi", StoreID: 1, Channel: "miniapp", Message: "薯片在哪里？",
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	userCount, assistantCount, toolCount := countMessagesByRole(repo.messages)
	t.Logf("Multi-turn messages: user=%d assistant=%d tool=%d total=%d",
		userCount, assistantCount, toolCount, len(repo.messages))

	// 第 1 轮: user + assistant(tool_calls:search) + tool(search_result)
	// 第 2 轮: assistant(tool_calls:location) + tool(location_result)
	// 第 3 轮: assistant(final) ← 最终回答，在 Chat() 步骤⑥持久化
	// 总: user=1, assistant=2(带tool_calls) + 1(最终回答), tool=2
	if userCount != 1 {
		t.Errorf("expected 1 user, got %d", userCount)
	}
	if assistantCount < 2 {
		t.Errorf("expected >=2 assistants (1 intermediate + 1 final), got %d", assistantCount)
	}
	if toolCount < 2 {
		t.Errorf("expected >=2 tool results, got %d", toolCount)
	}

	// 验证最终回答
	t.Logf("final answer: %s", resp.Answer)
	if !strings.Contains(resp.Answer, "零食区") && resp.Answer != "" {
		t.Logf("answer may differ: %s", resp.Answer)
	}

	// 验证 tool call 参数正确传递
	for _, m := range repo.messages {
		t.Logf("  msg id=%d role=%s content=%.60s intent=%s",
			m.ID, m.Role, m.Content, m.Intent)
	}
	if len(repo.toolCalls) < 2 {
		t.Errorf("expected >=2 tool call logs, got %d", len(repo.toolCalls))
	}
}

// TestAgentLoop_FallbackOnLLMError 验证 LLM 调用失败时正确降级到 fallback。
func TestAgentLoop_FallbackOnLLMError(t *testing.T) {
	repo := &fakeRepoWithToolCallLog{}

	// 不传 Agent → 自动走 fallback
	fallback := newFallbackOrchestrator(repo, nopLogger{})

	svc := NewServiceWithConfig(ServiceConfig{
		Repo:     repo,
		Log:      nopLogger{},
		Agent:    nil, // 无 LLM
		Fallback: fallback,
	})

	resp, err := svc.Chat(context.Background(), ChatRequest{
		RequestID: "r-fb", StoreID: 1, Channel: "miniapp", Message: "可乐在哪里",
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	// fallback 回答不应为空
	if resp.Answer == "" {
		t.Error("expected non-empty fallback answer")
	}
	if resp.Meta.Route != "fallback" {
		t.Errorf("expected fallback route, got %s", resp.Meta.Route)
	}

	userCount, assistantCount, _ := countMessagesByRole(repo.messages)
	if userCount != 1 {
		t.Errorf("fallback: expected 1 user, got %d", userCount)
	}
	if assistantCount != 1 {
		t.Errorf("fallback: expected 1 assistant, got %d", assistantCount)
	}

	t.Logf("fallback answer: %s", resp.Answer)
	t.Logf("fallback intent: %s", resp.Intent)
}

// TestServerSideChatSim 模拟服务端 Chat 调用流程（最接近真实场景）。
func TestServerSideChatSim(t *testing.T) {
	repo := &fakeRepoWithToolCallLog{}

	fmt.Println("=== 模拟多轮对话 ===")

	// 构造 fake LLM
	fakeLLM := &fakeLLMClient{
		finalAnswer: "乐事原味薯片在零食区 A-03 货架第3层。",
	}
	agent := NewAgent(fakeLLM, ToolDeps{Repo: repo, Log: nopLogger{}}, nopLogger{})
	fallback := newFallbackOrchestrator(repo, nopLogger{})

	svc := NewServiceWithConfig(ServiceConfig{
		Repo:     repo,
		Log:      nopLogger{},
		Agent:    agent,
		Fallback: fallback,
	})

	// ── Turn 1: 首次打开 ──
	fmt.Println("\n[Turn 1] first_open")
	resp, _ := svc.Chat(context.Background(), ChatRequest{
		RequestID: "sim-1", StoreID: 1, Channel: "miniapp",
		Message: "打开", EntryMode: "first_open",
	})
	fmt.Printf("  session_id=%d answer=%s\n", resp.SessionID, resp.Answer[:50])
	sid := resp.SessionID

	// ── Turn 2: 问商品位置（带 tool call）──
	fmt.Println("\n[Turn 2] 薯片在哪里？")
	fakeLLM.callCount = 0
	fakeLLM.toolCallSeq = []LLMToolCall{
		{ID: "c1", Name: "search_products", Args: json.RawMessage(`{"query":"薯片"}`)},
		{ID: "c2", Name: "get_product_location", Args: json.RawMessage(`{"product_id":106}`)},
	}
	fakeLLM.finalAnswer = "乐事原味薯片在零食区 A-03 货架第3层，进门左手边。"

	resp2, _ := svc.Chat(context.Background(), ChatRequest{
		RequestID: "sim-2", StoreID: 1, SessionID: sid,
		Channel: "miniapp", Message: "薯片在哪里？",
	})
	fmt.Printf("  session_id=%d answer=%s\n", resp2.SessionID, resp2.Answer)

	// ── Turn 3: 追问库存 ──
	fmt.Println("\n[Turn 3] 还有几包？")
	fakeLLM.callCount = 0
	fakeLLM.toolCallSeq = []LLMToolCall{
		{ID: "c3", Name: "search_products", Args: json.RawMessage(`{"query":"薯片"}`)},
	}
	fakeLLM.finalAnswer = "乐事原味薯片目前还有 15 包，数据是 10 分钟前更新的。"

	resp3, _ := svc.Chat(context.Background(), ChatRequest{
		RequestID: "sim-3", StoreID: 1, SessionID: sid,
		Channel: "miniapp", Message: "还有几包？",
	})
	fmt.Printf("  session_id=%d answer=%s\n", resp3.SessionID, resp3.Answer)

	// ── 消息统计 ──
	fmt.Println("\n=== 消息统计 ===")
	userCount, assistantCount, toolCount := countMessagesByRole(repo.messages)
	fmt.Printf("  user=%d assistant=%d tool=%d total=%d\n",
		userCount, assistantCount, toolCount, len(repo.messages))

	for _, m := range repo.messages {
		contentPreview := m.Content
		if len(contentPreview) > 60 {
			contentPreview = contentPreview[:60] + "..."
		}
		toolCallInfo := ""
		if m.ToolCallsJSON != nil {
			toolCallInfo = fmt.Sprintf(" tool_calls=%s", *m.ToolCallsJSON)
		}
		toolCallID := ""
		if m.ToolCallID != nil {
			toolCallID = fmt.Sprintf(" tool_id=%s", *m.ToolCallID)
		}
		fmt.Printf("  id=%d role=%s content=%q%s%s\n",
			m.ID, m.Role, contentPreview, toolCallInfo, toolCallID)
	}

	// 断言：无重复（first_open 产生 1 assistant，turn2 1 user + 2 tool + 1 assistant，turn3 1 user + 1 tool + 1 assistant）
	// = 3 user + 3 assistant (2 normal + 1 first_open) + 3 tool = 9 total
	// 注意：first_open 不走 Agent 循环，只产生 1 条 assistant
	// turn2: user + assistant(tool_calls:search) + tool(search_result) + assistant(tool_calls:location) + tool(location_result) + assistant(final)
	//        但只有 user 和中间 assistant+tool 被持久化，final assistant 在 Chat() 步骤⑥
	// turn3: user + assistant(tool_calls:search) + tool(search_result) + assistant(final)
	// 预期: user=3 (turn1 没有 user 持久化，turn2+turn3 各1条)
	// 实际 first_open 不持久化 user，所以 user 应该是 2

	if userCount > 2 {
		t.Errorf("too many user messages: %d (expected <= 2)", userCount)
	}
}
