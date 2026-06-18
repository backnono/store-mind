"""
小王 · 数字店员 — Python LLM Sidecar
FastAPI 服务，监听 localhost:9090
提供 3 个端点：意图分析、答案生成、指代消解
"""

import os
import logging
import time
from contextlib import asynccontextmanager

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
import uvicorn

from intent_analyzer import IntentAnalyzer
from answer_composer import AnswerComposer
from anaphora_resolver import AnaphoraResolver

# ── 配置 ────────────────────────────────────────────
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger("agent")

# DeepSeek 配置
DEEPSEEK_API_KEY = os.getenv("DEEPSEEK_API_KEY", "")
DEEPSEEK_BASE_URL = os.getenv("DEEPSEEK_BASE_URL", "https://api.deepseek.com/v1")
DEEPSEEK_MODEL = os.getenv("DEEPSEEK_MODEL", "deepseek-chat")

# ── 全局组件 ─────────────────────────────────────────
intent_analyzer: IntentAnalyzer
answer_composer: AnswerComposer
anaphora_resolver: AnaphoraResolver


@asynccontextmanager
async def lifespan(app: FastAPI):
    global intent_analyzer, answer_composer, anaphora_resolver
    logger.info("Initializing LLM components...")

    intent_analyzer = IntentAnalyzer(
        api_key=DEEPSEEK_API_KEY,
        base_url=DEEPSEEK_BASE_URL,
        model=DEEPSEEK_MODEL,
    )
    answer_composer = AnswerComposer(
        api_key=DEEPSEEK_API_KEY,
        base_url=DEEPSEEK_BASE_URL,
        model=DEEPSEEK_MODEL,
    )
    anaphora_resolver = AnaphoraResolver(
        api_key=DEEPSEEK_API_KEY,
        base_url=DEEPSEEK_BASE_URL,
        model=DEEPSEEK_MODEL,
    )

    if not DEEPSEEK_API_KEY:
        logger.warning(
            "DEEPSEEK_API_KEY is not set! LLM endpoints will fail. "
            "Set the environment variable before using."
        )

    logger.info(f"LLM Sidecar ready. model={DEEPSEEK_MODEL}")
    yield


app = FastAPI(
    title="小王 LLM Sidecar",
    version="0.1.0",
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)


# ── 端点 ─────────────────────────────────────────────


@app.post("/llm/intent")
async def analyze_intent(req: dict):
    """
    意图识别
    输入: { message, context_stack?, session_state? }
    输出: Decision { intent, route, confidence, rewritten_query, needs_handoff, reasoning_tags, fallback_used }
    超时: 3s
    """
    start = time.monotonic()
    try:
        result = await intent_analyzer.analyze(
            message=req.get("message", ""),
            context_stack=req.get("context_stack"),
            session_state=req.get("session_state"),
        )
        elapsed = time.monotonic() - start
        logger.info(f"intent analysis: {result.get('intent')} (confidence={result.get('confidence')}, {elapsed:.2f}s)")
        return result
    except Exception as e:
        elapsed = time.monotonic() - start
        logger.error(f"intent analysis failed ({elapsed:.2f}s): {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/llm/answer")
async def compose_answer(req: dict):
    """
    答案生成
    输入: { decision, message, evidence, cred_level? }
    输出: { answer, guidance_chips }
    超时: 5s
    """
    start = time.monotonic()
    try:
        result = await answer_composer.compose(
            decision=req.get("decision", {}),
            message=req.get("message", ""),
            evidence=req.get("evidence", []),
            cred_level=req.get("cred_level"),
        )
        elapsed = time.monotonic() - start
        answer_preview = result.get("answer", "")[:80]
        chips_count = len(result.get("guidance_chips", []))
        logger.info(f"answer composed ({elapsed:.2f}s): {answer_preview}... (chips={chips_count})")
        return result
    except Exception as e:
        elapsed = time.monotonic() - start
        logger.error(f"answer composition failed ({elapsed:.2f}s): {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/llm/resolve")
async def resolve_anaphora(req: dict):
    """
    指代消解
    输入: { message, context_stack, focus_entities? }
    输出: { resolved_entities, confidence }
    超时: 3s
    """
    start = time.monotonic()
    try:
        result = await anaphora_resolver.resolve(
            message=req.get("message", ""),
            context_stack=req.get("context_stack"),
            focus_entities=req.get("focus_entities"),
        )
        elapsed = time.monotonic() - start
        resolved = result.get("resolved_entities", [])
        logger.info(f"anaphora resolved ({elapsed:.2f}s): {resolved}")
        return result
    except Exception as e:
        elapsed = time.monotonic() - start
        logger.error(f"anaphora resolution failed ({elapsed:.2f}s): {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/health")
async def health():
    return {"status": "ok", "model": DEEPSEEK_MODEL}


# ── 入口 ─────────────────────────────────────────────
if __name__ == "__main__":
    port = int(os.getenv("AGENT_PORT", "9090"))
    logger.info(f"Starting LLM Sidecar on port {port}...")
    uvicorn.run(app, host="127.0.0.1", port=port, log_level="info")
