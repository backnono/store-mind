package customerqa

import (
	"context"
	"fmt"
	"strings"
	"time"

	domain "store-mind/domain/customerqa"
)

// —— 路由类型常量 ——
// 决定 orchestrator 使用哪种策略收集证据并生成回答。
const (
	RouteTool     = "tool"     // 直接查询结构化数据（库存、位置、活动）
	RouteRAG      = "rag"      // 从知识库中检索 FAQ 文本
	RouteHybrid   = "hybrid"   // 同时执行 tool + RAG，合并证据
	RouteFallback = "fallback" // 全部失败或意图不明时走降级编排器
)

// —— S2.2: 复合意图辅助 ——

// subIntents 将复合意图（逗号分隔）拆分为独立意图列表。
// 单个意图直接返回单元素切片。
func subIntents(intent string) []string {
	raw := strings.Split(intent, ",")
	result := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s != "" {
			result = append(result, s)
		}
	}
	if len(result) == 0 {
		return []string{intent}
	}
	return result
}

// isCompound 判断意图是否为复合意图。
func isCompound(intent string) bool {
	return strings.Contains(intent, ",")
}

// routeForIntent 将单个意图映射到路由类型。
// inventory / product_location / price / promotion → tool
// faq → rag
// 其他 → fallback
func routeForIntent(intent string) string {
	switch intent {
	case "inventory", "product_location", "price", "promotion":
		return RouteTool
	case "faq":
		return RouteRAG
	case "product_policy":
		return RouteHybrid
	default:
		return RouteFallback
	}
}

// —— 编排器接口与数据结构 ——

// Orchestrator 定义 Chat 流程的编排协议：意图识别 → 路由分发 → 证据收集 → 答案生成。
type Orchestrator interface {
	Run(ctx context.Context, req OrchestratorRequest) (OrchestratorResult, error)
}

// OrchestratorRequest 编排入参，由 service.Chat 组装后传入。
type OrchestratorRequest struct {
	RequestID        string
	StoreID          int64
	SessionID        int64
	MessageID        int64  // 已持久化的用户消息 ID，供 fallback 记录 tool_call 用
	UserID           *int64 // 可选用户 ID
	Channel          string
	Message          string                  // 用户原始消息
	SessionContext   *SessionContext         // S1: 会话上下文（状态 + context_stack）
	ResolvedEntities []domain.ResolvedEntity // S1: ContextResolver 消解后的实体
}

// Decision 单次编排决策，描述意图识别的完整输出。
// 由 IntentAnalyzer 生成，在编排器内部消费。
type Decision struct {
	Intent         string   // 意图标签（inventory, product_location, promotion, faq, handoff, unsupported, ...）
	RewrittenQuery string   // LLM 改写后的检索查询，为空时退回原文
	Route          string   // 最终路由（tool / rag / hybrid / fallback）
	NeedsHandoff   bool     // 是否需要转人工
	Confidence     float64  // 意图置信度 0-1
	ReasoningTags  []string // 推理标签，用于调试
	FallbackUsed   bool     // 标注是否实际使用了降级编排器
}

// Evidence 证据片段， 。
// 每条 Evidence 对应一个召回片段，由 AnswerComposer 组装为自然语言回答。
type Evidence struct {
	Source   string // 来源：tool / rag
	Kind     string // 种类：inventory / product_location / promotion / faq
	RecordID int64  // 关联的数据库记录 ID
	Title    string // 证据标题（商品名、FAQ 问题等）
	Content  string // 证据内容（描述文本、答案等）
}

// OrchestratorResult 编排输出，包含决策、回答及前端展示所需的附属数据。
type OrchestratorResult struct {
	Decision      Decision       // 本次编排的决策
	Answer        string         // 自然语言回答
	Cards         []ChatCard     // 结构化卡片
	Evidence      []Evidence     // 召回的证据列表
	GuidanceChips []GuidanceChip // 引导追问
}

// —— 主编排器实现 ——

// defaultOrchestrator 是主编排器，依赖 IntentAnalyzer + Retriever + AnswerComposer 三个抽象。
// 当任一抽象为 nil 时，自动降级到 fallbackOrchestrator（关键词匹配模式）。
type defaultOrchestrator struct {
	analyzer  IntentAnalyzer // 意图分析（LLM 或 fake）
	composer  AnswerComposer // 答案生成（LLM 或 fake）
	retriever Retriever      // 证据检索（MySQL 全文搜索）
	fallback  Orchestrator   // 降级编排器，analyzer/composer/retriever 任一不可用时启用
	repo      domain.Repository
	log       Logger
}

