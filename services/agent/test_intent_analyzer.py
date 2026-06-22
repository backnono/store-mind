import unittest
from types import SimpleNamespace
from unittest.mock import patch

from intent_analyzer import IntentAnalyzer


class IntentAnalyzerTimeoutTest(unittest.IsolatedAsyncioTestCase):
    async def test_deepseek_request_uses_five_second_timeout(self):
        analyzer = IntentAnalyzer(
            api_key="test-key",
            base_url="https://example.invalid/v1",
            model="test-model",
        )

        async def create_response(**_kwargs):
            return SimpleNamespace(
                choices=[
                    SimpleNamespace(
                        message=SimpleNamespace(
                            content='{"intent":"product_location","confidence":0.95}'
                        )
                    )
                ]
            )

        analyzer.client = SimpleNamespace(
            chat=SimpleNamespace(
                completions=SimpleNamespace(create=create_response)
            )
        )
        captured = {}

        async def capture_wait_for(awaitable, timeout):
            captured["timeout"] = timeout
            return await awaitable

        with patch("intent_analyzer.asyncio.wait_for", side_effect=capture_wait_for):
            await analyzer.analyze("椰树椰汁在哪里")

        self.assertEqual(captured["timeout"], 5.0)


if __name__ == "__main__":
    unittest.main()
