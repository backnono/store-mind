# Store Mind Backend Scaffold Template

> 目标：通过本文档在新项目中快速复用当前后端脚手架（Clean Architecture + Gin + GORM + MySQL + Zap + Harness + Smoke）。

## 1. 适用范围

本模板适用于：

- Go 单体后端（后续可扩展为模块化单体）
- 分层清晰、强调依赖方向约束
- 需要基础可观测性（请求日志 + request_id）
- 需要可复制的本地验证链路（test + migrate + smoke）

## 2. 技术栈基线

- Go `1.24`
- HTTP: `gin`
- ORM: `gorm` + `mysql driver`
- Logging: `zap`
- DB: MySQL 8.x（Docker Compose）

## 3. 目录结构（标准）

```text
backend/
  api/http/                     # HTTP 适配层（handler/router/middleware/response）
  application/<context>/        # 用例编排层（service + logger interface）
  domain/<context>/             # 领域层（entity/repository/errors）
  infra/
    config/                     # 环境变量配置
    logger/                     # zap 封装 + app logger adapter
    persistence/mysql/          # repository 实现 + gorm model + db open
  internal/bootstrap/           # 依赖装配
  cmd/server/                   # 启动入口
  db/
    migrations/                 # 结构迁移
    seeds/                      # 示例数据
  scripts/
    migrate.sh                  # 顺序执行 migration/seed
    smoke.sh                    # 一键冒烟
    lint_deps.py                # 分层依赖检查
    verify_action.py            # 改动动作预检
    validate.py                 # 统一验证流水线
  docs/scaffold/                # 本文档
  AGENTS.md                     # 后端约束规则
  Makefile
  docker-compose.yml
  go.mod
```

## 4. 分层依赖规则（必须）

依赖方向：

- `api -> application -> domain`
- `infra` 实现 `domain/application` 定义的契约
- `internal/bootstrap` 负责组装
- `cmd` 只做入口与启动

禁止：

- `domain` 依赖 `application/api/infra/cmd`
- `application` 依赖 `api/cmd`
- handler 直接写 SQL 或跳过 application

## 5. 快速搭建步骤（可复制）

### Step 1: 初始化模块

```bash
mkdir -p backend && cd backend
go mod init <your-module-name>
```

推荐依赖：

```bash
go get github.com/gin-gonic/gin@v1.10.0
go get gorm.io/gorm@v1.25.12
go get gorm.io/driver/mysql@v1.5.7
go get go.uber.org/zap@v1.27.0
go get github.com/google/uuid@v1.6.0
```

### Step 2: 建目录骨架

```bash
mkdir -p \
  api/http \
  application/customerqa \
  domain/customerqa \
  infra/config infra/logger infra/persistence/mysql \
  internal/bootstrap \
  cmd/server \
  db/migrations db/seeds \
  scripts docs/scaffold
```

### Step 3: 先写领域契约（domain）

- `domain/<context>/entity.go`
- `domain/<context>/repository.go`
- `domain/<context>/errors.go`

要求：只放业务模型与接口，不引入框架依赖。

### Step 4: 写用例服务（application）

- `application/<context>/service.go`
- `application/<context>/logger.go`（接口，不绑定 zap）

要求：

- 编排业务流程
- 调用 domain repository 契约
- 不依赖 gin/gorm

### Step 5: 写 infra 实现

- `infra/persistence/mysql/db.go`：初始化 gorm
- `infra/persistence/mysql/models.go`：表模型
- `infra/persistence/mysql/repository_<context>.go`：repository 实现
- `infra/logger/logger.go`：环境日志策略
- `infra/logger/app_logger.go`：application logger 适配器

### Step 6: 写 API 层

- `api/http/router.go`
- `api/http/handler_<context>.go`
- `api/http/middleware.go`
- `api/http/response.go`

要求：

- handler 只做参数解析、调用 service、响应映射
- 中间件注入 `request_id`，记录请求日志（含 query/body）
- 错误统一返回 `{code,message}`

### Step 7: 依赖装配与入口

- `internal/bootstrap/app.go`：组装 config/logger/db/repo/service/handler/router
- `cmd/server/main.go`：启动 server，处理 logger sync

### Step 8: DB 迁移与种子

- `db/migrations/001_*.sql`
- `db/seeds/001_*.sql`
- `scripts/migrate.sh` 支持顺序执行与 `WITH_SEED=1`

### Step 9: 本地基础设施

`docker-compose.yml`（MySQL 可配置端口）：

```yaml
ports:
  - "${MYSQL_PORT:-3307}:3306"
```

### Step 10: Makefile 标准命令

- `make tidy`
- `make test`
- `make run`
- `make migrate-up`
- `make seed`
- `make smoke`

## 6. 日志规范（当前模板）

### 开发环境

- `APP_ENV=dev`
- 输出到控制台（console encoder）

