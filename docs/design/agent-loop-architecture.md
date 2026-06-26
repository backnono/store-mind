# Agent 循环架构设计

> **状态**: 草案 · **日期**: 2026-06-26

---

## 一、架构对比

### 现状：显式编排器 + 三层消解 + 状态机

```
┌──────────────────────────────────────────────────────────────────────┐
│                        service.Chat()                                │
│                                                                      │
│  ① resolveSession → ② CreateMessage → ③ ContextResolver            │
│       ┌─────────────────────────────────────────────┐               │
│       │  L1: 规则继承 (hasPotentialEntity, emoji清洗)│               │
│       │  L2: LLM 指代消解 (AnaphoraClient)           │               │
│       │  L3: 澄清话术 (clarifyResponse)              │               │
│       │  shouldContinueToOrchestrator (话题切换检测) │               │
│       └─────────────────────────────────────────────┘               │
│     ↓                                                               │
│  ④ groundResolvedEntities (name→ID 反查)                             │
│     ↓                                                               │
│  ⑤ orchestrator.Run()                                               │
│       ┌─────────────────────────────────────────────┐               │
│       │  IntentAnalyzer → Decision(Route)            │               │
│       │  Retriever → Evidence[]                      │               │
│       │  AnswerComposer → Answer                     │               │
│       └─────────────────────────────────────────────┘               │
│     ↓                                                               │
│  ⑥ buildFocus / extractFocus / mergeFocus                           │
│  ⑦ CreateMessage + persistDecisionLog                                │
│                                                                      │
│  ❌ 问题: L1/L2/L3 三层间信息传递复杂，话题切换靠规则修补            │
│  ❌ 问题: focus_entity_ids 和 evidence 是两条独立通道，反复对齐      │
│  ❌ 问题: extractProductQuery 对 emoji/追问词做字符串替换，脆弱      │
└──────────────────────────────────────────────────────────────────────┘
```

### 目标：消息历史驱动的 Agent 循环

```
┌──────────────────────────────────────────────────────────────────────┐
│                     Agent 循环                                        │
│                                                                      │
│  ① 加载消息历史 (DB → []Message)                                     │
│  ② 追加用户消息到历史                                                │
│  ③ Tool 调用循环:                                                    │
│       ┌─────────────────────────────────────────────┐               │
│       │  LLM(history + tool_definitions) →           │               │
│       │    ├─ final_answer → 跳出循环                 │               │
│       │    ├─ tool_call: search_products("薯片")     │               │
│       │    │   → tool 返回 [{id:106, name, ...}]     │               │
│       │    │   → 追加 tool_result 到历史，继续循环    │               │
│       │    ├─ tool_call: get_product_location(106)   │               │
│       │    │   → tool 返回 {zone, shelf, layer}      │               │
│       │    │   → 追加 tool_result 到历史，继续    │               │
│       │    └─ final_answer → 跳出循环                │               │
│       └─────────────────────────────────────────────┘               │
│  ④ 剥离 final_answer，持久化完整消息历史到 DB                         │
│  ⑤ 返回 answer + cards + chips                                      │
│                                                                      │
│  ✅ 优势: 无需 L1/L2/L3，LLM attention 自然管理指代和话题切换        │
│  ✅ 优势: entity ID 内嵌在 tool_result 中，随消息历史自然传递        │
│  ✅ 优势: 话题切换自然发生: "椰奶在哪？" → LLM 从 history 找不到    │
│          椰奶 → 自行调用 search_products("椰奶") → 得到 id:113      │
└──────────────────────────────────────────────────────────────────────┘
```

---

## 二、模块存活矩阵

| 模块 | 命运 | 理由 |
|---|---|---|
| `orchestrator.go` | ✂️ 退役 | 意图路由/证据收集/编排逻辑全被 LLM 的 tool calling 取代 |
| `context_resolver.go` | ✂️ 退役 | L1/L2/L3 不再需要，指代消解由 LLM attention 层完成 |
| `service.go:Chat()` | 🔄 重写 | 从 7 步流程缩减为 4 步（加载历史 → agent 循环 → 持久化 → 返回） |
| `session_manager.go` | 🔄 简化 | 状态机退役，仅保留 ContextStack 的构建逻辑（用于 fallback/可观测） |
| `fallback.go` | ✅ 保留 | LLM 不可用时降级到关键词路由（完全不动） |
| `guide_engine.go` | ✅ 保留 | 引导芯片生成（不受影响） |
| `llm.go` | ✂️ 退役 | `IntentAnalyzer` + `AnswerComposer` 接口退役，LLM 通过 tool calling API 直接交互 |
| `inventory_credibility.go` | ✅ 保留 | 可信度计算工具，被 Tool 包装后使用 |
| `domain/*/entity.go` | ✅ 保留 | 数据结构不变 |
| `infra/persistence/mysql/*` | ✅ 保留 | Repository 不变，Tool 直接调用 |
| `api/http/handler_customer_qa.go` | 🔄 微调 | 请求/响应格式不变，内部调用改为 Agent 循环 |

