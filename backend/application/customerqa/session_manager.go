package customerqa

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	domain "store-mind/domain/customerqa"
)

// SessionState 会话状态机 5 种状态。
type SessionState string

const (
	StateIdle         SessionState = "idle"          // 初始/空闲
	StateProductFocus SessionState = "product_focus" // 用户聚焦于某个/某类商品
	StateListBrowse   SessionState = "list_browse"   // 用户浏览品类/活动列表
	StateTransaction  SessionState = "transaction"   // 用户发起结算/退款
	StateHandoff      SessionState = "handoff"       // 已转人工
)

// DecayAction 超时衰减后建议的动作。
type DecayAction string

const (
	DecayNone          DecayAction = "none"           // 无衰减，正常对话
	DecayWait          DecayAction = "wait"           // 30s 无输入 → 等待用户
	DecayLightSummary  DecayAction = "light_summary"  // 90s 无输入 → 轻度总结
	DecaySuspend       DecayAction = "suspend"        // 5min 无输入 → 挂起
	DecayConfirmResume DecayAction = "confirm_resume" // 30min+ 无输入 → 恢复确认
)

// SessionManager 管理会话状态机、Context Stack 压缩和超时衰减。
// 在每轮 Chat 前后分别调用 LoadSession / PersistSession，由 orchestrator 注入。
type SessionManager interface {
	// LoadSession 加载会话当前状态。
	// 返回当前状态、锁定实体、历史上下文栈、以及超时衰减建议。
	LoadSession(ctx context.Context, sessionID int64) (*SessionContext, error)

	// PersistSession 持久化本轮对话后的会话状态。
	// turnSummary: 本轮的结构化摘要，由 LLM（或 fallback）生成后传入。
	PersistSession(
		ctx context.Context,
		sessionID int64,
		newState SessionState,
		focusEntityIDs *domain.FocusEntityIDs,
		turnSummary *domain.ContextStackItem,
	) error

	// DecayCheck 根据会话最后活跃时间判断衰减动作。
	DecayCheck(ctx context.Context, sessionID int64) DecayAction
}

// SessionContext 单次 Chat 所需的完整会话上下文。
type SessionContext struct {
	State          SessionState              // 当前状态
	FocusEntityIDs *domain.FocusEntityIDs    // 锁定实体
	ContextStack   []domain.ContextStackItem // 最近 N 轮结构化摘要
	DecayAction    DecayAction               // 衰减建议
	LastActiveAt   time.Time                 // 最后活跃时间
}

// ── 默认实现 ───────────────────────────────────────

// defaultSessionManager 基于 domain.Repository 实现 SessionManager。
type defaultSessionManager struct {
	repo domain.Repository
	log  Logger
}

// NewSessionManager 创建默认的会话管理器。
func NewSessionManager(repo domain.Repository, log Logger) SessionManager {
	if log == nil {
		log = nopLogger{}
	}
	return &defaultSessionManager{repo: repo, log: log}
}

// LoadSession 加载会话上下文：
//  1. 获取会话记录
//  2. 查询最近 N 条 assistant 消息，提取 context_state / focus_entity_ids / context_stack
//  3. 衰减检查
func (m *defaultSessionManager) LoadSession(ctx context.Context, sessionID int64) (*SessionContext, error) {
	session, err := m.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("session_manager: get session: %w", err)
	}

	recentMessages, err := m.repo.ListRecentMessages(ctx, sessionID, 10)
	if err != nil {
		m.log.Warn("session_manager_load_messages", "session_id", sessionID, "error", err)
	}

	sc := &SessionContext{
		State:        StateIdle,
		LastActiveAt: session.StartedAt,
	}

	for i := len(recentMessages) - 1; i >= 0; i-- {
		msg := recentMessages[i]
		if msg.Role != "assistant" {
			continue
		}
		if msg.CreatedAt.After(sc.LastActiveAt) {
			sc.LastActiveAt = msg.CreatedAt
		}
		// 取最新一条 assistant 消息的状态信息
		if sc.State == StateIdle && msg.ContextState != nil {
			sc.State = SessionState(*msg.ContextState)
		}
		if sc.FocusEntityIDs == nil && msg.FocusEntityIDs != nil {
			sc.FocusEntityIDs = msg.FocusEntityIDs
		}
		if len(sc.ContextStack) == 0 && len(msg.ContextStack) > 0 {
			sc.ContextStack = msg.ContextStack
		}
	}

	// 取用户消息的最后时间作为活跃基准
	for i := len(recentMessages) - 1; i >= 0; i-- {
		msg := recentMessages[i]
		if msg.Role == "user" && msg.CreatedAt.After(sc.LastActiveAt) {
			sc.LastActiveAt = msg.CreatedAt
		}
	}

	// 确定衰减动作
	sc.DecayAction = m.decayCheck(sc.LastActiveAt)

	return sc, nil
}

