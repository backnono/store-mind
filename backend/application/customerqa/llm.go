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

type AnswerComposer interface {
	ComposeAnswer(ctx context.Context, req AnswerRequest) (string, error)
}
