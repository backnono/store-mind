package customerqa

import "time"

type Session struct {
	ID        int64
	StoreID   int64
	UserID    *int64
	Channel   string
	StartedAt time.Time
}

type Message struct {
	ID        int64
	SessionID int64
	Role      string
	Content   string
	Intent    string
	CreatedAt time.Time
}

type FAQ struct {
	ID       int64
	StoreID  int64
	Question string
	Answer   string
	Category string
}