// newDefaultOrchestrator 创建无 LLM 能力的主编排器，等价于纯 fallback 模式。
func newDefaultOrchestrator(repo domain.Repository, log Logger) Orchestrator {
	return NewDefaultOrchestrator(repo, log, nil, nil, nil)
}

// NewDefaultOrchestrator 创建主编排器。
// analyzer/composer/retriever 全为 nil 时，等同于纯 fallback 模式。
func NewDefaultOrchestrator(
	repo domain.Repository,
	log Logger,
	analyzer IntentAnalyzer,
	composer AnswerComposer,
	retriever Retriever,
) Orchestrator {
	if log == nil {
		log = nopLogger{}
	}
	return &defaultOrchestrator{
		analyzer:  analyzer,
		composer:  composer,
		retriever: retriever,
		repo:      repo,
		log:       log,
		fallback:  newFallbackOrchestrator(repo, log),
	}
}

// Run 主编排流程（S1 升级版）：
//  1. 检查衰减状态（长时间无输入 → 返回 Context Bridge）
//  2. 意图分析（传入 context_stack 实现多轮上下文）
//  3. 路由标准化
//  4. 证据收集（含库存可信度计算）
//  5. 答案生成
//
// 任何一步出错均尝试 fallback 兜底。
func (o *defaultOrchestrator) Run(ctx context.Context, req OrchestratorRequest) (OrchestratorResult, error) {
	// 无 LLM 能力时直接降级到 fallback 编排器
	if o.analyzer == nil {
		if o.fallback != nil {
			return o.fallback.Run(ctx, req)
		}
		return OrchestratorResult{}, nil
	}

	// S1: 衰减检查 — 长时间无输入时返回 Context Bridge
	if req.SessionContext != nil && req.SessionContext.DecayAction == DecayConfirmResume {
		return OrchestratorResult{
			Decision: Decision{
				Intent:       "resume",
				Route:        RouteFallback,
				Confidence:   1.0,
				FallbackUsed: true,
			},
			Answer: "对话已经结束很久了。请问这次需要我帮您什么吗？",
		}, nil
	}

	// 步骤 1：意图分析（传入 context_stack 实现多轮上下文）
	intentReq := IntentRequest{
		StoreID:   req.StoreID,
		SessionID: req.SessionID,
		Message:   req.Message,
	}
	// S1: 附加 context_stack 摘要到意图分析请求
	if req.SessionContext != nil && len(req.SessionContext.ContextStack) > 0 {
		intentReq.ContextStack = req.SessionContext.ContextStack
	}
	decision, err := o.analyzer.AnalyzeIntent(ctx, intentReq)
	if err != nil {
		if o.fallback != nil {
			return o.fallback.Run(ctx, req)
		}
		return OrchestratorResult{}, nil
	}

	// 步骤 2：路由标准化
	decision.Route = o.normalizeRoute(decision)

	// 步骤 3：按路由收集证据
	evidence, cards, err := o.collectEvidence(ctx, req, decision)
	if err != nil {
		if o.fallback != nil {
			return o.fallback.Run(ctx, req)
		}
		return OrchestratorResult{}, err
	}

	// 步骤 4：答案生成
	answer := conservativeAnswer(decision.Route) // 兜底模板
	var guidanceChips []GuidanceChip
	if len(evidence) > 0 && o.composer != nil {
		// 有证据 + composer 可用 → 走 LLM 生成自然语言答案
		result, err := o.composer.ComposeAnswer(
			ctx, AnswerRequest{
				Decision: decision,
				Message:  req.Message,
				Evidence: evidence,
			},
		)
		if err != nil {
			if o.fallback != nil {
				return o.fallback.Run(ctx, req)
			}
			return OrchestratorResult{}, err
		}
		answer = result.Answer
		guidanceChips = result.GuidanceChips
	} else if len(evidence) == 0 {
		// 无证据时给出通用的兜底话术
		answer = "暂时没有找到可靠依据回答这个问题，你可以换个问法，或联系人工客服。"
	}

	return OrchestratorResult{
		Decision:      decision,
		Answer:        answer,
		Cards:         cards,
		Evidence:      evidence,
		GuidanceChips: guidanceChips,
	}, nil
}

