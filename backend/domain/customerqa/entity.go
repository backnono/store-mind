package customerqa

import "time"

type Session struct {
	ID        int64      `json:"id"`
	StoreID   int64      `json:"store_id"`
	UserID    *int64     `json:"user_id,omitempty"`
	Channel   string     `json:"channel"`
	StartedAt time.Time  `json:"started_at"`
}

type Message struct {
	ID        int64     `json:"id"`
	SessionID int64     `json:"session_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	Intent    string    `json:"intent"`
	Confidence *float64 `json:"confidence,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type FAQ struct {
	ID       int64  `json:"id"`
	StoreID  int64  `json:"store_id"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Category string `json:"category"`
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
	ID          int64     `json:"id"`
	StoreID     int64     `json:"store_id"`
	SKUID       int64     `json:"sku_id"`
	ProductID   int64     `json:"product_id"`
	ProductName string    `json:"product_name"`
	SKUCode     string    `json:"sku_code"`
	Spec        string    `json:"spec"`
	Quantity    int       `json:"quantity"`
	SafetyStock int       `json:"safety_stock"`
	UpdatedAt   time.Time `json:"updated_at"`
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
