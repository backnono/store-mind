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
	ID             int64     `gorm:"primaryKey;autoIncrement"`
	SessionID      int64     `gorm:"not null;index"`
	Role           string    `gorm:"type:varchar(32);not null"`
	Content        string    `gorm:"type:text;not null"`
	Intent         string    `gorm:"type:varchar(64);not null;default:''"`
	Confidence     *float64  `gorm:"type:decimal(5,4)"`
	ContextState   *string   `gorm:"type:varchar(32)"`
	FocusEntityIDs *string   `gorm:"type:json"`
	ContextStack   *string   `gorm:"type:json"`
	ToolCallsJSON  *string   `gorm:"type:json"`
	ToolCallID     *string   `gorm:"type:varchar(64)"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
}

func (MessageModel) TableName() string { return "agent_message" }

type FeedbackModel struct {
	ID            int64     `gorm:"primaryKey;autoIncrement"`
	MessageID     int64     `gorm:"not null;index"`
	SessionID     int64     `gorm:"not null;index"`
	FeedbackValue int8      `gorm:"not null"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

func (FeedbackModel) TableName() string { return "agent_feedback" }

type DecisionLogModel struct {
	ID              int64     `gorm:"primaryKey;autoIncrement"`
	SessionID       int64     `gorm:"not null;index"`
	MessageID       int64     `gorm:"not null;index"`
	Intent          string    `gorm:"type:varchar(64);not null"`
	Route           string    `gorm:"type:varchar(32);not null"`
	RewriteQuery    *string   `gorm:"type:varchar(512)"`
	Confidence      float64   `gorm:"type:decimal(5,4);not null"`
	FallbackUsed    bool      `gorm:"not null"`
	HandoffRequired bool      `gorm:"not null"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
}

func (DecisionLogModel) TableName() string { return "agent_decision_log" }

type ToolCallModel struct {
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	SessionID    int64     `gorm:"not null;index"`
	MessageID    int64     `gorm:"not null;index"`
	ToolName     string    `gorm:"type:varchar(128);not null"`
	InputJSON    string    `gorm:"type:json"`
	OutputJSON   string    `gorm:"type:json"`
	LatencyMS    int       `gorm:"not null"`
	Success      bool      `gorm:"not null"`
	ErrorMessage string    `gorm:"type:text"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
}

func (ToolCallModel) TableName() string { return "agent_tool_call" }

type FAQModel struct {
	ID       int64  `gorm:"primaryKey;autoIncrement"`
	StoreID  int64  `gorm:"not null;index"`
	Question string `gorm:"type:varchar(255);not null"`
	Answer   string `gorm:"type:text;not null"`
	Category string `gorm:"type:varchar(64);not null"`
	Keywords string `gorm:"type:json"`
	Status   string `gorm:"type:varchar(32);not null"`
}

func (FAQModel) TableName() string { return "faq" }

type StoreModel struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	Name      string    `gorm:"type:varchar(255);not null"`
	Address   string    `gorm:"type:varchar(255);not null"`
	Status    string    `gorm:"type:varchar(32);not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (StoreModel) TableName() string { return "store" }

type ZoneModel struct {
	ID          int64  `gorm:"primaryKey;autoIncrement"`
	StoreID     int64  `gorm:"not null;index"`
	Code        string `gorm:"type:varchar(64);not null"`
	Name        string `gorm:"type:varchar(255);not null"`
	Description string `gorm:"type:varchar(255);not null"`
}

func (ZoneModel) TableName() string { return "zone" }

type ShelfModel struct {
	ID          int64  `gorm:"primaryKey;autoIncrement"`
	StoreID     int64  `gorm:"not null;index"`
	ZoneID      int64  `gorm:"not null;index"`
	Code        string `gorm:"type:varchar(64);not null"`
	Name        string `gorm:"type:varchar(255);not null"`
	Description string `gorm:"type:varchar(255);not null"`
}

func (ShelfModel) TableName() string { return "shelf" }

type ProductModel struct {
	ID       int64  `gorm:"primaryKey;autoIncrement"`
	Name     string `gorm:"type:varchar(255);not null"`
	Brand    string `gorm:"type:varchar(255);not null"`
	Category string `gorm:"type:varchar(128);not null"`
	Aliases  string `gorm:"type:json"`
	Tags     string `gorm:"type:json"`
	Status   string `gorm:"type:varchar(32);not null"`
}

func (ProductModel) TableName() string { return "product" }

type SKUModel struct {
	ID        int64   `gorm:"primaryKey;autoIncrement"`
	ProductID int64   `gorm:"not null;index"`
	Barcode   string  `gorm:"type:varchar(64);not null"`
	Spec      string  `gorm:"type:varchar(255);not null"`
	Price     float64 `gorm:"type:decimal(10,2);not null"`
	Status    string  `gorm:"type:varchar(32);not null"`
}

func (SKUModel) TableName() string { return "sku" }

type ProductLocationModel struct {
	ID           int64  `gorm:"primaryKey;autoIncrement"`
	StoreID      int64  `gorm:"not null;index"`
	ProductID    int64  `gorm:"not null;index"`
	SKUID        *int64 `gorm:"index"`
	ZoneID       int64  `gorm:"not null;index"`
	ShelfID      int64  `gorm:"not null;index"`
	LayerNo      int    `gorm:"not null"`
	PositionDesc string `gorm:"type:varchar(255);not null"`
}

func (ProductLocationModel) TableName() string { return "product_location" }

type InventoryModel struct {
	ID             int64      `gorm:"primaryKey;autoIncrement"`
	StoreID        int64      `gorm:"not null;index"`
	SKUID          int64      `gorm:"not null;index"`
	Quantity       int        `gorm:"not null"`
	SafetyStock    int        `gorm:"not null"`
	LastVerifiedAt *time.Time `gorm:"index"`
	UpdateSource   *string    `gorm:"type:varchar(32)"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime"`
}

func (InventoryModel) TableName() string { return "inventory" }

type PromotionModel struct {
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	StoreID      int64     `gorm:"not null;index"`
	Title        string    `gorm:"type:varchar(255);not null"`
	Description  string    `gorm:"type:text;not null"`
	ProductScope string    `gorm:"type:json"`
	StartAt      time.Time `gorm:"not null"`
	EndAt        time.Time `gorm:"not null"`
	Status       string    `gorm:"type:varchar(32);not null"`
}

func (PromotionModel) TableName() string { return "promotion" }
