package customerqa

import (
	"context"
	"fmt"
	"strings"
	"time"

	domain "store-mind/domain/customerqa"
)

// —— 请求 / 响应 DTO ——

// ChatRequest 对话请求，承载单轮 Chat 的输入参数。
// StoreID + Message 为必填；SessionID=0 时会自动创建新会话。
type ChatRequest struct {
	RequestID string // 请求追踪 ID，用于日志关联
	StoreID   int64  // 门店 ID，必填
	SessionID int64  // 会话 ID，0 表示创建新会话
	UserID    *int64 // 用户 ID，可选（未登录时为 nil）
	Channel   string // 渠道标识，默认 "miniapp"
	Message   string // 用户输入消息，必填

	// S1: 入口适配
	EntryMode string // 入口模式：first_open / zone_scan / resume / product_detail / promo
	ZoneID    *int64 // 货架区域 ID（zone_scan 入口时传入）
	ShelfID   *int64 // 货架 ID（zone_scan 入口时传入）
}

// ChatResponse 对话响应，一次 Chat 请求的完整输出。
// 包含 AI 回答、结构化卡片、引导建议、以及诊断元信息。
type ChatResponse struct {
	SessionID       int64            `json:"session_id"`       // 所属会话 ID
	MessageID       int64            `json:"message_id"`       // 本条 AI 回复的消息 ID
	Intent          string           `json:"intent"`           // 最终识别出的意图标签
	Answer          string           `json:"answer"`           // 自然语言回答
	Cards           []ChatCard       `json:"cards"`            // 结构化卡片（商品、库存、活动等）
	GuidanceChips   []GuidanceChip   `json:"guidance_chips"`   // 建议追问列表
	HandoffRequired bool             `json:"handoff_required"` // 是否需要转人工
	Meta            ChatResponseMeta `json:"meta,omitempty"`   // 调试 / 诊断元信息
}

// ChatResponseMeta 对话响应的诊断元信息，便于排查路由和置信度问题。
type ChatResponseMeta struct {
	Route         string  `json:"route,omitempty"`          // 路由类型（tool / rag / hybrid / fallback / agent）
	Confidence    float64 `json:"confidence,omitempty"`     // 意图置信度 0-1
	RewriteQuery  string  `json:"rewrite_query,omitempty"`  // LLM 改写后的检索查询
	FallbackUsed  bool    `json:"fallback_used,omitempty"`  // 是否走了降级逻辑
	EvidenceCount int     `json:"evidence_count,omitempty"` // 召回的证据条数
}

// ChatCard 结构化卡片，根据 Type 不同携带不同字段。
// Type=["product","inventory","promotion","faq"]
type ChatCard struct {
	Type     string `json:"type"`               // 卡片类型
	SKUID    int64  `json:"sku_id,omitempty"`   // [inventory] SKU ID
	Name     string `json:"name,omitempty"`     // [product,inventory] 商品名称
	Location string `json:"location,omitempty"` // [product,inventory] 货架位置描述
	Quantity int    `json:"quantity,omitempty"` // [inventory] 库存数量
	Title    string `json:"title,omitempty"`    // [promotion,faq] 标题
	Content  string `json:"content,omitempty"`  // [promotion,faq] 内容
	Validity string `json:"validity,omitempty"` // [promotion] 有效期
}

// FAQSearchRequest FAQ 检索请求。
type FAQSearchRequest struct {
	RequestID string
	StoreID   int64
	Query     string // 检索关键词
	Limit     int    // 最大返回条数，1-50，默认 10
}

// ProductSearchRequest 商品搜索请求。
type ProductSearchRequest struct {
	RequestID string
	StoreID   int64
	Query     string // 检索关键词
	Limit     int    // 最大返回条数，1-50，默认 10
}

// ProductLocationRequest 商品位置查询请求。
type ProductLocationRequest struct {
	RequestID string
	StoreID   int64
	ProductID int64 // 商品 ID
}

