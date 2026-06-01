package mysql

import "time"

type SessionModel struct {
	ID        int64 `gorm:"primaryKey;autoIncrement"`
	StoreID   int64 `gorm:"not null;index"`
	UserID    *int64
	Channel   string    `gorm:"type:varchar(64);not null"`
	StartedAt time.Time `gorm:"autoCreateTime"`
}

func (SessionModel) TableName() string { return "agent_session" }

type MessageModel struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	SessionID int64     `gorm:"not null;index"`
	Role      string    `gorm:"type:varchar(32);not null"`
	Content   string    `gorm:"type:text;not null"`
	Intent    string    `gorm:"type:varchar(64);not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (MessageModel) TableName() string { return "agent_message" }

type FAQModel struct {
	ID       int64  `gorm:"primaryKey;autoIncrement"`
	StoreID  int64  `gorm:"not null;index"`
	Question string `gorm:"type:varchar(255);not null"`
	Answer   string `gorm:"type:text;not null"`
	Category string `gorm:"type:varchar(64);not null"`
	Status   string `gorm:"type:varchar(32);not null"`
}

func (FAQModel) TableName() string { return "faq" }
