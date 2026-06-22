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
	Route         string  `json:"route,omitempty"`          // 路由类型（tool / rag / hybrid / fallback）
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
// 聚合了 Chat（核心对话）、FAQ/商品/库存/活动/会话/工具调用查询、反馈保存等能力。
type Service interface {
	// Chat 单轮对话：接收用户消息，完成意图识别 → 证据召回 → 答案生成 → 持久化。
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	SearchFAQ(ctx context.Context, req FAQSearchRequest) ([]domain.FAQ, error)
	SearchProducts(ctx context.Context, req ProductSearchRequest) ([]domain.Product, error)
	GetProductLocation(ctx context.Context, req ProductLocationRequest) (*domain.ProductLocation, error)
	GetInventory(ctx context.Context, req InventoryRequest) (*domain.Inventory, error)
	ListActivePromotions(ctx context.Context, req PromotionListRequest) ([]domain.Promotion, error)
	ListSessions(ctx context.Context, req SessionListRequest) ([]domain.Session, error)
	ListToolCalls(ctx context.Context, req ToolCallListRequest) ([]domain.ToolCall, error)
	// SaveFeedback 保存用户对回答的反馈（0=👎, 1=👍）。
	SaveFeedback(ctx context.Context, messageID, sessionID int64, feedbackValue int8) error
}

// service 是 Service 接口的默认实现。
// 通过 orchestrator 完成 Chat 流程编排，其余方法直接委托给 domain.Repository。
type service struct {
	repo            domain.Repository
	log             Logger
	orchestrator    Orchestrator
	sessionManager  SessionManager  // S1: 会话状态机
	contextResolver ContextResolver // S1: 指代消解器
	guideEngine     GuideEngine     // S1: 主动引导引擎
}

// ServiceConfig S1 模块注入配置。
type ServiceConfig struct {
	Repo            domain.Repository
	Log             Logger
	Orchestrator    Orchestrator
	SessionManager  SessionManager
	ContextResolver ContextResolver
	GuideEngine     GuideEngine
}

// NewService 创建默认的 customer-qa 应用服务。
// 编排器使用 fallback（关键词匹配）模式，适合 LLM 不可用时的降级场景。
func NewService(repo domain.Repository, log Logger) Service {
	return NewServiceWithOrchestrator(repo, log, nil)
}

// NewServiceWithOrchestrator 创建带编排器的应用服务。
// 当 orchestrator 为 nil 时，自动回退到 fallback 编排器（关键词路由）。
// 当 log 为 nil 时，使用 nopLogger 静默日志。
func NewServiceWithOrchestrator(repo domain.Repository, log Logger, orchestrator Orchestrator) Service {
	return NewServiceWithConfig(
		ServiceConfig{
			Repo:         repo,
			Log:          log,
			Orchestrator: orchestrator,
		},
	)
}

// NewServiceWithConfig 使用完整配置创建应用服务（S1 入口）。
// 未注入的 S1 组件为 nil 时，新流程自动降级为旧流程。
func NewServiceWithConfig(cfg ServiceConfig) Service {
	log := cfg.Log
	if log == nil {
		log = nopLogger{}
	}
	orch := cfg.Orchestrator
	if orch == nil {
		orch = newDefaultOrchestrator(cfg.Repo, log)
	}
	return &service{
		repo:            cfg.Repo,
		log:             log,
		orchestrator:    orch,
		sessionManager:  cfg.SessionManager,
		contextResolver: cfg.ContextResolver,
		guideEngine:     cfg.GuideEngine,
	}
}

// UsesPrimaryOrchestrator 判断当前 service 是否使用了主编排器（即 analyzer/composer/retriever 三者齐全）。
// 用于 bootstrap 层判断是否已接入 LLM sidecar。
func UsesPrimaryOrchestrator(svc Service) bool {
	impl, ok := svc.(*service)
	if !ok || impl.orchestrator == nil {
		return false
	}
	orch, ok := impl.orchestrator.(*defaultOrchestrator)
	if !ok {
		return false
	}
	return orch.analyzer != nil && orch.composer != nil && orch.retriever != nil
}

