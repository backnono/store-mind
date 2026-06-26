package customerqa

import (
	"context"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"
)

// —— LLM 客户端接口 ——

// LLMChatRequest Agent 循环中单次 LLM 调用的请求。
type LLMChatRequest struct {
	Messages        []AgentMessage   `json:"messages"`
	ToolDefinitions []map[string]any `json:"tools,omitempty"`
}

// LLMChatResponse Agent 循环中单次 LLM 调用的响应。
type LLMChatResponse struct {
	Content   string        `json:"content"`    // 文本内容（final_answer 或附带的思考）
	ToolCalls []LLMToolCall `json:"tool_calls"` // 工具调用列表
}

// LLMToolCall LLM 返回的单个工具调用。
type LLMToolCall struct {
	ID   string          `json:"id"`
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

// LLMClient Agent 循环所需的 LLM 客户端接口。
// 与旧的 IntentAnalyzer / AnswerComposer / AnaphoraClient 分离，
// 仅提供一个方法：传入消息 + 工具定义，返回文本或工具调用。
type LLMClient interface {
	// Chat 调用 LLM，支持 tool calling。
	// 返回的 LLMChatResponse 中，若 ToolCalls 非空则 LLM 请求调用工具；
	// 若 ToolCalls 为空则 Content 为最终回答。
	Chat(ctx context.Context, req LLMChatRequest) (*LLMChatResponse, error)
}

// —— Agent 结构 ——

// Agent 封装 Agent 循环的全部依赖和执行逻辑。
type Agent struct {
	llmClient LLMClient
	tools     []Tool
	log       Logger
}

// NewAgent 创建 Agent 实例。
func NewAgent(llmClient LLMClient, deps ToolDeps, log Logger) *Agent {
	if log == nil {
		log = nopLogger{}
	}
	return &Agent{
		llmClient: llmClient,
		tools:     AllAgentTools(deps),
		log:       log,
	}
}

// —— Agent 运行请求/响应 ——

// AgentRunRequest Agent 循环的输入参数。
type AgentRunRequest struct {
	StoreID     int64          // 门店 ID
	SessionID   int64          // 会话 ID
	MessageID   int64          // 已持久化的用户消息 ID
	History     []AgentMessage // 从 DB 加载的消息历史（已含本轮 user 消息）
	UserMessage string         // 本轮用户输入（用于日志）
}

// AgentRunResult Agent 循环的输出结果。
type AgentRunResult struct {
	FinalAnswer    string         // LLM 的最终自然语言回答
	Cards          []ChatCard     // 从 tool 返回结果中提取的结构化卡片
	UpdatedHistory []AgentMessage // 更新后的完整消息历史（含本轮所有消息）
}

// —— Agent 循环核心 ——

// nextToolCallID 自增计数器，用于生成唯一的 tool_call ID。
var nextToolCallID atomic.Int64

func generateToolCallID() string {
	return fmt.Sprintf("call_%d", nextToolCallID.Add(1))
}

// MaxAgentLoops Agent 循环最大轮数，防止死循环。
const MaxAgentLoops = 5

// Run 执行一次 Agent 多轮工具调用循环，直到 LLM 返回 final_answer。
//
// 流程:
//  1. 构建初始消息列表（system + history + user）
//  2. 循环: LLM → 解析 → tool 调用 → 观察 → 再推理
//  3. 返回 final_answer + cards + 更新后的消息历史
func (a *Agent) Run(ctx context.Context, req AgentRunRequest) (*AgentRunResult, error) {
	// 1. 构建消息列表：system prompt + 历史消息
	messages := make([]AgentMessage, 0, len(req.History)+1)
	messages = append(messages, AgentMessage{
		Role:    "system",
		Content: SystemPrompt(req.StoreID),
	})
	messages = append(messages, req.History...)

	// 2. 循环
	for loop := 0; loop < MaxAgentLoops; loop++ {
		a.log.Info("agent_loop_iteration",
			"session_id", req.SessionID,
			"loop", loop,
			"msg_count", len(messages),
		)

		// 2a. 调用 LLM
		resp, err := a.llmClient.Chat(ctx, LLMChatRequest{
			Messages:        messages,
			ToolDefinitions: toolDefinitions(a.tools),
		})
		if err != nil {
			return nil, fmt.Errorf("agent_loop_llm: %w", err)
		}

		// 2b. 有 tool_call → 执行
		if len(resp.ToolCalls) > 0 {
			// 追加 assistant 的 tool_call 消息
			assistantMsg := AgentMessage{Role: "assistant"}
			if resp.Content != "" {
				assistantMsg.Content = resp.Content
			}
			for _, tc := range resp.ToolCalls {
				assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, AgentToolCall{
					ID:   tc.ID,
					Name: tc.Name,
					Args: tc.Args,
				})
			}
			messages = append(messages, assistantMsg)

			// 执行每个 tool_call
			for _, tc := range resp.ToolCalls {
				tool := findTool(tc.Name, a.tools)
				if tool == nil {
					errMsg := fmt.Sprintf(`{"error": "unknown tool: %s"}`, tc.Name)
					messages = append(messages, AgentMessage{
						Role: "tool", ToolID: tc.ID, Content: errMsg,
					})
					a.log.Warn("agent_unknown_tool",
						"tool_name", tc.Name,
						"session_id", req.SessionID,
					)
					continue
				}

				toolStart := time.Now()
				result, toolErr := tool.Run(ctx, req.StoreID, req.SessionID, req.MessageID, tc.Args)
				elapsed := time.Since(toolStart)

				if toolErr != nil {
					a.log.Warn("agent_tool_error",
						"tool_name", tc.Name,
						"session_id", req.SessionID,
						"latency_ms", elapsed.Milliseconds(),
						"error", toolErr,
					)
					result = fmt.Sprintf(`{"error": "%s"}`, toolErr.Error())
				} else {
					a.log.Info("agent_tool_success",
						"tool_name", tc.Name,
						"session_id", req.SessionID,
						"latency_ms", elapsed.Milliseconds(),
					)
				}

				messages = append(messages, AgentMessage{
					Role: "tool", ToolID: tc.ID, Content: result,
				})
			}
			continue // 回到循环
		}

		// 2c. 无 tool_call → final answer
		if resp.Content == "" {
			// LLM 返回空内容，追加一条无 tool 的 assistant 消息并继续
			a.log.Warn("agent_empty_response", "session_id", req.SessionID, "loop", loop)
			messages = append(messages, AgentMessage{Role: "assistant", Content: ""})
			continue
		}

		// LLM 可能在没有 tool_call 时同时返回 content，这是 final answer
		// 追加 assistant 消息到历史
		messages = append(messages, AgentMessage{
			Role:    "assistant",
			Content: resp.Content,
		})

		return &AgentRunResult{
			FinalAnswer:    resp.Content,
			Cards:          extractCards(messages),
			UpdatedHistory: messages[1:], // 去掉 system prompt
		}, nil
	}

	// 超出最大循环次数 → 返回兜底回答
	fallbackAnswer := "暂时无法回答这个问题，你可以换个问法，或联系人工客服。"
	messages = append(messages, AgentMessage{
		Role:    "assistant",
		Content: fallbackAnswer,
	})
	a.log.Warn("agent_max_loops_exceeded",
		"session_id", req.SessionID,
		"loops", MaxAgentLoops,
	)
	return &AgentRunResult{
		FinalAnswer:    fallbackAnswer,
		Cards:          extractCards(messages),
		UpdatedHistory: messages[1:],
	}, nil
}

// UsesLLM 判断 Agent 是否具备 LLM 能力。
func (a *Agent) UsesLLM() bool {
	return a.llmClient != nil
}
