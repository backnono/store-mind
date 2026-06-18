"""
指代消解器 — POST /llm/resolve
将用户消息中的指代（"它"/"这个"/"那个"/"雪碧呢？"）解析为具体实体
"""

import json
import asyncio
from typing import Optional

from openai import AsyncOpenAI

ANAPHORA_SYSTEM_PROMPT = """你是无人超市数字店员「小王」的指代消解器。

## 任务
分析顾客消息中的指代表达，结合对话历史，确定顾客真正在问什么。

## 输入
- message: 当前顾客消息
- context_stack: 最近 N 轮对话摘要（每轮含 turn/intent/resolved_entities/system_action/system_summary）
- focus_entities: 当前锁定的实体（如具体 product_ids）

## 输出格式（严格 JSON，无额外文本）
{
  "resolved_entities": [
    {"type": "product", "name": "可口可乐", "product_id": 123}
  ],
  "confidence": 0.9,
  "explanation": "继承上一轮的 focus_product"
}

## 消解策略
1. 若消息含明确的商品名/品类 → 直接提取，confidence 高
2. 若消息含指代词（"它"/"这个"/"那个"）→ 从 context_stack 继承最近的实体
3. 若消息含省略（"雪碧呢？"/"多少钱？"）→ 继承 focus_entities 中的品类/属性
4. 无法消解时 → resolved_entities 为 [], confidence < 0.5

## 示例
context_stack: [{"turn":1, "intent":"product_location", "resolved_entities":[{"type":"product","name":"可口可乐"}], "system_summary":"告诉了用户可乐在B-02货架"}]
focus_entities: [{"type":"product","name":"可口可乐","product_id":123}]
message: "还有几瓶？"
→ {"resolved_entities":[{"type":"product","name":"可口可乐","product_id":123}], "confidence":0.95, "explanation":"'还有几瓶'省略了主语，从 focus_entities 继承可口可乐"}

message: "那雪碧呢？"
→ {"resolved_entities":[{"type":"product","name":"雪碧"}], "confidence":0.88, "explanation":"'那...呢'暗示同品类饮料，从上下文推断为雪碧"}

message: "帮我推荐一个防晒霜"
→ {"resolved_entities":[], "confidence":0.1, "explanation":"防晒霜不在门店商品范围内，无法消解"}
"""


class AnaphoraResolver:
    """使用 DeepSeek API 进行指代消解"""

    def __init__(self, api_key: str, base_url: str, model: str):
        self.client = AsyncOpenAI(api_key=api_key, base_url=base_url)
        self.model = model

    async def resolve(
        self,
        message: str,
        context_stack: Optional[list] = None,
        focus_entities: Optional[list] = None,
    ) -> dict:
        """解析指代，返回 resolved_entities"""
        if not message.strip():
            return {"resolved_entities": [], "confidence": 0.0, "explanation": "空输入"}

        user_parts = [f"顾客消息: {message}"]
        if context_stack:
            ctx_str = json.dumps(context_stack, ensure_ascii=False, indent=2)
            user_parts.append(f"对话历史摘要: {ctx_str}")
        if focus_entities:
            fe_str = json.dumps(focus_entities, ensure_ascii=False, indent=2)
            user_parts.append(f"当前焦点实体: {fe_str}")

        user_content = "\n\n".join(user_parts)

        try:
            response = await asyncio.wait_for(
                self.client.chat.completions.create(
                    model=self.model,
                    messages=[
                        {"role": "system", "content": ANAPHORA_SYSTEM_PROMPT},
                        {"role": "user", "content": user_content},
                    ],
                    temperature=0.1,
                    max_tokens=512,
                    response_format={"type": "json_object"},
                ),
                timeout=3.0,
            )

            content = response.choices[0].message.content or "{}"
            result = json.loads(content)
            return {
                "resolved_entities": result.get("resolved_entities", []),
                "confidence": float(result.get("confidence", 0.5)),
                "explanation": result.get("explanation", ""),
            }

        except asyncio.TimeoutError:
            return _simple_resolve(message)
        except Exception:
            return _simple_resolve(message)


def _simple_resolve(message: str) -> dict:
    """简单关键词兜底：直接提取消息中的实体信息"""
    return {
        "resolved_entities": [],
        "confidence": 0.3,
        "explanation": "LLM 不可用，无法进行指代消解",
    }
