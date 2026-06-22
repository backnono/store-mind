package customerqa

import (
	"context"
	"strings"

	domain "store-mind/domain/customerqa"
)

// ContextResolver 三层指代消解：L1 实体继承 → L2 LLM 指代解析 → L3 澄清询问。
// 在 orchestrator.Run() 中先于意图分析调用，将消解后的实体注入后续流程。
type ContextResolver interface {
	// Resolve 对用户消息进行指代消解。
	// 返回消解后的实体列表、消解层级、以及是否需要澄清。
	Resolve(ctx context.Context, req ResolveRequest) (*ResolveResult, error)
}

// ResolveRequest 指代消解请求。
type ResolveRequest struct {
	Message       string                    // 用户原始消息
	SessionState  SessionState              // 当前会话状态
	FocusEntities *domain.FocusEntityIDs    // 当前锁定的实体
	ContextStack  []domain.ContextStackItem // 最近 N 轮结构化摘要
}

// ResolveResult 指代消解结果。
type ResolveResult struct {
	ResolvedEntities []domain.ResolvedEntity // 消解后的实体列表
	Layer            string                  // 消解层级：L1 / L2 / L3
	NeedsClarify     bool                    // 是否需要向用户澄清
	ClarifyMessage   string                  // 澄清文案（当 NeedsClarify=true 时有效）
	Confidence       float64                 // 消解置信度 0-1
}

// ── 默认实现 ───────────────────────────────────────

// defaultContextResolver 实现三层指代消解策略。
// L2 层依赖 AnaphoraClient（LLM 调用接口），为 nil 时跳过 L2 直接进入 L3。
type defaultContextResolver struct {
	llmClient AnaphoraClient
	log       Logger
}

// AnaphoraClient LLM 指代消解接口，由 infra/ai 中的 Python LLM sidecar 实现。
type AnaphoraClient interface {
	ResolveAnaphora(
		ctx context.Context,
		message string,
		contextStack []domain.ContextStackItem,
		focusEntities *domain.FocusEntityIDs,
	) (*AnaphoraLLMResult, error)
}

// AnaphoraLLMResult LLM 指代消解的原始返回。
type AnaphoraLLMResult struct {
	ResolvedEntities []domain.ResolvedEntity `json:"resolved_entities"`
	Confidence       float64                 `json:"confidence"`
	Explanation      string                  `json:"explanation"`
}

// NewContextResolver 创建默认的三层指代消解器。
func NewContextResolver(llmClient AnaphoraClient, log Logger) ContextResolver {
	if log == nil {
		log = nopLogger{}
	}
	return &defaultContextResolver{llmClient: llmClient, log: log}
}

// Resolve 按 L1 → L2 → L3 顺序执行指代消解。
func (r *defaultContextResolver) Resolve(ctx context.Context, req ResolveRequest) (*ResolveResult, error) {
	// L1: 实体继承（基于会话状态和焦点实体的规则匹配）
	if result := r.resolveL1(req); result != nil {
		return result, nil
	}

	// L2: LLM 指代解析（将 context_stack + 消息传给 LLM）
	if r.llmClient != nil {
		if result, err := r.resolveL2(ctx, req); err == nil && result != nil {
			if result.Confidence >= 0.6 {
				return result, nil
			}
		}
	}

	// L3: 澄清询问
	return r.resolveL3(req), nil
}

// resolveL1 实体继承规则：
//
//	若当前状态为 product_focus 且用户输入为省略追问（无新实体词），
//	则直接继承 focus_product_ids 作为消解结果。
//	覆盖两种追问模式：
//	  - 显式指代："那个多少钱？"/"它还有吗？"
//	  - 省略追问："多少钱？"/"还有几瓶？"/"在哪里？"
func (r *defaultContextResolver) resolveL1(req ResolveRequest) *ResolveResult {
	if req.SessionState != StateProductFocus {
		return nil
	}
	if req.FocusEntities == nil || len(req.FocusEntities.ProductIDs) == 0 {
		return nil
	}

	// 检查消息中是否包含新商品名/品牌名（非追问信号）
	// 追问模式关键词：指代词 + 属性追问词
	anaphoraWords := []string{"那", "这个", "它", "这"}
	attributeQuestionWords := []string{
		"多少钱", "价格", "还有吗", "还有几", "还有多少",
		"库存", "在哪", "位置", "有货", "没货", "缺货",
		"几瓶", "几包", "几个", "几件", "几盒",
		"多少瓶", "多少包", "多少钱",
	}

	msg := req.Message
	isAnaphora := containsAny(msg, anaphoraWords)
	isAttributeQ := containsAny(msg, attributeQuestionWords)

	// L1 触发条件：
	// 1. 快速通道：消息是短追问（≤15 字）+ 包含属性追问词 → 直接继承
	// 2. 指代通道：消息包含指代词 + 不含潜在新实体词
	if isAttributeQ && len([]rune(msg)) <= 15 {
		// 省略追问，直接 L1
	} else if isAnaphora && !hasPotentialEntity(msg, anaphoraWords, attributeQuestionWords) {
		// 显式指代（如"那个多少钱？"），但排除含潜在新实体的（如"那雪碧呢？"）
	} else {
		// 可能包含新实体，交给 L2
		return nil
	}

	// L1 命中：继承焦点实体
	entities := make([]domain.ResolvedEntity, 0, len(req.FocusEntities.ProductIDs))
	for _, pid := range req.FocusEntities.ProductIDs {
		pid := pid // capture
		entities = append(
			entities, domain.ResolvedEntity{
				Type:      "product",
				Name:      "", // 名称由后续查询补充
				ProductID: &pid,
			},
		)
	}

	r.log.Info("context_resolver_l1_hit",
		"state", req.SessionState,
		"focus_count", len(req.FocusEntities.ProductIDs),
		"is_anaphora", isAnaphora,
		"is_attribute_q", isAttributeQ,
	)
	return &ResolveResult{
		ResolvedEntities: entities,
		Layer:            "L1",
		NeedsClarify:     false,
		Confidence:       0.95,
	}
}

