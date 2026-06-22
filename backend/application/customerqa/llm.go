package customerqa

import "context"

// —— 意图分析层 ——

// IntentRequest 意图分析请求，传入用户消息及上下文。
type IntentRequest struct {
	StoreID   int64  // 门店 ID，用于领域限定
	SessionID int64  // 会话 ID，用于多轮上下文（history 由实现自行获取）
	Message   string // 用户原始消息
}

// IntentAnalyzer 意图分析器，识别用户消息的意图并做出路由决策。
// 返回值 Decision 包含意图标签、置信度、改写后的查询等，供编排器决定后续执行路径。
type IntentAnalyzer interface {
	AnalyzeIntent(ctx context.Context, req IntentRequest) (Decision, error)
}

// —— 答案生成层 ——

// AnswerRequest 答案生成请求，传入编排决策和召回的 evidence。
type AnswerRequest struct {
	Decision Decision   // 编排决策（含意图、路由、置信度等）
	Message  string     // 用户原始消息，供 LLM 拼接 prompt
	Evidence []Evidence // 从知识库 / 工具层召回的证据列表
}

// GuidanceChip 引导建议芯片，展示在回答下方供用户快速点击追问。
type GuidanceChip struct {
	Text   string `json:"text"`   // 展示文案，如「还有库存吗」
	Prompt string `json:"prompt"` // 点击后发送的 prompt 文本
}

// AnswerResult 答案生成结果，包含自然语言答案和可选的引导芯片。
type AnswerResult struct {
	Answer        string         `json:"answer"`         // 自然语言回答
	GuidanceChips []GuidanceChip `json:"guidance_chips"` // 建议追问列表
}

// AnswerComposer 答案组装器，将证据与决策信息组合成面向用户的自然语言回答。
// 当前由 infra/ai 中的 Python LLM sidecar 实现。
type AnswerComposer interface {
	ComposeAnswer(ctx context.Context, req AnswerRequest) (*AnswerResult, error)
}
