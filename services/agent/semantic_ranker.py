"""
语义排序器 — POST /llm/semantic_search
将 FAQ 候选集按语义相关性重新排序，提升召回质量。
"""

import json
import asyncio
from typing import Optional

from openai import AsyncOpenAI

SEMANTIC_RANK_TIMEOUT_SECONDS = 5.0

SEMANTIC_RANK_SYSTEM_PROMPT = """你是无人超市数字店员「小王」的 FAQ 语义排序器。

你的任务：根据用户查询，对候选 FAQ 条目按语义相关性排序。

## 输入格式
- query: 用户的查询文本
- candidates: 候选 FAQ 列表，每项包含 id、question、answer、category

## 排序规则
1. 优先匹配用户查询的核心语义意图，而非字面关键词
2. 问题 (question) 与查询语义高度相关的条目应排在最前面
3. 答案 (answer) 内容若能直接回答查询，即使问题措辞不同也应提升排名
4. 类别 (category) 与查询领域匹配的条目给予适度加权
5. 无关条目排在最后，relevance_score 接近 0

## 输出格式（严格 JSON，无额外文本）
返回所有候选条目的排序结果，按相关性降序排列：
{
  "ranked": [
    {"id": 3, "relevance_score": 0.95},
    {"id": 1, "relevance_score": 0.72},
    {"id": 5, "relevance_score": 0.18}
  ]
}

注意：
- 必须包含所有输入候选，不得遗漏
- relevance_score 为 0.0-1.0 的浮点数
- 按 relevance_score 降序排列
"""


class SemanticRanker:
    """使用 DeepSeek API 对 FAQ 候选集进行语义重排序"""

    def __init__(self, api_key: str, base_url: str, model: str):
        self.client = AsyncOpenAI(api_key=api_key, base_url=base_url)
        self.model = model

    async def rank(
        self,
        query: str,
        candidates: list[dict],
    ) -> list[dict]:
        """
        对候选 FAQ 按语义相关性排序。

        Args:
            query: 用户查询文本
            candidates: 候选列表，每项 {"id": int, "question": str, "answer": str, "category": str}

        Returns:
            排序后的列表 [{"id": int, "relevance_score": float}, ...]，按相关性降序
        """
        if not query.strip():
            return _identity_rank(candidates)
        if len(candidates) == 0:
            return []
        if len(candidates) == 1:
            return [{"id": candidates[0]["id"], "relevance_score": 1.0}]

        # 构建用户消息：将 query + candidates 一并传入
        candidates_json = json.dumps(candidates, ensure_ascii=False, indent=2)
        user_content = f"用户查询: {query}\n\n候选 FAQ 列表:\n{candidates_json}"

        try:
            response = await asyncio.wait_for(
                self.client.chat.completions.create(
                    model=self.model,
                    messages=[
                        {"role": "system", "content": SEMANTIC_RANK_SYSTEM_PROMPT},
                        {"role": "user", "content": user_content},
                    ],
                    temperature=0.0,  # 零温度确保排序稳定
                    max_tokens=1024,
                    response_format={"type": "json_object"},
                ),
                timeout=SEMANTIC_RANK_TIMEOUT_SECONDS,
            )

            content = response.choices[0].message.content or "{}"
            result = json.loads(content)
            ranked = result.get("ranked", [])

            # 校验返回结果
            if not ranked:
                return _identity_rank(candidates)

            # 确保所有候选都返回了，缺少的补到末尾
            returned_ids = {item["id"] for item in ranked}
            for c in candidates:
                if c["id"] not in returned_ids:
                    ranked.append({"id": c["id"], "relevance_score": 0.0})

            # 标准化 relevance_score 为浮点数
            for item in ranked:
                item["relevance_score"] = float(item.get("relevance_score", 0.0))

            return ranked

        except asyncio.TimeoutError:
            return _identity_rank(candidates)
        except Exception:
            return _identity_rank(candidates)


def _identity_rank(candidates: list[dict]) -> list[dict]:
    """降级兜底：保持原始顺序，统一赋予 0.5 分"""
    return [{"id": c["id"], "relevance_score": 0.5} for c in candidates]
