# 无人超市顾客问答 Agent 系统需求设计

## 1. 项目定位

本项目从“大而全的无人超市 Agent 系统”收敛为“无人超市顾客问答 Agent 系统”。

一期目标是实现一个面向顾客的门店问答 Agent。顾客可以通过小程序、店内屏幕或语音入口询问商品位置、库存、优惠活动、门店规则和支付/售后流程，系统基于门店结构化数据和知识库给出准确、简短、可执行的回答。

系统一期不追求 Amazon Go 式纯视觉无人结算，也不优先建设复杂风控、摄像头追踪或多 Agent 自主协作。核心价值是先把“顾客在无人值守门店里遇到问题时能快速得到可靠答案”这件事做扎实。

## 2. 产品目标

### 2.1 一期目标

- 让顾客能自然语言查询商品位置。
- 让顾客能查询商品库存状态。
- 让顾客能查询优惠活动和推荐商品。
- 让顾客能查询支付、退款、客服、营业时间等门店 FAQ。
- 让系统能追踪每次 Agent 的意图识别、工具调用和最终回答。
- 让运营人员能维护商品、货架、库存、活动和 FAQ 数据。

### 2.2 非目标

一期不做以下能力：

- 自动无人结算。
- 视觉识别结算。
- 多摄像头用户轨迹追踪。
- 偷盗行为自动识别。
- 复杂风控 Agent。
- 自动补货决策。
- 多 Agent 自主协作。
- 全量运营数据分析。

## 3. 用户角色

### 3.1 顾客

顾客是一期核心用户。顾客进入门店后，通过文字或语音向 Agent 提问，获得商品、库存、活动和门店流程相关答案。

### 3.2 门店运营人员

运营人员负责维护商品信息、货架位置、库存、活动和 FAQ。运营人员不是一期 Agent 的主要问答对象，但后台数据维护能力必须存在，否则顾客 Agent 无法稳定回答。

### 3.3 平台管理员

平台管理员负责系统配置、门店配置、权限管理、日志查看和异常排查。一期只需要基础后台管理能力。

## 4. 核心使用场景

### 4.1 商品位置查询

顾客提问：

> 薯片在哪里？

系统回答：

> 薯片在零食区 A-03 货架第二层，进门后右手边。

系统需要支持商品名、商品别名、品类和模糊关键词。例如“可乐”“汽水”“无糖饮料”“纸巾”“儿童零食”。

### 4.2 商品库存查询

顾客提问：

> 可乐还有吗？

系统回答：

> 系统显示可口可乐 500ml 还有 12 瓶，在饮料区 B-02 货架。

库存回答需要避免过度承诺。推荐使用“系统显示”“当前库存记录显示”等表达，降低实时库存误差带来的风险。

### 4.3 商品推荐

顾客提问：

> 有没有低糖饮料？

系统回答：

> 有。当前可选的低糖饮料包括元气森林白桃味、无糖可乐和东方树叶乌龙茶，主要在饮料区 B-02 和 B-03 货架。

推荐逻辑一期以商品标签和品类匹配为主，不做复杂个性化推荐。

### 4.4 优惠活动查询

顾客提问：

> 今天有什么优惠？

系统回答：

> 今天饮料区有第二件半价活动，活动商品包括可口可乐、雪碧和无糖可乐，有效期到今晚 23:59。

系统只回答当前有效活动，不展示过期活动。

### 4.5 门店流程问答

顾客提问：

> 怎么付款？

系统回答：

> 你可以使用小程序扫码结算。确认购物车商品后选择微信或支付宝支付。如支付失败，可以点击页面右上角联系人工客服。

FAQ 覆盖支付、退款、发票、客服、营业时间、会员、异常处理等门店规则。

### 4.6 未命中或低置信度问题

顾客提问：

> 有没有适合糖尿病人的饮料？

如果系统没有可靠规则或商品标签，回答：

> 我暂时不能判断是否适合特定健康情况。你可以查看商品配料表和营养成分表，或选择标注为无糖/低糖的饮料。

