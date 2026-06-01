#!/usr/bin/env python3
"""Pre-check whether a structural action is architecture-safe."""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

from harness_rules import (
    LAYER_BY_COMPONENT,
    check_dependency,
    component_from_import,
    component_from_path,
)

CREATE_FILE_RE = re.compile(r"create file\s+([^\s]+)", re.IGNORECASE)
IMPORT_RE = re.compile(r"add import\s+([^\s]+)\s*->\s*([^\s]+)", re.IGNORECASE)


def explain_layers() -> str:
    """输出层级映射，便于在拒绝时给出可读反馈。"""

    parts = [f"{name}:L{layer}" for name, layer in sorted(LAYER_BY_COMPONENT.items())]
    return ", ".join(parts)


def verify_create_file(action: str) -> tuple[bool, str]:
    """校验 create file 动作是否落在已知组件目录下。"""

    match = CREATE_FILE_RE.search(action)
    if not match:
        return True, "no create-file pattern detected"

    rel = Path(match.group(1))
    component = component_from_path(rel)
    if component is None:
        return False, (
            f"unknown top-level component for {rel}. Known: "
            f"{', '.join(sorted(LAYER_BY_COMPONENT))}"
        )
    return True, f"file path belongs to component '{component}' (layer {LAYER_BY_COMPONENT[component]})"


def verify_import(action: str) -> tuple[bool, str]:
    """校验 add import 动作是否违反层级依赖方向。"""

    match = IMPORT_RE.search(action)
    if not match:
        return True, "no add-import pattern detected"

    from_file, to_import = match.groups()
    from_component = component_from_path(Path(from_file))
    if from_component is None:
        return False, f"unknown source component for file {from_file}"

    to_component = component_from_import(to_import)
    if to_component is None:
        return True, "target import is external or non-layered; no internal layer restriction"

    result = check_dependency(from_component, to_component)
    return result.allowed, result.reason


def main() -> int:
    """入口：汇总预校验结果，输出 ALLOWED/BLOCKED。"""

    parser = argparse.ArgumentParser()
    parser.add_argument("--action", required=True, help="Natural language action description")
    args = parser.parse_args()
    action = args.action.strip()

    checks = [verify_create_file(action), verify_import(action)]
    failed = [msg for ok, msg in checks if not ok]
    passed = [msg for ok, msg in checks if ok]

    if failed:
        print("BLOCKED")
        print(f"Action: {action}")
        print("Reasons:")
        for msg in failed:
            print(f"- {msg}")
        print(f"Layer map: {explain_layers()}")
        return 1

    print("ALLOWED")
    print(f"Action: {action}")
    for msg in passed:
        print(f"- {msg}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
