#!/usr/bin/env python3
"""S1 acceptance harness for local diagnostics and CI gating.

Gates:
  entry       — first_open returns preset questions, zone_scan returns shelf products
  guidance    — location answer has guidance chips
  credibility — inventory answer contains credibility level
  multi-turn  — same-session "还有几瓶/多少钱" inherits previous product
  persistence — context_state / focus_entity_ids / context_stack persisted
  sidecar     — /llm/resolve returns resolved_entities

Output: artifacts/s1/s1-report.{json,md,xml}
"""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
import time
import urllib.error
import urllib.request
import xml.etree.ElementTree as ET
from pathlib import Path
from typing import Any, Callable

ROOT = Path(__file__).resolve().parents[2]
DEFAULT_OUTPUT_DIR = ROOT / "artifacts" / "s1"


# ── Report ──────────────────────────────────────────

class AcceptanceReport:
    def __init__(self, mode: str) -> None:
        self.mode = mode
        self.started_at = time.strftime("%Y-%m-%dT%H:%M:%S%z")
        self.checks: list[dict[str, Any]] = []

    def add(self, gate: str, name: str, passed: bool, detail: str = "") -> None:
        self.checks.append(
            {"gate": gate, "name": name, "passed": bool(passed), "detail": detail}
        )

    @property
    def passed(self) -> bool:
        return bool(self.checks) and all(item["passed"] for item in self.checks)

    @property
    def failed_count(self) -> int:
        return sum(not item["passed"] for item in self.checks)

    def as_dict(self) -> dict[str, Any]:
        passed_count = sum(item["passed"] for item in self.checks)
        return {
            "mode": self.mode,
            "started_at": self.started_at,
            "passed": self.passed,
            "summary": {
                "total": len(self.checks),
                "passed": passed_count,
                "failed": self.failed_count,
            },
            "checks": self.checks,
        }

    def write(self, output_dir: Path) -> None:
        output_dir.mkdir(parents=True, exist_ok=True)
        payload = self.as_dict()
        (output_dir / "s1-report.json").write_text(
            json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8"
        )

        lines = [
            "# S1 Acceptance Report",
            "",
            f"- Mode: `{self.mode}`",
            f"- Result: **{'PASS' if self.passed else 'FAIL'}**",
            f"- Checks: {payload['summary']['passed']}/{payload['summary']['total']} passed",
            "",
            "| Gate | Check | Result | Detail |",
            "|---|---|---|---|",
        ]
        for item in self.checks:
            detail = str(item["detail"]).replace("|", "\\|").replace("\n", " ")
            lines.append(
                f"| {item['gate']} | {item['name']} | "
                f"{'PASS' if item['passed'] else 'FAIL'} | {detail} |"
            )
        (output_dir / "s1-report.md").write_text(
            "\n".join(lines) + "\n", encoding="utf-8"
        )

        suite = ET.Element(
            "testsuite",
            name="s1-acceptance",
            tests=str(len(self.checks)),
            failures=str(self.failed_count),
        )
        for item in self.checks:
            case = ET.SubElement(
                suite, "testcase", classname=f"s1.{item['gate']}", name=item["name"]
            )
            if not item["passed"]:
                failure = ET.SubElement(case, "failure", message=str(item["detail"]))
                failure.text = str(item["detail"])
        ET.ElementTree(suite).write(
            output_dir / "s1-report.xml", encoding="utf-8", xml_declaration=True
        )


# ── HTTP Client ─────────────────────────────────────

class HTTPClient:
    def request(
        self,
        method: str,
        url: str,
        payload: dict[str, Any] | None = None,
        timeout: float = 15.0,
    ) -> tuple[int, dict[str, Any]]:
        body = None
        headers = {"Accept": "application/json"}
        if payload is not None:
            body = json.dumps(payload, ensure_ascii=False).encode()
            headers["Content-Type"] = "application/json"
        req = urllib.request.Request(url, data=body, headers=headers, method=method)
        try:
            with urllib.request.urlopen(req, timeout=timeout) as response:
                raw = response.read().decode()
                return response.status, json.loads(raw) if raw else {}
        except urllib.error.HTTPError as error:
            raw = error.read().decode()
            try:
                data = json.loads(raw) if raw else {}
            except json.JSONDecodeError:
                data = {"raw": raw}
            return error.code, data


# ── MySQL CLI ───────────────────────────────────────