// PersistSession 将更新后的会话状态持久化到最新一条 assistant 消息。
// 实际持久化在 service.Chat 中的 CreateMessage 已完成，
// 此处返回需要写入的字段，由调用方合并。
func (m *defaultSessionManager) PersistSession(
	ctx context.Context,
	sessionID int64,
	newState SessionState,
	focusEntityIDs *domain.FocusEntityIDs,
	turnSummary *domain.ContextStackItem,
) error {
	// Session state 和 context_stack 是在 service.Chat 中
	// 通过 CreateMessage 的 domain.Message 字段持久化的。
	// 本方法作为接口约定的验证入口，实际写入由调用方负责。
	m.log.Info("session_manager_persist", "session_id", sessionID, "state", newState)
	return nil
}

// DecayCheck 根据会话最后活跃时间返回衰减动作。
func (m *defaultSessionManager) DecayCheck(ctx context.Context, sessionID int64) DecayAction {
	sc, err := m.LoadSession(ctx, sessionID)
	if err != nil {
		return DecayNone
	}
	return m.decayCheck(sc.LastActiveAt)
}

// decayCheck 核心衰减逻辑：
//
//	≤30s → 正常
//	30s~90s → 等待用户
//	90s~5min → 轻度总结
//	5min~30min → 挂起
//	>30min → 恢复确认
func (m *defaultSessionManager) decayCheck(lastActiveAt time.Time) DecayAction {
	elapsed := time.Since(lastActiveAt)
	switch {
	case elapsed < 30*time.Second:
		return DecayNone
	case elapsed < 90*time.Second:
		return DecayWait
	case elapsed < 5*time.Minute:
		return DecayLightSummary
	case elapsed < 30*time.Minute:
		return DecaySuspend
	default:
		return DecayConfirmResume
	}
}

// ── Context Stack 辅助方法 ─────────────────────────

// BuildTurnSummary 构建单轮对话的结构化摘要。
// turn 从当前 context_stack 长度 + 1 自动推断。
func BuildTurnSummary(
	currentStack []domain.ContextStackItem,
	intent string,
	resolvedEntities []domain.ResolvedEntity,
	systemAction, systemSummary string,
) domain.ContextStackItem {
	turn := len(currentStack) + 1
	return domain.ContextStackItem{
		Turn:             turn,
		Intent:           intent,
		ResolvedEntities: resolvedEntities,
		SystemAction:     systemAction,
		SystemSummary:    systemSummary,
	}
}

// AppendContextStack 向 context_stack 追加新轮次摘要，保留最近 N 轮。
func AppendContextStack(
	stack []domain.ContextStackItem,
	item domain.ContextStackItem,
	maxKeep int,
) []domain.ContextStackItem {
	result := append(stack, item)
	if len(result) > maxKeep {
		result = result[len(result)-maxKeep:]
	}
	return result
}

// StateTransition 状态转移规则。
// 根据当前状态和本轮意图，返回新状态。
func StateTransition(current SessionState, intent string, userMessage string) SessionState {
	switch intent {
	case "handoff":
		return StateHandoff
	case "product_location", "inventory", "product_policy":
		if current == StateHandoff {
			return StateHandoff
		}
		return StateProductFocus
	case "promotion", "faq":
		if current == StateProductFocus || current == StateListBrowse {
			return current
		}
		return StateListBrowse
	case "transaction", "checkout", "refund":
		return StateTransaction
	case "unsupported":
		if current == StateHandoff {
			return StateHandoff
		}
		return StateIdle
	default:
		return current
	}
}

// ── 序列化辅助 ─────────────────────────────────────

// MarshalContextStack 将 ContextStack 序列化为 JSON 字符串。
func MarshalContextStack(stack []domain.ContextStackItem) (string, error) {
	if len(stack) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal(stack)
	if err != nil {
		return "", fmt.Errorf("marshal context_stack: %w", err)
	}
	return string(b), nil
}

// MarshalFocusEntityIDs 将 FocusEntityIDs 序列化为 JSON 字符串。
func MarshalFocusEntityIDs(ids *domain.FocusEntityIDs) (string, error) {
	if ids == nil {
		return "null", nil
	}
	b, err := json.Marshal(ids)
	if err != nil {
		return "", fmt.Errorf("marshal focus_entity_ids: %w", err)
	}
	return string(b), nil
}
