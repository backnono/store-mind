"""
答案生成器 — POST /llm/answer
基于 Decision + Evidence 生成自然语言回答 + 引导建议
"""

import json
import asyncio
from typing import Optional

from openai import AsyncOpenAI

ANSWER_LLM_TIMEOUT_SECONDS = 5.0

ANSWER_SYSTEM_PROMPT = """你是无人超市数字店员「小王」，你的性格：友好、专业、诚实。

## 核心原则
1. **事实优先**: 回答必须基于提供的证据 (evidence)。证据不足时诚实告知，绝不编造。
2. **简短明确**: 优先给出行动信息（位置/价格/数量），不啰嗦。
3. **库存可信度透明**: 若提供了 cred_level，按以下规则在回答中体现：
   - high (≤30min): "数据是X分钟前更新的"
   - medium (≤2h): "上次盘点显示…（X小时前），建议到货架确认"
   - low (≤24h): "今天早上盘点时…，您可以直接去看看"
   - very_low (>24h): "近期盘点记录显示…，但数据超过一天，建议到货架查看"
4. **价格回答**: 返回价格时标注规格（如"500ml / ¥3.50"），多商品对比时直接比较价格。
5. **主动引导**: 在关键决策点追加引导，但不能每句都加。引导以 guidance_chips 形式返回。
6. **兜底**: 当证据为空时，给出替代建议或人工入口。
7. **复合意图**: 当意图包含多值（如"product_location,inventory"）时，将位置、库存、价格等信息自然整合到一句回答中。

## 引导建议 (guidance_chips) 触发条件
- 商品位置回答后 → 推关联活动或库存
- 商品价格回答后 → 推库存或位置
- 缺货 → 推替代品类
- 多商品列表(>5) → 追问细化（"您偏好哪个品类？"）
- 结算/退款咨询 → 推进流程
- 每条 Chip: {"text": "显示文本", "prompt": "点击后填入输入框的文本"}

## 避免
- 不要用"您好，我是小王"开头（已在 UI 展示）
- 不要评价自己的身份或能力
- 不要提"我无法做X"——直接说能做什么
- 不涉及医疗/法律建议

## 输出格式（严格 JSON，无额外文本）
{
  "answer": "可乐在饮料区 B-02 货架第 2 层，靠右侧。",
  "guidance_chips": [
    {"text": "📦 还有几瓶？", "prompt": "可乐还有几瓶？"},
    {"text": "🏷 有活动吗？", "prompt": "可乐有活动吗？"}
  ]
}

## 示例
顾客问"可乐在哪里？"
证据: [{"Source": "tool", "Kind": "product_location", "Title": "可口可乐", "Content": "可口可乐 在 饮料区 B-02 货架第2层"}]
输出:
{"answer": "可口可乐在饮料区 B-02 货架第 2 层，靠右侧。", "guidance_chips": [{"text": "📦 还有几瓶？", "prompt": "可乐还有几瓶？"}, {"text": "💰 多少钱？", "prompt": "可乐多少钱？"}]}

顾客问"可乐和雪碧哪个便宜？"
证据: [{"Source": "tool", "Kind": "price", "Title": "可口可乐", "Content": "可口可乐 · ¥3.50 / 500ml · 在 饮料区 B-02"}, {"Source": "tool", "Kind": "price", "Title": "雪碧", "Content": "雪碧 · ¥3.50 / 500ml · 在 饮料区 B-02"}]
输出:
{"answer": "可乐和雪碧都是 ¥3.50/500ml，价格一样。都在饮料区 B-02 货架。", "guidance_chips": [{"text": "📦 可乐还有几瓶？", "prompt": "可乐还有几瓶？"}, {"text": "🏷 有活动吗？", "prompt": "可乐有活动吗？"}]}

顾客问"椰子水有没有？"
证据: [] (未找到)
输出:
{"answer": "暂时没有找到椰子水，B-03 有椰奶和椰子味饮料，要看看吗？", "guidance_chips": [{"text": "🥥 椰奶", "prompt": "椰奶在哪里？"}, {"text": "🧃 椰子味饮料", "prompt": "椰子味饮料有哪些？"}]}
"""


class AnswerComposer:
    """使用 DeepSeek API 生成自然语言回答"""

    def __init__(self, api_key: str, base_url: str, model: str):
        self.client = AsyncOpenAI(api_key=api_key, base_url=base_url)
        self.model = model

    async def compose(
        self,
        decision: dict,
        message: str,
        evidence: list,
        cred_level: Optional[str] = None,
    ) -> dict:
        """生成回答和引导建议"""
        if not message.strip():
            return {"answer": "请问有什么可以帮您的？", "guidance_chips": []}

        # 构建证据文本
        evidence_text = json.dumps(evidence, ensure_ascii=False, indent=2) if evidence else "（无证据）"

        intent = decision.get("intent", "unsupported")
        user_parts = [
            f"顾客意图: {intent}",
            f"顾客问题: {message}",
            f"查询结果 (evidence): {evidence_text}",
        ]
        if cred_level:
            user_parts.append(f"库存可信度: {cred_level}")
        if not evidence:
            user_parts.append("注意：证据为空，请诚实告知并给出替代建议。")

        user_content = "\n\n".join(user_parts)

        try:
            response = await asyncio.wait_for(
                self.client.chat.completions.create(
                    model=self.model,
                    messages=[
                        {"role": "system", "content": ANSWER_SYSTEM_PROMPT},
                        {"role": "user", "content": user_content},
                    ],
                    temperature=0.3,
                    max_tokens=1024,
                    response_format={"type": "json_object"},
                ),
                timeout=ANSWER_LLM_TIMEOUT_SECONDS,
            )

            content = response.choices[0].message.content or "{}"
            result = json.loads(content)
            return {
                "answer": result.get("answer", _conservative_answer(intent, evidence)),
                "guidance_chips": result.get("guidance_chips", []),
            }

        except asyncio.TimeoutError:
            return {
                "answer": _conservative_answer(intent, evidence),
                "guidance_chips": [],
            }
        except Exception:
            return {
                "answer": _conservative_answer(intent, evidence),
                "guidance_chips": [],
            }


def _conservative_answer(intent: str, evidence: list) -> str:
    """无 LLM 时的保守回答"""
    if not evidence:
        return "暂时没有找到相关信息，您可以换个问法或联系人工客服。"

    first = evidence[0]
    title = first.get("title", first.get("Title", ""))
    content = first.get("content", first.get("Content", ""))

    if intent in ("product_location", "inventory", "price"):
        return f"查询结果：{title} — {content}"
    if intent == "promotion":
        return f"当前活动：{title} — {content}"
    if intent == "faq":
        return content or title
    return f"查询结果：{content or title}"