// normalizeRoute 将意图标签映射为标准路由类型。
// 映射规则：
//   - inventory / product_location / price / promotion → RouteTool（查结构化数据）
//   - faq → RouteRAG（查知识库文本）
//   - product_policy → RouteHybrid（先查结构化再补 FAQ）
//   - S2.2: 复合意图 → 合并各子意图路由（混合 type+rag → hybrid，纯 type → tool，纯 rag → rag）
//   - 若 IntentAnalyzer 已返回有效 Route 则直接沿用
//   - 其余 → RouteFallback
func (o *defaultOrchestrator) normalizeRoute(decision Decision) string {
	intent := strings.TrimSpace(decision.Intent)

	// S2.2: 复合意图路由合并
	if isCompound(intent) {
		hasTool, hasRAG := false, false
		for _, si := range subIntents(intent) {
			r := routeForIntent(si)
			if r == RouteTool || r == RouteHybrid {
				hasTool = true
			}
			if r == RouteRAG || r == RouteHybrid {
				hasRAG = true
			}
		}
		if hasTool && hasRAG {
			return RouteHybrid
		}
		if hasTool {
			return RouteTool
		}
		if hasRAG {
			return RouteRAG
		}
		return RouteFallback
	}

	switch {
	case intent == "inventory" || intent == "product_location" || intent == "price" || intent == "promotion":
		return RouteTool
	case intent == "faq":
		return RouteRAG
	case intent == "product_policy":
		return RouteHybrid
	case decision.Route == RouteTool || decision.Route == RouteRAG || decision.Route == RouteHybrid || decision.Route == RouteFallback:
		return decision.Route
	default:
		return RouteFallback
	}
}

// collectEvidence 按路由类型分派到不同的证据收集策略：
//   - tool:   查询数据库获取结构化数据（商品、库存、位置、活动、价格）
//   - rag:    从知识库检索 FAQ 文本
//   - hybrid: 先 tool 再 rag，合并证据
//   - S2.2: 复合意图 → 对各子意图并行收集证据，合并结果
//   - 其他:   委托 fallback 编排器
func (o *defaultOrchestrator) collectEvidence(
	ctx context.Context,
	req OrchestratorRequest,
	decision Decision,
) ([]Evidence, []ChatCard, error) {
	// S2.2: 复合意图 — 对各子意图分别收集证据并合并
	if isCompound(decision.Intent) {
		return o.collectCompoundEvidence(ctx, req, decision)
	}

	switch decision.Route {
	case RouteTool:
		return o.collectToolEvidence(ctx, req, decision)
	case RouteRAG:
		return o.collectRAGEvidence(ctx, req, decision)
	case RouteHybrid:
		// Hybrid 模式：同时收集 tool 和 rag 的证据
		toolEvidence, cards, err := o.collectToolEvidence(
			ctx,
			req,
			Decision{Intent: "inventory", RewrittenQuery: decision.RewrittenQuery},
		)
		if err != nil {
			return nil, nil, err
		}
		ragEvidence, _, err := o.collectRAGEvidence(ctx, req, decision)
		if err != nil {
			return nil, nil, err
		}
		return append(toolEvidence, ragEvidence...), cards, nil
	default:
		if o.fallback != nil {
			result, err := o.fallback.Run(ctx, req)
			if err != nil {
				return nil, nil, err
			}
			return result.Evidence, result.Cards, nil
		}
		return nil, nil, nil
	}
}

// collectCompoundEvidence S2.2: 复合意图证据收集。
// 将意图字符串按逗号拆分为多个子意图，对每个子意图分别收集证据，最后合并。
func (o *defaultOrchestrator) collectCompoundEvidence(
	ctx context.Context,
	req OrchestratorRequest,
	decision Decision,
) ([]Evidence, []ChatCard, error) {
	allEvidence := make([]Evidence, 0)
	allCards := make([]ChatCard, 0)
	query := decision.RewrittenQuery
	if strings.TrimSpace(query) == "" {
		query = req.Message
	}
	resolvedProductID := o.resolvedProductID(req)

	seenEvidence := map[string]bool{} // 去重

	for _, si := range subIntents(decision.Intent) {
		subDecision := Decision{
			Intent:         si,
			RewrittenQuery: query,
			Route:          routeForIntent(si),
		}

		var evidence []Evidence
		var cards []ChatCard
		var err error

		if routeForIntent(si) == RouteTool {
			evidence, cards, err = o.collectSingleIntentToolEvidence(ctx, req, subDecision, query, resolvedProductID)
		} else if routeForIntent(si) == RouteRAG {
			evidence, _, err = o.collectRAGEvidence(ctx, req, subDecision)
		}

		if err != nil {
			o.log.Warn("compound_evidence_partial_fail", "sub_intent", si, "error", err)
			continue // 子意图失败不中断整体流程
		}

		// 去重合并
		for _, ev := range evidence {
			key := ev.Source + ":" + ev.Kind + ":" + fmt.Sprintf("%d", ev.RecordID)
			if !seenEvidence[key] {
				seenEvidence[key] = true
				allEvidence = append(allEvidence, ev)
			}
		}
		allCards = append(allCards, cards...)
	}

	return allEvidence, allCards, nil
}

