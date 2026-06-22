"""
离线意图评估 — 本地关键词模式 (无需 LLM API)
使用 Python Sidecar 的 _fallback_decision() 关键词规则进行评估。
这个模式不需要 DEEPSEEK_API_KEY，可以直接评估关键词兜底逻辑的准确率。

用法: python test_intent_eval_local.py
"""

import json
import os
import sys

# 直接复用 intent_analyzer.py 中的关键词兜底逻辑
INTENT_RULES = [
    (["在哪", "哪里", "位置", "在哪儿", "哪个货架", "帮我找", "哪儿有", "告诉我.*位置"], "product_location"),
    (["还有", "库存", "几瓶", "几个", "有没有", "断货", "有货"], "inventory"),
    (["多少钱", "价格", "报价", "什么价格", "贵不贵", "便宜", "卖多少", "哪个便宜"], "price"),
    (["活动", "优惠", "打折", "特价", "促销", "买二送一", "满减"], "promotion"),
    (["退款", "退货", "发票", "营业", "几点关", "几点开", "会员", "卫生间", "密码", "刷卡", "过期"], "faq"),
    (["人工", "转人工", "工作人员", "客服"], "handoff"),
    (["天气", "写文章", "电影", "唱歌", "股市"], "unsupported"),
]


def classify(message: str) -> str:
    """关键词规则分类——与 Python intent_analyzer._fallback_decision 一致"""
    msg = message.strip()
    for keywords, intent in INTENT_RULES:
        for kw in keywords:
            if kw in msg:
                return intent
    return "product_location"  # 默认：兜底到 product_location（最常见）


def is_correct(expected: str, actual: str) -> bool:
    expected_parts = set(expected.split(","))
    actual_parts = {actual}  # 关键词规则不会返回复合意图
    # 宽松匹配：actual 在 expected 中就算对
    return actual_parts.issubset(expected_parts) or expected_parts == actual_parts


def main():
    test_path = os.path.join(os.path.dirname(__file__), "test_intent_cases.json")
    with open(test_path, "r", encoding="utf-8") as f:
        suite = json.load(f)
    cases = suite["cases"]

    results = []
    for i, case in enumerate(cases, 1):
        actual = classify(case["message"])
        correct = is_correct(case["expected_intent"], actual)
        status = "✅" if correct else "❌"
        print(f"  [{i:3d}] {status} '{case['message'][:35]:35s}'  expected={case['expected_intent']:25s}  actual={actual:20s}")
        results.append({**case, "actual": actual, "correct": correct})

    correct_count = sum(1 for r in results if r["correct"])
    accuracy = correct_count / len(results) * 100

    print(f"\n{'='*60}")
    print(f"  关键词规则准确率: {accuracy:.1f}% ({correct_count}/{len(results)})")
    print(f"  注：关键词规则的准确率通常较低，LLM 可显著提升。")
    print(f"  如果 LLM 接入后准确率应 ≥ 85%。")

    # 错误详情
    wrong = [r for r in results if not r["correct"]]
    if wrong:
        print(f"\n  错误用例 ({len(wrong)}):")
        for r in wrong:
            print(f"    [{r['id']}] '{r['message']}' → actual={r['actual']}")


if __name__ == "__main__":
    main()
