#!/usr/bin/env python3
"""Lint internal Go dependency direction by architecture layers."""

from __future__ import annotations

import re
import sys
from pathlib import Path
from typing import Iterable, List, Tuple

from harness_rules import (
    check_dependency,
    component_from_import,
    component_from_path,
)

IMPORT_BLOCK_RE = re.compile(r"import\s*\((.*?)\)", re.DOTALL | re.MULTILINE)
IMPORT_LINE_RE = re.compile(r'^\s*"([^"]+)"\s*$', re.MULTILINE)
SINGLE_IMPORT_RE = re.compile(r'import\s+"([^"]+)"')


def extract_imports(content: str) -> List[str]:
    """提取 Go 文件中的 import（同时支持 block 和单行写法）。"""

    imports: List[str] = []
    for block in IMPORT_BLOCK_RE.findall(content):
        imports.extend(IMPORT_LINE_RE.findall(block))
    imports.extend(SINGLE_IMPORT_RE.findall(content))
    return imports


def go_files(root: Path) -> Iterable[Path]:
    """遍历仓库内 Go 文件，跳过隐藏目录与 web 前端目录。"""

    for path in root.rglob("*.go"):
        if any(part.startswith(".") for part in path.parts):
            continue
        if "web" in path.parts:
            continue
        yield path


def main() -> int:
    """扫描内部 import 并按层级规则输出违规项。"""

    repo_root = Path(__file__).resolve().parent.parent
    violations: List[Tuple[Path, str, str]] = []

    for path in go_files(repo_root):
        rel_path = path.relative_to(repo_root)
        from_component = component_from_path(rel_path)
        if not from_component:
            continue

        content = path.read_text(encoding="utf-8")
        for imp in extract_imports(content):
            to_component = component_from_import(imp)
            if not to_component:
                continue
            result = check_dependency(from_component, to_component)
            if not result.allowed:
                violations.append((rel_path, imp, result.reason))

    if violations:
        print("Dependency layer violations detected:")
        for rel_path, imp, reason in violations:
            print(f"- {rel_path}: imports {imp}; {reason}")
        return 1

    print("Dependency lint passed: no layer violations found.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
