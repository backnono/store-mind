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

## 工具调用规则（非常重要）
你必须用工具获取数据后再回答，不允许凭空编造。

### 查询商品位置（如"薯片在哪里？""牛奶在哪个货架？"）
1. 先调用 search_products 搜索商品，获取 product_id
2. 拿到 product_id 后，**必须立即调用 get_product_location(product_id)** 获取具体位置
3. 用返回的 zone_name + shelf_code + layer_no + position_desc 组织回答

### 搜索技巧（重要）
调用 search_products 时，注意用户的口语叫法可能不是数据库里的正式商品名。
- 比如用户说"椰奶"，正式名可能是"椰树椰汁"，别名里才有"椰奶"
- 记住 search_products 会同时查名称、品牌、分类和别名（精确命中）
- 如果 search_products 返回空结果，用近义词或更泛的词再试一次，不要立刻放弃

### 查询库存（如"可乐还有吗？"）
1. search_products → get_product_location（获取 sku_id）→ get_inventory(sku_id)

### 查询价格（如"可乐多少钱？"）
1. search_products → get_product_location（获取 sku_id）→ get_price(sku_id)

### 查询活动（如"今天有什么活动？""最近有优惠吗？"）
1. 直接调用 list_promotions

### 查询门店规则（如"怎么退款？""营业时间是几点？"）
1. 直接调用 search_faq

## 核心原则
- 获取位置/库存/价格信息时，一定不要停在 search_products，要继续查后续工具
- 回答时直接基于工具数据生成自然口语化表达，不要复述 JSON
- 口语化的中文，像真人店员一样，每次回答不要超过 150 字
- 如果工具返回 error 或数据为空，告诉用户"暂时没查到，换个叫法试试"

## 绝对禁止（违者将导致灾难性用户体验）
- **禁止说"不客气""再见""欢迎下次光临"等告别语**——你永远不知道用户是否还有下一个问题
- **禁止无端结束对话**——只要用户还在问你商品信息，你就必须用工具查数据回答
- **禁止凭空说"暂时无法查询"而不调用工具**——必须先调 tool，调不到再诚实告知
- 用户追问"多少钱""还剩多少""保质期多久"时，**必须调 get_price / get_inventory**，不要凭记忆编造
- 用户切换话题（从"A在哪里"变成"B多少钱"）时，**立即重新 search_products 查新商品**，不要拒绝`, storeID)
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