---

## 三、核心伪代码

### 3.1 `agent.go` —— 新建文件

```go
package customerqa

import (
    "context"
    "encoding/json"
    "fmt"
)

// ── Tool 定义 ──────────────────────────────────────

// Tool 是 LLM 可调用的工具接口。
// Description 会注入 system prompt，告诉 LLM 何时使用此工具。
type Tool interface {
    Name() string
    Description() string
    Run(ctx context.Context, storeID, sessionID, messageID int64, input json.RawMessage) (string, error)
}

// 工具注册表
var agentTools = []Tool{
    &SearchProductsTool{},      // 按名称搜索商品
    &GetProductLocationTool{},  // 查询商品位置
    &GetInventoryTool{},        // 查询库存
    &ListPromotionsTool{},      // 列出活动
    &SearchFAQTool{},           // 搜索 FAQ
    &GetPriceTool{},            // 查询价格
}

// ── Tool 实现示例: search_products ─────────────────

type SearchProductsTool struct{}

func (t *SearchProductsTool) Name() string        { return "search_products" }
func (t *SearchProductsTool) Description() string {
    return `按关键词搜索门店商品。当用户询问商品是否存在、在哪里能买到时使用。
输入: {"query": "薯片"}
输出: [{"product_id":106, "name":"乐事原味薯片", "brand":"乐事", "category":"零食"}]`
}

func (t *SearchProductsTool) Run(ctx context.Context, storeID, sessionID, messageID int64, raw json.RawMessage) (string, error) {
    var input struct{ Query string `json:"query"` }
    json.Unmarshal(raw, &input)
    
    products, err := repo.SearchProducts(ctx, storeID, input.Query, 5)
    if err != nil {
        return fmt.Sprintf(`{"error": "%s"}`, err.Error()), err
    }
    
    type ProductResult struct {
        ProductID int64  `json:"product_id"`
        Name      string `json:"name"`
        Brand     string `json:"brand"`
        Category  string `json:"category"`
    }
    results := make([]ProductResult, len(products))
    for i, p := range products {
        results[i] = ProductResult{p.ID, p.Name, p.Brand, p.Category}
    }
    b, _ := json.Marshal(results)
    return string(b), nil
}

// ── 消息格式 ──────────────────────────────────────

// Message 是 Agent 循环中的标准消息格式。
// 对标 OpenAI ChatMessage 格式: system / user / assistant / tool
type AgentMessage struct {
    Role      string          `json:"role"`       // system | user | assistant | tool
    Content   string          `json:"content"`    // 文本内容
    ToolCalls []AgentToolCall `json:"tool_calls,omitempty"` // assistant 发起的工具调用
    ToolID    string          `json:"tool_id,omitempty"`    // tool 消息关联的调用 ID
}

type AgentToolCall struct {
    ID   string          `json:"id"`
    Name string          `json:"name"`
    Args json.RawMessage `json:"args"`
}

// ── Agent 循环核心 ─────────────────────────────────

// AgentLoop 执行一次多轮工具调用循环，直到 LLM 返回 final_answer。
//
// 入参:
//   - history: 从 DB 加载的完整消息历史（含前序轮次的 tool_call + tool_result）
//   - userMessage: 本轮用户输入
//   - storeID/sessionID/messageID: 上下文 ID，传给工具记录 tool_call 日志
//
// 出参:
//   - finalAnswer: LLM 的最终自然语言回答
//   - cards: 从 tool 返回结果中提取的结构化卡片
//   - updatedHistory: 更新后的完整消息历史（含本轮所有消息），用于持久化
func AgentLoop(
    ctx context.Context,
    llm LLMClient,
    history []AgentMessage,
    userMessage string,
    storeID, sessionID, messageID int64,
) (finalAnswer string, cards []ChatCard, updatedHistory []AgentMessage, err error) {

    // 1. 构建初始消息列表
    messages := append(history,
        AgentMessage{Role: "user", Content: userMessage},
    )

    // 2. 循环: LLM → 解析 → tool 调用 → 观察 → 再推理
    const maxLoops = 5 // 安全上限，防止死循环
    for loop := 0; loop < maxLoops; loop++ {
        
        // 2a. 调用 LLM (附带 tool 定义)
        resp, err := llm.Chat(ctx, messages, toolDefinitions())
        if err != nil {
            return "", nil, nil, fmt.Errorf("agent_loop_llm: %w", err)
        }

        // 2b. 有 tool_call → 执行
        if len(resp.ToolCalls) > 0 {
            // 将 assistant 的 tool_call 消息加入历史
            assistantMsg := AgentMessage{Role: "assistant", Content: resp.Content}
            for _, tc := range resp.ToolCalls {
                assistantMsg.ToolCalls = append(assistantMsg.ToolCalls, AgentToolCall{
                    ID: tc.ID, Name: tc.Name, Args: tc.Args,
                })
            }
            messages = append(messages, assistantMsg)

            // 执行每个 tool_call
            for _, tc := range resp.ToolCalls {
                tool := findTool(tc.Name)
                if tool == nil {
                    messages = append(messages, AgentMessage{
                        Role: "tool", ToolID: tc.ID,
                        Content: `{"error": "unknown tool: ` + tc.Name + `"}`,
                    })
                    continue
                }
                result, toolErr := tool.Run(ctx, storeID, sessionID, messageID, tc.Args)
                if toolErr != nil {
                    result = fmt.Sprintf(`{"error": "%s"}`, toolErr.Error())
                }
                messages = append(messages, AgentMessage{
                    Role: "tool", ToolID: tc.ID, Content: result,
                })
            }
            continue // 回到循环，让 LLM 看到 tool_result 后继续推理
        }

        // 2c. 无 tool_call → final answer，跳出循环
        finalAnswer = resp.Content
        break
    }

    if finalAnswer == "" {
        finalAnswer = "暂时无法回答这个问题，你可以换个问法，或联系人工客服。"
    }

    // 3. 从 tool 返回结果中提取结构化卡片
    cards = extractCards(messages)

    return finalAnswer, cards, messages, nil
}
```

