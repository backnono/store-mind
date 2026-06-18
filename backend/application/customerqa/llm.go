package customerqa

import "context"

type IntentRequest struct {
	StoreID   int64
	SessionID int64
	Message   string
}

type IntentAnalyzer interface {
	AnalyzeIntent(ctx context.Context, req IntentRequest) (Decision, error)
}

type AnswerRequest struct {
	Decision Decision
	Message  string
	Evidence []Evidence
}

// GuidanceChip 引导建议芯片
type GuidanceChip struct {
	Text   string `json:"text"`
	Prompt string `json:"prompt"`
}

// AnswerResult 答案生成结果
type AnswerResult struct {
	Answer        string         `json:"answer"`
	GuidanceChips []GuidanceChip `json:"guidance_chips"`
}

type AnswerComposer interface {
	ComposeAnswer(ctx context.Context, req AnswerRequest) (*AnswerResult, error)
}