// InventoryRequest 库存查询请求。
type InventoryRequest struct {
	RequestID string
	StoreID   int64
	SKUID     int64 // SKU ID
}

// PromotionListRequest 活动列表查询请求。
type PromotionListRequest struct {
	RequestID string
	StoreID   int64
	Limit     int       // 最大返回条数，1-50，默认 10
	Now       time.Time // 当前时间，用于判断活动是否在有效期；为零值时取 time.Now()
}

// SessionListRequest 会话列表查询请求。
type SessionListRequest struct {
	RequestID string
	StoreID   int64
	Limit     int // 最大返回条数，1-100，默认 20
}

// ToolCallListRequest 工具调用记录查询请求。
type ToolCallListRequest struct {
	RequestID string
	SessionID int64
	Limit     int // 最大返回条数，1-100，默认 20
}

// —— 应用服务接口 ——

// Service 是 customer-qa 子域的应用服务入口。
type Service interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	SearchFAQ(ctx context.Context, req FAQSearchRequest) ([]domain.FAQ, error)
	SearchProducts(ctx context.Context, req ProductSearchRequest) ([]domain.Product, error)
	GetProductLocation(ctx context.Context, req ProductLocationRequest) (*domain.ProductLocation, error)
	GetInventory(ctx context.Context, req InventoryRequest) (*domain.Inventory, error)
	ListActivePromotions(ctx context.Context, req PromotionListRequest) ([]domain.Promotion, error)
	ListSessions(ctx context.Context, req SessionListRequest) ([]domain.Session, error)
	ListToolCalls(ctx context.Context, req ToolCallListRequest) ([]domain.ToolCall, error)
	SaveFeedback(ctx context.Context, messageID, sessionID int64, feedbackValue int8) error
}

// —— service 实现 ——

// service 是 Service 接口的默认实现。
// Agent 循环架构：用 Agent（LLM tool calling）替代了 orchestrator + contextResolver。
type service struct {
	repo        domain.Repository
	log         Logger
	agent       *Agent       // Agent 循环（LLM tool calling）
	fallback    Orchestrator // 降级编排器（LLM 不可用时）
	guideEngine GuideEngine  // 引导引擎
}

// ServiceConfig Agent 循环模式的服务注入配置。
type ServiceConfig struct {
	Repo        domain.Repository
	Log         Logger
	Agent       *Agent
	Fallback    Orchestrator
	GuideEngine GuideEngine
}

// NewService 创建仅使用降级编排器的应用服务（无 LLM）。
func NewService(repo domain.Repository, log Logger) Service {
	return NewServiceWithConfig(ServiceConfig{
		Repo: repo,
		Log:  log,
	})
}

// NewServiceWithConfig 使用完整配置创建应用服务（Agent 循环模式）。
func NewServiceWithConfig(cfg ServiceConfig) Service {
	log := cfg.Log
	if log == nil {
		log = nopLogger{}
	}
	fallback := cfg.Fallback
	if fallback == nil {
		fallback = newFallbackOrchestrator(cfg.Repo, log)
	}
	return &service{
		repo:        cfg.Repo,
		log:         log,
		agent:       cfg.Agent,
		fallback:    fallback,
		guideEngine: cfg.GuideEngine,
	}
}

// UsesPrimaryOrchestrator 判断当前 service 是否使用了 Agent 循环（即具备 LLM 能力）。
func UsesPrimaryOrchestrator(svc Service) bool {
	impl, ok := svc.(*service)
	if !ok || impl.agent == nil {
		return false
	}
	return impl.agent.UsesLLM()
}

// —— Chat 核心方法（Agent 循环版）——