### 3.2 `service.go` —— 简化后的 Chat

```go
func (s *service) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    // ① 参数校验 (不变)
    // ② resolveSession (不变)
    // ③ 加载消息历史 (替代 ContextResolver + SessionManager)
    
    history, err := s.repo.ListRecentMessages(ctx, session.ID, 20)
    agentHistory := convertToAgentMessages(history) // domain.Message → AgentMessage
    
    // ④ Agent 循环 (替代 orchestrator.Run + ContextResolver + focus 提取)
    answer, cards, updatedHistory, err := AgentLoop(
        ctx, s.llmClient, agentHistory, req.Message,
        req.StoreID, session.ID, msg.ID,
    )
    if err != nil {
        // 退回到 fallback orchestrator
        return s.fallbackChat(ctx, req, session, msg)
    }
    
    // ⑤ 持久化 (变简单了 — 整段历史一次性写入)
    s.persistAgentMessages(ctx, session.ID, updatedHistory)
    
    // ⑥ 返回 (提取最后一条 assistant 消息中的 cards/answer)
    return &ChatResponse{
        SessionID: session.ID,
        MessageID: lastAssistantMsgID(updatedHistory),
        Answer:    answer,
        Cards:     cards,
        // ...
    }, nil
}
```

### 3.3 `fallback.go` —— 完全不动

```go
// 当 LLM 不可用时，AgentLoop 内部或 service.Chat 捕获错误后，
// 退回到 fallbackOrchestrator.Run()。
// fallback 路径不受任何影响。
```

---

## 四、关键差异逐项对比

### 4.1 "薯片在哪里？"

| 维度 | 现状 | Agent 模式 |
|---|---|---|
| 实现 | entry_mode→orchestrator.Run→SearchProducts→GetProductLocation→AnswerComposer | LLM 收到 "薯片在哪里？" → tool_call: search_products("薯片") → tool_result: [{product_id:106,...}] → tool_call: get_product_location(106) → tool_result: {zone:"零食区",shelf:"A-03"} → 组织回答 |
| 焦点记录 | extractFocusFromEvidence → DB: focus_entity_ids={"product_ids":[106]} | tool_result 中已含 product_id:106，作为消息历史的一部分持久化 |
| 代码路径 | service.go 374-383 (10行) + orchestrator.go 400-530 (130行) | agent.go 约 80 行循环 |

### 4.2 "📦 还有几包？"（追问）