// Chat 处理单轮对话的完整流程（S1 升级版）：
//  1. 参数校验 & Session 解析（无则创建）
//  2. 持久化用户消息
//  3. S1: 加载会话上下文（状态机 + context_stack）
//  4. S1: 入口适配（first_open / zone_scan / resume）
//  5. 交由 orchestrator 执行意图识别 → 召回 → 答案生成
//  6. S1: GuideEngine 追加引导芯片
//  7. 持久化 AI 回复（含会话状态 + context_stack）
func (s *service) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// 参数校验
	if req.StoreID <= 0 || strings.TrimSpace(req.Message) == "" {
		s.log.Warn("app_chat_invalid_argument", "request_id", req.RequestID, "store_id", req.StoreID)
		return nil, domain.ErrInvalidArgument
	}
	if strings.TrimSpace(req.Channel) == "" {
		req.Channel = "miniapp" // 默认小程序渠道
	}
	req.Message = strings.TrimSpace(req.Message)

	s.log.Info(
		"app_chat_start",
		"request_id",
		req.RequestID,
		"store_id",
		req.StoreID,
		"channel",
		req.Channel,
		"entry_mode",
		req.EntryMode,
	)

	// 步骤 1：解析或创建 Session
	session, err := s.resolveSession(ctx, req)
	if err != nil {
		s.log.Error(
			"app_chat_resolve_session_failed",
			"request_id",
			req.RequestID,
			"session_id",
			req.SessionID,
			"error",
			err,
		)
		return nil, err
	}

	// S1: 加载会话上下文
	var sessionCtx *SessionContext
	if s.sessionManager != nil {
		sessionCtx, err = s.sessionManager.LoadSession(ctx, session.ID)
		if err != nil {
			s.log.Warn(
				"app_chat_session_load_failed",
				"request_id",
				req.RequestID,
				"session_id",
				session.ID,
				"error",
				err,
			)
		}
	}
	if sessionCtx == nil {
		sessionCtx = &SessionContext{State: StateIdle}
	}

	// S1: 入口适配 — 当 entry_mode 指定且为首轮对话时，生成适应回答
	if s.isFirstMessage(sessionCtx) {
		switch req.EntryMode {
		case "first_open":
			return s.entryFirstOpen(ctx, req, session)
		case "zone_scan":
			return s.entryZoneScan(ctx, req, session)
		case "resume":
			return s.entryResume(ctx, req, session, sessionCtx)
		case "promo":
			return s.entryPromo(ctx, req, session)
		case "product_detail":
			return s.entryProductDetail(ctx, req, session)
		}
	}

	// 步骤 2：持久化用户消息
	msg, err := s.repo.CreateMessage(
		ctx,
		&domain.Message{SessionID: session.ID, Role: "user", Content: req.Message, Intent: ""},
	)
	if err != nil {
		s.log.Error(
			"app_chat_create_message_failed",
			"request_id",
			req.RequestID,
			"session_id",
			session.ID,
			"error",
			err,
		)
		return nil, err
	}

	// S1: ContextResolver 指代消解（L1/L2/L3）
	var resolvedEntities []domain.ResolvedEntity
	if s.contextResolver != nil && !s.isFirstMessage(sessionCtx) {
		resolveResult, err := s.contextResolver.Resolve(ctx, ResolveRequest{
			Message:       req.Message,
			SessionState:  sessionCtx.State,
			FocusEntities: sessionCtx.FocusEntityIDs,
			ContextStack:  sessionCtx.ContextStack,
		})
		if err != nil {
			s.log.Warn("app_chat_resolve_failed", "request_id", req.RequestID, "error", err)
		} else if resolveResult != nil && resolveResult.NeedsClarify {
			// L3: 需要澄清 → 直接返回澄清话术
			return s.clarifyResponse(ctx, session, resolveResult)
		} else if resolveResult != nil {
			resolvedEntities = resolveResult.ResolvedEntities
		}
	}

	// 步骤 3：编排执行
	orchReq := OrchestratorRequest{
		RequestID:        req.RequestID,
		StoreID:          req.StoreID,
		SessionID:        session.ID,
		MessageID:        msg.ID,
		UserID:           req.UserID,
		Channel:          req.Channel,
		Message:          req.Message,
		SessionContext:   sessionCtx,       // S1: 传入会话上下文
		ResolvedEntities: resolvedEntities, // S1: 传入消解后的实体
	}
	result, err := s.orchestrator.Run(ctx, orchReq)
	if err != nil {
		s.log.Error("app_chat_orchestrator_failed", "request_id", req.RequestID, "session_id", session.ID, "error", err)
		return nil, err
	}

	// S1: GuideEngine 追加引导芯片（传入 Products 供替代推荐等）
	guidanceChips := result.GuidanceChips
	if s.guideEngine != nil && len(guidanceChips) == 0 {
		products := s.extractProductsFromEvidence(result.Evidence)
		// fallback orchestrator 不产生 Evidence，从 Cards 补充
		if len(products) == 0 {
			products = s.extractProductsFromCards(result.Cards)
		}
		guidanceChips = s.guideEngine.Evaluate(
			GuideContext{
				Intent:       result.Decision.Intent,
				Decision:     result.Decision,
				Message:      req.Message,
				Evidence:     result.Evidence,
				SessionState: sessionCtx.State,
				Products:     products,
			},
		)
	}

	// S1: 更新 FocusEntityIDs — 从消解结果中提取
	newFocusIDs := buildFocusFromResolution(resolvedEntities, sessionCtx.FocusEntityIDs)

	// S1: 计算新的会话状态和 context_stack
	newState := StateTransition(sessionCtx.State, result.Decision.Intent, req.Message)
	stateStr := string(newState)
	turnSummary := BuildTurnSummary(
		sessionCtx.ContextStack,
		result.Decision.Intent,
		resolvedEntities,
		result.Decision.Route,
		result.Answer,
	)
	newStack := AppendContextStack(sessionCtx.ContextStack, turnSummary, 5)

	// 序列化
	ctxStackJSON, _ := MarshalContextStack(newStack)
	focusJSON, _ := MarshalFocusEntityIDs(newFocusIDs)

	// 步骤 4：持久化 AI 回复（含会话状态）
	_ = ctxStackJSON // 序列化已在 CreateMessage 内通过 Marshal 处理
	_ = focusJSON
	assistantMsg, err := s.repo.CreateMessage(
		ctx, &domain.Message{
			SessionID:      session.ID,
			Role:           "assistant",
			Content:        result.Answer,
			Intent:         result.Decision.Intent,
			ContextState:   &stateStr,
			FocusEntityIDs: newFocusIDs,
			ContextStack:   newStack,
		},
	)
	if err != nil {
		s.log.Error(
			"app_chat_create_assistant_message_failed",
			"request_id",
			req.RequestID,
			"session_id",
			session.ID,
			"error",
			err,
		)
		return nil, err
	}

	// 步骤 5：持久化决策日志
	s.persistDecisionLog(ctx, session.ID, msg.ID, result.Decision)

	s.log.Info(
		"app_chat_success",
		"request_id",
		req.RequestID,
		"session_id",
		session.ID,
		"message_id",
		assistantMsg.ID,
		"intent",
		result.Decision.Intent,
		"state",
		newState,
	)
	return &ChatResponse{
		SessionID:       session.ID,
		MessageID:       assistantMsg.ID,
		Intent:          result.Decision.Intent,
		Answer:          result.Answer,
		Cards:           result.Cards,
		GuidanceChips:   guidanceChips,
		HandoffRequired: result.Decision.NeedsHandoff,
		Meta: ChatResponseMeta{
			Route:         result.Decision.Route,
			Confidence:    result.Decision.Confidence,
			RewriteQuery:  result.Decision.RewrittenQuery,
			FallbackUsed:  result.Decision.FallbackUsed,
			EvidenceCount: len(result.Evidence),
		},
	}, nil
}