// Chat 处理单轮对话的 Agent 循环流程：
//
//	① 参数校验 & Session 解析
//	② 入口适配（first_open / zone_scan / resume / promo / product_detail）
//	③ 持久化用户消息
//	④ 加载历史 → Agent 循环
//	⑤ LLM 不可用时退回到 fallback
//	⑥ 持久化 AI 回复 + 引导芯片
func (s *service) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// ① 参数校验
	if req.StoreID <= 0 || strings.TrimSpace(req.Message) == "" {
		s.log.Warn("app_chat_invalid_argument", "request_id", req.RequestID, "store_id", req.StoreID)
		return nil, domain.ErrInvalidArgument
	}
	if strings.TrimSpace(req.Channel) == "" {
		req.Channel = "miniapp"
	}
	req.Message = strings.TrimSpace(req.Message)

	s.log.Info("app_chat_start",
		"request_id", req.RequestID, "store_id", req.StoreID,
		"channel", req.Channel, "entry_mode", req.EntryMode,
	)

	// 解析或创建 Session
	session, err := s.resolveSession(ctx, req)
	if err != nil {
		s.log.Error("app_chat_resolve_session_failed", "request_id", req.RequestID, "error", err)
		return nil, err
	}

	// ② 入口适配（首轮消息时触发）
	isFirst := s.isFirstMessageInSession(ctx, session.ID)
	if isFirst {
		switch req.EntryMode {
		case "first_open":
			return s.entryFirstOpen(ctx, req, session)
		case "zone_scan":
			return s.entryZoneScan(ctx, req, session)
		case "resume":
			return s.entryResume(ctx, req, session)
		case "promo":
			return s.entryPromo(ctx, req, session)
		case "product_detail":
			return s.entryProductDetail(ctx, req, session)
		}
	}

	// ③ 持久化用户消息
	msg, err := s.repo.CreateMessage(ctx, &domain.Message{
		SessionID: session.ID,
		Role:      "user",
		Content:   req.Message,
	})
	if err != nil {
		s.log.Error("app_chat_create_message_failed", "request_id", req.RequestID, "session_id", session.ID, "error", err)
		return nil, err
	}

	// ④ Agent 循环（尝试 LLM tool calling）
	loopResult := s.runAgentLoop(ctx, req, session.ID, msg.ID)

	// ⑤ 引导芯片
	var guidanceChips []GuidanceChip
	if s.guideEngine != nil {
		guidanceChips = s.guideEngine.Evaluate(GuideContext{
			Intent:   loopResult.Intent,
			Message:  req.Message,
			Products: extractGuideProducts(loopResult.Cards),
		})
	}

	// ⑥ 持久化 AI 回复
	assistantMsg, err := s.repo.CreateMessage(ctx, &domain.Message{
		SessionID: session.ID,
		Role:      "assistant",
		Content:   loopResult.Answer,
		Intent:    loopResult.Intent,
	})
	if err != nil {
		s.log.Error("app_chat_create_assistant_message_failed", "request_id", req.RequestID, "session_id", session.ID, "error", err)
		return nil, err
	}

	s.log.Info("app_chat_success",
		"request_id", req.RequestID, "session_id", session.ID,
		"message_id", assistantMsg.ID, "route", loopResult.Route,
	)
	return &ChatResponse{
		SessionID:       session.ID,
		MessageID:       assistantMsg.ID,
		Intent:          loopResult.Intent,
		Answer:          loopResult.Answer,
		Cards:           loopResult.Cards,
		GuidanceChips:   guidanceChips,
		HandoffRequired: loopResult.Handoff,
		Meta: ChatResponseMeta{
			Route:         loopResult.Route,
			Confidence:    loopResult.Confidence,
			FallbackUsed:  !loopResult.AgentUsed,
			EvidenceCount: loopResult.EvidenceCount,
		},
	}, nil
}

// agentLoopResult Agent 循环的内部结果，用于传递给 GuideEngine 和组装 ChatResponse。
type agentLoopResult struct {
	Answer        string
	Cards         []ChatCard
	Intent        string
	Handoff       bool
	AgentUsed     bool
	Confidence    float64
	Route         string
	EvidenceCount int
}