| 维度 | 现状 | Agent 模式 |
|---|---|---|
| 实现 | L1 规则: hasAttrQ + emoji清洗 → 继承 focus 106 → orchestrator inventory 路径 | LLM 从历史中看到上一轮 tool_result 里有 product_id:106 → tool_call: get_inventory(106) → 回答 |
| 可维护性 | 规则: hasPotentialEntity, emoji 清洗, "还有"关键词匹配 | 零规则。LLM 从历史中自提取 |

### 4.3 "椰奶在哪里？"（话题切换）⚠️ 当前最痛的点

| 维度 | 现状 | Agent 模式 |
|---|---|---|
| 实现 | L2 返回空→buildFocusFromResolution返回旧106→extractFocusFromEvidence补丁→mergeFocusIDs | LLM 看到历史中是薯片(id:106)，当前问"椰奶"→直接调用 search_products("椰奶") →得到113 → get_product_location(113) → 回答 |
| 代码路径 | 4 个函数，约 120 行，多个分支 | Agent 循环 80 行，无分支 |
| 边界 case | 每个新关键词需要加到 extractProductQuery 的 replacements 列表 | 不需要。LLM 自行判断"椰奶"是独立实体 |

### 4.4 "可乐在哪里，还有货吗？"（复合意图）

| 维度 | 现状 | Agent 模式 |
|---|---|---|
| 实现 | subIntents() 拆分 → 意图A走tool→意图B走tool→合并证据 | LLM 自然处理: search_products("可乐") → get_product_location(id) → get_inventory(sku) → 组织回答 |
| 支持度 | 需要显式的复合意图拆分逻辑 | LLM 在循环中自然地连续调用多个 tool |

---

## 五、风险矩阵

| 风险 | 等级 | 缓解措施 |
|---|---|---|
| LLM 幻觉 tool_call 参数 | 🟡 中 | Tool.Run 内做参数校验；无效调用返回 error JSON，LLM sees 后重试 |
| Tool 调用死循环 | 🟡 中 | `maxLoops=5` 硬上限；超限后返回 fallback 回答 |
| LLM 不调用 tool 直接回答 | 🟢 低 | FAQ 类问题恰好不需要 tool，直接回答是正确行为 |
| LLM 返回的 tool_args 格式错 | 🟡 中 | `json.Unmarshal` 容错 + 返回结构化 error |
| LLM 不可用时全链路挂 | 🔴 高 | **fallback 完全保留**，LLM 调用失败 → 退回到 `fallbackOrchestrator.Run()` |
| 延迟增加 | 🟡 中 | 每轮 Chat 可能触发 2-3 次 LLM 调用（vs 现状 1-2 次）；但并行 tool_call 可减少轮次 |

---

## 六、目录结构 & 文件拆分

### 6.1 项目分层（DDD-lite 不减层）

```
backend/
├── domain/customerqa/          ← 不动
│   ├── entity.go               # Message / Session / Product / Inventory ...
│   ├── repository.go           # Repository 接口（Tool 直接调用）
│   ├── knowledge.go
│   └── errors.go
│
├── application/customerqa/     ← 核心变更区
│   │
│   │  -- ✨ 新增 3 个文件 --
│   ├── agent.go                # Agent 循环 + LLM 交互
│   ├── agent_tools.go          # Tool 接口 + 6 个 Tool 实现
│   └── agent_message.go        # AgentMessage 结构 + domain.Message 互转
│   │
│   │  -- 🔄 保留但修改 --
│   ├── service.go              # Chat() 从 250 行精简到 ~60 行
│   ├── session_manager.go      # 删除状态机，保留 ContextStack 构建
│   └── guide_engine.go         # 不动，引导芯片生成
│   │
│   │  -- ✅ 保留不动 --
│   ├── fallback.go             # LLM 不可用时降级到关键词路由
│   ├── inventory_credibility.go # 可信度计算，被 Tool 包装后使用
│   └── logger.go               # 日志接口，不动
│   │
│   │  -- ✂️ 删除 --
│   ×   orchestrator.go         # 680 行 → 删除
│   ×   context_resolver.go     # 270 行 → 删除
│   ×   llm.go                  # IntentAnalyzer / AnswerComposer → 删除
│   ×   retriever.go            # Retriever → 删除（Tool 直接用 repo.SearchFAQ）
│
├── infra/                      ← 几乎不动
│   ├── ai/
│   │   ├── python_llm_client.go    # LLM 侧车适配（接口简化为 Chat 单方法）
│   │   └── fake_client.go          # 测试用 fake
│   ├── config/
│   │   └── config.go               # 不动
│   ├── logger/
│   │   └── ...                     # 不动
│   └── persistence/
│       └── mysql/
│           ├── db.go               # 不动
│           ├── models.go           # 不动
│           └── repository_*.go     # 不动
│
├── api/http/                   ← 微调
│   ├── handler_customer_qa.go      # 请求/响应格式不变，内部调用 agent.Chat
│   └── router.go                   # 不动
│
└── internal/
    └── bootstrap/
        └── app.go                  # 依赖注入简化
```

