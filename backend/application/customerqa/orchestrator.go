package customerqa

import (
	"context"

	domain "store-mind/domain/customerqa"
)

// —— 路由类型常量 ——

const (
	RouteTool     = "tool"
	RouteRAG      = "rag"
	RouteHybrid   = "hybrid"
	RouteFallback = "fallback"
)

// —— 编排器接口与共享类型 ——

// Orchestrator 定义编排协议（Agent 循环中仅 fallback 路径使用）。
type Orchestrator interface {
	Run(ctx context.Context, req OrchestratorRequest) (OrchestratorResult, error)
}

// OrchestratorRequest 编排入参。
type OrchestratorRequest struct {
	RequestID        string
	StoreID          int64
	SessionID        int64
	MessageID        int64
	UserID           *int64
	Channel          string
	Message          string
	SessionContext   *SessionContext         // 保留用于向后兼容
	ResolvedEntities []domain.ResolvedEntity // 保留用于向后兼容
}

// Decision 单次编排决策。
type Decision struct {
	Intent         string
	RewrittenQuery string
	Route          string
	NeedsHandoff   bool
	Confidence     float64
	ReasoningTags  []string
	FallbackUsed   bool
}

// Evidence 证据片段。
type Evidence struct {
	Source   string
	Kind     string
	RecordID int64
	Title    string
	Content  string
}

// OrchestratorResult 编排输出。
type OrchestratorResult struct {
	Decision      Decision
	Answer        string
	Cards         []ChatCard
	Evidence      []Evidence
	GuidanceChips []GuidanceChip
}

// —— 兼容性构造函数 ——

// NewDefaultOrchestrator 创建降级编排器（LLM 不可用时使用）。
// 当 analyzer/composer/retriever 全为 nil 时，等价于纯 fallback 模式。
func NewDefaultOrchestrator(
	repo domain.Repository,
	log Logger,
	analyzer IntentAnalyzer,
	composer AnswerComposer,
	retriever Retriever,
) Orchestrator {
	return newFallbackOrchestrator(repo, log)
}