// runAgentLoop 尝试通过 Agent 循环处理用户消息。
func (s *service) runAgentLoop(
	ctx context.Context,
	req ChatRequest,
	sessionID, messageID int64,
) agentLoopResult {
	// 无 Agent（无 LLM）→ 直接走 fallback
	if s.agent == nil || !s.agent.UsesLLM() {
		result, err := s.fallback.Run(ctx, OrchestratorRequest{
			RequestID: req.RequestID,
			StoreID:   req.StoreID,
			SessionID: sessionID,
			MessageID: messageID,
			UserID:    req.UserID,
			Channel:   req.Channel,
			Message:   req.Message,
		})
		if err != nil {
			s.log.Error("app_chat_fallback_failed", "request_id", req.RequestID, "error", err)
			return agentLoopResult{
				Answer: "暂时无法处理你的问题，请稍后再试或联系人工客服。",
				Intent: "fallback",
				Route:  "fallback",
			}
		}
		return agentLoopResult{
			Answer:        result.Answer,
			Cards:         result.Cards,
			Intent:        result.Decision.Intent,
			Handoff:       result.Decision.NeedsHandoff,
			Confidence:    result.Decision.Confidence,
			Route:         "fallback",
			EvidenceCount: len(result.Evidence),
		}
	}

	// 加载消息历史
	history, _ := s.repo.ListRecentMessages(ctx, sessionID, 20)
	agentHistory := ConvertToAgentMessages(history)

	// 追加本轮用户消息
	agentHistory = append(agentHistory, AgentMessage{
		Role:    "user",
		Content: req.Message,
	})

	// 运行 Agent 循环
	result, err := s.agent.Run(ctx, AgentRunRequest{
		StoreID:     req.StoreID,
		SessionID:   sessionID,
		MessageID:   messageID,
		History:     agentHistory,
		UserMessage: req.Message,
	})
	if err != nil {
		s.log.Warn("app_chat_agent_failed", "request_id", req.RequestID, "error", err)
		// 退回到 fallback
		fbResult, fbErr := s.fallback.Run(ctx, OrchestratorRequest{
			RequestID: req.RequestID,
			StoreID:   req.StoreID,
			SessionID: sessionID,
			MessageID: messageID,
			UserID:    req.UserID,
			Channel:   req.Channel,
			Message:   req.Message,
		})
		if fbErr != nil {
			return agentLoopResult{
				Answer: "暂时无法处理你的问题，请稍后再试或联系人工客服。",
				Intent: "fallback",
				Route:  "fallback",
			}
		}
		return agentLoopResult{
			Answer:        fbResult.Answer,
			Cards:         fbResult.Cards,
			Intent:        fbResult.Decision.Intent,
			Handoff:       fbResult.Decision.NeedsHandoff,
			Confidence:    fbResult.Decision.Confidence,
			Route:         "fallback",
			EvidenceCount: len(fbResult.Evidence),
		}
	}

	// Agent 循环成功 → 持久化中间消息（tool call + tool result）
	s.persistAgentMessages(ctx, sessionID, result.UpdatedHistory)

	return agentLoopResult{
		Answer:    result.FinalAnswer,
		Cards:     result.Cards,
		Intent:    "agent",
		AgentUsed: true,
		Route:     "agent",
	}
}

// persistAgentMessages 持久化 Agent 循环中产生的 tool 和 assistant 消息。
// 用户消息已在循环前持久化，这里只持久化新增的中间消息。
func (s *service) persistAgentMessages(ctx context.Context, sessionID int64, history []AgentMessage) {
	domainMsgs := ConvertToDomainMessages(sessionID, history)
	for _, m := range domainMsgs {
		// 跳过已在循环前持久化的 user 消息
		if m.Role == "user" && m.ID != 0 {
			continue
		}
		if _, err := s.repo.CreateMessage(ctx, &m); err != nil {
			s.log.Warn("agent_persist_message_failed",
				"role", m.Role,
				"session_id", sessionID,
				"error", err,
			)
		}
	}
}