### 6.2 三个新文件的职责边界

| 文件 | 职责 | 约行数 | 改动频率 |
|---|---|---|---|
| `agent.go` | `AgentLoop()` 核心循环：组装 messages → 调 LLM → 解析 tool_call → 执行 → 追加结果 → 循环 → 返回 final_answer。**不含具体 tool 逻辑** | 120 | 低（稳定后基本不动） |
| `agent_tools.go` | `Tool` 接口定义 + 6 个具体 Tool 的 `Name()/Description()/Run()` 实现。每个 `Run()` 内部调用 `domain.Repository` 方法 | 250 | 中（随业务增长的唯一文件，新增 Tool 只改这里） |
| `agent_message.go` | `AgentMessage` 结构体 + `domain.Message` ↔ `AgentMessage` 互转 + `extractCards()` 从 tool_result 提取 ChatCard | 80 | 低 |

### 6.3 为什么拆成 3 个而非 1 个

```
agent.go          — 循环逻辑，是"框架"，改动频率最低
agent_tools.go    — Tool 定义，是"业务"，随新增能力而变化
agent_message.go  — 数据转换，是"协议"，几乎不变
```

三者关注点完全不同，合并会导致 450 行的巨石文件。

### 6.4 依赖注入的变化

```go
// 现状 bootstrap/app.go
orchestrator.NewDefaultOrchestrator(repo, log, analyzer, composer, retriever)
contextResolver := NewContextResolver(llmClient, log)
sessionManager := NewSessionManager(repo, log)
svc := NewService(repo, orchestrator, contextResolver, sessionManager, guideEngine, log)
// 6 个构造调用，4 个中间对象

// 改造后 bootstrap/app.go
agent := NewAgent(repo, llmClient, log)   // 3 个入参，1 个对象
svc := NewService(repo, agent, fallbackOrch, guideEngine, log)
// 2 个构造调用，无中间对象
```

### 6.5 Chat() 精简后形态

```go
func (s *service) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
    // ① 参数校验 (不变)
    // ② resolveSession (不变)
    // ③ 持久化用户消息 (不变)
    session, msg := ...
    
    // ④ 加载历史 → Agent 循环 (替代原步骤 ③④⑤⑥)
    history, _ := s.repo.ListRecentMessages(ctx, session.ID, 20)
    answer, cards, updatedMsgs, err := s.agent.Run(ctx, AgentRunRequest{
        StoreID:     req.StoreID,
        SessionID:   session.ID,
        MessageID:   msg.ID,
        History:     convertToAgentMessages(history),
        UserMessage: req.Message,
    })
    
    if err != nil {
        return s.fallbackChat(...)  // ⑤ LLM 不行 → 回退 fallback
    }
    
    s.persistAgentMessages(ctx, updatedMsgs)  // ⑥ 持久化
    return buildResponse(session, answer, cards) // ⑦ 组装返回
}
```

---

## 七、文件变更清单

| 操作 | 文件 | 说明 |
|---|---|---|
| ✨ 新建 | `agent.go` | AgentLoop 循环 + LLM 交互 |
| ✨ 新建 | `agent_tools.go` | Tool 接口 + 6 个 Tool 实现 |
| ✨ 新建 | `agent_message.go` | AgentMessage 结构 + DB 互转 |
| 🔄 重写 | `service.go` | Chat() 从 250 行精简到 ~60 行 |
| ✂️ 删除 | `orchestrator.go` | 680 行退役 |
| ✂️ 删除 | `context_resolver.go` | 270 行退役 |
| ✂️ 删除 | `llm.go` | 60 行退役 |
| ✂️ 删除 | `retriever.go` | Tool 直接用 repo.SearchFAQ |
| 🔄 简化 | `session_manager.go` | 状态机退役，保留 ContextStack 构建 |
| ✅ 不动 | `fallback.go` `guide_engine.go` `inventory_credibility.go` `entity.go` `repository.go` `handler_customer_qa.go` `bootstrap/app.go` 微调 |
