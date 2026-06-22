# S1 核心体验验证方案

> 目标：验证 `docs/design/gap-analysis/gap-analysis.html` 中 S1「核心体验升级」是否真正实现且体验合理，而不是只完成了局部模块或字段。

## 1. 验证范围

S1 覆盖五类能力：

1. 多轮对话：同一会话内连续追问“多少钱？”“还有几瓶？”能继承上一轮商品。
2. 库存可信度：库存回答带有可信度等级或口径。
3. 主动引导：位置回答后返回可点击的追问建议。
4. 首次打开：`entry_mode=first_open` 返回预设问题列表。
5. 货架扫码：`entry_mode=zone_scan` 返回当前货架商品。

本方案分为四层验证：

- 单元测试：证明核心规则本身正确。
- 服务层集成测试：证明 S1 模块进入 `Chat` 主链路。
- HTTP 验收：用真实接口验证用户可见行为。
- DB 检查：验证会话状态、焦点实体和上下文摘要被持久化。

## 2. 当前实现重点风险

阅读当前代码后，建议把以下点作为高优先级检查项：

| 风险 | 相关文件 | 验证方式 |
|---|---|---|
| `ContextResolver` 已注入 `service`，但 `Chat` 主流程看起来没有调用它 | `backend/application/customerqa/service.go` | 多轮追问 E2E + mock resolver 调用次数 |
| `OrchestratorRequest` 没有 resolved entities 字段，消解结果可能无法参与 tool 查询 | `backend/application/customerqa/orchestrator.go` | 第二轮“还有几瓶？”必须查到上一轮商品库存 |
| L1 规则似乎只在消息包含“那/这个/它”时继承，可能不覆盖“多少钱？”这种省略追问 | `backend/application/customerqa/context_resolver.go` | `product_focus + focus_product_ids + "多少钱？"` 单测 |
| Go 侧 `/llm/resolve` 客户端可能没有解析 `resolved_entities` | `backend/infra/ai/python_llm_client.go` | sidecar contract + Go client 单测 |
| `zone_scan` 可能未使用 `zone_id/shelf_id` 过滤当前货架 | `backend/application/customerqa/service.go` | HTTP 验收检查返回商品位置是否属于指定货架 |
| `GuideEngine` 在 `service.Chat` 后置调用时未传入 `Products` | `backend/application/customerqa/service.go` | 位置回答后 chips 是否稳定出现 |

## 3. 单元测试建议

### 3.1 Session Manager

建议新增：`backend/application/customerqa/session_manager_test.go`

测试点：

- `StateTransition`：
  - `idle + product_location -> product_focus`
  - `product_focus + inventory -> product_focus`
  - `idle + promotion -> list_browse`
  - `idle + faq(退款/结算相关) -> list_browse` 或按当前设计记录为行为差异
  - `any + handoff -> handoff`
- `AppendContextStack`：
  - 连续追加 6 轮时只保留最近 5 轮。
- `DecayCheck`：
  - `<30s -> none`
  - `30s~90s -> wait`
  - `90s~5min -> light_summary`
  - `5min~30min -> suspend`
  - `>=30min -> confirm_resume`

通过条件：所有边界值符合 S1 文档阈值，且上下文栈不会无限增长。

### 3.2 Context Resolver

建议新增：`backend/application/customerqa/context_resolver_test.go`

必须覆盖：

```text
state=product_focus
focus_product_ids=[101]
message="还有几瓶？"
预期：Layer=L1，NeedsClarify=false，ResolvedEntities 包含 product_id=101
```

```text
state=product_focus
focus_product_ids=[101]
message="多少钱？"
预期：Layer=L1，NeedsClarify=false，ResolvedEntities 包含 product_id=101
```

```text
state=idle
message="还有吗？"
预期：Layer=L3，NeedsClarify=true
```

```text
L2 返回 confidence=0.59
预期：进入 L3 澄清
```

通过条件：省略追问和显式指代词都能继承当前焦点商品；低置信度不会盲目消解。

### 3.3 Inventory Credibility

建议新增：`backend/application/customerqa/inventory_credibility_test.go`

测试矩阵：

| `last_verified_at` | 预期等级 |
|---|---|
| 10 分钟前 | `high` |
| 30 分钟前 | `high` |
| 1 小时前 | `medium` |
| 2 小时前 | `medium` |
| 3 小时前 | `low` |
| 24 小时前 | `low` |
| 25 小时前 | `reference_only` |
| nil inventory | `reference_only` |

通过条件：等级边界与 S1 文档一致，库存证据中包含可信度标签。

### 3.4 Guide Engine

建议新增：`backend/application/customerqa/guide_engine_test.go`

测试点：

- `product_location` 且有 evidence/cards 后，返回“还有几瓶？”和“这个有活动吗？”。
- `inventory` 且 evidence 为空时，返回替代推荐 chips。
- 商品列表超过 5 个时，返回细化追问 chips。
- 支付/退款类消息返回人工客服或退款流程 chips。