// isFirstMessageInSession 判断会话是否为首条消息（无历史 assistant 消息）。
func (s *service) isFirstMessageInSession(ctx context.Context, sessionID int64) bool {
	msgs, err := s.repo.ListRecentMessages(ctx, sessionID, 5)
	if err != nil {
		return true
	}
	for _, m := range msgs {
		if m.Role == "assistant" {
			return false
		}
	}
	return true
}

// resolveSession 解析或创建会话。
func (s *service) resolveSession(ctx context.Context, req ChatRequest) (*domain.Session, error) {
	if req.SessionID > 0 {
		session, err := s.repo.GetSession(ctx, req.SessionID)
		if err != nil {
			return nil, err
		}
		if session.StoreID != req.StoreID {
			return nil, domain.ErrInvalidArgument
		}
		return session, nil
	}
	return s.repo.CreateSession(ctx, &domain.Session{
		StoreID: req.StoreID,
		UserID:  req.UserID,
		Channel: req.Channel,
	})
}

// —— 入口适配方法 ——

// entryFirstOpen 首次破冰入口。
func (s *service) entryFirstOpen(ctx context.Context, req ChatRequest, session *domain.Session) (*ChatResponse, error) {
	answer := "您好！我是小王，您身边的数字店员。\n\n店里的一切都可以问我：商品在哪儿、还有没有货、今天有什么活动、怎么付款……"
	guidanceChips := []GuidanceChip{
		{Text: "📍 薯片在哪里？", Prompt: "薯片在哪里？"},
		{Text: "🏷 今天有什么活动？", Prompt: "今天有什么活动？"},
		{Text: "🥤 低糖饮料有哪些？", Prompt: "低糖饮料有哪些？"},
		{Text: "💳 怎么付款？", Prompt: "怎么付款？"},
	}
	msg, err := s.repo.CreateMessage(ctx, &domain.Message{
		SessionID: session.ID,
		Role:      "assistant",
		Content:   answer,
		Intent:    "greeting",
	})
	if err != nil {
		return nil, err
	}
	return &ChatResponse{
		SessionID:     session.ID,
		MessageID:     msg.ID,
		Intent:        "greeting",
		Answer:        answer,
		GuidanceChips: guidanceChips,
		Meta:          ChatResponseMeta{Route: "entry_first_open"},
	}, nil
}

// entryZoneScan 货架扫码入口。
func (s *service) entryZoneScan(ctx context.Context, req ChatRequest, session *domain.Session) (*ChatResponse, error) {
	products, err := s.repo.ListProductsByLocation(ctx, req.StoreID, req.ZoneID, req.ShelfID, 10)
	if err != nil {
		s.log.Warn("app_entry_zone_scan_list_failed", "error", err)
		products, _ = s.repo.SearchProducts(ctx, req.StoreID, "", 5)
	}
	if len(products) == 0 {
		products, _ = s.repo.SearchProducts(ctx, req.StoreID, "", 5)
	}

	zoneLabel := "当前区域"
	if req.ZoneID != nil {
		zoneLabel = fmt.Sprintf("区域 %d", *req.ZoneID)
	}
	answer := "我看到您在" + zoneLabel + "，这个货架主要有："
	cards := make([]ChatCard, 0, len(products))
	for _, p := range products {
		loc, err := s.repo.GetProductLocation(ctx, req.StoreID, p.ID)
		if err == nil {
			cards = append(cards, ChatCard{
				Type:     "product",
				Name:     p.Name,
				Location: loc.ZoneName + " " + loc.ShelfCode + " 货架",
			})
		}
	}

	msg, err := s.repo.CreateMessage(ctx, &domain.Message{
		SessionID: session.ID,
		Role:      "assistant",
		Content:   answer,
		Intent:    "zone_scan",
	})
	if err != nil {
		return nil, err
	}
	return &ChatResponse{
		SessionID: session.ID,
		MessageID: msg.ID,
		Intent:    "zone_scan",
		Answer:    answer,
		Cards:     cards,
		Meta:      ChatResponseMeta{Route: "entry_zone_scan"},
	}, nil
}