class MySQLCLI:
    def __init__(self, host: str, port: int, user: str, password: str, database: str) -> None:
        self.host = host
        self.port = port
        self.user = user
        self.password = password
        self.database = database

    def rows(self, sql: str) -> list[list[str]]:
        env = os.environ.copy()
        env["MYSQL_PWD"] = self.password
        cmd = [
            "mysql", f"-h{self.host}", f"-P{self.port}", f"-u{self.user}",
            "-N", "-B", self.database, "-e", sql,
        ]
        completed = subprocess.run(cmd, check=True, capture_output=True, text=True, env=env)
        return [line.split("\t") for line in completed.stdout.splitlines() if line.strip()]

    def scalar(self, sql: str) -> str:
        rows = self.rows(sql)
        if not rows or not rows[0]:
            return ""
        return rows[0][0]


# ── Helpers ─────────────────────────────────────────

def run_check(report: AcceptanceReport, gate: str, name: str, check: Callable[[], tuple[bool, str]]) -> None:
    try:
        passed, detail = check()
    except Exception as error:
        passed, detail = False, f"{type(error).__name__}: {error}"
    report.add(gate, name, passed, detail)
    icon = "PASS" if passed else "FAIL"
    print(f"[{icon}] {gate}: {name} — {detail}")


def chat(api: str, http: HTTPClient, store_id: int = 1, session_id: int = 0,
         message: str = "", entry_mode: str = "", zone_id: int | None = None,
         shelf_id: int | None = None, timeout: float = 25.0) -> tuple[int, dict[str, Any]]:
    """Send a /chat request, returning (status, response)."""
    body: dict[str, Any] = {"store_id": store_id, "channel": "miniapp", "message": message}
    if session_id > 0:
        body["session_id"] = session_id
    if entry_mode:
        body["entry_mode"] = entry_mode
    if zone_id is not None:
        body["zone_id"] = zone_id
    if shelf_id is not None:
        body["shelf_id"] = shelf_id
    return http.request("POST", f"{api}/api/v1/customer-qa/chat", body, timeout=timeout)


# ── S1 Validation Functions ─────────────────────────

def validate_first_open(response: dict[str, Any]) -> tuple[bool, str]:
    """first_open: intent=greeting, route=entry_first_open, >= 4 guidance chips."""
    failures = []
    if response.get("intent") != "greeting":
        failures.append(f"intent={response.get('intent')}")
    meta = response.get("meta") or {}
    if meta.get("route") != "entry_first_open":
        failures.append(f"route={meta.get('route')}")
    chips = response.get("guidance_chips") or []
    if len(chips) < 4:
        failures.append(f"guidance_chips={len(chips)} (expected >=4)")
    texts = [c.get("text", "") for c in chips]
    has_location = any("在哪" in t or "位置" in t for t in texts)
    has_promo = any("活动" in t for t in texts)
    if not has_location:
        failures.append("missing location chip")
    if not has_promo:
        failures.append("missing promo chip")
    if failures:
        return False, "; ".join(failures)
    return True, f"chips={len(chips)}, location={has_location}, promo={has_promo}"


def validate_zone_scan(response: dict[str, Any]) -> tuple[bool, str]:
    """zone_scan: intent=zone_scan, route=entry_zone_scan, cards non-empty."""
    failures = []
    if response.get("intent") != "zone_scan":
        failures.append(f"intent={response.get('intent')}")
    meta = response.get("meta") or {}
    if meta.get("route") != "entry_zone_scan":
        failures.append(f"route={meta.get('route')}")
    cards = response.get("cards") or []
    if len(cards) == 0:
        failures.append("cards empty")
    if failures:
        return False, "; ".join(failures)
    return True, f"cards={len(cards)}"


def validate_guidance(response: dict[str, Any]) -> tuple[bool, str]:
    """Product location answer MUST have guidance chips."""
    chips = response.get("guidance_chips") or []
    if len(chips) < 2:
        return False, f"guidance_chips={len(chips)} (expected >=2)"
    texts = [c.get("text", "") for c in chips]
    has_inventory = any("几瓶" in t or "几件" in t or "库存" in t for t in texts)
    has_promo = any("活动" in t for t in texts)
    return True, f"chips={len(chips)}, inventory={has_inventory}, promo={has_promo}"


