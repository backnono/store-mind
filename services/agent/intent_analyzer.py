"""
意图分析器 — POST /llm/intent
将用户消息 + 上下文分析为结构化 Decision
"""

import json
import asyncio
from typing import Optional

from openai import AsyncOpenAI

INTENT_LLM_TIMEOUT_SECONDS = 5.0

# ── 7 类意图分类体系 ────────────────────────────────
# product_location  - 商品位置查询（"可乐在哪里？"）
# inventory          - 库存查询（"可乐还有吗？"）
# price              - 价格查询（"可乐多少钱？"）
# promotion          - 活动查询（"今天有什么活动？"）
# faq                - 门店 FAQ（"怎么退款？""营业时间？"）
# handoff            - 转人工（"我要人工客服"）
# unsupported        - 无关问题（"今天天气怎么样？"）
# 复合意图支持：intent 字段可以是逗号分隔的多值，如 "product_location,inventory"

INTENT_SYSTEM_PROMPT = """你是无人超市数字店员「小王」的意图分析器。

你的任务：分析顾客消息，输出结构化 JSON。

## 意图分类
- product_location: 询问具体商品的位置/在哪儿
- inventory: 询问库存/还有没有/还有几瓶
- price: 询问价格/多少钱
- promotion: 询问活动/优惠/打折/特价
- faq: 询问支付/退款/售后/营业时间/门店规则
- handoff: 明确要求人工客服
- unsupported: 与门店购物无关的问题

## 规则
1. 优先识别为单一意图；仅当问题明确包含多个独立意图时，intent 字段才用逗号分隔，如 "product_location,inventory"
2. rewritten_query: 如果是泛指/口语/不完整表达，改写为完整查询；否则保持不变
3. needs_handoff: 仅当用户明确要求人工客服或问题涉及退款/投诉等需人工处理时
4. confidence: 0.0-1.0，复合意图取各子意图置信度的最小值
5. 绝不编造：若无法确定意图，confidence < 0.5 时使用 route=fallback
6. reasoning_tags: 简短标注判断依据的关键词

## 注意
- context_stack 提供最近的对话摘要，可用于消解指代
- session_state 是当前的对话状态（idle/product_focus/list_browse/transaction/handoff）
- 若当前状态为 product_focus 且用户输入不含新实体词，应继承 focus 实体

## 输出格式（严格 JSON，无额外文本）
{
  "intent": "product_location",
  "rewritten_query": "可乐在哪个货架",
  "route": "tool",
  "needs_handoff": false,
  "confidence": 0.95,
  "reasoning_tags": ["产品名:可乐", "位置关键词:在哪"],
  "fallback_used": false
}
"""


class IntentAnalyzer:
    """使用 DeepSeek API 进行意图分析"""

    def __init__(self, api_key: str, base_url: str, model: str):
        self.client = AsyncOpenAI(api_key=api_key, base_url=base_url)
        self.model = model

    async def analyze(
        self,
        message: str,
        context_stack: Optional[list] = None,
        session_state: Optional[str] = None,
    ) -> dict:
        """分析用户意图，返回 Decision 字典"""
        if not message.strip():
            return _empty_decision()

        # 构建用户消息
        user_parts = [f"顾客消息: {message}"]
        if session_state:
            user_parts.append(f"当前对话状态: {session_state}")
        if context_stack:
            ctx_str = json.dumps(context_stack, ensure_ascii=False, indent=2)
            user_parts.append(f"对话历史摘要: {ctx_str}")

        user_content = "\n\n".join(user_parts)

        try:
            response = await asyncio.wait_for(
                self.client.chat.completions.create(
                    model=self.model,
                    messages=[
                        {"role": "system", "content": INTENT_SYSTEM_PROMPT},
                        {"role": "user", "content": user_content},
                    ],
                    temperature=0.1,  # 低温度以获得更稳定的分类
                    max_tokens=512,
                    response_format={"type": "json_object"},
                ),
                timeout=INTENT_LLM_TIMEOUT_SECONDS,
            )

            content = response.choices[0].message.content or "{}"
            result = json.loads(content)

            # 标准化字段
            decision = {
                "intent": result.get("intent", "unsupported"),
                "rewritten_query": result.get("rewritten_query", message),
                "route": result.get("route", _infer_route(result.get("intent", ""))),
                "needs_handoff": result.get("needs_handoff", False),
                "confidence": float(result.get("confidence", 0.5)),
                "reasoning_tags": result.get("reasoning_tags", []),
                "fallback_used": result.get("fallback_used", False),
            }
            return decision

        except asyncio.TimeoutError:
            return _fallback_decision(message)
        except Exception:
            return _fallback_decision(message)


def _infer_route(intent: str) -> str:
    """从意图推断路由"""
    if intent in ("product_location", "inventory", "price", "promotion"):
        return "tool"
    if intent == "faq":
        return "rag"
    if intent == "handoff":
        return "fallback"
    # 复合意图包含 tool 相关的，用 hybrid
    if "," in intent:
        tools = {"product_location", "inventory", "price", "promotion"}
        rags = {"faq"}
        parts = set(intent.split(","))
        if parts & tools and parts & rags:
            return "hybrid"
        if parts & tools:
            return "tool"
        if parts & rags:
            return "rag"
    return "fallback"


def _empty_decision() -> dict:
    return {
        "intent": "unsupported",
        "rewritten_query": "",
        "route": "fallback",
        "needs_handoff": False,
        "confidence": 0.0,
        "reasoning_tags": ["空输入"],
        "fallback_used": True,
    }


def _fallback_decision(message: str) -> dict:
    # 简单关键词兜底
    msg = message.lower()
    if any(kw in msg for kw in ["在哪", "哪里", "位置", "在哪儿"]):
        intent = "product_location"
    elif any(kw in msg for kw in ["还有", "库存", "几瓶", "几个"]):
        intent = "inventory"
    elif any(kw in msg for kw in ["多少钱", "价格", "报价"]):
        intent = "price"
    elif any(kw in msg for kw in ["活动", "优惠", "打折", "特价"]):
        intent = "promotion"
    elif any(kw in msg for kw in ["退款", "支付", "售后", "营业", "客服"]):
        intent = "faq"
    elif any(kw in msg for kw in ["人工", "转人工"]):
        intent = "handoff"
    else:
        intent = "unsupported"

    return {
        "intent": intent,
        "rewritten_query": message,
        "route": _infer_route(intent),
        "needs_handoff": intent == "handoff",
        "confidence": 0.6,
        "reasoning_tags": ["关键词兜底"],
        "fallback_used": True,
    }
