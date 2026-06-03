package customerqa

import (
	"context"
	"time"
)

type KnowledgeChunk struct {
	ID            int64     `json:"id"`
	DocID         string    `json:"doc_id"`
	StoreID       int64     `json:"store_id"`
	KnowledgeType string    `json:"knowledge_type"`
	Title         string    `json:"title"`
	Content       string    `json:"content"`
	Tags          []string  `json:"tags,omitempty"`
	ProductID     *int64    `json:"product_id,omitempty"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type ChatDecisionLog struct {
	ID              int64     `json:"id"`
	SessionID       int64     `json:"session_id"`
	MessageID       int64     `json:"message_id"`
	Intent          string    `json:"intent"`
	Route           string    `json:"route"`
	RewriteQuery    string    `json:"rewrite_query,omitempty"`
	Confidence      float64   `json:"confidence"`
	FallbackUsed    bool      `json:"fallback_used"`
	HandoffRequired bool      `json:"handoff_required"`
	CreatedAt       time.Time `json:"created_at"`
}

type KnowledgeRepository interface {
	SearchKnowledge(ctx context.Context, storeID int64, query string, knowledgeTypes []string, limit int) ([]KnowledgeChunk, error)
}

type DecisionLogRepository interface {
	CreateDecisionLog(ctx context.Context, log *ChatDecisionLog) (*ChatDecisionLog, error)
}