// collectToolEvidence 按意图类型查询数据库获取结构化证据：
//   - inventory:      搜索商品 → 查位置 → 查库存 → 合并返回 evidence + card
//   - product_policy: 同 inventory（产品政策类问题常包含库存信息）
//   - product_location: 搜索商品 → 查位置 → 返回位置 evidence + card
//   - price:          搜索商品 → 查库存（含价格）→ 返回价格 evidence + card
//   - promotion:      列活动列表 → 返回活动 evidence + card
//
// S1: 当 req.ResolvedEntities 中有消解后的 product_id 时，跳过 SearchProducts 直接查询。
func (o *defaultOrchestrator) collectToolEvidence(
	ctx context.Context,
	req OrchestratorRequest,
	decision Decision,
) ([]Evidence, []ChatCard, error) {
	query := decision.RewrittenQuery
	if strings.TrimSpace(query) == "" {
		query = req.Message // 无改写则用原始消息
	}
	resolvedProductID := o.resolvedProductID(req)
	return o.collectSingleIntentToolEvidence(ctx, req, decision, query, resolvedProductID)
}

// collectSingleIntentToolEvidence S2.2: 对单个意图执行 Tool 层证据收集。
// 与 collectToolEvidence 等价，但接受显式传递的 query 和 resolvedProductID
// 供复合意图场景复用。
func (o *defaultOrchestrator) collectSingleIntentToolEvidence(
	ctx context.Context,
	req OrchestratorRequest,
	decision Decision,
	query string,
	resolvedProductID *int64,
) ([]Evidence, []ChatCard, error) {
	switch decision.Intent {
	case "inventory", "product_policy":
		return o.collectInventoryEvidence(ctx, req, decision, query, resolvedProductID)
	case "product_location":
		return o.collectLocationEvidence(ctx, req, decision, query, resolvedProductID)
	case "price":
		return o.collectPriceEvidence(ctx, req, decision, query, resolvedProductID)
	case "promotion":
		return o.collectPromotionEvidence(ctx, req, decision)
	default:
		return nil, nil, nil
	}
}

// resolvedProductID 从 ResolvedEntities 中提取首个 product_id。
func (o *defaultOrchestrator) resolvedProductID(req OrchestratorRequest) *int64 {
	for _, e := range req.ResolvedEntities {
		if e.Type == "product" && e.ProductID != nil {
			return e.ProductID
		}
	}
	return nil
}

// collectInventoryEvidence 收集库存证据（S1 重构版）。
func (o *defaultOrchestrator) collectInventoryEvidence(
	ctx context.Context,
	req OrchestratorRequest,
	decision Decision,
	query string,
	resolvedProductID *int64,
) ([]Evidence, []ChatCard, error) {
	productID, productName, err := o.resolveProduct(ctx, req, query, resolvedProductID)
	if err != nil {
		return nil, nil, err
	}
	if productID == nil {
		return nil, nil, nil
	}

	location, err := o.repo.GetProductLocation(ctx, req.StoreID, *productID)
	if err != nil {
		return nil, nil, err
	}

	if location.SKUID == nil {
		return []Evidence{{
				Source: "tool", Kind: "product_location", RecordID: *productID,
				Title: productName, Content: fmt.Sprintf("%s 在 %s %s", productName, location.ZoneName, location.ShelfCode),
			}}, []ChatCard{{
				Type: "product", Name: productName,
				Location: fmt.Sprintf("%s %s 货架", location.ZoneName, location.ShelfCode),
			}}, nil
	}

	inventory, err := o.repo.GetInventory(ctx, req.StoreID, *location.SKUID)
	if err != nil {
		return nil, nil, err
	}
	credTag := CredibilityTag(inventory)
	evidence := []Evidence{{
		Source: "tool", Kind: "inventory", RecordID: inventory.SKUID,
		Title:   productName,
		Content: fmt.Sprintf("系统显示%s还有 %d 件 · %s", productName, inventory.Quantity, credTag),
	}}
	card := ChatCard{
		Type: "inventory", SKUID: inventory.SKUID, Name: productName,
		Location: fmt.Sprintf("%s %s 货架", location.ZoneName, location.ShelfCode),
		Quantity: inventory.Quantity,
	}
	return evidence, []ChatCard{card}, nil
}

