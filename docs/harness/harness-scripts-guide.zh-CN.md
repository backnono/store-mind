# Harness 脚本规则说明（本项目）

更新时间：2026-05-29

本文介绍当前 4 个脚本的全部规则与判定逻辑（以 `backend/` 子项目为作用域）：

- `backend/scripts/harness_rules.py`
- `backend/scripts/lint_deps.py`
- `backend/scripts/verify_action.py`
- `backend/scripts/validate.py`

## 1. `harness_rules.py`：分层规则中心

### 1.1 模块前缀规则

- `MODULE_PREFIX = "store-mind/"`
- 只有 import 路径以这个前缀开头，才会被视为“项目内部依赖”并参与层级校验。
- 外部依赖（例如 `github.com/...`）不参与层级判断。

### 1.2 层级映射规则

`LAYER_BY_COMPONENT`：

- `domain`: 0
- `application`: 1
- `infra`: 2
- `api`: 3
- `internal`: 4
- `cmd`: 5

含义：

- 数字越小越底层。
- 允许高层依赖低层。
- 允许同层依赖同层。
- 禁止低层依赖高层。

### 1.3 组件识别规则

- `component_from_path(path)`：
  - 从文件相对路径第一段识别组件。
  - 例如 `application/customerqa/service.go` -> `application`。
  - 第一段不在映射表中则返回 `None`。
- `component_from_import(go_import)`：
  - 仅当 import 以 `MODULE_PREFIX` 开头时才解析组件。
  - 例如 `store-mind/application/customerqa` -> `application`。
  - 非内部 import 返回 `None`。

### 1.4 依赖方向判定规则

`check_dependency(from_component, to_component)`：

- 同组件：允许。
- 若 `from_layer >= to_layer`：允许（向下依赖或同层）。
- 若 `from_layer < to_layer`：拒绝（向上依赖）。

返回结构：

- `allowed: bool`
- `reason: str`

## 2. `lint_deps.py`：静态依赖扫描

### 2.1 扫描范围规则

- 扫描 `backend/` 下所有 `*.go` 文件。
- 跳过隐藏目录（路径段以 `.` 开头）。
- 跳过 `web/` 目录（前端代码）。
- 如果文件不在已知组件目录下（如不在 `domain/application/infra/...`），则该文件不参与内部层级判断。

### 2.2 import 提取规则

- 支持两类写法：
  - `import ( ... )`
  - `import "xxx"`
- 使用正则提取 import 路径文本。

### 2.3 违规判定规则

对每个内部 import：

1. 识别源文件组件（`from_component`）
2. 识别目标 import 组件（`to_component`）
3. 调用 `check_dependency`
4. 若 `allowed == False`，记录违规项

### 2.4 输出与退出码规则

- 有违规：
  - 打印 `Dependency layer violations detected:`
  - 每条违规显示文件、import、原因
  - 退出码 `1`
- 无违规：
  - 打印 `Dependency lint passed: no layer violations found.`
  - 退出码 `0`

## 3. `verify_action.py`：动作预校验

该脚本用于“执行结构性修改前”的快速风险拦截。

### 3.1 输入规则

- 必填参数：`--action "<自然语言动作描述>"`
- 动作文本中目前识别两类模式：
  - `create file <path>`
  - `add import <from_file> -> <to_import>`

### 3.2 create file 规则

`verify_create_file(action)`：

- 若未匹配 `create file` 模式：视为通过（不阻塞）。
- 若匹配：
  - 解析 `<path>` 的顶层目录组件。
  - 顶层组件未知：拒绝。
  - 顶层组件已知：允许。

### 3.3 add import 规则

`verify_import(action)`：

- 若未匹配 `add import` 模式：视为通过（不阻塞）。
- 若匹配：
  - 先校验 `<from_file>` 的源组件是否可识别；不可识别则拒绝。
  - 再解析 `<to_import>`：
    - 外部依赖或非分层组件：允许（不做内部层级限制）。
    - 内部依赖：调用 `check_dependency` 判定方向是否合法。

### 3.4 汇总规则与输出

`main()` 会同时执行：

- `verify_create_file`
- `verify_import`

判定逻辑：

- 任一检查失败 -> 输出 `BLOCKED`，退出码 `1`
- 全部通过 -> 输出 `ALLOWED`，退出码 `0`

拒绝时会附带层级地图（如 `api:L3, domain:L0, ...`）用于定位。

## 4. `validate.py`：统一验证入口

### 4.1 Go 二进制选择规则

- 解析优先级：`GO_BIN` > 登录 shell 的 `command -v go` > 当前进程 PATH 的 `go`。
- 可通过环境变量覆盖：
  - `GO_BIN=/path/to/go python3 backend/scripts/validate.py`
- 启动时会先执行 `<go_bin> version`：
  - 成功：打印实际版本。
  - 失败：直接退出码 `1`。

### 4.2 固定流水线规则

执行顺序固定为：

1. `build`：`<go_bin> build ./...`
2. `lint-arch`：`python3 scripts/lint_deps.py`
3. `test`：`<go_bin> test ./...`
4. `verify`：`python3 scripts/verify/runner.py`

### 4.3 失败短路规则

- 任意步骤失败：
  - 立即打印 `[FAIL] <step>`
  - 直接退出码 `1`
  - 后续步骤不再执行
- 全部成功：
  - 打印 `Validation pipeline passed.`
  - 退出码 `0`

### 4.4 verify 步骤默认行为

- `validate.py` 中 `verify` 步骤默认执行：
  - `python3 scripts/verify/runner.py`
- 默认不带 `--live`，因此只做 verify 骨架自检，不请求真实后端接口。
- 若需要在统一流水线中强制真实链路验证，可改为：
  - `python3 scripts/verify/runner.py --live`

## 5. 推荐使用方式

### 5.1 日常统一验证

```bash
cd backend
GO_BIN=/opt/homebrew/opt/go@1.24/bin/go python3 scripts/validate.py
```

### 5.1.1 业务链路 live 验证（可选）

```bash
cd backend
python3 scripts/verify/runner.py --live
```

### 5.2 结构改动前预检

```bash
cd backend
python3 scripts/verify_action.py --action "create file application/new_feature/service.go"
python3 scripts/verify_action.py --action "add import api/http/handler.go -> store-mind/application/customerqa"
```

### 5.3 单独跑依赖层级检查

```bash
cd backend
python3 scripts/lint_deps.py
```
