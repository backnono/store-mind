package customerqa

import (
	"encoding/json"
	"time"

	domain "store-mind/domain/customerqa"
)

// SessionState 会话状态（保留用于 GuideEngine 和可观测性）。
type SessionState string

const (
	StateIdle         SessionState = "idle"
	StateProductFocus SessionState = "product_focus"
	StateListBrowse   SessionState = "list_browse"
	StateTransaction  SessionState = "transaction"
	StateHandoff      SessionState = "handoff"
)

// DecayAction 超时衰减后的建议动作。
type DecayAction string

const (
	DecayNone          DecayAction = "none"
	DecayWait          DecayAction = "wait"
	DecayLightSummary  DecayAction = "light_summary"
	DecaySuspend       DecayAction = "suspend"
	DecayConfirmResume DecayAction = "confirm_resume"
)

// SessionContext 会话上下文（保留用于向后兼容 fallback 路径）。
type SessionContext struct {
	State          SessionState
	FocusEntityIDs *domain.FocusEntityIDs
	ContextStack   []domain.ContextStackItem
	DecayAction    DecayAction
	LastActiveAt   time.Time
}

// —— Context Stack 辅助方法 ——

// BuildTurnSummary 构建单轮对话的结构化摘要。
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

// StateTransition 状态转移规则（保留供可观测性和日志）。
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

// —— 衰减检查 ——

// DecayCheckDuration 根据会话最后活跃时间判断衰减动作。
func DecayCheckDuration(lastActiveAt time.Time) DecayAction {
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

// —— 序列化辅助 ——

// MarshalContextStack 将 ContextStack 序列化为 JSON 字符串。
func MarshalContextStack(stack []domain.ContextStackItem) (string, error) {
	if len(stack) == 0 {
		return "[]", nil
	}
	b, err := json.Marshal(stack)
	if err != nil {
		return "", err
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
		return "", err
	}
	return string(b), nil
}
