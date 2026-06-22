"""
离线意图识别评估脚本
用法：
  # 1. 确保 LLM Sidecar 已启动 (python server.py)
  # 2. 运行评估
  python test_intent_eval.py

环境变量：
  DEEPSEEK_API_KEY   DeepSeek API Key
  AGENT_PORT         LLM Sidecar 端口（默认 9090）
"""

import json
import sys
import os
import time
import asyncio
from collections import defaultdict

import httpx

# ── 配置 ────────────────────────────────────────────
AGENT_PORT = int(os.getenv("AGENT_PORT", "9090"))
BASE_URL = f"http://127.0.0.1:{AGENT_PORT}"

# 加载测试集
TEST_CASES_PATH = os.path.join(os.path.dirname(__file__), "test_intent_cases.json")

# ── 判断标准 ─────────────────────────────────────────
# 允许的等价意图映射：某些意图即使名称不完全一样也算正确
EQUIVALENT_INTENTS = {
    "product_location": {"product_location"},
    "inventory": {"inventory"},
    "price": {"price"},
    "promotion": {"promotion"},
    "faq": {"faq"},
    "handoff": {"handoff"},
    "unsupported": {"unsupported"},
}


def is_correct(expected: str, actual: str) -> bool:
    """判断实际意图是否匹配预期意图。
    复合意图（如 "product_location,inventory"）需同时包含所有子意图。
    """
    expected_parts = set(expected.split(","))
    actual_parts = set(actual.split(","))

    # 精确匹配
    if expected_parts == actual_parts:
        return True

    # 对复合意图（expected 是复合），检查 actual 是否至少包含所有 expected 子意图
    if len(expected_parts) > 1:
        return expected_parts.issubset(actual_parts)

    # 单意图时检查等价类
    for eq_class in EQUIVALENT_INTENTS.values():
        if expected in eq_class and actual in eq_class:
            return True

    return False


async def evaluate_one(client: httpx.AsyncClient, case: dict, idx: int, total: int) -> dict:
    """评估单条用例"""
    try:
        resp = await client.post(
            f"{BASE_URL}/llm/intent",
            json={
                "message": case["message"],
                "context_stack": None,
                "session_state": None,
            },
            timeout=8.0,
        )
        data = resp.json()
        actual_intent = data.get("intent", "unsupported")
        confidence = data.get("confidence", 0)
        correct = is_correct(case["expected_intent"], actual_intent)
        status = "✅" if correct else "❌"
        print(f"  [{idx:3d}/{total}] {status} '{case['message'][:30]:30s}'  expected={case['expected_intent']:30s}  actual={actual_intent:30s}  conf={confidence:.2f}")
        return {
            "id": case["id"],
            "message": case["message"],
            "expected": case["expected_intent"],
            "actual": actual_intent,
            "confidence": confidence,
            "correct": correct,
            "note": case.get("note", ""),
        }
    except Exception as e:
        print(f"  [{idx:3d}/{total}] ❌ '{case['message'][:30]:30s}'  ERROR: {e}")
        return {
            "id": case["id"],
            "message": case["message"],
            "expected": case["expected_intent"],
            "actual": "ERROR",
            "confidence": 0,
            "correct": False,
            "error": str(e),
            "note": case.get("note", ""),
        }


async def main():
    # 加载测试集
    with open(TEST_CASES_PATH, "r", encoding="utf-8") as f:
        suite = json.load(f)
    cases = suite["cases"]

    print("=" * 60)
    print(f" 小王小王 离线意图评估")
    print(f" 测试集: {TEST_CASES_PATH}")
    print(f" 总用例数: {len(cases)}")
    print(f" LLM Sidecar: {BASE_URL}")
    print("=" * 60)

    # 检查 sidecar 健康
    async with httpx.AsyncClient() as client:
        try:
            health = await client.get(f"{BASE_URL}/health", timeout=5)
            print(f" Sidecar 状态: {health.json()}")
        except Exception as e:
            print(f" ❌ 无法连接 LLM Sidecar ({BASE_URL}): {e}")
            print("   请先启动: cd services/agent && python server.py")
            print("   (或设置 DEEPSEEK_API_KEY 为 'mock' 进行离线关键词评估)")
            return 1

        print()
        print("开始评估...\n")
        start_time = time.monotonic()

        results = []
        for i, case in enumerate(cases, 1):
            result = await evaluate_one(client, case, i, len(cases))
            results.append(result)

        elapsed = time.monotonic() - start_time

    # ── 统计 ──────────────────────────────────────
    correct = [r for r in results if r["correct"]]
    wrong = [r for r in results if not r["correct"]]
    accuracy = len(correct) / len(results) * 100 if results else 0

    # 按意图分类统计
    by_intent = defaultdict(lambda: {"total": 0, "correct": 0})
    for r in results:
        for intent in r["expected"].split(","):
            by_intent[intent]["total"] += 1.0 / len(r["expected"].split(","))  # 复合意图按比例分配
        if r["correct"]:
            for intent in r["expected"].split(","):
                by_intent[intent]["correct"] += 1.0 / len(r["expected"].split(","))

    print()
    print("=" * 60)
    print("  评估结果")
    print("=" * 60)
    print(f"  总 用 例: {len(results)}")
    print(f"  正    确: {len(correct)}")
    print(f"  错    误: {len(wrong)}")
    print(f"  准 确 率: {accuracy:.1f}%")
    print(f"  总 耗 时: {elapsed:.1f}s ({elapsed/len(results):.2f}s/条)")
    print(f"  验收标准: ≥ 85% {'✅ 达标' if accuracy >= 85 else '❌ 未达标'}")
    print()

    # 按意图分类准确率
    print("  按意图分类准确率:")
    intent_order = ["product_location", "inventory", "price", "promotion", "faq", "handoff", "unsupported"]
    for intent in intent_order:
        stats = by_intent.get(intent)
        if stats and stats["total"] > 0:
            acc = stats["correct"] / stats["total"] * 100
            bar = "█" * int(acc / 5) + "░" * (20 - int(acc / 5))
            print(f"    {intent:25s} {acc:5.1f}%  {bar}  ({stats['correct']:.0f}/{stats['total']:.0f})")

    # 错误用例详情
    if wrong:
        print()
        print("  ❌ 错误用例详情:")
        for r in wrong:
            print(f"    [{r['id']}] '{r['message']}'")
            print(f"         预期: {r['expected']}  实际: {r['actual']}  (confidence={r['confidence']:.2f})  {r.get('note','')}")

    # 保存完整结果
    report_path = os.path.join(os.path.dirname(__file__), "test_intent_report.json")
    with open(report_path, "w", encoding="utf-8") as f:
        json.dump({
            "accuracy": accuracy,
            "total": len(results),
            "correct": len(correct),
            "wrong": len(wrong),
            "elapsed_s": round(elapsed, 2),
            "by_intent": {k: {"accuracy": round(v["correct"]/v["total"]*100, 1) if v["total"]>0 else 0, **v} for k, v in by_intent.items()},
            "results": results,
        }, f, ensure_ascii=False, indent=2)
    print(f"\n  详细报告已保存: {report_path}")
    return 0 if accuracy >= 85 else 1


if __name__ == "__main__":
    sys.exit(asyncio.run(main()))
