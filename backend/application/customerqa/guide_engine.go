package customerqa

import (
	"strings"

	domain "store-mind/domain/customerqa"
)

// GuideEngine 主动引导决策引擎。
// 在 AnswerComposer 生成回答后调用，根据当前决策和证据状态决定是否触发引导及引导内容。
type GuideEngine interface {
	// Evaluate 根据当前上下文评估是否应触发引导。
	// 返回引导芯片列表，为空表示不触发。
	Evaluate(ctx GuideContext) []GuidanceChip
}

// GuideContext 引导引擎的输入上下文。
type GuideContext struct {
	Intent       string           // 当前意图
	Decision     Decision         // 编排决策
	Message      string           // 用户原始消息
	Evidence     []Evidence       // 证据列表
	SessionState SessionState     // 当前会话状态
	Products     []domain.Product // 检索到的商品（用于替代推荐等）
}

// ── 默认实现（规则引擎）───────────────────────────

// defaultGuideEngine 基于规则表实现主动引导。
type defaultGuideEngine struct {
	log Logger
}

// NewGuideEngine 创建基于规则的引导引擎。
func NewGuideEngine(log Logger) GuideEngine {
	if log == nil {
		log = nopLogger{}
	}
	return &defaultGuideEngine{log: log}
}

// Evaluate 按优先级评估 5 个引导触发条件。
func (e *defaultGuideEngine) Evaluate(ctx GuideContext) []GuidanceChip {
	var chips []GuidanceChip

	// 规则 1：商品位置回答后 → 推活动/库存追问
	if ctx.Intent == "product_location" && len(ctx.Products) > 0 {
		chips = append(chips, e.locationGuidance(ctx)...)
	}

	// 规则 2：缺货或未命中 → 推替代品
	if ctx.Intent == "inventory" && len(ctx.Evidence) == 0 {
		chips = append(chips, e.outOfStockGuidance(ctx)...)
	}

	// 规则 3：多商品列表（>5 个）→ 追问细化
	if ctx.Intent == "product_location" && len(ctx.Products) > 5 {
		chips = append(chips, e.listRefineGuidance(ctx)...)
	}

	// 规则 4：结算/退款 → 推进流程
	if ctx.Intent == "faq" && isTransactionRelated(ctx.Message) {
		chips = append(chips, e.transactionGuidance(ctx)...)
	}

	// 规则 5：沉默 >30s → 温和唤醒
	if ctx.SessionState == StateIdle || ctx.Decision.FallbackUsed {
		// 不在 AnswerComposer 之后触发（由前端/定时器控制），此处预留
	}

	if len(chips) > 0 {
		e.log.Info("guide_engine_triggered", "intent", ctx.Intent, "chip_count", len(chips))
	}
	return chips
}

// locationGuidance 商品位置回答后的引导：推活动和库存追问。
func (e *defaultGuideEngine) locationGuidance(ctx GuideContext) []GuidanceChip {
	return []GuidanceChip{
		{Text: "📦 还有几瓶？", Prompt: "还有几瓶？"},
		{Text: "🏷 这个有活动吗？", Prompt: "这个有活动吗？"},
	}
}

// outOfStockGuidance 缺货时的替代推荐引导。
func (e *defaultGuideEngine) outOfStockGuidance(ctx GuideContext) []GuidanceChip {
	return []GuidanceChip{
		{Text: "🥤 同品类还有什么？", Prompt: "同品类的都有什么？"},
		{Text: "🆕 有什么新品推荐？", Prompt: "有什么新品推荐？"},
	}
}

// listRefineGuidance 多商品列表时的细化引导。
func (e *defaultGuideEngine) listRefineGuidance(ctx GuideContext) []GuidanceChip {
	return []GuidanceChip{
		{Text: "💬 碳酸饮料有哪些？", Prompt: "碳酸饮料有哪些？"},
		{Text: "💬 茶饮有哪些？", Prompt: "茶饮有哪些？"},
	}
}

// transactionGuidance 支付/退款时的流程推进引导。
func (e *defaultGuideEngine) transactionGuidance(ctx GuideContext) []GuidanceChip {
	return []GuidanceChip{
		{Text: "🆘 联系人工客服", Prompt: "帮我联系人工客服"},
		{Text: "📋 查看退款流程", Prompt: "退款流程是什么？"},
	}
}

// isTransactionRelated 判断消息是否与结算/退款相关。
func isTransactionRelated(message string) bool {
	keywords := []string{"退款", "支付", "付款", "结算", "退货", "退款流程", "怎么退"}
	for _, kw := range keywords {
		if strings.Contains(message, kw) {
			return true
		}
	}
	return false
}