Agent 不能编造医学、法律、金融等高风险建议。

## 5. 功能范围

### 5.1 顾客问答入口

- 支持文字输入。
- 支持后续扩展语音输入。
- 展示 Agent 回答。
- 支持展示商品卡片、货架位置、活动卡片。
- 支持“联系人工”入口。

### 5.2 Agent 意图识别

系统需要识别以下意图：

- 商品位置查询。
- 商品库存查询。
- 商品推荐。
- 活动查询。
- 门店 FAQ。
- 人工客服请求。
- 无关问题或无法回答问题。

### 5.3 工具调用

Agent 不直接凭模型记忆回答门店事实问题，而是通过工具查询结构化数据。

一期工具包括：

- `search_products`：按商品名、别名、品类、标签搜索商品。
- `get_product_location`：查询商品所在区域、货架和陈列描述。
- `get_inventory`：查询 SKU 当前库存。
- `search_promotions`：查询有效活动。
- `search_faq`：查询门店 FAQ。
- `create_handoff_request`：生成转人工请求。

### 5.4 答案生成

答案需要满足：

- 简短明确。
- 优先给行动信息。
- 不编造商品、库存、活动和门店规则。
- 对库存使用谨慎表达。
- 找不到结果时给出替代建议或人工入口。
- 不回答与门店服务无关的问题。

### 5.5 后台管理

后台至少支持：

- 门店管理。
- 区域管理。
- 货架管理。
- 商品管理。
- SKU 管理。
- 库存管理。
- 活动管理。
- FAQ 管理。
- Agent 会话日志查看。
- Agent 工具调用日志查看。

## 6. 系统架构

### 6.1 MVP 架构

```text
顾客入口
  ↓
Chat UI / Voice UI
  ↓
Agent API
  ↓
Intent Router
  ↓
Tool Calling
  ├── Product Tool
  ├── Location Tool
  ├── Inventory Tool
  ├── Promotion Tool
  └── FAQ Tool
  ↓
Answer Generator
  ↓
顾客回答
```

### 6.2 推荐技术选型

- 前端：Web / 小程序 / 店内屏 WebView。
- 后端：Go 或 Node.js。
- Agent 服务：Python + LangGraph 或轻量 Tool Calling 服务。
- LLM：OpenAI / Qwen / Claude，根据部署和成本选择。
- 数据库：MySQL。
- 缓存：Redis，可选。
- 向量库：一期可选，FAQ 量较小时可先用 MySQL 全文索引或关键词检索。
- ASR/TTS：二期引入。
- MQTT/IoT：后续阶段引入。

### 6.3 架构原则

- 一期优先业务闭环，不堆复杂中间件。
- 门店事实类答案必须查工具。
- Agent 服务与业务数据服务解耦。
- 所有工具调用必须记录日志。
- 无法回答时必须有兜底策略。

## 7. 数据模型

### 7.1 门店表 `store`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | bigint | 门店 ID |
| name | varchar | 门店名称 |
| address | varchar | 地址 |
| status | varchar | 状态 |
| created_at | datetime | 创建时间 |
| updated_at | datetime | 更新时间 |

### 7.2 区域表 `zone`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | bigint | 区域 ID |
| store_id | bigint | 门店 ID |
| code | varchar | 区域编码 |
| name | varchar | 区域名称 |
| description | varchar | 位置描述 |

### 7.3 货架表 `shelf`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | bigint | 货架 ID |
| store_id | bigint | 门店 ID |
| zone_id | bigint | 区域 ID |
| code | varchar | 货架编码 |
| name | varchar | 货架名称 |
| description | varchar | 货架描述 |

### 7.4 商品表 `product`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | bigint | 商品 ID |
| name | varchar | 商品名称 |
| brand | varchar | 品牌 |
| category | varchar | 分类 |
| aliases | json | 别名 |
| tags | json | 标签 |
| status | varchar | 状态 |

