#!/usr/bin/env python3
"""S0 acceptance harness for local diagnostics and CI gating."""

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
DEFAULT_OUTPUT_DIR = ROOT / "artifacts" / "s0"
INTENT_CASES = ROOT / "services" / "agent" / "test_intent_cases.json"


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
        (output_dir / "s0-report.json").write_text(
            json.dumps(payload, ensure_ascii=False, indent=2) + "\n",
            encoding="utf-8",
        )

        lines = [
            "# S0 Acceptance Report",
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
        (output_dir / "s0-report.md").write_text(
            "\n".join(lines) + "\n", encoding="utf-8"
        )

        suite = ET.Element(
            "testsuite",
            name="s0-acceptance",
            tests=str(len(self.checks)),
            failures=str(self.failed_count),
        )
        for item in self.checks:
            case = ET.SubElement(
                suite,
                "testcase",
                classname=f"s0.{item['gate']}",
                name=item["name"],
            )
            if not item["passed"]:
                failure = ET.SubElement(case, "failure", message=str(item["detail"]))
                failure.text = str(item["detail"])
        ET.ElementTree(suite).write(
            output_dir / "s0-report.xml", encoding="utf-8", xml_declaration=True
        )


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
        request = urllib.request.Request(
            url, data=body, headers=headers, method=method
        )
        try:
            with urllib.request.urlopen(request, timeout=timeout) as response:
                raw = response.read().decode()
                return response.status, json.loads(raw) if raw else {}
        except urllib.error.HTTPError as error:
            raw = error.read().decode()
            try:
                data = json.loads(raw) if raw else {}
            except json.JSONDecodeError:
                data = {"raw": raw}
            return error.code, data


class MySQLCLI:
    def __init__(
        self, host: str, port: int, user: str, password: str, database: str
    ) -> None:
        self.host = host
        self.port = port
        self.user = user
        self.password = password
        self.database = database

    def rows(self, sql: str) -> list[list[str]]:
        env = os.environ.copy()
        env["MYSQL_PWD"] = self.password
        command = [
            "mysql",
            f"-h{self.host}",
            f"-P{self.port}",
            f"-u{self.user}",
            "-N",
            "-B",
            self.database,
            "-e",
            sql,
        ]
        completed = subprocess.run(
            command,
            check=True,
            capture_output=True,
            text=True,
            env=env,
        )
        return [
            line.split("\t")
            for line in completed.stdout.splitlines()
            if line.strip()
        ]

    def scalar(self, sql: str) -> str:
        rows = self.rows(sql)
        if not rows or not rows[0]:
            return ""
        return rows[0][0]


def validate_e2e_response(
    response: dict[str, Any], expected_fragments: list[str]
) -> tuple[bool, str]:
    meta = response.get("meta") or {}
    evidence_count = int(meta.get("evidence_count") or 0)
    cards = response.get("cards") or []
    answer = str(response.get("answer") or "")
    failures = []
    if response.get("intent") != "product_location":
        failures.append(f"intent={response.get('intent')!r}")
    if meta.get("route") != "tool":
        failures.append(f"route={meta.get('route')!r}")
    if meta.get("fallback_used") is not False:
        failures.append(f"fallback_used={meta.get('fallback_used')!r}")
    if evidence_count <= 0:
        failures.append(f"evidence_count={evidence_count}")
    if not cards:
        failures.append("cards empty")
    missing = [part for part in expected_fragments if part not in answer]
    if missing:
        failures.append(f"answer missing database facts: {missing}")
    if failures:
        return False, "; ".join(failures)
    return True, f"evidence_count={evidence_count}, cards={len(cards)}"


def validate_no_fabrication_response(
    response: dict[str, Any],
) -> tuple[bool, str]:
    answer = str(response.get("answer") or "")
    evidence_count = int((response.get("meta") or {}).get("evidence_count") or 0)
    if evidence_count != 0:
        return False, f"expected evidence_count=0, got {evidence_count}"
    specific_location = re.search(
        r"(在.{0,16}(区|货架)|[A-Z]-\d{1,3}|货架第?\d+层)", answer
    )
    if specific_location:
        return False, f"specific location without evidence: {specific_location.group(0)}"
    honest_markers = ("没有找到", "未找到", "暂无", "暂时", "可靠依据", "换个问法")
    if not any(marker in answer for marker in honest_markers):
        return False, f"answer does not acknowledge insufficient evidence: {answer[:100]}"
    return True, answer[:120]


def validate_intent_accuracy(
    correct: int, total: int, threshold: float = 85.0
) -> tuple[bool, str]:
    accuracy = correct / total * 100 if total else 0.0
    detail = f"{accuracy:.1f}% ({correct}/{total}), required >= {threshold:.1f}%"
    return accuracy >= threshold, detail


def is_intent_correct(expected: str, actual: str) -> bool:
    expected_parts = {part.strip() for part in expected.split(",") if part.strip()}
    actual_parts = {part.strip() for part in actual.split(",") if part.strip()}
    return expected_parts == actual_parts


def run_check(
    report: AcceptanceReport,
    gate: str,
    name: str,
    check: Callable[[], tuple[bool, str]],
) -> None:
    try:
        passed, detail = check()
    except Exception as error:  # acceptance reports must collect all failures
        passed, detail = False, f"{type(error).__name__}: {error}"
    report.add(gate, name, passed, detail)
    icon = "PASS" if passed else "FAIL"
    print(f"[{icon}] {gate}: {name} — {detail}")


class S0Acceptance:
    def __init__(
        self,
        api_base: str,
        sidecar_base: str,
        database: MySQLCLI,
        http: HTTPClient,
        skip_intent_eval: bool,
    ) -> None:
        self.api = api_base.rstrip("/")
        self.sidecar = sidecar_base.rstrip("/")
        self.db = database
        self.http = http
        self.skip_intent_eval = skip_intent_eval

    def run(self, report: AcceptanceReport) -> None:
        self._environment(report)
        self._schema(report)
        self._sidecar_contract(report)
        if self.skip_intent_eval:
            report.add("intent-quality", "56-case LLM accuracy", True, "SKIPPED by option")
            print("[PASS] intent-quality: 56-case LLM accuracy — SKIPPED by option")
        else:
            self._intent_quality(report)
        chat = self._e2e(report)
        self._no_fabrication(report)
        if chat:
            self._persistence(report, chat)
            self._feedback(report, chat)

    def _environment(self, report: AcceptanceReport) -> None:
        run_check(
            report,
            "environment",
            "Go backend health",
            lambda: self._health(f"{self.api}/healthz"),
        )
        run_check(
            report,
            "environment",
            "Python sidecar health",
            lambda: self._health(f"{self.sidecar}/health"),
        )
        run_check(
            report,
            "environment",
            "MySQL connectivity",
            lambda: (self.db.scalar("SELECT 1") == "1", "SELECT 1"),
        )

    def _health(self, url: str) -> tuple[bool, str]:
        status, payload = self.http.request("GET", url, timeout=5)
        return status == 200, f"HTTP {status}: {payload}"

    def _schema(self, report: AcceptanceReport) -> None:
        required = {
            "inventory": {"last_verified_at", "update_source"},
            "agent_message": {"context_state", "focus_entity_ids", "context_stack"},
            "agent_feedback": {
                "id",
                "message_id",
                "session_id",
                "feedback_value",
                "created_at",
            },
            "agent_decision_log": {
                "intent",
                "route",
                "rewrite_query",
                "confidence",
                "fallback_used",
                "handoff_required",
            },
        }
        for table, columns in required.items():
            def check(table: str = table, columns: set[str] = columns) -> tuple[bool, str]:
                rows = self.db.rows(
                    "SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS "
                    f"WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME='{table}'"
                )
                actual = {row[0] for row in rows}
                missing = sorted(columns - actual)
                return not missing, "all required columns present" if not missing else f"missing={missing}"

            run_check(report, "schema", table, check)

        run_check(
            report,
            "schema",
            "legacy catalog data preserved",
            lambda: self._legacy_data_check(),
        )

    def _legacy_data_check(self) -> tuple[bool, str]:
        count = int(
            self.db.scalar("SELECT COUNT(*) FROM product WHERE id IN (101,102,103)")
            or 0
        )
        return count == 3, f"known products present={count}/3"

    def _sidecar_contract(self, report: AcceptanceReport) -> None:
        def intent() -> tuple[bool, str]:
            status, payload = self.http.request(
                "POST", f"{self.sidecar}/llm/intent", {"message": "可乐在哪里"}
            )
            required = {
                "intent",
                "route",
                "confidence",
                "rewritten_query",
                "fallback_used",
            }
            missing = required - payload.keys()
            valid = (
                status == 200
                and not missing
                and 0 <= float(payload.get("confidence", -1)) <= 1
                and payload.get("fallback_used") is False
            )
            return valid, f"HTTP {status}, missing={sorted(missing)}, payload={payload}"

        def answer() -> tuple[bool, str]:
            status, payload = self.http.request(
                "POST",
                f"{self.sidecar}/llm/answer",
                {
                    "decision": {
                        "intent": "product_location",
                        "route": "tool",
                        "confidence": 0.95,
                    },
                    "message": "可乐在哪里",
                    "evidence": [
                        {
                            "source": "tool",
                            "kind": "product_location",
                            "record_id": 1,
                            "title": "可口可乐",
                            "content": "可口可乐在饮料区 B-02 货架第2层",
                        }
                    ],
                },
            )
            valid = (
                status == 200
                and bool(str(payload.get("answer") or "").strip())
                and isinstance(payload.get("guidance_chips"), list)
            )
            return valid, f"HTTP {status}, answer={str(payload.get('answer') or '')[:100]}"

        def resolve() -> tuple[bool, str]:
            status, payload = self.http.request(
                "POST",
                f"{self.sidecar}/llm/resolve",
                {
                    "message": "它还有吗",
                    "context_stack": [
                        {
                            "turn": 1,
                            "intent": "product_location",
                            "resolved_entities": [
                                {"type": "product", "name": "可口可乐"}
                            ],
                        }
                    ],
                    "focus_entities": [{"type": "product", "name": "可口可乐"}],
                },
            )
            valid = status == 200 and "resolved_entities" in payload
            return valid, f"HTTP {status}, keys={sorted(payload.keys())}"

        run_check(report, "sidecar-contract", "/llm/intent", intent)
        run_check(report, "sidecar-contract", "/llm/answer", answer)
        run_check(report, "sidecar-contract", "/llm/resolve", resolve)

    def _intent_quality(self, report: AcceptanceReport) -> None:
        def check() -> tuple[bool, str]:
            suite = json.loads(INTENT_CASES.read_text(encoding="utf-8"))
            cases = suite["cases"]
            correct = 0
            errors = 0
            for case in cases:
                try:
                    status, payload = self.http.request(
                        "POST",
                        f"{self.sidecar}/llm/intent",
                        {
                            "message": case["message"],
                            "context_stack": None,
                            "session_state": None,
                        },
                        timeout=10,
                    )
                    actual = str(payload.get("intent") or "")
                    if status == 200 and is_intent_correct(
                        case["expected_intent"], actual
                    ):
                        correct += 1
                    elif status != 200:
                        errors += 1
                except Exception:
                    errors += 1
            passed, detail = validate_intent_accuracy(correct, len(cases))
            return passed and errors == 0, f"{detail}, request_errors={errors}"

        run_check(report, "intent-quality", "56-case LLM accuracy", check)

    def _e2e(self, report: AcceptanceReport) -> dict[str, Any] | None:
        result: dict[str, Any] = {}

        def check() -> tuple[bool, str]:
            nonlocal result
            status, result = self.http.request(
                "POST",
                f"{self.api}/api/v1/customer-qa/chat",
                {"store_id": 1, "channel": "miniapp", "message": "可乐在哪里"},
                timeout=25,
            )
            if status != 200:
                return False, f"HTTP {status}: {result}"
            expected = self.db.rows(
                "SELECT z.name, s.code FROM product p "
                "JOIN product_location pl ON pl.product_id=p.id "
                "JOIN zone z ON z.id=pl.zone_id "
                "JOIN shelf s ON s.id=pl.shelf_id "
                "WHERE pl.store_id=1 AND p.status='active' "
                "AND (p.name LIKE '%可乐%' OR p.aliases LIKE '%可乐%') LIMIT 1"
            )
            if not expected:
                return False, "fixture missing: no 可乐 location in database"
            return validate_e2e_response(result, expected[0])

        run_check(report, "e2e", "intent → tool → answer", check)
        return result if result.get("message_id") and result.get("session_id") else None

    def _no_fabrication(self, report: AcceptanceReport) -> None:
        def check() -> tuple[bool, str]:
            status, payload = self.http.request(
                "POST",
                f"{self.api}/api/v1/customer-qa/chat",
                {
                    "store_id": 1,
                    "channel": "miniapp",
                    "message": "本店绝对不存在的火星防晒霜在哪里",
                },
                timeout=25,
            )
            if status != 200:
                return False, f"HTTP {status}: {payload}"
            return validate_no_fabrication_response(payload)

        run_check(report, "no-fabrication", "missing product has no invented facts", check)

    def _persistence(
        self, report: AcceptanceReport, chat: dict[str, Any]
    ) -> None:
        session_id = int(chat["session_id"])
        assistant_id = int(chat["message_id"])

        def check() -> tuple[bool, str]:
            rows = self.db.rows(
                "SELECT um.id, am.id, dl.intent, dl.route, dl.confidence, dl.fallback_used "
                "FROM agent_message am "
                "JOIN agent_message um ON um.session_id=am.session_id AND um.role='user' "
                "JOIN agent_decision_log dl ON dl.session_id=am.session_id AND dl.message_id=um.id "
                f"WHERE am.id={assistant_id} AND am.session_id={session_id} "
                "ORDER BY um.id DESC LIMIT 1"
            )
            if not rows:
                return False, "message/decision-log relationship not found"
            row = rows[0]
            expected_meta = chat.get("meta") or {}
            valid = (
                row[2] == chat.get("intent")
                and row[3] == expected_meta.get("route")
                and abs(float(row[4]) - float(expected_meta.get("confidence") or 0)) < 0.0001
                and int(row[5]) == int(bool(expected_meta.get("fallback_used")))
            )
            return valid, f"user_message={row[0]}, assistant_message={row[1]}, decision={row[2:]}"

        run_check(report, "persistence", "messages and decision log are linked", check)

    def _feedback(self, report: AcceptanceReport, chat: dict[str, Any]) -> None:
        session_id = int(chat["session_id"])
        message_id = int(chat["message_id"])

        def submit(value: int) -> tuple[bool, str]:
            before = int(
                self.db.scalar(
                    "SELECT COUNT(*) FROM agent_feedback "
                    f"WHERE message_id={message_id} AND session_id={session_id} "
                    f"AND feedback_value={value}"
                )
                or 0
            )
            status, payload = self.http.request(
                "POST",
                f"{self.api}/api/v1/customer-qa/feedback",
                {
                    "message_id": message_id,
                    "session_id": session_id,
                    "feedback_value": value,
                },
            )
            after = int(
                self.db.scalar(
                    "SELECT COUNT(*) FROM agent_feedback "
                    f"WHERE message_id={message_id} AND session_id={session_id} "
                    f"AND feedback_value={value}"
                )
                or 0
            )
            valid = status == 200 and payload.get("status") == "ok" and after == before + 1
            return valid, f"HTTP {status}, rows {before}→{after}, payload={payload}"

        run_check(report, "feedback", "thumbs up persists", lambda: submit(1))
        run_check(report, "feedback", "thumbs down persists", lambda: submit(0))

        def invalid() -> tuple[bool, str]:
            status, payload = self.http.request(
                "POST",
                f"{self.api}/api/v1/customer-qa/feedback",
                {
                    "message_id": message_id,
                    "session_id": session_id,
                    "feedback_value": 2,
                },
            )
            return status == 400, f"HTTP {status}: {payload}"

        run_check(report, "feedback", "invalid value rejected", invalid)


def parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--mode", choices=("local", "ci"), default="local")
    parser.add_argument("--api-base", default=os.getenv("S0_API_BASE", "http://127.0.0.1:8080"))
    parser.add_argument(
        "--sidecar-base",
        default=os.getenv("S0_SIDECAR_BASE", "http://127.0.0.1:9090"),
    )
    parser.add_argument(
        "--output-dir",
        type=Path,
        default=Path(os.getenv("S0_OUTPUT_DIR", str(DEFAULT_OUTPUT_DIR))),
    )
    parser.add_argument("--skip-intent-eval", action="store_true")
    parser.add_argument("--mysql-host", default=os.getenv("S0_MYSQL_HOST", "127.0.0.1"))
    parser.add_argument("--mysql-port", type=int, default=int(os.getenv("S0_MYSQL_PORT", "3307")))
    parser.add_argument("--mysql-user", default=os.getenv("S0_MYSQL_USER", "app"))
    parser.add_argument("--mysql-password", default=os.getenv("S0_MYSQL_PASSWORD", "app"))
    parser.add_argument("--mysql-database", default=os.getenv("S0_MYSQL_DATABASE", "store_mind"))
    return parser.parse_args(argv)


def main(argv: list[str] | None = None) -> int:
    args = parse_args(argv)
    report = AcceptanceReport(mode=args.mode)
    runner = S0Acceptance(
        api_base=args.api_base,
        sidecar_base=args.sidecar_base,
        database=MySQLCLI(
            args.mysql_host,
            args.mysql_port,
            args.mysql_user,
            args.mysql_password,
            args.mysql_database,
        ),
        http=HTTPClient(),
        skip_intent_eval=args.skip_intent_eval,
    )
    runner.run(report)
    report.write(args.output_dir)
    print(
        f"\nS0 {'PASS' if report.passed else 'FAIL'}: "
        f"{len(report.checks) - report.failed_count}/{len(report.checks)} checks passed"
    )
    print(f"Reports: {args.output_dir}")
    return 0 if report.passed else 1


if __name__ == "__main__":
    sys.exit(main())
