package mysql

import (
	"context"
	"fmt"

	domain "store-mind/domain/customerqa"

	"gorm.io/gorm"
)

type CustomerQARepository struct {
	db *gorm.DB
}

func NewCustomerQARepository(db *gorm.DB) *CustomerQARepository {
	return &CustomerQARepository{db: db}
}

func (r *CustomerQARepository) CreateSession(ctx context.Context, session *domain.Session) (*domain.Session, error) {
	m := SessionModel{StoreID: session.StoreID, UserID: session.UserID, Channel: session.Channel}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	session.ID = m.ID
	session.StartedAt = m.StartedAt
	return session, nil
}

func (r *CustomerQARepository) CreateMessage(ctx context.Context, message *domain.Message) (*domain.Message, error) {
	m := MessageModel{SessionID: message.SessionID, Role: message.Role, Content: message.Content, Intent: message.Intent}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	message.ID = m.ID
	message.CreatedAt = m.CreatedAt
	return message, nil
}

func (r *CustomerQARepository) SearchFAQ(ctx context.Context, storeID int64, query string, limit int) ([]domain.FAQ, error) {
	pattern := fmt.Sprintf("%%%s%%", query)
	var rows []FAQModel
	if err := r.db.WithContext(ctx).
		Where("store_id = ? AND status = ? AND (question LIKE ? OR answer LIKE ?)", storeID, "active", pattern, pattern).
		Order("id DESC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]domain.FAQ, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.FAQ{ID: row.ID, StoreID: row.StoreID, Question: row.Question, Answer: row.Answer, Category: row.Category})
	}
	return out, nil
}
