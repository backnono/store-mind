# Backend Agent 约束（严格版）

本文件作用于 `backend/` 子树。凡是后端相关任务，必须遵守以下规则。

## 1. 作用域与边界

- 仅在 `backend/` 目录内实施后端变更。
- 禁止在后端任务中修改 `frontend/` 目录内容。
- 若需求必须跨前后端改动，先明确拆分为两个提交或两个任务批次。

## 2. 分层依赖规则（强制）

当前目录分层：

- `api/`：接口适配层（HTTP）
- `application/`：用例编排层
- `domain/`：领域模型与仓储契约层
- `infra/`：基础设施实现层（DB、配置、日志）
- `internal/bootstrap/`：依赖装配
- `cmd/`：程序入口

强制依赖方向：

- 允许：`api -> application -> domain`
- 允许：`infra` 实现 `domain/application` 定义的接口
- 允许：`cmd/internal/bootstrap` 组装依赖
- 禁止：`domain` 依赖 `application/api/infra/cmd`
- 禁止：`application` 依赖 `api/cmd`
- 禁止：`api` 直接访问 `infra` 具体存储实现（必须经 application/domain 抽象）

## 3. 变更入口约束

- 新增业务能力时，必须按“契约先行”推进：
  1. 先定义 `domain` 实体与仓储接口
  2. 再实现 `application` 用例
  3. 再接入 `api` handler/router
  4. 最后补 `infra` 具体实现与迁移脚本
- 禁止跳层直接在 handler 中写业务逻辑或 SQL。

## 4. 数据与迁移约束

- 结构变更必须落在 `db/migrations/*.sql`，命名按递增前缀。
- 示例数据必须落在 `db/seeds/*.sql`，不得混入 migration。
- 任何新增表/字段，需同步更新：
  - `infra/persistence/mysql/models.go`
  - 对应 repository 实现
  - 必要的查询索引

## 5. 脚本与命令约束

- 默认在 `backend/` 目录执行命令。
- 标准命令：
  - 测试：`/opt/homebrew/opt/go@1.24/bin/go test ./...`
  - 迁移：`make migrate-up`
  - 种子：`make seed`
  - 冒烟：`make smoke`（端口冲突时使用 `DB_PORT=3308 bash scripts/smoke.sh`）
- 修改脚本后，必须至少验证一次 `go test`。

## 6. 测试与提交流程（强制）

- 无测试依据不得宣称“完成/修复”。
- 涉及行为变更至少满足：
  - 1 个 application 层测试
  - 1 个 api 层测试（`httptest`）
- 提交前至少执行：
  1. `go test ./...`
  2. 若涉及 DB/接口链路，执行 `make smoke` 或等效命令

## 7. 错误处理与可观测性

- API 统一返回结构化错误（`code` + `message`）。
- 禁止将底层数据库错误直接透传给外部响应。
- 新增关键链路应补充日志点（请求入口、关键分支、失败路径）。

## 8. 安全与配置

- 禁止提交真实密钥、密码、token。
- 配置通过环境变量注入；默认值仅用于本地开发。
- `.env` 不入库，仅保留 `.env.example`（如需要）。

## 9. 禁止事项（硬性）

- 禁止跨层循环依赖。
- 禁止在 `domain` 中引入框架依赖（Gin、GORM 等）。
- 禁止为“临时可跑”绕过分层约束并不修复。
- 禁止在未验证测试的情况下声明任务完成。

## 10. 文档同步

出现以下情况必须更新文档：

- 新增/调整接口：更新 `backend/README.md` 或对应 API 文档
- 新增迁移/seed：更新 `docs/testing/smoke.md` 或相关执行说明
- 架构约束变化：同步更新本文件 `backend/AGENTS.md`
