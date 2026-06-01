#!/usr/bin/env python3
"""Shared Harness rules for dependency validation."""

from __future__ import annotations

from dataclasses import dataclass
from pathlib import Path
from typing import Dict, Optional

MODULE_PREFIX = "store-mind/"

# 组件层级映射：数字越小越底层，依赖方向只能“高 -> 低”或“同层”。
LAYER_BY_COMPONENT: Dict[str, int] = {
    "domain": 0,
    "application": 1,
    "infra": 2,
    "api": 3,
    "internal": 4,
    "cmd": 5,
}


@dataclass(frozen=True)
class DependencyDecision:
    """依赖判定结果：是否允许 + 解释原因。"""

    allowed: bool
    reason: str


def component_from_path(path: Path) -> Optional[str]:
    """从相对路径推断顶层组件（如 application、api）。"""

    try:
        first = path.parts[0]
    except IndexError:
        return None
    return first if first in LAYER_BY_COMPONENT else None


def component_from_import(go_import: str) -> Optional[str]:
    """从 Go import 路径提取内部组件；外部依赖返回 None。"""

    if not go_import.startswith(MODULE_PREFIX):
        return None
    rest = go_import[len(MODULE_PREFIX) :]
    component = rest.split("/", 1)[0]
    return component if component in LAYER_BY_COMPONENT else None


def check_dependency(from_component: str, to_component: str) -> DependencyDecision:
    """根据层级规则判断 from -> to 是否允许。"""

    if from_component == to_component:
        return DependencyDecision(True, "same component")

    from_layer = LAYER_BY_COMPONENT[from_component]
    to_layer = LAYER_BY_COMPONENT[to_component]
    if from_layer >= to_layer:
        return DependencyDecision(
            True, f"layer {from_layer} -> {to_layer} is downward/flat"
        )
    return DependencyDecision(
        False,
        (
            f"forbidden upward dependency: {from_component} (layer {from_layer}) "
            f"-> {to_component} (layer {to_layer})"
        ),
    )
