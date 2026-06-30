"""
Agent 循环推理器 — POST /llm/chat
接收 Go 侧 Agent 循环的 messages + tools，归一化后透传给 DeepSeek。
"""

import json
import asyncio
import logging
from copy import deepcopy
from typing import Any

from openai import AsyncOpenAI

logger = logging.getLogger("chat_agent")

CHAT_LLM_TIMEOUT_SECONDS = 12.0


class ChatAgent:
    """Go 侧 Agent 循环的 LLM 推理桥接。

    关键职责：
    1. 归一化：Go 的 tool_calls 格式（{name, args}）→ OpenAI 标准格式
       （{type: "function", function: {name, arguments: "..."}}）
    2. reasoning_content 保留：DeepSeek v4-pro thinking 模式要求在后续
       assistant 消息中保留 reasoning_content，否则 400 错误
    3. tool 角色字段：Go 用 tool_id，OpenAI 用 tool_call_id
    """

    def __init__(self, api_key: str, base_url: str, model: str):
        self.client = AsyncOpenAI(api_key=api_key, base_url=base_url)
        self.model = model

    async def chat(
        self,
        messages: list[dict[str, Any]],
        tools: list[dict[str, Any]],
    ) -> dict[str, Any]:
        if not messages:
            return _fallback_chat()

        # ── 归一化：Go 格式 → OpenAI 标准格式 ──
        normalized = _normalize_messages(messages)

        try:
            kwargs: dict[str, Any] = {
                "model": self.model,
                "messages": normalized,
                "temperature": 0.3,
                "max_tokens": 1024,
            }

            if tools:
                kwargs["tools"] = tools
                kwargs["tool_choice"] = "auto"

            response = await asyncio.wait_for(
                self.client.chat.completions.create(**kwargs),
                timeout=CHAT_LLM_TIMEOUT_SECONDS,
            )

            choice = response.choices[0]
            msg = choice.message

            result: dict[str, Any] = {"content": msg.content or ""}

            if msg.tool_calls:
                tool_calls = []
                for tc in msg.tool_calls:
                    try:
                        args = json.loads(tc.function.arguments)
                    except (json.JSONDecodeError, TypeError):
                        args = tc.function.arguments
                    tool_calls.append({
                        "id": tc.id,
                        "name": tc.function.name,
                        "args": args,
                    })
                result["tool_calls"] = tool_calls
                logger.info("chat_agent → tool_calls: %s",
                            ", ".join(tc.function.name for tc in msg.tool_calls))
            else:
                logger.info("chat_agent → answer: %.80s", msg.content or "")

            return result

        except asyncio.TimeoutError:
            logger.error("chat_agent timeout after %.1fs", CHAT_LLM_TIMEOUT_SECONDS)
            return _fallback_chat()
        except Exception as e:
            logger.error("chat_agent LLM call failed: %s", e)
            return _fallback_chat()


# ── 归一化逻辑 ──────────────────────────────────────

def _normalize_messages(messages: list[dict[str, Any]]) -> list[dict[str, Any]]:
    """将 Go 侧的消息格式归一化为 OpenAI 标准格式。

    Go → OpenAI 映射：
      assistant.tool_calls[i].name  → tool_calls[i].function.name
      assistant.tool_calls[i].args  → tool_calls[i].function.arguments (JSON string)
      tool.tool_id                  → tool.tool_call_id
    """
    result = []
    for m in messages:
        norm = dict(m)  # shallow copy
        role = norm.get("role", "")

        if role == "assistant":
            raw_calls = norm.get("tool_calls")
            if raw_calls and isinstance(raw_calls, list):
                fixed = []
                for tc in raw_calls:
                    fn_name = tc.get("name", "")
                    fn_args = tc.get("args")
                    # args 可能是 dict 或 JSON string → 统一为 JSON string
                    if isinstance(fn_args, dict):
                        args_str = json.dumps(fn_args, ensure_ascii=False)
                    elif isinstance(fn_args, str):
                        args_str = fn_args
                    else:
                        args_str = "{}"
                    fixed.append({
                        "id": tc.get("id", ""),
                        "type": "function",
                        "function": {
                            "name": fn_name,
                            "arguments": args_str,
                        },
                    })
                norm["tool_calls"] = fixed

        elif role == "tool":
            # Go 用 tool_id，OpenAI 用 tool_call_id
            if "tool_id" in norm and "tool_call_id" not in norm:
                norm["tool_call_id"] = norm["tool_id"]

        result.append(norm)
    return result


def _fallback_chat() -> dict[str, Any]:
    return {
        "content": "暂时无法处理你的问题，你可以换个问法，或联系人工客服。",
    }