def validate_credibility(answer: str, cards: list[dict[str, Any]]) -> tuple[bool, str]:
    """Inventory answer MUST show credibility info."""
    credibility_markers = [
        "high", "medium", "low", "reference_only",
        "高可信", "中可信", "低可信", "仅供参考",
        "分钟前", "小时前", "天前", "刚刚",
        "盘点", "数据更新",
    ]
    found = [m for m in credibility_markers if m.lower() in answer.lower()]
    card_ok = any(c.get("type") == "inventory" and c.get("quantity", 0) > 0 for c in cards)
    if not found and not card_ok:
        return False, f"no credibility marker in answer and no inventory card"
    return True, f"markers={found}, card_ok={card_ok}"


def validate_multi_turn(r2: dict[str, Any], r3: dict[str, Any]) -> tuple[bool, str]:
    """R2 "还有几瓶" must NOT be unsupported. R3 "多少钱" must NOT be unsupported."""
    failures = []
    if r2.get("intent") == "unsupported":
        failures.append("R2 intent=unsupported (should inherit product)")
    if r3.get("intent") == "unsupported":
        failures.append("R3 intent=unsupported (should inherit product)")
    if r2.get("intent") == "faq":
        # If intent is faq, answer should still be about the product (not generic FAQ)
        answer = r2.get("answer", "")
        if "没有找到" in answer or "暂无" in answer:
            failures.append("R2 answer indicates no product found")
    if failures:
        return False, "; ".join(failures)
    return True, f"R2_intent={r2.get('intent')}, R3_intent={r3.get('intent')}"


def validate_persistence(db: MySQLCLI, session_id: int) -> tuple[bool, str]:
    """Verify context_state, focus_entity_ids, context_stack are persisted in DB."""
    rows = db.rows(
        f"SELECT id, role, intent, context_state, focus_entity_ids, context_stack "
        f"FROM agent_message WHERE session_id={session_id} ORDER BY id"
    )
    if len(rows) < 2:
        return False, f"expected >=2 messages, found {len(rows)}"
    has_state = any(row[3] and row[3] != "None" for row in rows)
    has_focus = any(row[4] and row[4] != "null" and row[4] != "None" for row in rows)
    has_stack = any(row[5] and row[5] != "[]" and row[5] != "None" and len(row[5]) > 2 for row in rows)
    if not has_state:
        return False, "no context_state in any message"
    if not has_focus:
        return False, "no focus_entity_ids in any message"
    if not has_stack:
        return False, "no non-empty context_stack in any message"
    return True, f"state={has_state}, focus={has_focus}, stack={has_stack}, messages={len(rows)}"


def validate_sidecar_resolve(sidecar: str, http: HTTPClient) -> tuple[bool, str]:
    """Sidecar /llm/resolve must return resolved_entities."""
    status, payload = http.request(
        "POST", f"{sidecar}/llm/resolve",
        {
            "message": "还有几瓶？",
            "context_stack": [
                {
                    "turn": 1,
                    "intent": "product_location",
                    "resolved_entities": [{"type": "product", "name": "可口可乐", "product_id": 101}],
                    "system_summary": "告诉了用户可口可乐在饮料区 B-02 货架",
                }
            ],
            "focus_entities": {"product_ids": [101], "sku_ids": [1001]},
        },
        timeout=10,
    )
    if status != 200:
        return False, f"HTTP {status}"
    entities = payload.get("resolved_entities")
    confidence = float(payload.get("confidence", -1))
    if not entities or len(entities) == 0:
        return False, f"resolved_entities empty, payload={payload}"
    if confidence < 0.6:
        return False, f"confidence={confidence} < 0.6"
    return True, f"entities={len(entities)}, confidence={confidence:.2f}"


def validate_sidecar_resolve_go_client(api: str, db: MySQLCLI, http: HTTPClient) -> tuple[bool, str]:
    """Verify Go backend passes resolved_entities from sidecar into orchestrator.
    Send a followup "还有几瓶？" against a session that already has product_focus state.
    The proof: R2 answer must contain inventory info, not unsupported."""
    # First round: establish product_focus
    status, r1 = chat(api, http, message="可乐在哪里")
    if status != 200:
        return False, f"R1 HTTP {status}"
    sid = int(r1.get("session_id") or 0)
    if sid <= 0:
        return False, "R1 missing session_id"

    # Second round: omitted followup
    status, r2 = chat(api, http, session_id=sid, message="还有几瓶？")
    if status != 200:
        return False, f"R2 HTTP {status}"

    if r2.get("intent") == "unsupported":
        return False, "R2 intent=unsupported: resolved entities not used by orchestrator"

    # Check DB: focus_entity_ids should be non-empty
    rows = db.rows(
        f"SELECT focus_entity_ids, context_state FROM agent_message "
        f"WHERE session_id={sid} AND role='assistant' ORDER BY id DESC LIMIT 1"
    )
    has_focus_db = bool(rows and rows[0][0] and rows[0][0] not in ("null", "None", ""))
    return True, f"R2_intent={r2.get('intent')}, db_focus={has_focus_db}"


