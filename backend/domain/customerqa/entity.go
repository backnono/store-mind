package customerqa

import "time"

type Session struct {
	ID        int64     `json:"id"`
	StoreID   int64     `json:"store_id"`
	UserID    *int64    `json:"user_id,omitempty"`
	Channel   string    `json:"channel"`
	StartedAt time.Time `json:"started_at"`
}

type Message struct {
	ID             int64              `json:"id"`
	SessionID      int64              `json:"session_id"`
	Role           string             `json:"role"`
	Content        string             `json:"content"`
	Intent         string             `json:"intent"`
	Confidence     *float64           `json:"confidence,omitempty"`
	ContextState   *string            `json:"context_state,omitempty"`
	FocusEntityIDs *FocusEntityIDs    `json:"focus_entity_ids,omitempty"`
	ContextStack   []ContextStackItem `json:"context_stack,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
}

// FocusEntityIDs 当前对话锁定的实体
type FocusEntityIDs struct {
	ProductIDs []int64 `json:"product_ids,omitempty"`
	SKUIDs     []int64 `json:"sku_ids,omitempty"`
	ZoneIDs    []int64 `json:"zone_ids,omitempty"`
}

// ContextStackItem 单轮对话的结构化摘要
type ContextStackItem struct {
	Turn             int              `json:"turn"`
	Intent           string           `json:"intent"`
	ResolvedEntities []ResolvedEntity `json:"resolved_entities,omitempty"`
	SystemAction     string           `json:"system_action"`
	SystemSummary    string           `json:"system_summary"`
}

// ResolvedEntity 消解后的实体
type ResolvedEntity struct {
	Type      string `json:"type"` // product / sku / zone / category
	Name      string `json:"name"`
	ProductID *int64 `json:"product_id,omitempty"`
	SKUID     *int64 `json:"sku_id,omitempty"`
}

type FAQ struct {
	ID       int64    `json:"id"`
	StoreID  int64    `json:"store_id"`
	Question string   `json:"question"`
	Answer   string   `json:"answer"`
	Category string   `json:"category"`
	Keywords []string `json:"keywords,omitempty"`
	Status   string   `json:"status"`
}

type Product struct {
	ID       int64    `json:"id"`
	Name     string   `json:"name"`
	Brand    string   `json:"brand"`
	Category string   `json:"category"`
	Aliases  []string `json:"aliases,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Status   string   `json:"status"`
}

type ProductLocation struct {
	ID           int64  `json:"id"`
	StoreID      int64  `json:"store_id"`
	ProductID    int64  `json:"product_id"`
	SKUID        *int64 `json:"sku_id,omitempty"`
	ZoneID       int64  `json:"zone_id"`
	ZoneName     string `json:"zone_name"`
	ShelfID      int64  `json:"shelf_id"`
	ShelfCode    string `json:"shelf_code"`
	ShelfName    string `json:"shelf_name"`
	LayerNo      int    `json:"layer_no"`
	PositionDesc string `json:"position_desc"`
}

type Inventory struct {
	ID             int64      `json:"id"`
	StoreID        int64      `json:"store_id"`
	SKUID          int64      `json:"sku_id"`
	ProductID      int64      `json:"product_id"`
	ProductName    string     `json:"product_name"`
	SKUCode        string     `json:"sku_code"`
	Spec           string     `json:"spec"`
	Price          float64    `json:"price"` // S2.3: SKU 价格
	Quantity       int        `json:"quantity"`
	SafetyStock    int        `json:"safety_stock"`
	LastVerifiedAt *time.Time `json:"last_verified_at,omitempty"`
	UpdateSource   *string    `json:"update_source,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type Promotion struct {
	ID           int64     `json:"id"`
	StoreID      int64     `json:"store_id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	ProductScope []string  `json:"product_scope,omitempty"`
	StartAt      time.Time `json:"start_at"`
	EndAt        time.Time `json:"end_at"`
	Status       string    `json:"status"`
}

type Feedback struct {
	ID            int64     `json:"id"`
	MessageID     int64     `json:"message_id"`
	SessionID     int64     `json:"session_id"`
	FeedbackValue int8      `json:"feedback_value"` // 1=👍 / 0=👎
	CreatedAt     time.Time `json:"created_at"`
}

type ToolCall struct {
	ID           int64     `json:"id"`
	SessionID    int64     `json:"session_id"`
	MessageID    int64     `json:"message_id"`
	ToolName     string    `json:"tool_name"`
	InputJSON    string    `json:"input_json"`
	OutputJSON   string    `json:"output_json"`
	LatencyMS    int       `json:"latency_ms"`
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// ---------- Admin resource entities ----------

type Store struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Zone struct {
	ID          int64  `json:"id"`
	StoreID     int64  `json:"store_id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Shelf struct {
	ID          int64  `json:"id"`
	StoreID     int64  `json:"store_id"`
	ZoneID      int64  `json:"zone_id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type SKU struct {
	ID        int64   `json:"id"`
	ProductID int64   `json:"product_id"`
	Barcode   string  `json:"barcode"`
	Spec      string  `json:"spec"`
	Price     float64 `json:"price"`
	Status    string  `json:"status"`
}