// collectLocationEvidence 收集位置证据（S1 重构版）。
func (o *defaultOrchestrator) collectLocationEvidence(
	ctx context.Context,
	req OrchestratorRequest,
	decision Decision,
	query string,
	resolvedProductID *int64,
) ([]Evidence, []ChatCard, error) {
	productID, productName, err := o.resolveProduct(ctx, req, query, resolvedProductID)
	if err != nil {
		return nil, nil, err
	}
	if productID == nil {
		return nil, nil, nil
	}

	location, err := o.repo.GetProductLocation(ctx, req.StoreID, *productID)
	if err != nil {
		return nil, nil, err
	}
	evidence := []Evidence{{
		Source: "tool", Kind: "product_location", RecordID: *productID,
		Title:   productName,
		Content: fmt.Sprintf("%s 在 %s %s 货架第%d层", productName, location.ZoneName, location.ShelfCode, location.LayerNo),
	}}
	card := ChatCard{
		Type: "product", Name: productName,
		Location: fmt.Sprintf("%s %s 货架第%d层", location.ZoneName, location.ShelfCode, location.LayerNo),
	}
	if location.SKUID != nil {
		card.SKUID = *location.SKUID
	}
	return evidence, []ChatCard{card}, nil
}

// collectPromotionEvidence 收集活动证据。
func (o *defaultOrchestrator) collectPromotionEvidence(
	ctx context.Context,
	req OrchestratorRequest,
	decision Decision,
) ([]Evidence, []ChatCard, error) {
	items, err := o.repo.ListActivePromotions(ctx, req.StoreID, time.Now(), 5)
	if err != nil {
		return nil, nil, err
	}
	if len(items) == 0 {
		return nil, nil, nil
	}
	evidence := []Evidence{{
		Source: "tool", Kind: "promotion", RecordID: items[0].ID,
		Title: items[0].Title, Content: items[0].Description,
	}}
	card := ChatCard{
		Type: "promotion", Title: items[0].Title,
		Content: items[0].Description, Validity: items[0].EndAt.Format("01-02 15:04"),
	}
	return evidence, []ChatCard{card}, nil
}

// collectPriceEvidence S2.3: 收集价格证据。
// 支持单商品价格查询（"可乐多少钱"）和多商品价格对比（"可乐和雪碧哪个便宜"）。
// 通过 resolveProduct 查找商品 → 获取 Inventory（含 Price）→ 返回价格证据。
func (o *defaultOrchestrator) collectPriceEvidence(
	ctx context.Context,
	req OrchestratorRequest,
	decision Decision,
	query string,
	resolvedProductID *int64,
) ([]Evidence, []ChatCard, error) {
	// 路径 A: ResolvedEntities 有消解结果 — 直接按 product_id 查询价格
	if resolvedProductID != nil {
		return o.collectSingleProductPrice(ctx, req, *resolvedProductID)
	}

	// 路径 B: 搜索所有匹配商品，可能为多商品价格对比
	products, err := o.repo.SearchProducts(ctx, req.StoreID, extractProductQuery(query), 5)
	if err != nil {
		return nil, nil, err
	}
	if len(products) == 0 {
		return nil, nil, nil
	}

	evidence := make([]Evidence, 0, len(products))
	cards := make([]ChatCard, 0, len(products))
	for _, p := range products {
		loc, err := o.repo.GetProductLocation(ctx, req.StoreID, p.ID)
		if err != nil || loc.SKUID == nil {
			// 无位置/SKU 时仅返回位置信息
			if loc != nil {
				evidence = append(evidence, Evidence{
					Source: "tool", Kind: "price", RecordID: p.ID,
					Title: p.Name, Content: fmt.Sprintf("%s 在 %s %s，价格暂未查到", p.Name, loc.ZoneName, loc.ShelfCode),
				})
			}
			continue
		}
		inv, err := o.repo.GetInventory(ctx, req.StoreID, *loc.SKUID)
		if err != nil {
			continue
		}
		priceStr := fmt.Sprintf("¥%.2f", inv.Price)
		if inv.Spec != "" {
			priceStr += fmt.Sprintf(" / %s", inv.Spec)
		}
		evContent := fmt.Sprintf("%s · %s · 在 %s %s",
			p.Name, priceStr, loc.ZoneName, loc.ShelfCode)
		evidence = append(evidence, Evidence{
			Source: "tool", Kind: "price", RecordID: p.ID,
			Title: p.Name, Content: evContent,
		})
		cards = append(cards, ChatCard{
			Type: "price", Name: p.Name,
			Location: fmt.Sprintf("%s %s 货架", loc.ZoneName, loc.ShelfCode),
			SKUID:    inv.SKUID,
		})
	}
	return evidence, cards, nil
}

