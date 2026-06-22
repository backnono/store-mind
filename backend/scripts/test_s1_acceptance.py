import importlib.util
import json
import tempfile
import unittest
import xml.etree.ElementTree as ET
from pathlib import Path


MODULE_PATH = Path(__file__).with_name("s1_acceptance.py")
SPEC = importlib.util.spec_from_file_location("s1_acceptance", MODULE_PATH)
s1 = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(s1)


class AcceptanceReportTest(unittest.TestCase):
    def test_failed_check_makes_report_fail_and_writes_all_formats(self):
        report = s1.AcceptanceReport(mode="ci")
        report.add("entry", "first_open", True, "HTTP 200, chips=4")
        report.add("multi-turn", "follow-up inherits product", False, "R2 intent=unsupported")

        self.assertFalse(report.passed)
        self.assertEqual(report.failed_count, 1)

        with tempfile.TemporaryDirectory() as tmp:
            output = Path(tmp)
            report.write(output)

            payload = json.loads((output / "s1-report.json").read_text())
            self.assertFalse(payload["passed"])
            self.assertEqual(payload["summary"]["failed"], 1)
            self.assertIn("R2 intent=unsupported", (output / "s1-report.md").read_text())

            root = ET.parse(output / "s1-report.xml").getroot()
            self.assertEqual(root.attrib["failures"], "1")

    def test_all_pass_marks_report_passed(self):
        report = s1.AcceptanceReport(mode="local")
        report.add("entry", "first_open", True, "")
        report.add("guidance", "location chips", True, "")
        self.assertTrue(report.passed)
        self.assertEqual(report.failed_count, 0)


class S1ValidationTest(unittest.TestCase):
    # ── first_open ──────────────────────────────────

    def test_first_open_valid(self):
        response = {
            "session_id": 1, "message_id": 2,
            "intent": "greeting",
            "meta": {"route": "entry_first_open"},
            "guidance_chips": [
                {"text": "📍 薯片在哪里？", "prompt": "薯片在哪里？"},
                {"text": "🏷 今天有什么活动？", "prompt": "今天有什么活动？"},
                {"text": "🥤 低糖饮料有哪些？", "prompt": "低糖饮料有哪些？"},
                {"text": "💳 怎么付款？", "prompt": "怎么付款？"},
            ],
        }
        ok, detail = s1.validate_first_open(response)
        self.assertTrue(ok, detail)

    def test_first_open_missing_chips(self):
        response = {
            "intent": "greeting",
            "meta": {"route": "entry_first_open"},
            "guidance_chips": [{"text": "hello", "prompt": "hello"}],
        }
        ok, _ = s1.validate_first_open(response)
        self.assertFalse(ok)

    def test_first_open_wrong_intent(self):
        response = {"intent": "unsupported", "meta": {"route": "entry_first_open"}, "guidance_chips": []}
        ok, detail = s1.validate_first_open(response)
        self.assertFalse(ok)
        self.assertIn("intent", detail)

    # ── zone_scan ───────────────────────────────────

    def test_zone_scan_valid(self):
        response = {
            "intent": "zone_scan",
            "meta": {"route": "entry_zone_scan"},
            "cards": [{"type": "product", "name": "test"}],
        }
        ok, detail = s1.validate_zone_scan(response)
        self.assertTrue(ok, detail)

    def test_zone_scan_empty_cards(self):
        response = {"intent": "zone_scan", "meta": {"route": "entry_zone_scan"}, "cards": []}
        ok, _ = s1.validate_zone_scan(response)
        self.assertFalse(ok)

    # ── guidance ────────────────────────────────────

    def test_guidance_valid(self):
        response = {
            "guidance_chips": [
                {"text": "📦 还有几瓶？", "prompt": "还有几瓶？"},
                {"text": "🏷 这个有活动吗？", "prompt": "这个有活动吗？"},
            ],
        }
        ok, detail = s1.validate_guidance(response)
        self.assertTrue(ok, detail)

    def test_guidance_insufficient(self):
        response = {"guidance_chips": []}
        ok, detail = s1.validate_guidance(response)
        self.assertFalse(ok)
        self.assertIn("0", detail)

    # ── credibility ─────────────────────────────────

    def test_credibility_via_high_marker(self):
        ok, detail = s1.validate_credibility(
            "系统显示可口可乐还有 12 件 · high · 10分钟前更新", []
        )
        self.assertTrue(ok, detail)

    def test_credibility_via_chinese_marker(self):
        ok, detail = s1.validate_credibility(
            "上次盘点显示还有 12 瓶（2小时前），建议到货架确认。", []
        )
        self.assertTrue(ok, detail)

    def test_credibility_via_card(self):
        ok, detail = s1.validate_credibility(
            "可乐还有货", [{"type": "inventory", "quantity": 12}]
        )
        self.assertTrue(ok, detail)

    def test_credibility_missing(self):
        ok, detail = s1.validate_credibility("可乐还有货", [])
        self.assertFalse(ok)

    # ── multi-turn ──────────────────────────────────

    def test_multi_turn_valid(self):
        r2 = {"intent": "inventory", "answer": "可乐还有 12 瓶"}
        r3 = {"intent": "inventory", "answer": "可乐 ¥3.50"}
        ok, detail = s1.validate_multi_turn(r2, r3)
        self.assertTrue(ok, detail)

    def test_multi_turn_r2_unsupported_rejected(self):
        r2 = {"intent": "unsupported"}
        r3 = {"intent": "unsupported"}
        ok, _ = s1.validate_multi_turn(r2, r3)
        self.assertFalse(ok)

    def test_multi_turn_r2_faq_no_answer_rejected(self):
        r2 = {"intent": "faq", "answer": "没有找到相关商品信息"}
        r3 = {"intent": "faq"}
        ok, _ = s1.validate_multi_turn(r2, r3)
        self.assertFalse(ok)


if __name__ == "__main__":
    unittest.main()