// persistDecisionLog 将编排决策（意图、路由、置信度等）持久化到 ChatDecisionLog 表。
// 仅当 repo 实现了 DecisionLogRepository 接口时才执行，否则静默跳过。
// 写入失败仅记录日志不中断流程，因为决策日志属于可观测性数据，不影响主链路。
func (s *service) persistDecisionLog(ctx context.Context, sessionID, messageID int64, decision Decision) {
	repo, ok := s.repo.(domain.DecisionLogRepository)
	if !ok {
		return
	}
	_, err := repo.CreateDecisionLog(
		ctx, &domain.ChatDecisionLog{
			SessionID:       sessionID,
			MessageID:       messageID,
			Intent:          decision.Intent,
			Route:           decision.Route,
			RewriteQuery:    decision.RewrittenQuery,
			Confidence:      decision.Confidence,
			FallbackUsed:    decision.FallbackUsed,
			HandoffRequired: decision.NeedsHandoff,
		},
	)
	if err != nil {
		s.log.Error("app_chat_decision_log_failed", "session_id", sessionID, "message_id", messageID, "error", err)
	}
}

// SearchFAQ 按门店和关键词检索 FAQ，limit 范围 [1,50]，默认 10。
func (s *service) SearchFAQ(ctx context.Context, req FAQSearchRequest) ([]domain.FAQ, error) {
	if req.StoreID <= 0 || strings.TrimSpace(req.Query) == "" {
		s.log.Warn("app_faq_invalid_argument", "request_id", req.RequestID, "store_id", req.StoreID)
		return nil, domain.ErrInvalidArgument
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 50 {
		req.Limit = 50
	}

	items, err := s.repo.SearchFAQ(ctx, req.StoreID, req.Query, req.Limit)
	if err != nil {
		s.log.Error("app_faq_search_failed", "request_id", req.RequestID, "store_id", req.StoreID, "error", err)
		return nil, err
	}
	s.log.Info("app_faq_search_success", "request_id", req.RequestID, "store_id", req.StoreID, "count", len(items))
	return items, nil
}

// SearchProducts 按门店和关键词检索商品，limit 范围 [1,50]，默认 10。
func (s *service) SearchProducts(ctx context.Context, req ProductSearchRequest) ([]domain.Product, error) {
	if req.StoreID <= 0 || strings.TrimSpace(req.Query) == "" {
		s.log.Warn("app_product_search_invalid_argument", "request_id", req.RequestID, "store_id", req.StoreID)
		return nil, domain.ErrInvalidArgument
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 50 {
		req.Limit = 50
	}

	items, err := s.repo.SearchProducts(ctx, req.StoreID, req.Query, req.Limit)
	if err != nil {
		s.log.Error("app_product_search_failed", "request_id", req.RequestID, "store_id", req.StoreID, "error", err)
		return nil, err
	}
	s.log.Info("app_product_search_success", "request_id", req.RequestID, "store_id", req.StoreID, "count", len(items))
	return items, nil
}

// GetProductLocation 查询指定商品在门店的具体位置（区域 + 货架 + 层）。
func (s *service) GetProductLocation(ctx context.Context, req ProductLocationRequest) (*domain.ProductLocation, error) {
	if req.StoreID <= 0 || req.ProductID <= 0 {
		s.log.Warn(
			"app_product_location_invalid_argument",
			"request_id",
			req.RequestID,
			"store_id",
			req.StoreID,
			"product_id",
			req.ProductID,
		)
		return nil, domain.ErrInvalidArgument
	}

	item, err := s.repo.GetProductLocation(ctx, req.StoreID, req.ProductID)
	if err != nil {
		s.log.Error(
			"app_product_location_failed",
			"request_id",
			req.RequestID,
			"store_id",
			req.StoreID,
			"product_id",
			req.ProductID,
			"error",
			err,
		)
		return nil, err
	}
	s.log.Info(
		"app_product_location_success",
		"request_id",
		req.RequestID,
		"store_id",
		req.StoreID,
		"product_id",
		req.ProductID,
	)
	return item, nil
}

// GetInventory 查询指定 SKU 的库存数量。
func (s *service) GetInventory(ctx context.Context, req InventoryRequest) (*domain.Inventory, error) {
	if req.StoreID <= 0 || req.SKUID <= 0 {
		s.log.Warn(
			"app_inventory_invalid_argument",
			"request_id",
			req.RequestID,
			"store_id",
			req.StoreID,
			"sku_id",
			req.SKUID,
		)
		return nil, domain.ErrInvalidArgument
	}

	item, err := s.repo.GetInventory(ctx, req.StoreID, req.SKUID)
	if err != nil {
		s.log.Error(
			"app_inventory_failed",
			"request_id",
			req.RequestID,
			"store_id",
			req.StoreID,
			"sku_id",
			req.SKUID,
			"error",
			err,
		)
		return nil, err
	}
	s.log.Info("app_inventory_success", "request_id", req.RequestID, "store_id", req.StoreID, "sku_id", req.SKUID)
	return item, nil
}

// ListActivePromotions 列出门店当前在有效期内的活动，limit 范围 [1,50]，默认 10。
func (s *service) ListActivePromotions(ctx context.Context, req PromotionListRequest) ([]domain.Promotion, error) {
	if req.StoreID <= 0 {
		s.log.Warn("app_promotion_list_invalid_argument", "request_id", req.RequestID, "store_id", req.StoreID)
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

	items, err := s.repo.ListActivePromotions(ctx, req.StoreID, req.Now, req.Limit)
	if err != nil {
		s.log.Error("app_promotion_list_failed", "request_id", req.RequestID, "store_id", req.StoreID, "error", err)
		return nil, err
	}
	s.log.Info("app_promotion_list_success", "request_id", req.RequestID, "store_id", req.StoreID, "count", len(items))
	return items, nil
}

// ListSessions 分页获取门店的会话列表，limit 范围 [1,100]，默认 20。
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

// ListToolCalls 查询指定会话的工具调用记录，limit 范围 [1,100]，默认 20。
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

// SaveFeedback 保存用户对回答的反馈（👍=1, 👎=0）。
// feedbackValue 仅接受 0 或 1，其余值视为非法参数。
// 要求 repo 实现了 FeedbackRepository 接口，否则返回错误。
func (s *service) SaveFeedback(ctx context.Context, messageID, sessionID int64, feedbackValue int8) error {
	if messageID <= 0 || sessionID <= 0 {
		return domain.ErrInvalidArgument
	}
	if feedbackValue != 0 && feedbackValue != 1 {
		return domain.ErrInvalidArgument
	}
	feedbackRepo, ok := s.repo.(domain.FeedbackRepository)
	if !ok {
		s.log.Error("app_feedback_repo_not_available")
		return domain.ErrInvalidArgument
	}
	_, err := feedbackRepo.CreateFeedback(
		ctx, &domain.Feedback{
			MessageID:     messageID,
			SessionID:     sessionID,
			FeedbackValue: feedbackValue,
		},
	)
	if err != nil {
		s.log.Error("app_feedback_create_failed", "message_id", messageID, "session_id", sessionID, "error", err)
		return err
	}
	s.log.Info("app_feedback_saved", "message_id", messageID, "session_id", sessionID, "value", feedbackValue)
	return nil
}

// —— S1: 入口适配方法 ——

// isFirstMessage 判断是否为会话的首条消息（无历史 assistant 消息）。
func (s *service) isFirstMessage(ctx *SessionContext) bool {
	return ctx.State == StateIdle && len(ctx.ContextStack) == 0
}

// entryFirstOpen 首次破冰入口：返回预设问题列表。
func (s *service) entryFirstOpen(ctx context.Context, req ChatRequest, session *domain.Session) (*ChatResponse, error) {
	answer := "您好！我是小王，您身边的数字店员。\n\n店里的一切都可以问我：商品在哪儿、还有没有货、今天有什么活动、怎么付款……"
	guidanceChips := []GuidanceChip{
		{Text: "📍 薯片在哪里？", Prompt: "薯片在哪里？"},
		{Text: "🏷 今天有什么活动？", Prompt: "今天有什么活动？"},
		{Text: "🥤 低糖饮料有哪些？", Prompt: "低糖饮料有哪些？"},
		{Text: "💳 怎么付款？", Prompt: "怎么付款？"},
	}

	stateStr := string(StateIdle)
	msg, err := s.repo.CreateMessage(
		ctx, &domain.Message{
			SessionID:    session.ID,
			Role:         "assistant",
			Content:      answer,
			Intent:       "greeting",
			ContextState: &stateStr,
		},
	)
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

// entryZoneScan 货架扫码入口：展示当前区域商品列表。
func (s *service) entryZoneScan(ctx context.Context, req ChatRequest, session *domain.Session) (*ChatResponse, error) {
	// S1: 使用 zone_id/shelf_id 过滤当前货架商品
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
			cards = append(
				cards, ChatCard{
					Type:     "product",
					Name:     p.Name,
					Location: loc.ZoneName + " " + loc.ShelfCode + " 货架",
				},
			)
		}
	}

	stateStr := string(StateListBrowse)
	msg, err := s.repo.CreateMessage(
		ctx, &domain.Message{
			SessionID:    session.ID,
			Role:         "assistant",
			Content:      answer,
			Intent:       "zone_scan",
			ContextState: &stateStr,
		},
	)
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

// entryResume 历史会话恢复入口：生成 Context Bridge 文案。
func (s *service) entryResume(
	ctx context.Context,
	req ChatRequest,
	session *domain.Session,
	sessionCtx *SessionContext,
) (*ChatResponse, error) {
	answer := "欢迎回来！" + s.buildContextBridge(sessionCtx)

	stateStr := string(sessionCtx.State)
	msg, err := s.repo.CreateMessage(
		ctx, &domain.Message{
			SessionID:    session.ID,
			Role:         "assistant",
			Content:      answer,
			Intent:       "resume",
			ContextState: &stateStr,
		},
	)
	if err != nil {
		return nil, err
	}

	return &ChatResponse{
		SessionID: session.ID,
		MessageID: msg.ID,
		Intent:    "resume",
		Answer:    answer,
		Meta:      ChatResponseMeta{Route: "entry_resume", FallbackUsed: true},
	}, nil
}

// entryPromo 活动入口：展开活动详情。
func (s *service) entryPromo(ctx context.Context, req ChatRequest, session *domain.Session) (*ChatResponse, error) {
	items, err := s.repo.ListActivePromotions(ctx, req.StoreID, time.Now(), 5)
	if err != nil || len(items) == 0 {
		return s.entryFirstOpen(ctx, req, session)
	}

	answer := "今天的活动是" + items[0].Title + "，参与商品有："
	cards := []ChatCard{
		{
			Type:     "promotion",
			Title:    items[0].Title,
			Content:  items[0].Description,
			Validity: items[0].EndAt.Format("01-02 15:04"),
		},
	}
	guidanceChips := []GuidanceChip{
		{Text: "📍 活动商品在哪里？", Prompt: "活动商品在哪里？"},
	}

	stateStr := string(StateListBrowse)
	msg, err := s.repo.CreateMessage(
		ctx, &domain.Message{
			SessionID:    session.ID,
			Role:         "assistant",
			Content:      answer,
			Intent:       "promotion",
			ContextState: &stateStr,
		},
	)
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

// entryProductDetail 商品详情入口：展示商品卡片 + 引导。
func (s *service) entryProductDetail(ctx context.Context, req ChatRequest, session *domain.Session) (
	*ChatResponse,
	error,
) {
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
	cards := []ChatCard{
		{
			Type:     "product",
			Name:     p.Name,
			Location: locStr,
		},
	}

	if loc != nil && loc.SKUID != nil {
		cards[0].SKUID = *loc.SKUID
	}

	guidanceChips := []GuidanceChip{
		{Text: "📦 还有几瓶？", Prompt: "还有几瓶？"},
		{Text: "🥤 同品类还有什么？", Prompt: "同品类还有什么？"},
	}

	focusIDs := &domain.FocusEntityIDs{ProductIDs: []int64{p.ID}}
	if loc != nil && loc.SKUID != nil {
		focusIDs.SKUIDs = []int64{*loc.SKUID}
	}
	stateStr := string(StateProductFocus)
	msg, err := s.repo.CreateMessage(
		ctx, &domain.Message{
			SessionID:      session.ID,
			Role:           "assistant",
			Content:        answer,
			Intent:         "product_detail",
			ContextState:   &stateStr,
			FocusEntityIDs: focusIDs,
		},
	)
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

// buildContextBridge 根据会话上下文生成 Context Bridge 文案。
func (s *service) buildContextBridge(ctx *SessionContext) string {
	if ctx.DecayAction == DecayConfirmResume {
		return "对话已经结束很久了，请问需要我帮您什么吗？"
	}
	lastAction := ""
	if len(ctx.ContextStack) > 0 {
		lastItem := ctx.ContextStack[len(ctx.ContextStack)-1]
		lastAction = "您之前在看「" + lastItem.SystemSummary + "」，需要继续吗？"
	}
	if lastAction != "" {
		return lastAction
	}
	return "有什么需要帮您的吗？"
}

// —— S1: 消解与引导辅助方法 ——

// clarifyResponse 当 ContextResolver L3 触发时，返回澄清话术。
func (s *service) clarifyResponse(ctx context.Context, session *domain.Session, resolveResult *ResolveResult) (*ChatResponse, error) {
	stateStr := string(StateIdle)
	msg, err := s.repo.CreateMessage(ctx, &domain.Message{
		SessionID:    session.ID,
		Role:         "assistant",
		Content:      resolveResult.ClarifyMessage,
		Intent:       "clarify",
		ContextState: &stateStr,
	})
	if err != nil {
		return nil, err
	}
	return &ChatResponse{
		SessionID: session.ID,
		MessageID: msg.ID,
		Intent:    "clarify",
		Answer:    resolveResult.ClarifyMessage,
		Meta:      ChatResponseMeta{Route: RouteFallback},
	}, nil
}

// extractProductsFromEvidence 从 evidence 列表中提取商品名称列表。
// 用于 GuideEngine 的替代推荐等场景。
func (s *service) extractProductsFromEvidence(evidence []Evidence) []domain.Product {
	if len(evidence) == 0 {
		return nil
	}
	products := make([]domain.Product, 0, len(evidence))
	for _, ev := range evidence {
		if ev.Kind == "product_location" || ev.Kind == "inventory" {
			products = append(products, domain.Product{ID: ev.RecordID, Name: ev.Title})
		}
	}
	return products
}

// extractProductsFromCards 从 ChatCard 列表中提取商品信息（fallback 编排器不产生 Evidence）。
func (s *service) extractProductsFromCards(cards []ChatCard) []domain.Product {
	products := make([]domain.Product, 0, len(cards))
	for _, card := range cards {
		if card.Type == "product" || card.Type == "inventory" {
			products = append(products, domain.Product{Name: card.Name})
		}
	}
	return products
}

// buildFocusFromResolution 从消解结果构建新的 FocusEntityIDs。
// 从 resolvedEntities 提取 product_id/sku_id，若无消解结果则保持原焦点。
func buildFocusFromResolution(resolved []domain.ResolvedEntity, current *domain.FocusEntityIDs) *domain.FocusEntityIDs {
	if len(resolved) == 0 {
		return current
	}
	focus := &domain.FocusEntityIDs{}
	for _, e := range resolved {
		switch e.Type {
		case "product":
			if e.ProductID != nil {
				focus.ProductIDs = append(focus.ProductIDs, *e.ProductID)
			}
		case "sku":
			if e.SKUID != nil {
				focus.SKUIDs = append(focus.SKUIDs, *e.SKUID)
			}
		}
	}
	if len(focus.ProductIDs) == 0 && len(focus.SKUIDs) == 0 {
		return current
	}
	return focus
}

// resolveSession 解析或创建会话：
//   - SessionID > 0 : 查询已有会话，校验 StoreID 归属
//   - SessionID = 0 : 创建新会话
func (s *service) resolveSession(ctx context.Context, req ChatRequest) (*domain.Session, error) {
	if req.SessionID > 0 {
		session, err := s.repo.GetSession(ctx, req.SessionID)
		if err != nil {
			return nil, err
		}
		// 安全校验：不允许跨门店复用 Session
		if session.StoreID != req.StoreID {
			return nil, domain.ErrInvalidArgument
		}
		return session, nil
	}
	return s.repo.CreateSession(ctx, &domain.Session{StoreID: req.StoreID, UserID: req.UserID, Channel: req.Channel})
}