通过条件：位置回答后至少有 2 个引导建议，且建议文本可直接作为下一轮 prompt。

## 4. 服务层集成测试建议

建议补充到：`backend/application/customerqa/service_test.go`

### 4.1 首次打开入口

输入：

```go
ChatRequest{
    StoreID: 1,
    Channel: "miniapp",
    Message: "打开",
    EntryMode: "first_open",
}
```

通过条件：

- `Intent == "greeting"`
- `Meta.Route == "entry_first_open"`
- `len(GuidanceChips) >= 4`
- assistant message 被持久化
- `ContextState == "idle"`

### 4.2 货架扫码入口

输入：

```go
zoneID := int64(2)
shelfID := int64(3)
ChatRequest{
    StoreID: 1,
    Channel: "miniapp",
    Message: "扫码",
    EntryMode: "zone_scan",
    ZoneID: &zoneID,
    ShelfID: &shelfID,
}
```

通过条件：

- `Intent == "zone_scan"`
- `Meta.Route == "entry_zone_scan"`
- `len(Cards) > 0`
- 每张商品卡的位置属于 `zone_id=2, shelf_id=3` 对应货架。
- 如果实现未按 zone/shelf 过滤，应标为 S1 不完全通过。

### 4.3 多轮追问主链路

流程：

1. 第 1 轮：`可乐在哪里`
2. 第 2 轮：同一个 `session_id` 问 `还有几瓶？`
3. 第 3 轮：同一个 `session_id` 问 `多少钱？`

通过条件：

- 第 1 轮：`Intent == "product_location"`，有商品卡。
- 第 2 轮：`Intent == "inventory"`，回答/卡片指向第 1 轮同一商品。
- 第 3 轮：不应变成 `unsupported`，也不应泛化成 FAQ。
- 三轮 assistant message 的 `context_stack` 持续追加。
- 当前焦点实体被写入 `focus_entity_ids`。

建议用 mock/stub 增加一个硬性断言：

- `ContextResolver.Resolve` 在第 2/3 轮至少被调用一次。
- 若当前代码调用次数为 0，应判定为 S1 主链路未接通。

## 5. HTTP 验收用例

前置条件：

- MySQL 已执行 migrations。
- 已加载 `backend/db/seeds/001_faq_seed.sql`、`002_customer_qa_catalog_seed.sql`、`003_s0_test_data.sql`。
- Go 后端监听：`http://127.0.0.1:8080`
- Python sidecar 监听：`http://127.0.0.1:9090`

### 5.1 first_open

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/customer-qa/chat \
  -H 'Content-Type: application/json' \
  -d '{"store_id":1,"channel":"miniapp","message":"打开","entry_mode":"first_open"}'
```

通过条件：

- HTTP 200
- `intent == "greeting"`
- `meta.route == "entry_first_open"`
- `guidance_chips` 数量不少于 4
- chips 覆盖位置、活动、推荐、支付等常见入口。

### 5.2 zone_scan

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/customer-qa/chat \
  -H 'Content-Type: application/json' \
  -d '{"store_id":1,"channel":"miniapp","message":"扫码","entry_mode":"zone_scan","zone_id":2,"shelf_id":3}'
```

通过条件：

- HTTP 200
- `intent == "zone_scan"`
- `meta.route == "entry_zone_scan"`
- `cards` 非空
- cards 中商品位置属于指定 zone/shelf。

辅助 DB 校验：

```sql
SELECT p.id, p.name, z.name AS zone_name, s.code AS shelf_code
FROM product p
JOIN product_location pl ON pl.product_id = p.id
JOIN zone z ON z.id = pl.zone_id
JOIN shelf s ON s.id = pl.shelf_id
WHERE pl.store_id = 1 AND pl.zone_id = 2 AND pl.shelf_id = 3;
```

### 5.3 位置回答后主动引导

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/customer-qa/chat \
  -H 'Content-Type: application/json' \
  -d '{"store_id":1,"channel":"miniapp","message":"可乐在哪里"}'
```

通过条件：

- `intent == "product_location"`
- `cards` 非空
- `guidance_chips` 包含类似“还有几瓶？”、“这个有活动吗？”的建议。

### 5.4 库存可信度

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/customer-qa/chat \
  -H 'Content-Type: application/json' \
  -d '{"store_id":1,"channel":"miniapp","message":"可乐还有几瓶"}'
```

通过条件：

- `intent == "inventory"`
- `cards[0].type == "inventory"`
- 回答中出现可信度信息，例如：
  - `high`
  - `medium`
  - `low`
  - `reference_only`
  - 或中文等价表达：“高可信 / 中可信 / 低可信 / 仅供参考 / 几分钟前更新 / 小时前更新”

建议同时查数据库确认该 SKU 的时间：

```sql
SELECT i.sku_id, i.quantity, i.last_verified_at, i.update_source
FROM inventory i
WHERE i.store_id = 1 AND i.sku_id = 1001;
```