// collectSingleProductPrice S2.3: 查询单个商品的价格（按 product_id 直查）。
func (o *defaultOrchestrator) collectSingleProductPrice(
	ctx context.Context,
	req OrchestratorRequest,
	productID int64,
) ([]Evidence, []ChatCard, error) {
	loc, err := o.repo.GetProductLocation(ctx, req.StoreID, productID)
	if err != nil {
		return nil, nil, err
	}
	if loc.SKUID == nil {
		return []Evidence{{
			Source: "tool", Kind: "price", RecordID: productID,
			Title: "商品", Content: fmt.Sprintf("在 %s %s，价格暂未查到", loc.ZoneName, loc.ShelfCode),
		}}, nil, nil
	}
	inv, err := o.repo.GetInventory(ctx, req.StoreID, *loc.SKUID)
	if err != nil {
		return nil, nil, err
	}
	priceStr := fmt.Sprintf("¥%.2f", inv.Price)
	if inv.Spec != "" {
		priceStr += fmt.Sprintf(" / %s", inv.Spec)
	}
	return []Evidence{{
			Source: "tool", Kind: "price", RecordID: productID,
			Title: inv.ProductName,
			Content: fmt.Sprintf("%s · %s · 在 %s %s",
				inv.ProductName, priceStr, loc.ZoneName, loc.ShelfCode),
		}}, []ChatCard{{
			Type: "price", Name: inv.ProductName,
			Location: fmt.Sprintf("%s %s 货架", loc.ZoneName, loc.ShelfCode),
			SKUID:    inv.SKUID,
		}}, nil
}

// resolveProduct 解析查询到的商品：优先用 resolvedProductID 直接查，否则 SearchProducts。
// 返回 productID、productName、error。
func (o *defaultOrchestrator) resolveProduct(
	ctx context.Context,
	req OrchestratorRequest,
	query string,
	resolvedProductID *int64,
) (*int64, string, error) {
	if resolvedProductID != nil {
		// S1: L1/L2 已消解出具体 product_id → 验证存在即可
		_, err := o.repo.GetProductLocation(ctx, req.StoreID, *resolvedProductID)
		if err != nil {
			return nil, "", err
		}
		return resolvedProductID, "商品", nil // productName 由后续 inventory 查询补充
	}

	// 传统路径：SearchProducts → 取第一条
	products, err := o.repo.SearchProducts(ctx, req.StoreID, extractProductQuery(query), 5)
	if err != nil {
		return nil, "", err
	}
	if len(products) == 0 {
		return nil, "", nil
	}
	pid := products[0].ID
	return &pid, products[0].Name, nil
}

// collectRAGEvidence 通过 Retriever 从知识库召回 FAQ 证据。
// 查询优先使用改写后的 RewrittenQuery，为空时退回原始消息。
func (o *defaultOrchestrator) collectRAGEvidence(
	ctx context.Context,
	req OrchestratorRequest,
	decision Decision,
) ([]Evidence, []ChatCard, error) {
	if o.retriever == nil {
		return nil, nil, nil
	}
	query := decision.RewrittenQuery
	if strings.TrimSpace(query) == "" {
		query = req.Message
	}
	items, err := o.retriever.Retrieve(
		ctx, RetrievalRequest{
			StoreID: req.StoreID,
			Query:   query,
			Intent:  decision.Intent,
			Limit:   5,
		},
	)
	if err != nil {
		return nil, nil, err
	}
	return items, nil, nil
}

// conservativeAnswer 按路由类型返回保守的模板回答。
// 用作 composer 不可用或无证据时的兜底话术，避免暴露内部技术细节。
func conservativeAnswer(route string) string {
	switch route {
	case RouteTool:
		return "系统已查询到相关门店数据。"
	case RouteRAG, RouteHybrid:
		return "已根据门店知识整理答案。"
	default:
		return ""
	}
}
