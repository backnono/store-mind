#!/usr/bin/env python3
"""业务链路验证骨架入口。

默认模式（非 --live）只做场景注册与参数检查，适合接入 CI/本地流水线。
开启 --live 后会实际请求后端接口，执行端到端业务链路验证。
"""

from __future__ import annotations

import argparse
import json
import os
import time
import urllib.error
import urllib.request
from dataclasses import dataclass
from typing import Any, Callable


@dataclass(frozen=True)
class VerifyConfig:
    """业务验证运行参数。"""

    base_url: str
    username: str
    password: str
    plant_id: str
    device_id: str
    command_type: str
    timeout_seconds: int
    poll_interval_seconds: float


def _join_url(base_url: str, path: str) -> str:
    return base_url.rstrip("/") + path


def _http_json(
    method: str,
    url: str,
    *,
    token: str | None = None,
    payload: dict[str, Any] | None = None,
    timeout: float = 10.0,
) -> tuple[int, dict[str, Any]]:
    """发送 HTTP 请求并尽量解析 JSON 响应。"""

    headers = {"Accept": "application/json"}
    data: bytes | None = None
    if token:
        headers["Authorization"] = f"Bearer {token}"
    if payload is not None:
        headers["Content-Type"] = "application/json"
        data = json.dumps(payload).encode("utf-8")

    req = urllib.request.Request(url=url, method=method, headers=headers, data=data)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read().decode("utf-8")
            status = resp.getcode()
    except urllib.error.HTTPError as err:
        raw = err.read().decode("utf-8", errors="replace")
        status = err.code
    except urllib.error.URLError as err:
        raise RuntimeError(f"request failed: {method} {url}: {err}") from err

    if not raw.strip():
        return status, {}
    try:
        parsed = json.loads(raw)
        if isinstance(parsed, dict):
            return status, parsed
        return status, {"_raw": parsed}
    except json.JSONDecodeError:
        return status, {"_text": raw}


def _pick(obj: dict[str, Any], *keys: str) -> Any:
    for key in keys:
        if key in obj and obj[key] not in (None, ""):
            return obj[key]
    return None


def _login_and_get_token(cfg: VerifyConfig) -> str:
    """登录并提取 access token（兼容几种常见响应字段）。"""

    status, data = _http_json(
        "POST",
        _join_url(cfg.base_url, "/auth/login"),
        payload={"username": cfg.username, "password": cfg.password},
    )
    if status >= 400:
        raise RuntimeError(f"login failed: status={status}, body={data}")

    token = _pick(data, "access_token", "accessToken")
    if token is None and isinstance(data.get("data"), dict):
        token = _pick(data["data"], "access_token", "accessToken", "token")
    if not isinstance(token, str) or not token.strip():
        raise RuntimeError(f"cannot find access token in login response: {data}")
    return token


def scenario_login_and_me(cfg: VerifyConfig) -> None:
    """链路1：登录 -> 查询当前用户。"""

    token = _login_and_get_token(cfg)
    status, data = _http_json("GET", _join_url(cfg.base_url, "/me"), token=token)
    if status >= 400:
        raise RuntimeError(f"/me failed: status={status}, body={data}")
    print("[PASS] scenario_login_and_me")


def _extract_command_id(body: dict[str, Any]) -> str | None:
    cid = _pick(body, "command_id", "commandId", "id")
    if isinstance(cid, str) and cid:
        return cid
    data = body.get("data")
    if isinstance(data, dict):
        cid = _pick(data, "command_id", "commandId", "id")
        if isinstance(cid, str) and cid:
            return cid
    return None


def _extract_status(body: dict[str, Any]) -> str | None:
    status = _pick(body, "status", "command_status", "commandStatus")
    if isinstance(status, str) and status:
        return status.lower()
    data = body.get("data")
    if isinstance(data, dict):
        status = _pick(data, "status", "command_status", "commandStatus")
        if isinstance(status, str) and status:
            return status.lower()
    return None


def scenario_command_ack_closed_loop(cfg: VerifyConfig) -> None:
    """链路2：下发命令 -> 轮询命令详情直到终态。"""

    token = _login_and_get_token(cfg)
    send_payload = {
        "plantId": cfg.plant_id,
        "commandType": cfg.command_type,
        "params": {"on": True},
        "timeoutMs": max(cfg.timeout_seconds * 1000, 1000),
    }
    send_status, send_body = _http_json(
        "POST",
        _join_url(cfg.base_url, f"/devices/{cfg.device_id}/commands"),
        token=token,
        payload=send_payload,
    )
    if send_status >= 400:
        raise RuntimeError(f"send command failed: status={send_status}, body={send_body}")

    command_id = _extract_command_id(send_body)
    if not command_id:
        raise RuntimeError(f"cannot extract command id from response: {send_body}")

    deadline = time.time() + cfg.timeout_seconds
    terminal = {"ok", "success", "acked", "failed", "timeout", "cancelled"}
    last_status: str | None = None
    last_body: dict[str, Any] = {}
    while time.time() < deadline:
        poll_status, poll_body = _http_json(
            "GET",
            _join_url(cfg.base_url, f"/devices/{cfg.device_id}/commands/{command_id}"),
            token=token,
        )
        if poll_status < 400:
            state = _extract_status(poll_body)
            last_status = state
            last_body = poll_body
            if state in terminal:
                print(f"[PASS] scenario_command_ack_closed_loop (final_status={state})")
                return
        time.sleep(cfg.poll_interval_seconds)

    raise RuntimeError(
        f"command did not reach terminal state in time: command_id={command_id}, "
        f"last_status={last_status}, last_body={last_body}"
    )


SCENARIOS: dict[str, Callable[[VerifyConfig], None]] = {
    "login_and_me": scenario_login_and_me,
    "command_ack_closed_loop": scenario_command_ack_closed_loop,
}


def _load_config() -> VerifyConfig:
    return VerifyConfig(
        base_url=os.environ.get("VERIFY_BASE_URL", "http://127.0.0.1:8080"),
        username=os.environ.get("VERIFY_USERNAME", "admin"),
        password=os.environ.get("VERIFY_PASSWORD", "admin123"),
        plant_id=os.environ.get("VERIFY_PLANT_ID", "p1"),
        device_id=os.environ.get("VERIFY_DEVICE_ID", "dev-1"),
        command_type=os.environ.get("VERIFY_COMMAND_TYPE", "device_power"),
        timeout_seconds=int(os.environ.get("VERIFY_TIMEOUT_SECONDS", "30")),
        poll_interval_seconds=float(os.environ.get("VERIFY_POLL_INTERVAL_SECONDS", "2")),
    )


def _parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--scenario",
        default="all",
        help="指定场景名，默认 all（可选: login_and_me, command_ack_closed_loop）",
    )
    parser.add_argument(
        "--live",
        action="store_true",
        help="开启真实接口调用；默认只执行骨架自检。",
    )
    return parser.parse_args()


def main() -> int:
    args = _parse_args()
    cfg = _load_config()

    names = list(SCENARIOS.keys()) if args.scenario == "all" else [args.scenario]
    for name in names:
        if name not in SCENARIOS:
            print(f"[FAIL] unknown scenario: {name}. available={', '.join(SCENARIOS)}")
            return 1

    if not args.live:
        print("[SKIP] verify live checks disabled (run with --live to execute API flows)")
        print(f"[INFO] registered scenarios: {', '.join(names)}")
        print("[PASS] verify skeleton check")
        return 0

    for name in names:
        print(f"[RUN] {name}")
        SCENARIOS[name](cfg)
    print("[PASS] all selected live scenarios")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