### 7.5 SKU 表 `sku`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | bigint | SKU ID |
| product_id | bigint | 商品 ID |
| barcode | varchar | 条码 |
| spec | varchar | 规格 |
| price | decimal | 售价 |
| status | varchar | 状态 |

### 7.6 陈列表 `product_location`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | bigint | 陈列 ID |
| store_id | bigint | 门店 ID |
| product_id | bigint | 商品 ID |
| sku_id | bigint | SKU ID，可为空 |
| zone_id | bigint | 区域 ID |
| shelf_id | bigint | 货架 ID |
| layer_no | int | 层号 |
| position_desc | varchar | 描述性位置 |

### 7.7 库存表 `inventory`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | bigint | 库存 ID |
| store_id | bigint | 门店 ID |
| sku_id | bigint | SKU ID |
| quantity | int | 库存数量 |
| safety_stock | int | 安全库存 |
| updated_at | datetime | 更新时间 |

### 7.8 活动表 `promotion`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | bigint | 活动 ID |
| store_id | bigint | 门店 ID |
| title | varchar | 活动名称 |
| description | text | 活动说明 |
| product_scope | json | 适用商品或分类 |
| start_at | datetime | 开始时间 |
| end_at | datetime | 结束时间 |
| status | varchar | 状态 |

### 7.9 FAQ 表 `faq`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | bigint | FAQ ID |
| store_id | bigint | 门店 ID |
| question | varchar | 标准问题 |
| answer | text | 标准答案 |
| category | varchar | 分类 |
| keywords | json | 关键词 |
| status | varchar | 状态 |

### 7.10 Agent 会话表 `agent_session`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | bigint | 会话 ID |
| store_id | bigint | 门店 ID |
| user_id | bigint | 用户 ID，可为空 |
| channel | varchar | 来源渠道 |
| started_at | datetime | 开始时间 |
| ended_at | datetime | 结束时间 |

### 7.11 Agent 消息表 `agent_message`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | bigint | 消息 ID |
| session_id | bigint | 会话 ID |
| role | varchar | user/assistant/system |
| content | text | 消息内容 |
| intent | varchar | 识别意图 |
| confidence | decimal | 置信度 |
| created_at | datetime | 创建时间 |

### 7.12 Agent 工具调用表 `agent_tool_call`

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | bigint | 调用 ID |
| session_id | bigint | 会话 ID |
| message_id | bigint | 消息 ID |
| tool_name | varchar | 工具名称 |
| input_json | json | 输入参数 |
| output_json | json | 输出结果 |
| latency_ms | int | 耗时 |
| success | tinyint | 是否成功 |
| error_message | text | 错误信息 |
| created_at | datetime | 创建时间 |

## 8. Agent 回答规则

### 8.1 基本规则

- 商品、库存、活动、门店规则必须基于工具结果回答。
- 工具无结果时，不能编造。
- 库存回答必须提示数据来源或状态。
- 回答优先包含位置、数量、有效期、下一步动作。
- 遇到退款、支付失败、投诉等问题时，提供人工客服入口。
- 遇到医疗、法律、金融等高风险问题时，不给专业判断。

### 8.2 示例

商品位置：

> 找到了。薯片在零食区 A-03 货架第二层，进门后右手边。

库存：

> 系统显示可口可乐 500ml 还有 12 瓶，在饮料区 B-02 货架。

未找到：

> 我没有找到“椰子水”的在售信息。你可以看看饮料区 B-03 的果汁和茶饮，或联系人工客服确认。

无关问题：

> 我主要负责回答本门店的商品、库存、优惠和购物流程问题。你可以问我商品在哪里，或者今天有什么活动。

## 9. 接口草案

### 9.1 发送消息

`POST /api/agent/chat`

请求：

```json
{
  "store_id": 1,
  "session_id": "optional",
  "user_id": "optional",
  "channel": "miniapp",
  "message": "可乐在哪里？"
}
```

响应：

