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

// —— 编排器接口与数据结构 ——

// Orchestrator 定义 Chat 流程的编排协议：意图识别 → 路由分发 → 证据收集 → 答案生成。
type Orchestrator interface {
	Run(ctx context.Context, req OrchestratorRequest) (OrchestratorResult, error)
}

// OrchestratorRequest 编排入参，由 service.Chat 组装后传入。
type OrchestratorRequest struct {
	RequestID string
	StoreID   int64
	SessionID int64
	MessageID int64  // 已持久化的用户消息 ID，供 fallback 记录 tool_call 用
	UserID    *int64 // 可选用户 ID
	Channel   string
	Message   string // 用户原始消息
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

// Run 主编排流程：
//  1. 意图分析（analyzer 为 nil 时直接走 fallback）
//  2. 路由标准化（按意图映射为 tool/rag/hybrid/fallback）
//  3. 证据收集（按路由分派到 tool/rag/hybrid 收集器）
//  4. 答案生成（有证据时调用 composer，无证据返回兜底话术）
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

	// 步骤 1：意图分析
	decision, err := o.analyzer.AnalyzeIntent(
		ctx, IntentRequest{
			StoreID:   req.StoreID,
			SessionID: req.SessionID,
			Message:   req.Message,
		},
	)
	if err != nil {
		// 意图分析失败 → fallback
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
//   - inventory / product_location → RouteTool（查结构化数据）
//   - faq → RouteRAG（查知识库文本）
//   - product_policy → RouteHybrid（先查结构化再补 FAQ）
//   - 若 IntentAnalyzer 已返回有效 Route 则直接沿用
//   - 其余 → RouteFallback
func (o *defaultOrchestrator) normalizeRoute(decision Decision) string {
	intent := strings.TrimSpace(decision.Intent)
	switch {
	case intent == "inventory" || intent == "product_location":
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
//   - tool:   查询数据库获取结构化数据（商品、库存、位置、活动）
//   - rag:    从知识库检索 FAQ 文本
//   - hybrid: 先 tool 再 rag，合并证据
//   - 其他:   委托 fallback 编排器
func (o *defaultOrchestrator) collectEvidence(
	ctx context.Context,
	req OrchestratorRequest,
	decision Decision,
) ([]Evidence, []ChatCard, error) {
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

// collectToolEvidence 按意图类型查询数据库获取结构化证据：
//   - inventory:      搜索商品 → 查位置 → 查库存 → 合并返回 evidence + card
//   - product_policy: 同 inventory（产品政策类问题常包含库存信息）
//   - product_location: 搜索商品 → 查位置 → 返回位置 evidence + card
//   - promotion:      列活动列表 → 返回活动 evidence + card
func (o *defaultOrchestrator) collectToolEvidence(
	ctx context.Context,
	req OrchestratorRequest,
	decision Decision,
) ([]Evidence, []ChatCard, error) {
	query := decision.RewrittenQuery
	if strings.TrimSpace(query) == "" {
		query = req.Message // 无改写则用原始消息
	}

	switch decision.Intent {
	case "inventory", "product_policy":
		// 库存 / 产品政策：搜索商品 → 查位置 → 查库存 → 拼装 evidence+card
		products, err := o.repo.SearchProducts(ctx, req.StoreID, extractProductQuery(query), 5)
		if err != nil {
			return nil, nil, err
		}
		if len(products) == 0 {
			return nil, nil, nil
		}
		location, err := o.repo.GetProductLocation(ctx, req.StoreID, products[0].ID)
		if err != nil {
			return nil, nil, err
		}
		// 商品只有位置、无 SKU 关联时仅返回位置信息
		if location.SKUID == nil {
			return []Evidence{
					{
						Source:   "tool",
						Kind:     "product_location",
						RecordID: products[0].ID,
						Title:    products[0].Name,
						Content:  fmt.Sprintf("%s 在 %s %s", products[0].Name, location.ZoneName, location.ShelfCode),
					},
				}, []ChatCard{
					{
						Type:     "product",
						Name:     products[0].Name,
						Location: fmt.Sprintf("%s %s 货架", location.ZoneName, location.ShelfCode),
					},
				}, nil
		}
		// 查库存
		inventory, err := o.repo.GetInventory(ctx, req.StoreID, *location.SKUID)
		if err != nil {
			return nil, nil, err
		}
		evidence := []Evidence{
			{
				Source:   "tool",
				Kind:     "inventory",
				RecordID: inventory.SKUID,
				Title:    products[0].Name,
				Content:  fmt.Sprintf("系统显示%s还有 %d 件", products[0].Name, inventory.Quantity),
			},
		}
		card := ChatCard{
			Type:     "inventory",
			SKUID:    inventory.SKUID,
			Name:     products[0].Name,
			Location: fmt.Sprintf("%s %s 货架", location.ZoneName, location.ShelfCode),
			Quantity: inventory.Quantity,
		}
		return evidence, []ChatCard{card}, nil

	case "product_location":
		// 商品位置：搜索商品 → 查位置 → 拼装 evidence+card
		products, err := o.repo.SearchProducts(ctx, req.StoreID, extractProductQuery(query), 5)
		if err != nil {
			return nil, nil, err
		}
		if len(products) == 0 {
			return nil, nil, nil
		}
		location, err := o.repo.GetProductLocation(ctx, req.StoreID, products[0].ID)
		if err != nil {
			return nil, nil, err
		}
		evidence := []Evidence{
			{
				Source:   "tool",
				Kind:     "product_location",
				RecordID: products[0].ID,
				Title:    products[0].Name,
				Content: fmt.Sprintf(
					"%s 在 %s %s 货架第%d层",
					products[0].Name,
					location.ZoneName,
					location.ShelfCode,
					location.LayerNo,
				),
			},
		}
		card := ChatCard{
			Type:     "product",
			Name:     products[0].Name,
			Location: fmt.Sprintf("%s %s 货架第%d层", location.ZoneName, location.ShelfCode, location.LayerNo),
		}
		if location.SKUID != nil {
			card.SKUID = *location.SKUID
		}
		return evidence, []ChatCard{card}, nil

	case "promotion":
		// 活动：列出当前有效活动 → 拼装 evidence+card
		items, err := o.repo.ListActivePromotions(ctx, req.StoreID, time.Now(), 5)
		if err != nil {
			return nil, nil, err
		}
		if len(items) == 0 {
			return nil, nil, nil
		}
		evidence := []Evidence{
			{
				Source:   "tool",
				Kind:     "promotion",
				RecordID: items[0].ID,
				Title:    items[0].Title,
				Content:  items[0].Description,
			},
		}
		card := ChatCard{
			Type:     "promotion",
			Title:    items[0].Title,
			Content:  items[0].Description,
			Validity: items[0].EndAt.Format("01-02 15:04"),
		}
		return evidence, []ChatCard{card}, nil

	default:
		return nil, nil, nil
	}
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
