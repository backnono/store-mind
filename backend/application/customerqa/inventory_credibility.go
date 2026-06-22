package customerqa

import (
	"time"

	domain "store-mind/domain/customerqa"
)

// CredLevel 库存可信度等级。
type CredLevel string

const (
	CredHigh          CredLevel = "high"           // 🟢 高可信：≤30min
	CredMedium        CredLevel = "medium"         // 🟡 中可信：≤2h
	CredLow           CredLevel = "low"            // 🟠 低可信：≤24h
	CredReferenceOnly CredLevel = "reference_only" // 🔴 仅供参考：>24h 或无数据
)

// CredInfo 库存可信度详情。
type CredInfo struct {
	Level          CredLevel  // 可信度等级
	Label          string     // 中文标签
	Color          string     // 颜色代码
	VerifiedAt     *time.Time // 最后验证时间
	AdviceTemplate string     // 建议口径模板后缀
}

// AllCredLevels 返回所有可信度等级定义，用于前端展示和 LLM prompt。
func AllCredLevels() []CredInfo {
	return []CredInfo{
		{
			Level:          CredHigh,
			Label:          "高可信",
			Color:          "#4CAF85",
			AdviceTemplate: "数据是{minutes}分钟前更新的",
		},
		{
			Level:          CredMedium,
			Label:          "中可信",
			Color:          "#DBB840",
			AdviceTemplate: "上次盘点显示还有{quantity}件（{time_ago}前），建议到货架确认",
		},
		{
			Level:          CredLow,
			Label:          "低可信",
			Color:          "#E8913A",
			AdviceTemplate: "今天早上盘点时还有货，在{location}，您可以直接去看看",
		},
		{
			Level:          CredReferenceOnly,
			Label:          "仅供参考",
			Color:          "#D45A3A",
			AdviceTemplate: "近期盘点记录显示有库存，但数据超过一天，建议到{location}查看",
		},
	}
}

// ComputeCredibility 根据库存记录的 last_verified_at 计算可信度。
// 返回 CredInfo 包含等级、标签、颜色和回答口径模板。
func ComputeCredibility(inv *domain.Inventory) CredInfo {
	levels := AllCredLevels()
	if inv == nil || inv.LastVerifiedAt == nil {
		return levels[3] // 无数据 → 仅供参考
	}

	elapsed := time.Since(*inv.LastVerifiedAt)
	switch {
	case elapsed <= 30*time.Minute:
		return levels[0] // 🟢 高可信
	case elapsed <= 2*time.Hour:
		return levels[1] // 🟡 中可信
	case elapsed <= 24*time.Hour:
		return levels[2] // 🟠 低可信
	default:
		return levels[3] // 🔴 仅供参考
	}
}

// CredibilityTag 返回库存可信度标签，用于注入 Evidence 的 Content。
// 例如："🟢 高可信 · 10分钟前更新"
func CredibilityTag(inv *domain.Inventory) string {
	cred := ComputeCredibility(inv)
	if inv == nil || inv.LastVerifiedAt == nil {
		return string(CredReferenceOnly)
	}

	elapsed := time.Since(*inv.LastVerifiedAt)
	timeDesc := formatTimeAgo(elapsed)

	return string(cred.Level) + " · " + timeDesc
}

// formatTimeAgo 将时长格式化为人类可读的中文描述。
func formatTimeAgo(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "刚刚更新"
	case d < time.Hour:
		return time.Duration(d.Minutes()).String() + "分钟前更新"
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1小时前更新"
		}
		return time.Duration(hours).String() + "小时前更新"
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1天前更新"
		}
		return time.Duration(days).String() + "天前更新"
	}
}
