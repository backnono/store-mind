import importlib.util
import json
import tempfile
import unittest
import xml.etree.ElementTree as ET
from pathlib import Path


MODULE_PATH = Path(__file__).with_name("s0_acceptance.py")
SPEC = importlib.util.spec_from_file_location("s0_acceptance", MODULE_PATH)
s0 = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(s0)


class AcceptanceReportTest(unittest.TestCase):
    def test_failed_check_makes_report_fail_and_writes_all_formats(self):
        report = s0.AcceptanceReport(mode="ci")
        report.add("environment", "Go health", True, "HTTP 200")
        report.add("e2e", "LLM answer path", False, "evidence_count=0")

        self.assertFalse(report.passed)
        self.assertEqual(report.failed_count, 1)

        with tempfile.TemporaryDirectory() as tmp:
            output = Path(tmp)
            report.write(output)

            payload = json.loads((output / "s0-report.json").read_text())
            self.assertFalse(payload["passed"])
            self.assertEqual(payload["summary"]["failed"], 1)
            self.assertIn("LLM answer path", (output / "s0-report.md").read_text())

            root = ET.parse(output / "s0-report.xml").getroot()
            self.assertEqual(root.attrib["failures"], "1")


class AcceptanceAssertionTest(unittest.TestCase):
    def test_e2e_requires_real_llm_tool_answer_path(self):
        response = {
            "intent": "product_location",
            "answer": "德芙在零食区 A-01 货架第2层。",
            "cards": [{"type": "product"}],
            "meta": {
                "route": "tool",
                "fallback_used": False,
                "evidence_count": 1,
            },
        }

        ok, detail = s0.validate_e2e_response(response, ["零食区", "A-01"])

        self.assertTrue(ok, detail)

    def test_e2e_rejects_template_response_without_evidence(self):
        response = {
            "intent": "product_location",
            "answer": "暂时没有找到可靠依据回答这个问题。",
            "cards": None,
            "meta": {
                "route": "tool",
                "fallback_used": False,
                "evidence_count": 0,
            },
        }

        ok, detail = s0.validate_e2e_response(response, ["零食区", "A-01"])

        self.assertFalse(ok)
        self.assertIn("evidence_count", detail)

    def test_no_fabrication_rejects_specific_location_without_evidence(self):
        response = {
            "answer": "防晒霜在日用品区 C-01 货架第3层。",
            "meta": {"evidence_count": 0},
        }

        ok, detail = s0.validate_no_fabrication_response(response)

        self.assertFalse(ok)
        self.assertIn("specific location", detail)

    def test_no_fabrication_accepts_evidence_insufficiency_answer(self):
        response = {
            "answer": "暂时没有找到可靠依据回答这个问题，你可以换个问法。",
            "meta": {"evidence_count": 0},
        }

        ok, detail = s0.validate_no_fabrication_response(response)

        self.assertTrue(ok, detail)


class IntentAccuracyTest(unittest.TestCase):
    def test_composite_intent_accepts_expected_parts_in_any_order(self):
        self.assertTrue(
            s0.is_intent_correct(
                "product_location,inventory", "inventory,product_location"
            )
        )
        self.assertFalse(
            s0.is_intent_correct("product_location,inventory", "product_location")
        )

    def test_accuracy_gate_requires_eighty_five_percent(self):
        ok, detail = s0.validate_intent_accuracy(correct=47, total=56, threshold=85.0)
        self.assertFalse(ok)
        self.assertIn("83.9%", detail)

        ok, detail = s0.validate_intent_accuracy(correct=48, total=56, threshold=85.0)
        self.assertTrue(ok, detail)


if __name__ == "__main__":
    unittest.main()
