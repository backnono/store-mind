# 业务链路验证骨架（`scripts/verify/`）

该目录用于承载项目级端到端验证（verify）脚本。

当前提供了两类业务链路场景：

- `login_and_me`：登录 -> 调用 `/me`
- `command_ack_closed_loop`：下发命令 -> 轮询命令详情直到终态

## 1. 默认行为（骨架自检）

默认不请求真实服务，仅检查场景注册和参数解析，适合本地/CI 稳定接入：

```bash
python3 scripts/verify/runner.py
```

## 2. 真实链路验证（live 模式）

当本地后端、数据库、MQTT 等依赖都已准备好后，可开启 live 模式：

```bash
python3 scripts/verify/runner.py --live
```

只跑单个场景：

```bash
python3 scripts/verify/runner.py --live --scenario login_and_me
python3 scripts/verify/runner.py --live --scenario command_ack_closed_loop
```

## 3. 环境变量

- `VERIFY_BASE_URL`：后端地址，默认 `http://127.0.0.1:8080`
- `VERIFY_USERNAME`：登录用户名，默认 `admin`
- `VERIFY_PASSWORD`：登录密码，默认 `admin123`
- `VERIFY_PLANT_ID`：命令场景 plant，默认 `p1`
- `VERIFY_DEVICE_ID`：命令场景 device，默认 `dev-1`
- `VERIFY_COMMAND_TYPE`：命令类型，默认 `device_power`
- `VERIFY_TIMEOUT_SECONDS`：轮询超时秒数，默认 `30`
- `VERIFY_POLL_INTERVAL_SECONDS`：轮询间隔秒数，默认 `2`

示例：

```bash
VERIFY_BASE_URL=http://127.0.0.1:8080 \
VERIFY_USERNAME=admin \
VERIFY_PASSWORD=admin123 \
VERIFY_DEVICE_ID=dev-001 \
python3 scripts/verify/runner.py --live --scenario command_ack_closed_loop
```

## 4. 与 `scripts/validate.py` 的关系

`scripts/validate.py` 的 `verify` 步骤已经接入本骨架入口：

```bash
python3 scripts/verify/runner.py
```

也就是默认只做“骨架级校验”，不强依赖运行中的后端环境。

如需把 live 校验变成强制步骤，可把 `validate.py` 的 verify 命令改为：

```bash
python3 scripts/verify/runner.py --live
```
