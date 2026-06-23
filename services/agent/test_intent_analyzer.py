import unittest
from types import SimpleNamespace
from unittest.mock import patch

from anaphora_resolver import AnaphoraResolver
from answer_composer import AnswerComposer
from intent_analyzer import IntentAnalyzer
from semantic_ranker import SemanticRanker


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


class LLMTimeoutTest(unittest.IsolatedAsyncioTestCase):
    async def test_anaphora_request_uses_five_second_timeout(self):
        resolver = AnaphoraResolver(
            api_key="test-key",
            base_url="https://example.invalid/v1",
            model="test-model",
        )
        resolver.client = _fake_client('{"resolved_entities":[],"confidence":0.9}')
        captured = {}

        async def capture_wait_for(awaitable, timeout):
            captured["timeout"] = timeout
            return await awaitable

        with patch("anaphora_resolver.asyncio.wait_for", side_effect=capture_wait_for):
            await resolver.resolve("还有几瓶？")

        self.assertEqual(captured["timeout"], 5.0)

    async def test_answer_request_uses_five_second_timeout(self):
        composer = AnswerComposer(
            api_key="test-key",
            base_url="https://example.invalid/v1",
            model="test-model",
        )
        composer.client = _fake_client('{"answer":"在 B-02","guidance_chips":[]}')
        captured = {}

        async def capture_wait_for(awaitable, timeout):
            captured["timeout"] = timeout
            return await awaitable

        with patch("answer_composer.asyncio.wait_for", side_effect=capture_wait_for):
            await composer.compose({"intent": "product_location"}, "可乐在哪里", [])

        self.assertEqual(captured["timeout"], 5.0)

    async def test_semantic_rank_request_uses_five_second_timeout(self):
        ranker = SemanticRanker(
            api_key="test-key",
            base_url="https://example.invalid/v1",
            model="test-model",
        )
        ranker.client = _fake_client('{"ranked":[{"id":2,"relevance_score":0.9},{"id":1,"relevance_score":0.2}]}')
        captured = {}

        async def capture_wait_for(awaitable, timeout):
            captured["timeout"] = timeout
            return await awaitable

        candidates = [
            {"id": 1, "question": "怎么退款", "answer": "联系人工", "category": "售后"},
            {"id": 2, "question": "可乐在哪里", "answer": "B-02", "category": "商品"},
        ]
        with patch("semantic_ranker.asyncio.wait_for", side_effect=capture_wait_for):
            await ranker.rank("可乐在哪", candidates)

        self.assertEqual(captured["timeout"], 5.0)


def _fake_client(content):
    async def create_response(**_kwargs):
        return SimpleNamespace(
            choices=[
                SimpleNamespace(message=SimpleNamespace(content=content))
            ]
        )

    return SimpleNamespace(
        chat=SimpleNamespace(
            completions=SimpleNamespace(create=create_response)
        )
    )


if __name__ == "__main__":
    unittest.main()
