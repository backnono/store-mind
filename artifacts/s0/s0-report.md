# S0 Acceptance Report

- Mode: `local`
- Result: **PASS**
- Checks: 18/18 passed

| Gate | Check | Result | Detail |
|---|---|---|---|
| environment | Go backend health | PASS | HTTP 200: {'status': 'ok'} |
| environment | Python sidecar health | PASS | HTTP 200: {'status': 'ok', 'model': 'deepseek-v4-pro'} |
| environment | MySQL connectivity | PASS | SELECT 1 |
| schema | inventory | PASS | all required columns present |
| schema | agent_message | PASS | all required columns present |
| schema | agent_feedback | PASS | all required columns present |
| schema | agent_decision_log | PASS | all required columns present |
| schema | legacy catalog data preserved | PASS | known products present=3/3 |
| sidecar-contract | /llm/intent | PASS | HTTP 200, missing=[], payload={'intent': 'product_location', 'rewritten_query': '可乐在哪里', 'route': 'tool', 'needs_handoff': False, 'confidence': 0.95, 'reasoning_tags': ['产品名:可乐', '位置关键词:在哪里'], 'fallback_used': False} |
| sidecar-contract | /llm/answer | PASS | HTTP 200, answer=可口可乐在饮料区 B-02 货架第2层。 |
| sidecar-contract | /llm/resolve | PASS | HTTP 200, keys=['confidence', 'explanation', 'resolved_entities'] |
| intent-quality | 56-case LLM accuracy | PASS | 85.7% (48/56), required >= 85.0%, request_errors=0 |
| e2e | intent → tool → answer | PASS | evidence_count=1, cards=1 |
| no-fabrication | missing product has no invented facts | PASS | 暂时没有找到可靠依据回答这个问题，你可以换个问法，或联系人工客服。 |
| persistence | messages and decision log are linked | PASS | user_message=10, assistant_message=11, decision=['product_location', 'tool', '0.9500', '0'] |
| feedback | thumbs up persists | PASS | HTTP 200, rows 0→1, payload={'status': 'ok'} |
| feedback | thumbs down persists | PASS | HTTP 200, rows 0→1, payload={'status': 'ok'} |
| feedback | invalid value rejected | PASS | HTTP 400: {'code': 'bad_request', 'message': 'feedback_value must be 0 (👎) or 1 (👍)'} |