// containsAny 检查消息是否包含任意关键词。
func containsAny(message string, words []string) bool {
	for _, w := range words {
		if strings.Contains(message, w) {
			return true
		}
	}
	return false
}

// hasPotentialEntity 检查消息除去已知追问词后是否还有剩余内容词（潜在新实体）。
// 例如："那雪碧呢？" 去掉"那"/"呢"后剩"雪碧"→ true（可能新实体）
// "那个多少钱？" 去掉"那"/"个"/"多少钱"/"？"后剩空 → false（纯指代追问）
func hasPotentialEntity(message string, anaphoraWords, questionWords []string) bool {
	cleaned := message
	for _, w := range anaphoraWords {
		cleaned = strings.ReplaceAll(cleaned, w, "")
	}
	for _, w := range questionWords {
		cleaned = strings.ReplaceAll(cleaned, w, "")
	}
	// 去掉常见标点和语气词
	extraNoise := []string{"？", "?", "呢", "吗", "吧", "啊", "的", "了", "个"}
	for _, w := range extraNoise {
		cleaned = strings.ReplaceAll(cleaned, w, "")
	}
	cleaned = strings.TrimSpace(cleaned)
	// 如果清理后仍有非空白内容 → 可能存在新实体
	return len([]rune(cleaned)) > 0
}

// resolveL2 调用 LLM 进行指代消解。
func (r *defaultContextResolver) resolveL2(ctx context.Context, req ResolveRequest) (*ResolveResult, error) {
	llmResult, err := r.llmClient.ResolveAnaphora(ctx, req.Message, req.ContextStack, req.FocusEntities)
	if err != nil {
		r.log.Warn("context_resolver_l2_llm_failed", "error", err)
		return nil, err
	}

	r.log.Info(
		"context_resolver_l2_hit",
		"confidence",
		llmResult.Confidence,
		"entities",
		len(llmResult.ResolvedEntities),
	)
	return &ResolveResult{
		ResolvedEntities: llmResult.ResolvedEntities,
		Layer:            "L2",
		NeedsClarify:     false,
		Confidence:       llmResult.Confidence,
	}, nil
}

// resolveL3 生成澄清询问。
func (r *defaultContextResolver) resolveL3(req ResolveRequest) *ResolveResult {
	// 根据当前状态生成不同的澄清文案
	var clarifyMsg string
	switch req.SessionState {
	case StateProductFocus:
		if req.FocusEntities != nil && len(req.FocusEntities.ProductIDs) > 0 {
			clarifyMsg = "您是问这个商品的价格、库存还是位置呢？"
		} else {
			clarifyMsg = "您想了解哪方面的信息呢？"
		}
	case StateListBrowse:
		clarifyMsg = "您想进一步了解哪个商品呢？"
	default:
		clarifyMsg = "能再具体说说您想问什么吗？"
	}

	r.log.Info("context_resolver_l3_clarify", "state", req.SessionState)
	return &ResolveResult{
		ResolvedEntities: nil,
		Layer:            "L3",
		NeedsClarify:     true,
		ClarifyMessage:   clarifyMsg,
		Confidence:       0.0,
	}
}

// ── 辅助：消息中是否包含新实体信号 ──────────────

// HasNewEntitySignal 简易启发式检查消息是否引入了新的商品/品类实体。
// 返回 true 意味着不应走 L1 继承，应进入 L2/L3。
func HasNewEntitySignal(message string) bool {
	newEntityWords := []string{
		"多少钱", "价格", "还有吗", "库存", "在哪", "位置",
		"还有几", "还有多少", "有货", "没货", "缺货",
	}
	for _, w := range newEntityWords {
		if strings.Contains(message, w) {
			return false // 这些问句通常是在追问当前焦点商品
		}
	}
	// 如果消息长度较长且不含上述追问词，可能引入了新实体
	if len([]rune(message)) > 10 {
		return true
	}
	return false
}