### 生产环境

- `APP_ENV=production`
- 输出到 `LOG_DIR`（默认 `logs`）
- 按天切文件：`app-YYYY-MM-DD.log`

### 建议环境变量

- `APP_ENV`
- `LOG_LEVEL=debug|info|warn|error`
- `LOG_DIR`

## 7. HTTP 日志与链路追踪规范

中间件最少记录：

- `request_id`
- method/path/status/latency
- client_ip/user_agent
- query
- body（JSON，敏感字段脱敏，长度截断）

传递规则：

- 请求头优先 `X-Request-Id`
- 无则生成 UUID
- 回写响应头 `X-Request-Id`
- 写入 `context.Context`
- 业务日志（application/handler）带 `request_id`
- API 响应 `meta.request_id`

## 8. Harness 与约束接入

建议接入：

- `scripts/harness_rules.py`
- `scripts/lint_deps.py`
- `scripts/verify_action.py`
- `scripts/validate.py`
- `AGENTS.md`

关键配置：

- `MODULE_PREFIX` 必须与 `go.mod` 模块名一致
- `LAYER_BY_COMPONENT` 必须与真实目录一致

## 9. 测试基线（必须）

至少包含：

- application 单测（fake repo）
- api 单测（httptest）
- logger/middleware 单测（关键行为，如 request_id）

标准验证：

```bash
cd backend
/opt/homebrew/opt/go@1.24/bin/go test ./...
```

## 10. 冒烟测试基线（推荐）

`scripts/smoke.sh` 覆盖：

1. 启动 MySQL
2. 执行 migration + seed
3. 启动应用
4. 校验 `/healthz`
5. 校验一条核心业务接口
6. 输出 `SMOKE PASS`

端口冲突示例：

```bash
cd backend
DB_PORT=3308 bash scripts/smoke.sh
```

## 11. 新项目复用清单（Checklist）

创建新项目时按顺序执行：

1. 复制目录骨架
2. 初始化 `go.mod` 与依赖
3. 替换模块名并同步 `harness_rules.py` 的 `MODULE_PREFIX`
4. 按业务上下文实现 domain/application/api/infra
5. 补 migration/seed
6. 补 Makefile + scripts
7. 执行 `go test ./...`
8. 执行 `smoke.sh`
9. 落地 `backend/AGENTS.md`

## 12. 迁移到多上下文的建议

当业务扩展时：

- 增加 `domain/<new-context>` 与 `application/<new-context>`
- API 以资源分组路由，不破坏现有上下文
- 共享能力放 `infra`（日志、配置、db、中间件）
- 保持“上下文内高内聚，上下文间低耦合”

---

如果你把本模板复制到新仓库，请优先修改以下 4 项：

1. `go.mod` 模块名
2. `scripts/harness_rules.py` 的 `MODULE_PREFIX`
3. `docker-compose.yml` 的端口与数据库名
4. `infra/config/config.go` 默认 DSN

## 13. 最小复制包清单（用于新项目落地）

建议按下列顺序复制（从“基础能力”到“业务样例”）：

### A. 基础运行骨架（必选）

- `go.mod`
- `cmd/server/main.go`
- `internal/bootstrap/app.go`
- `infra/config/config.go`
- `infra/logger/logger.go`
- `infra/logger/app_logger.go`
- `api/http/router.go`
- `api/http/middleware.go`
- `api/http/request_id.go`
- `api/http/response.go`

### B. 数据与环境（必选）

- `docker-compose.yml`
- `db/migrations/001_init_customer_qa.sql`（可改名为你的业务初始化脚本）
- `scripts/migrate.sh`
- `scripts/smoke.sh`
- `Makefile`

### C. 架构约束与验证（强烈建议）

- `AGENTS.md`
- `scripts/harness_rules.py`
- `scripts/lint_deps.py`
- `scripts/verify_action.py`
- `scripts/validate.py`
- `scripts/verify/runner.py`
- `scripts/verify/README.zh-CN.md`

### D. 业务样例（可选但推荐）

- `domain/customerqa/*`
- `application/customerqa/*`
- `infra/persistence/mysql/*`
- `api/http/handler_customer_qa.go`
- `api/http/handler_customer_qa_test.go`

### E. 测试样例（推荐）

- `application/customerqa/service_test.go`
- `api/http/middleware_test.go`

### 复制后第一时间需要替换的配置

1. `go.mod` 模块名
2. `scripts/harness_rules.py` 的 `MODULE_PREFIX`
3. `infra/config/config.go` 默认 DSN
4. `docker-compose.yml` 端口与数据库名
5. `db/migrations/*.sql` 与 `db/seeds/*.sql` 的业务表结构

### 复制后第一轮验证命令

```bash
cd backend
/opt/homebrew/opt/go@1.24/bin/go test ./...
DB_PORT=3308 bash scripts/smoke.sh
```
