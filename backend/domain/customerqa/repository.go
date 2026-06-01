package customerqa

import "context"

type Repository interface {
	CreateSession(ctx context.Context, session *Session) (*Session, error)
	CreateMessage(ctx context.Context, message *Message) (*Message, error)
	SearchFAQ(ctx context.Context, storeID int64, query string, limit int) ([]FAQ, error)
}