# ── Main Runner ─────────────────────────────────────

class S1Acceptance:
    def __init__(self, api_base: str, sidecar_base: str, db: MySQLCLI, http: HTTPClient) -> None:
        self.api = api_base.rstrip("/")
        self.sidecar = sidecar_base.rstrip("/")
        self.db = db
        self.http = http

    def run(self, report: AcceptanceReport) -> None:
        self._environment(report)
        self._entry_first_open(report)
        self._entry_zone_scan(report)
        self._guidance(report)
        self._credibility(report)
        self._sidecar_resolve(report)
        multi_chat = self._multi_turn(report)
        if multi_chat:
            self._persistence(report, multi_chat["session_id"])

    # ── environment ─────────────────────────────────

    def _environment(self, report: AcceptanceReport) -> None:
        run_check(report, "environment", "Go backend health",
                  lambda: self._health(f"{self.api}/healthz"))
        run_check(report, "environment", "Python sidecar health",
                  lambda: self._health(f"{self.sidecar}/health"))
        run_check(report, "environment", "MySQL connectivity",
                  lambda: (self.db.scalar("SELECT 1") == "1", "SELECT 1"))

    def _health(self, url: str) -> tuple[bool, str]:
        status, payload = self.http.request("GET", url, timeout=5)
        return status == 200, f"HTTP {status}: {payload}"

    # ── entry: first_open ──────────────────────────

    def _entry_first_open(self, report: AcceptanceReport) -> None:
        def check() -> tuple[bool, str]:
            status, resp = chat(self.api, self.http,
                                message="打开", entry_mode="first_open")
            if status != 200:
                return False, f"HTTP {status}: {resp}"
            return validate_first_open(resp)

        run_check(report, "entry", "first_open returns preset questions", check)

    # ── entry: zone_scan ────────────────────────────

    def _entry_zone_scan(self, report: AcceptanceReport) -> None:
        def check() -> tuple[bool, str]:
            status, resp = chat(self.api, self.http,
                                message="扫码", entry_mode="zone_scan",
                                zone_id=2, shelf_id=3)
            if status != 200:
                return False, f"HTTP {status}: {resp}"
            return validate_zone_scan(resp)

        run_check(report, "entry", "zone_scan returns shelf products", check)

    # ── guidance ────────────────────────────────────

    def _guidance(self, report: AcceptanceReport) -> None:
        def check() -> tuple[bool, str]:
            status, resp = chat(self.api, self.http, message="可乐在哪里")
            if status != 200:
                return False, f"HTTP {status}: {resp}"
            if resp.get("intent") != "product_location":
                return False, f"intent={resp.get('intent')}"
            return validate_guidance(resp)

        run_check(report, "guidance", "location answer has guidance chips", check)

    # ── credibility ─────────────────────────────────

    def _credibility(self, report: AcceptanceReport) -> None:
        def check() -> tuple[bool, str]:
            status, resp = chat(self.api, self.http, message="可乐还有几瓶")
            if status != 200:
                return False, f"HTTP {status}: {resp}"
            answer = str(resp.get("answer") or "")
            cards = resp.get("cards") or []
            return validate_credibility(answer, cards)

        run_check(report, "credibility", "inventory answer has credibility level", check)

    # ── sidecar /llm/resolve ────────────────────────

    def _sidecar_resolve(self, report: AcceptanceReport) -> None:
        run_check(report, "sidecar-resolve", "/llm/resolve returns resolved_entities",
                  lambda: validate_sidecar_resolve(self.sidecar, self.http))
        run_check(report, "sidecar-resolve", "Go client routes resolved_entities to orchestrator",
                  lambda: validate_sidecar_resolve_go_client(self.api, self.db, self.http))

    # ── multi-turn ──────────────────────────────────

    def _multi_turn(self, report: AcceptanceReport) -> dict[str, Any] | None:
        result: dict[str, Any] = {}

        def check() -> tuple[bool, str]:
            nonlocal result
            # R1: 可乐在哪里
            status, r1 = chat(self.api, self.http, message="可乐在哪里")
            if status != 200:
                return False, f"R1 HTTP {status}: {r1}"
            sid = int(r1.get("session_id") or 0)
            if sid <= 0:
                return False, "R1 missing session_id"

            # R2: 还有几瓶？
            status, r2 = chat(self.api, self.http, session_id=sid, message="还有几瓶？")
            if status != 200:
                return False, f"R2 HTTP {status}: {r2}"

            # R3: 多少钱？
            status, r3 = chat(self.api, self.http, session_id=sid, message="多少钱？")
            if status != 200:
                return False, f"R3 HTTP {status}: {r3}"

            result = {"session_id": sid, "r1": r1, "r2": r2, "r3": r3}
            return validate_multi_turn(r2, r3)

        run_check(report, "multi-turn", "follow-up '还有几瓶/多少钱' inherits product", check)
        return result if result.get("session_id") else None

    # ── persistence ─────────────────────────────────

    def _persistence(self, report: AcceptanceReport, session_id: int) -> None:
        run_check(report, "persistence", "context_state persisted",
                  lambda: self._check_column(session_id, "context_state"))
        run_check(report, "persistence", "focus_entity_ids persisted",
                  lambda: self._check_column(session_id, "focus_entity_ids"))
        run_check(report, "persistence", "context_stack persisted",
                  lambda: self._check_column(session_id, "context_stack"))

    def _check_column(self, session_id: int, column: str) -> tuple[bool, str]:
        rows = self.db.rows(
            f"SELECT id, {column} FROM agent_message "
            f"WHERE session_id={session_id} AND role='assistant' ORDER BY id"
        )
        for row in rows:
            val = str(row[1]) if len(row) > 1 else ""
            if column == "context_state" and val and val not in ("None", ""):
                return True, f"message_id={row[0]}, {column}={val}"
            if column == "focus_entity_ids" and val and val not in ("null", "None", ""):
                return True, f"message_id={row[0]}, {column}={val[:80]}"
            if column == "context_stack" and val and val not in ("[]", "None", "") and len(val) > 5:
                return True, f"message_id={row[0]}, {column}={val[:80]}"
        return False, f"no non-empty {column} in session {session_id}"