// entryResume 历史会话恢复入口。
func (s *service) entryResume(ctx context.Context, req ChatRequest, session *domain.Session) (*ChatResponse, error) {
	answer := "欢迎回来！有什么需要帮您的吗？"
	msg, err := s.repo.CreateMessage(ctx, &domain.Message{
		SessionID: session.ID,
		Role:      "assistant",
		Content:   answer,
		Intent:    "resume",
	})
	if err != nil {
		return nil, err
	}
	return &ChatResponse{
		SessionID: session.ID,
		MessageID: msg.ID,
		Intent:    "resume",
		Answer:    answer,
		Meta:      ChatResponseMeta{Route: "entry_resume"},
	}, nil
}

// entryPromo 活动入口。
func (s *service) entryPromo(ctx context.Context, req ChatRequest, session *domain.Session) (*ChatResponse, error) {
	items, err := s.repo.ListActivePromotions(ctx, req.StoreID, time.Now(), 5)
	if err != nil || len(items) == 0 {
		return s.entryFirstOpen(ctx, req, session)
	}

	answer := "今天的活动是" + items[0].Title + "，参与商品有："
	cards := []ChatCard{{
		Type:     "promotion",
		Title:    items[0].Title,
		Content:  items[0].Description,
		Validity: items[0].EndAt.Format("01-02 15:04"),
	}}
	guidanceChips := []GuidanceChip{
		{Text: "📍 活动商品在哪里？", Prompt: "活动商品在哪里？"},
	}

	msg, err := s.repo.CreateMessage(ctx, &domain.Message{
		SessionID: session.ID,
		Role:      "assistant",
		Content:   answer,
		Intent:    "promotion",
	})
	if err != nil {
		return nil, err
	}
	return &ChatResponse{
		SessionID:     session.ID,
		MessageID:     msg.ID,
		Intent:        "promotion",
		Answer:        answer,
		Cards:         cards,
		GuidanceChips: guidanceChips,
		Meta:          ChatResponseMeta{Route: "entry_promo"},
	}, nil
}

// entryProductDetail 商品详情入口。
func (s *service) entryProductDetail(ctx context.Context, req ChatRequest, session *domain.Session) (*ChatResponse, error) {
	products, err := s.repo.SearchProducts(ctx, req.StoreID, req.Message, 1)
	if err != nil || len(products) == 0 {
		return s.entryFirstOpen(ctx, req, session)
	}

	p := products[0]
	loc, _ := s.repo.GetProductLocation(ctx, req.StoreID, p.ID)
	locStr := ""
	if loc != nil {
		locStr = loc.ZoneName + " " + loc.ShelfCode + " 货架"
	}
	answer := "为您找到了这个："
	cards := []ChatCard{{
		Type:     "product",
		Name:     p.Name,
		Location: locStr,
	}}
	if loc != nil && loc.SKUID != nil {
		cards[0].SKUID = *loc.SKUID
	}
	guidanceChips := []GuidanceChip{
		{Text: "📦 还有几瓶？", Prompt: "还有几瓶？"},
		{Text: "🥤 同品类还有什么？", Prompt: "同品类还有什么？"},
	}

	msg, err := s.repo.CreateMessage(ctx, &domain.Message{
		SessionID: session.ID,
		Role:      "assistant",
		Content:   answer,
		Intent:    "product_detail",
	})
	if err != nil {
		return nil, err
	}
	return &ChatResponse{
		SessionID:     session.ID,
		MessageID:     msg.ID,
		Intent:        "product_detail",
		Answer:        answer,
		Cards:         cards,
		GuidanceChips: guidanceChips,
		Meta:          ChatResponseMeta{Route: "entry_product_detail"},
	}, nil
}

// —— 其他服务方法 ——

