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
	ListRecentMessages(ctx context.Context, sessionID int64, limit int) ([]Message, error)
	SearchFAQ(ctx context.Context, storeID int64, query string, limit int) ([]FAQ, error)
	SearchProducts(ctx context.Context, storeID int64, query string, limit int) ([]Product, error)
	ListProductsByLocation(ctx context.Context, storeID int64, zoneID, shelfID *int64, limit int) ([]Product, error)
	GetProductLocation(ctx context.Context, storeID, productID int64) (*ProductLocation, error)
	GetInventory(ctx context.Context, storeID, skuID int64) (*Inventory, error)
	ListActivePromotions(ctx context.Context, storeID int64, now time.Time, limit int) ([]Promotion, error)
}

// FeedbackRepository 反馈持久化接口
type FeedbackRepository interface {
	CreateFeedback(ctx context.Context, feedback *Feedback) (*Feedback, error)
}

// AdminRepository 管理后台 CRUD 接口
type AdminRepository interface {
	// Store
	CreateStore(ctx context.Context, store *Store) (*Store, error)
	UpdateStore(ctx context.Context, store *Store) (*Store, error)
	DeleteStore(ctx context.Context, id int64) error

	// Zone
	CreateZone(ctx context.Context, zone *Zone) (*Zone, error)
	UpdateZone(ctx context.Context, zone *Zone) (*Zone, error)
	DeleteZone(ctx context.Context, id int64) error

	// Shelf
	CreateShelf(ctx context.Context, shelf *Shelf) (*Shelf, error)
	UpdateShelf(ctx context.Context, shelf *Shelf) (*Shelf, error)
	DeleteShelf(ctx context.Context, id int64) error

	// Product
	CreateProduct(ctx context.Context, product *Product) (*Product, error)
	UpdateProduct(ctx context.Context, product *Product) (*Product, error)
	DeleteProduct(ctx context.Context, id int64) error

	// SKU
	CreateSKU(ctx context.Context, sku *SKU) (*SKU, error)
	UpdateSKU(ctx context.Context, sku *SKU) (*SKU, error)
	DeleteSKU(ctx context.Context, id int64) error

	// ProductLocation
	CreateProductLocation(ctx context.Context, pl *ProductLocation) (*ProductLocation, error)
	UpdateProductLocation(ctx context.Context, pl *ProductLocation) (*ProductLocation, error)
	DeleteProductLocation(ctx context.Context, id int64) error

	// Inventory
	CreateInventory(ctx context.Context, inv *Inventory) (*Inventory, error)
	UpdateInventory(ctx context.Context, inv *Inventory) (*Inventory, error)
	DeleteInventory(ctx context.Context, id int64) error

	// Promotion
	CreatePromotion(ctx context.Context, promo *Promotion) (*Promotion, error)
	UpdatePromotion(ctx context.Context, promo *Promotion) (*Promotion, error)
	DeletePromotion(ctx context.Context, id int64) error

	// FAQ
	CreateFAQ(ctx context.Context, faq *FAQ) (*FAQ, error)
	UpdateFAQ(ctx context.Context, faq *FAQ) (*FAQ, error)
	DeleteFAQ(ctx context.Context, id int64) error
}