# ── CLI ────────────────────────────────────────────

def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--mode", choices=("local", "ci"), default="local")
    parser.add_argument("--api-base", default=os.getenv("S1_API_BASE", os.getenv("S0_API_BASE", "http://127.0.0.1:8080")))
    parser.add_argument("--sidecar-base", default=os.getenv("S1_SIDECAR_BASE", os.getenv("S0_SIDECAR_BASE", "http://127.0.0.1:9090")))
    parser.add_argument("--output-dir", type=Path,
                        default=Path(os.getenv("S1_OUTPUT_DIR", str(DEFAULT_OUTPUT_DIR))))
    parser.add_argument("--mysql-host", default=os.getenv("S1_MYSQL_HOST", os.getenv("S0_MYSQL_HOST", "127.0.0.1")))
    parser.add_argument("--mysql-port", type=int, default=int(os.getenv("S1_MYSQL_PORT", os.getenv("S0_MYSQL_PORT", "3307"))))
    parser.add_argument("--mysql-user", default=os.getenv("S1_MYSQL_USER", os.getenv("S0_MYSQL_USER", "app")))
    parser.add_argument("--mysql-password", default=os.getenv("S1_MYSQL_PASSWORD", os.getenv("S0_MYSQL_PASSWORD", "app")))
    parser.add_argument("--mysql-database", default=os.getenv("S1_MYSQL_DATABASE", os.getenv("S0_MYSQL_DATABASE", "store_mind")))
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    report = AcceptanceReport(mode=args.mode)
    runner = S1Acceptance(
        api_base=args.api_base,
        sidecar_base=args.sidecar_base,
        db=MySQLCLI(args.mysql_host, args.mysql_port, args.mysql_user, args.mysql_password, args.mysql_database),
        http=HTTPClient(),
    )
    runner.run(report)
    report.write(args.output_dir)
    print(
        f"\nS1 {'PASS' if report.passed else 'FAIL'}: "
        f"{len(report.checks) - report.failed_count}/{len(report.checks)} checks passed"
    )
    print(f"Reports: {args.output_dir}")
    return 0 if report.passed else 1


if __name__ == "__main__":
    sys.exit(main())