// SearchFAQ 按门店和关键词检索 FAQ。
func (s *service) SearchFAQ(ctx context.Context, req FAQSearchRequest) ([]domain.FAQ, error) {
	if req.StoreID <= 0 || strings.TrimSpace(req.Query) == "" {
		return nil, domain.ErrInvalidArgument
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 50 {
		req.Limit = 50
	}
	return s.repo.SearchFAQ(ctx, req.StoreID, req.Query, req.Limit)
}

// SearchProducts 按门店和关键词检索商品。
func (s *service) SearchProducts(ctx context.Context, req ProductSearchRequest) ([]domain.Product, error) {
	if req.StoreID <= 0 || strings.TrimSpace(req.Query) == "" {
		return nil, domain.ErrInvalidArgument
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 50 {
		req.Limit = 50
	}
	return s.repo.SearchProducts(ctx, req.StoreID, req.Query, req.Limit)
}

// GetProductLocation 查询指定商品在门店的具体位置。
func (s *service) GetProductLocation(ctx context.Context, req ProductLocationRequest) (*domain.ProductLocation, error) {
	if req.StoreID <= 0 || req.ProductID <= 0 {
		return nil, domain.ErrInvalidArgument
	}
	return s.repo.GetProductLocation(ctx, req.StoreID, req.ProductID)
}

// GetInventory 查询指定 SKU 的库存数量。
func (s *service) GetInventory(ctx context.Context, req InventoryRequest) (*domain.Inventory, error) {
	if req.StoreID <= 0 || req.SKUID <= 0 {
		return nil, domain.ErrInvalidArgument
	}
	return s.repo.GetInventory(ctx, req.StoreID, req.SKUID)
}

// ListActivePromotions 列出门店当前在有效期内的活动。
func (s *service) ListActivePromotions(ctx context.Context, req PromotionListRequest) ([]domain.Promotion, error) {
	if req.StoreID <= 0 {
		return nil, domain.ErrInvalidArgument
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 50 {
		req.Limit = 50
	}
	if req.Now.IsZero() {
		req.Now = time.Now()
	}
	return s.repo.ListActivePromotions(ctx, req.StoreID, req.Now, req.Limit)
}

// ListSessions 分页获取门店的会话列表。
func (s *service) ListSessions(ctx context.Context, req SessionListRequest) ([]domain.Session, error) {
	if req.StoreID <= 0 {
		return nil, domain.ErrInvalidArgument
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	return s.repo.ListSessions(ctx, req.StoreID, req.Limit)
}

// ListToolCalls 查询指定会话的工具调用记录。
func (s *service) ListToolCalls(ctx context.Context, req ToolCallListRequest) ([]domain.ToolCall, error) {
	if req.SessionID <= 0 {
		return nil, domain.ErrInvalidArgument
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	return s.repo.ListToolCalls(ctx, req.SessionID, req.Limit)
}

// SaveFeedback 保存用户对回答的反馈。
func (s *service) SaveFeedback(ctx context.Context, messageID, sessionID int64, feedbackValue int8) error {
	if messageID <= 0 || sessionID <= 0 {
		return domain.ErrInvalidArgument
	}
	if feedbackValue != 0 && feedbackValue != 1 {
		return domain.ErrInvalidArgument
	}
	feedbackRepo, ok := s.repo.(domain.FeedbackRepository)
	if !ok {
		return domain.ErrInvalidArgument
	}
	_, err := feedbackRepo.CreateFeedback(ctx, &domain.Feedback{
		MessageID:     messageID,
		SessionID:     sessionID,
		FeedbackValue: feedbackValue,
	})
	return err
}

// extractGuideProducts 从 ChatCard 列表中提取商品信息，供 GuideEngine 使用。
func extractGuideProducts(cards []ChatCard) []domain.Product {
	products := make([]domain.Product, 0, len(cards))
	for _, card := range cards {
		if card.Type == "product" || card.Type == "inventory" || card.Type == "price" {
			products = append(products, domain.Product{Name: card.Name})
		}
	}
	return products
}
