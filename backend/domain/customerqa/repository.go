package customerqa

import (
	"context"
	"time"
)

type Repository interface {
	CreateSession(ctx context.Context, session *Session) (*Session, error)
	GetSession(ctx context.Context, sessionID int64) (*Session, error)
	CreateMessage(ctx context.Context, message *Message) (*Message, error)
	CreateToolCall(ctx context.Context, toolCall *ToolCall) (*ToolCall, error)
	ListSessions(ctx context.Context, storeID int64, limit int) ([]Session, error)
	ListToolCalls(ctx context.Context, sessionID int64, limit int) ([]ToolCall, error)
	SearchFAQ(ctx context.Context, storeID int64, query string, limit int) ([]FAQ, error)
	SearchProducts(ctx context.Context, storeID int64, query string, limit int) ([]Product, error)
	GetProductLocation(ctx context.Context, storeID, productID int64) (*ProductLocation, error)
	GetInventory(ctx context.Context, storeID, skuID int64) (*Inventory, error)
	ListActivePromotions(ctx context.Context, storeID int64, now time.Time, limit int) ([]Promotion, error)
}

// FeedbackRepository 反馈持久化接口
type FeedbackRepository interface {
	CreateFeedback(ctx context.Context, feedback *Feedback) (*Feedback, error)
}
