package customerqa

import (
	"encoding/json"
	"fmt"
	"time"

	domain "store-mind/domain/customerqa"
)

// —— Agent 消息格式 ——

// AgentMessage 是 Agent 循环中的标准消息格式。
// 对标 OpenAI ChatMessage 格式: system / user / assistant / tool
type AgentMessage struct {
	Role      string          `json:"role"`                 // system | user | assistant | tool
	Content   string          `json:"content"`              // 文本内容
	ToolCalls []AgentToolCall `json:"tool_calls,omitempty"` // assistant 发起的工具调用
	ToolID    string          `json:"tool_id,omitempty"`    // tool 消息关联的调用 ID
}

// AgentToolCall 单次工具调用。
type AgentToolCall struct {
	ID   string          `json:"id"`
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

// —— domain.Message ↔ AgentMessage 互转 ——

// ConvertToAgentMessages 将 domain.Message 列表转换为 AgentMessage 列表。
// 遍历 DB 中的消息历史，将 user/assistant 消息及其附带的 tool_call 展平
// 为 Agent 循环所需的顺序消息流。
func ConvertToAgentMessages(msgs []domain.Message) []AgentMessage {
	result := make([]AgentMessage, 0, len(msgs))
	for _, m := range msgs {
		switch m.Role {
		case "user":
			result = append(result, AgentMessage{
				Role:    "user",
				Content: m.Content,
			})
		case "assistant":
			// assistant 消息可能携带 content 和 tool_calls
			am := AgentMessage{
				Role:    "assistant",
				Content: m.Content,
			}
			// 从 Content 中尝试反序列化 tool_calls（如果持久化时嵌入了）
			if m.ToolCallsJSON != nil && *m.ToolCallsJSON != "" {
				var tcs []AgentToolCall
				if err := json.Unmarshal([]byte(*m.ToolCallsJSON), &tcs); err == nil {
					am.ToolCalls = tcs
				}
			}
			result = append(result, am)
		case "tool":
			toolID := ""
			if m.ToolCallID != nil {
				toolID = *m.ToolCallID
			}
			result = append(result, AgentMessage{
				Role:    "tool",
				Content: m.Content,
				ToolID:  toolID,
			})
		}
	}
	return result
}

// ConvertToDomainMessages 将 AgentMessage 列表转换回 domain.Message 列表用于持久化。
// 将 assistant 消息拆分为：assistant 文本消息 + 可能的 tool_call 记录。
// 将 tool 消息转换为 tool 角色消息。
func ConvertToDomainMessages(sessionID int64, agentMsgs []AgentMessage) []domain.Message {
	result := make([]domain.Message, 0, len(agentMsgs))
	for _, am := range agentMsgs {
		msg := domain.Message{
			SessionID: sessionID,
			Role:      am.Role,
			Content:   am.Content,
		}
		if am.Role == "assistant" && len(am.ToolCalls) > 0 {
			// 将 tool_calls 序列化存入 ToolCallsJSON
			b, _ := json.Marshal(am.ToolCalls)
			s := string(b)
			msg.ToolCallsJSON = &s
		}
		if am.Role == "tool" && am.ToolID != "" {
			msg.ToolCallID = &am.ToolID
		}
		result = append(result, msg)
	}
	return result
}

// SystemPrompt 返回 Agent 循环的 system prompt，含工具使用说明。
func SystemPrompt(storeID int64) string {
	return fmt.Sprintf(`你是「小王」，一位友好、专业的零售门店数字店员。
门店 ID: %d

## 你的能力
你可以通过调用工具来回答用户关于以下方面的问题：
- 商品搜索：某商品是否在售、在哪里能买到
- 商品位置：商品在哪个区域、货架、第几层
- 库存查询：商品还有多少件
- 价格查询：商品售价
- 促销活动：当前有哪些活动
- FAQ：门店规则、营业时间、退换货政策等

## 工作流程
1. 理解用户问题，判断需要哪些工具
2. 按需调用工具获取数据
3. 基于工具返回的数据组织自然、友好的回答
4. 如果没有工具能回答用户问题，友好地告知用户

## 回答要求
- 使用口语化的中文，像真人店员一样
- 如果有商品位置和库存数据，主动告知
- 如果商品未找到，建议用户换个叫法或联系人工客服
- 回答要简洁，不要超过 200 字
- 不要编造没有从工具获得的数据`, storeID)
}

// extractCards 从 Agent 循环的消息历史中提取 ChatCard 结构化卡片。
// 遍历所有 tool 消息，解析其 JSON 内容，抽取 product/inventory/promotion/faq 卡片。
func extractCards(msgs []AgentMessage) []ChatCard {
	cards := make([]ChatCard, 0)
	for _, m := range msgs {
		if m.Role != "tool" {
			continue
		}
		// 尝试解析为结构化卡片
		if card := parseCardFromToolResult(m.Content); card != nil {
			cards = append(cards, *card)
		}
	}
	return cards
}

// parseCardFromToolResult 从单个 tool_result JSON 中提取 ChatCard。
func parseCardFromToolResult(raw string) *ChatCard {
	if raw == "" {
		return nil
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil
	}
	// 根据 card_type 字段提取对应卡片
	cardType, _ := data["card_type"].(string)
	switch cardType {
	case "product":
		card := &ChatCard{Type: "product"}
		if v, ok := data["name"].(string); ok {
			card.Name = v
		}
		if v, ok := data["location"].(string); ok {
			card.Location = v
		}
		if v, ok := data["sku_id"].(float64); ok {
			card.SKUID = int64(v)
		}
		return card
	case "inventory":
		card := &ChatCard{Type: "inventory"}
		if v, ok := data["name"].(string); ok {
			card.Name = v
		}
		if v, ok := data["location"].(string); ok {
			card.Location = v
		}
		if v, ok := data["quantity"].(float64); ok {
			card.Quantity = int(v)
		}
		if v, ok := data["sku_id"].(float64); ok {
			card.SKUID = int64(v)
		}
		return card
	case "promotion":
		card := &ChatCard{Type: "promotion"}
		if v, ok := data["title"].(string); ok {
			card.Title = v
		}
		if v, ok := data["content"].(string); ok {
			card.Content = v
		}
		if v, ok := data["validity"].(string); ok {
			card.Validity = v
		}
		return card
	case "faq":
		card := &ChatCard{Type: "faq"}
		if v, ok := data["title"].(string); ok {
			card.Title = v
		}
		if v, ok := data["content"].(string); ok {
			card.Content = v
		}
		return card
	case "price":
		card := &ChatCard{Type: "price"}
		if v, ok := data["name"].(string); ok {
			card.Name = v
		}
		if v, ok := data["location"].(string); ok {
			card.Location = v
		}
		if v, ok := data["sku_id"].(float64); ok {
			card.SKUID = int64(v)
		}
		return card
	}
	return nil
}

// lastAssistantContent 从消息列表中提取最后一条 assistant 消息的文本内容。
func lastAssistantContent(msgs []AgentMessage) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "assistant" && msgs[i].Content != "" {
			return msgs[i].Content
		}
	}
	return ""
}

// AgentLoopTimestamp 返回当前时间戳，供 Agent 循环记录耗时。
var AgentLoopTimestamp = time.Now
