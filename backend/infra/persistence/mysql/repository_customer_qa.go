package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

func (r *CustomerQARepository) GetSession(ctx context.Context, sessionID int64) (*domain.Session, error) {
	var row SessionModel
	if err := r.db.WithContext(ctx).Where("id = ?", sessionID).First(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &domain.Session{
		ID:        row.ID,
		StoreID:   row.StoreID,
		UserID:    row.UserID,
		Channel:   row.Channel,
		StartedAt: row.StartedAt,
	}, nil
}

func (r *CustomerQARepository) CreateMessage(ctx context.Context, message *domain.Message) (*domain.Message, error) {
	m := MessageModel{SessionID: message.SessionID, Role: message.Role, Content: message.Content, Intent: message.Intent, Confidence: message.Confidence}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	message.ID = m.ID
	message.CreatedAt = m.CreatedAt
	return message, nil
}

func (r *CustomerQARepository) CreateToolCall(ctx context.Context, toolCall *domain.ToolCall) (*domain.ToolCall, error) {
	m := ToolCallModel{
		SessionID:    toolCall.SessionID,
		MessageID:    toolCall.MessageID,
		ToolName:     toolCall.ToolName,
		InputJSON:    toolCall.InputJSON,
		OutputJSON:   toolCall.OutputJSON,
		LatencyMS:    toolCall.LatencyMS,
		Success:      toolCall.Success,
		ErrorMessage: toolCall.ErrorMessage,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		return nil, err
	}
	toolCall.ID = m.ID
	toolCall.CreatedAt = m.CreatedAt
	return toolCall, nil
}

func (r *CustomerQARepository) ListSessions(ctx context.Context, storeID int64, limit int) ([]domain.Session, error) {
	var rows []SessionModel
	if err := r.db.WithContext(ctx).Where("store_id = ?", storeID).Order("id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.Session, 0, len(rows))
	for _, row := range rows {
		items = append(items, domain.Session{
			ID:        row.ID,
			StoreID:   row.StoreID,
			UserID:    row.UserID,
			Channel:   row.Channel,
			StartedAt: row.StartedAt,
		})
	}
	return items, nil
}

func (r *CustomerQARepository) ListToolCalls(ctx context.Context, sessionID int64, limit int) ([]domain.ToolCall, error) {
	var rows []ToolCallModel
	if err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("id ASC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]domain.ToolCall, 0, len(rows))
	for _, row := range rows {
		items = append(items, domain.ToolCall{
			ID:           row.ID,
			SessionID:    row.SessionID,
			MessageID:    row.MessageID,
			ToolName:     row.ToolName,
			InputJSON:    row.InputJSON,
			OutputJSON:   row.OutputJSON,
			LatencyMS:    row.LatencyMS,
			Success:      row.Success,
			ErrorMessage: row.ErrorMessage,
			CreatedAt:    row.CreatedAt,
		})
	}
	return items, nil
}

func (r *CustomerQARepository) SearchFAQ(ctx context.Context, storeID int64, query string, limit int) ([]domain.FAQ, error) {
	pattern := fmt.Sprintf("%%%s%%", query)
	var rows []FAQModel
	if err := r.db.WithContext(ctx).
		Where("store_id = ? AND status = ? AND (question LIKE ? OR answer LIKE ? OR keywords LIKE ?)", storeID, "active", pattern, pattern, pattern).
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

func (r *CustomerQARepository) SearchProducts(ctx context.Context, storeID int64, query string, limit int) ([]domain.Product, error) {
	pattern := fmt.Sprintf("%%%s%%", query)
	var rows []ProductModel
	if err := r.db.WithContext(ctx).
		Table("product AS p").
		Select("DISTINCT p.id, p.name, p.brand, p.category, p.aliases, p.tags, p.status").
		Joins("JOIN product_location AS pl ON pl.product_id = p.id AND pl.store_id = ?", storeID).
		Where("p.status = ? AND (p.name LIKE ? OR p.category LIKE ? OR p.aliases LIKE ?)", "active", pattern, pattern, pattern).
		Order("p.id DESC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]domain.Product, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Product{
			ID:       row.ID,
			Name:     row.Name,
			Brand:    row.Brand,
			Category: row.Category,
			Aliases:  decodeJSONList(row.Aliases),
			Tags:     decodeJSONList(row.Tags),
			Status:   row.Status,
		})
	}
	return out, nil
}

func (r *CustomerQARepository) GetProductLocation(ctx context.Context, storeID, productID int64) (*domain.ProductLocation, error) {
	type locationRow struct {
		ID           int64
		StoreID      int64
		ProductID    int64
		SKUID        *int64
		ZoneID       int64
		ZoneName     string
		ShelfID      int64
		ShelfCode    string
		ShelfName    string
		LayerNo      int
		PositionDesc string
	}

	var row locationRow
	err := r.db.WithContext(ctx).
		Table("product_location AS pl").
		Select("pl.id, pl.store_id, pl.product_id, pl.sku_id, pl.zone_id, z.name AS zone_name, pl.shelf_id, s.code AS shelf_code, s.name AS shelf_name, pl.layer_no, pl.position_desc").
		Joins("JOIN zone AS z ON z.id = pl.zone_id").
		Joins("JOIN shelf AS s ON s.id = pl.shelf_id").
		Where("pl.store_id = ? AND pl.product_id = ?", storeID, productID).
		Order("pl.id ASC").
		First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &domain.ProductLocation{
		ID:           row.ID,
		StoreID:      row.StoreID,
		ProductID:    row.ProductID,
		SKUID:        row.SKUID,
		ZoneID:       row.ZoneID,
		ZoneName:     row.ZoneName,
		ShelfID:      row.ShelfID,
		ShelfCode:    row.ShelfCode,
		ShelfName:    row.ShelfName,
		LayerNo:      row.LayerNo,
		PositionDesc: row.PositionDesc,
	}, nil
}

func (r *CustomerQARepository) GetInventory(ctx context.Context, storeID, skuID int64) (*domain.Inventory, error) {
	type inventoryRow struct {
		ID          int64
		StoreID     int64
		SKUID       int64
		ProductID   int64
		ProductName string
		SKUCode     string
		Spec        string
		Quantity    int
		SafetyStock int
		UpdatedAt   time.Time
	}

	var row inventoryRow
	err := r.db.WithContext(ctx).
		Table("inventory AS i").
		Select("i.id, i.store_id, i.sku_id, s.product_id, p.name AS product_name, s.barcode AS sku_code, s.spec, i.quantity, i.safety_stock, i.updated_at").
		Joins("JOIN sku AS s ON s.id = i.sku_id").
		Joins("JOIN product AS p ON p.id = s.product_id").
		Where("i.store_id = ? AND i.sku_id = ?", storeID, skuID).
		First(&row).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return &domain.Inventory{
		ID:          row.ID,
		StoreID:     row.StoreID,
		SKUID:       row.SKUID,
		ProductID:   row.ProductID,
		ProductName: row.ProductName,
		SKUCode:     row.SKUCode,
		Spec:        row.Spec,
		Quantity:    row.Quantity,
		SafetyStock: row.SafetyStock,
		UpdatedAt:   row.UpdatedAt,
	}, nil
}

func (r *CustomerQARepository) ListActivePromotions(ctx context.Context, storeID int64, now time.Time, limit int) ([]domain.Promotion, error) {
	var rows []PromotionModel
	if err := r.db.WithContext(ctx).
		Where("store_id = ? AND status = ? AND start_at <= ? AND end_at >= ?", storeID, "active", now, now).
		Order("start_at ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	out := make([]domain.Promotion, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.Promotion{
			ID:           row.ID,
			StoreID:      row.StoreID,
			Title:        row.Title,
			Description:  row.Description,
			ProductScope: decodeJSONList(row.ProductScope),
			StartAt:      row.StartAt,
			EndAt:        row.EndAt,
			Status:       row.Status,
		})
	}
	return out, nil
}

func decodeJSONList(raw string) []string {
	if raw == "" {
		return nil
	}

	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return items
}