### 5.5 多轮指代消解

第一轮：

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/customer-qa/chat \
  -H 'Content-Type: application/json' \
  -d '{"store_id":1,"channel":"miniapp","message":"可乐在哪里"}'
```

记录返回的 `session_id`。

第二轮：

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/customer-qa/chat \
  -H 'Content-Type: application/json' \
  -d '{"store_id":1,"session_id":<SESSION_ID>,"channel":"miniapp","message":"还有几瓶？"}'
```

第三轮：

```bash
curl -s -X POST http://127.0.0.1:8080/api/v1/customer-qa/chat \
  -H 'Content-Type: application/json' \
  -d '{"store_id":1,"session_id":<SESSION_ID>,"channel":"miniapp","message":"多少钱？"}'
```

通过条件：

- 第二轮能回答上一轮商品库存，不应要求用户重新说明商品。
- 第三轮能围绕同一商品回答，不应变成“没有找到可靠依据”。
- 后两轮不应把“还有几瓶/多少钱”当作独立商品词去搜索。

DB 检查：

```sql
SELECT id, role, intent, context_state, focus_entity_ids, context_stack, created_at
FROM agent_message
WHERE session_id = <SESSION_ID>
ORDER BY id;
```

通过条件：

- assistant 消息包含 `context_state`。
- 至少一条 assistant 消息包含非空 `context_stack`。
- 聚焦商品后应存在非空 `focus_entity_ids`。
- `context_stack` 中应包含 `resolved_entities`，否则后续 L2 指代消解缺少实体基础。

### 5.6 Sidecar 指代消解契约

```bash
curl -s -X POST http://127.0.0.1:9090/llm/resolve \
  -H 'Content-Type: application/json' \
  -d '{
    "message":"还有几瓶？",
    "context_stack":[
      {
        "turn":1,
        "intent":"product_location",
        "resolved_entities":[{"type":"product","name":"可口可乐","product_id":101}],
        "system_summary":"告诉了用户可口可乐在饮料区 B-02 货架"
      }
    ],
    "focus_entities":{"product_ids":[101],"sku_ids":[1001]}
  }'
```

通过条件：

- HTTP 200
- `resolved_entities` 非空
- `confidence >= 0.6`

Go client 还需要单独验证：`PythonLLMClient.ResolveAnaphora` 必须把 JSON 里的 `resolved_entities` 解析到 `AnaphoraLLMResult.ResolvedEntities`，否则 sidecar 正确也无法进入 Go 主链路。

## 6. 建议自动化验收脚本

如果要延续 S0 风格，建议新增：

```text
backend/scripts/s1_acceptance.py
backend/scripts/s1_verify.sh
backend/scripts/test_s1_acceptance.py
```

建议 gate：

| Gate | 检查 |
|---|---|
| `entry` | `first_open` 和 `zone_scan` |
| `guidance` | 位置回答后有 guidance chips |
| `credibility` | 库存回答包含可信度 |
| `multi-turn` | 同 session 的“还有几瓶/多少钱”正确继承商品 |
| `persistence` | `context_state/focus_entity_ids/context_stack` 持久化 |
| `sidecar-resolve` | `/llm/resolve` 和 Go client 都能返回实体 |

输出目录建议：

```text
artifacts/s1/
├── s1-report.json
├── s1-report.md
└── s1-report.xml
```

## 7. S1 通过/不通过判定

### 必须全部通过

- `first_open` 返回预设问题列表。
- `zone_scan` 返回当前货架商品，而不是泛商品列表。
- 位置回答后有主动引导 chips。
- 库存回答用户可见地包含可信度信息。
- 同一会话内“还有几瓶？”和“多少钱？”能继承上一轮商品。

### 可接受但需记录为缺口

- LLM 回答措辞不固定，但事实、卡片和 meta 正确。
- 可信度以英文等级返回，但后续应产品化为中文徽章。
- `context_stack` 摘要由规则生成而非 LLM 压缩，只要结构稳定且可用于后续消解。

### 一票否决

- 多轮追问无法继承商品。
- `ContextResolver` 没有进入 `Chat` 主链路。
- `/llm/resolve` 返回了实体，但 Go client 丢失实体。
- `zone_scan` 忽略 `zone_id/shelf_id`。
- 库存可信度只存在内部 evidence，最终回答完全不可见。

## 8. 推荐执行顺序

1. 先跑现有回归：

   ```bash
   cd backend
   /opt/homebrew/opt/go@1.24/bin/go test ./...
   ```

2. 补 S1 单元测试，先暴露失败点。
3. 补服务层多轮集成测试，确认 `ContextResolver` 是否接入。
4. 启动后端和 sidecar，按第 5 节跑 HTTP 验收。
5. 根据 DB 检查判断是否只是“响应看起来对”，还是会话状态真的可持续。
6. 若要纳入 CI，再实现 `s1_acceptance.py` 和 `s1_verify.sh`。
