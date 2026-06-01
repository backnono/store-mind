#!/usr/bin/env python3
"""Unified validation pipeline for Harness-style local checks."""

from __future__ import annotations

import subprocess
import sys
import os
import shlex
from pathlib import Path


def run_step(name: str, cmd: list[str]) -> bool:
    """执行单个验证步骤并打印统一 PASS/FAIL 日志。"""

    print(f"\n==> {name}")
    print("$ " + " ".join(cmd))
    result = subprocess.run(cmd, cwd=Path(__file__).resolve().parent.parent)
    if result.returncode != 0:
        print(f"[FAIL] {name} (exit {result.returncode})")
        return False
    print(f"[PASS] {name}")
    return True


def resolve_go_bin() -> str:
    """解析 Go 可执行路径，优先级：GO_BIN > 登录 shell 的 go > PATH go。"""

    explicit = os.environ.get("GO_BIN", "").strip()
    if explicit:
        return explicit

    shell = os.environ.get("SHELL", "").strip()
    if shell:
        # 使用登录 shell，尽量与开发者终端一致（如 asdf / brew PATH 注入）。
        probe = subprocess.run(
            [shell, "-lic", "command -v go"],
            capture_output=True,
            text=True,
        )
        if probe.returncode == 0:
            resolved = probe.stdout.strip()
            if resolved:
                return resolved

    return "go"


def main() -> int:
    """统一验证入口：build -> lint-arch -> test -> verify。"""

    go_bin = resolve_go_bin()
    version = subprocess.run(
        [go_bin, "version"],
        cwd=Path(__file__).resolve().parent.parent,
        capture_output=True,
        text=True,
    )
    if version.returncode != 0:
        print(f"[FAIL] unable to execute {go_bin} version")
        print(version.stderr.strip() or version.stdout.strip())
        return 1
    print(f"Using Go: {version.stdout.strip()}")
    print(f"Go binary: {shlex.quote(go_bin)}")

    steps = [
        ("build", [go_bin, "build", "./..."]),
        ("lint-arch", ["python3", "scripts/lint_deps.py"]),
        ("test", [go_bin, "test", "./..."]),
        ("verify", ["python3", "scripts/verify/runner.py"]),
    ]

    for name, cmd in steps:
        if not run_step(name, cmd):
            return 1
    print("\nValidation pipeline passed.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