```json
{
  "session_id": "s_123",
  "message_id": "m_456",
  "intent": "product_location",
  "answer": "可乐在饮料区 B-02 货架第一层。",
  "cards": [
    {
      "type": "product",
      "sku_id": 1001,
      "name": "可口可乐 500ml",
      "location": "饮料区 B-02 货架第一层"
    }
  ],
  "handoff_required": false
}
```

### 9.2 查询会话历史

`GET /api/agent/sessions/{session_id}/messages`

### 9.3 后台维护接口

后台接口按资源拆分：

- `/api/admin/stores`
- `/api/admin/zones`
- `/api/admin/shelves`
- `/api/admin/products`
- `/api/admin/skus`
- `/api/admin/inventory`
- `/api/admin/promotions`
- `/api/admin/faqs`
- `/api/admin/agent/sessions`
- `/api/admin/agent/tool-calls`

## 10. 异常与兜底

### 10.1 商品未命中

处理策略：

- 尝试别名、关键词、品类匹配。
- 返回相近商品或相近品类。
- 提供人工入口。

### 10.2 库存异常

处理策略：

- 如果库存数据为空，回答“暂时无法确认库存”。
- 如果库存更新时间过旧，提示数据更新时间。
- 不承诺“绝对有货”。

### 10.3 工具调用失败

处理策略：

- 重试一次。
- 仍失败则返回兜底回答。
- 记录失败日志。
- 必要时触发后台告警。

### 10.4 LLM 低置信度

处理策略：

- 请求用户澄清。
- 展示候选商品或候选问题。
- 不直接编造答案。

## 11. MVP 验收标准

### 11.1 功能验收

- 输入明确商品名，系统能返回正确位置。
- 输入商品别名，系统能匹配正确商品。
- 输入品类，系统能返回相关商品列表。
- 询问库存，系统能返回库存状态。
- 询问活动，系统能返回当前有效活动。
- 询问支付/退款/客服，系统能返回 FAQ。
- 无法回答时，系统不会编造事实。
- 每次回答可追踪意图和工具调用。
- 后台可以维护商品、货架、库存、活动和 FAQ。

### 11.2 性能验收

- 常见问题平均响应时间小于 3 秒。
- 工具查询平均响应时间小于 500 毫秒。
- 单门店支持至少 1000 个 SKU。
- FAQ 支持至少 200 条。

### 11.3 质量验收

- 商品位置问答准确率达到 95% 以上。
- 库存问答命中率达到 95% 以上。
- FAQ 命中率达到 90% 以上。
- 无关问题拒答率达到 95% 以上。
- 工具调用日志完整率达到 100%。

## 12. 阶段规划

### Phase 1：文字问答 MVP

- 商品查询。
- 位置查询。
- 库存查询。
- 活动查询。
- FAQ。
- 工具调用日志。
- 后台基础配置。

### Phase 2：语音问答

- ASR 接入。
- TTS 接入。
- 店内屏幕或语音设备接入。
- 多轮对话上下文。
- 常见追问支持。

### Phase 3：运营 Agent

- 运营人员查询库存。
- 运营人员查询补货建议。
- 运营人员查询设备状态。
- 销售和库存摘要。
- 异常事件摘要。

### Phase 4：IoT 与风控增强

- MQTT 设备接入。
- 门锁、摄像头、传感器状态查询。
- 异常事件流。
- 基础风控事件解释。
- 人工告警协同。

## 13. 后续演进

顾客问答 Agent 跑通后，可以逐步演进为完整无人超市智能系统：

- 从顾客问答扩展到运营问答。
- 从静态库存查询扩展到 IoT 实时库存。
- 从 FAQ 扩展到门店知识库。
- 从单门店扩展到多门店 SaaS。
- 从规则推荐扩展到个性化推荐。
- 从人工维护数据扩展到自动巡检和异常发现。

## 14. 关键取舍

一期最重要的取舍是：先做可靠问答，而不是复杂自动化。

可靠问答的核心不是模型本身，而是结构化门店数据、清晰工具边界、可追踪工具调用和严格兜底策略。只要这个闭环稳定，后续语音、IoT、运营 Agent 和风控 Agent 都可以在同一套基础上演进。
