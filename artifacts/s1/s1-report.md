# S1 Acceptance Report

- Mode: `local`
- Result: **FAIL**
- Checks: 12/13 passed

| Gate | Check | Result | Detail |
|---|---|---|---|
| environment | Go backend health | PASS | HTTP 200: {'status': 'ok'} |
| environment | Python sidecar health | PASS | HTTP 200: {'status': 'ok', 'model': 'deepseek-v4-pro'} |
| environment | MySQL connectivity | PASS | SELECT 1 |
| entry | first_open returns preset questions | PASS | chips=4, location=True, promo=True |
| entry | zone_scan returns shelf products | PASS | cards=2 |
| guidance | location answer has guidance chips | PASS | chips=2, inventory=True, promo=False |
| credibility | inventory answer has credibility level | FAIL | no credibility marker in answer and no inventory card |
| sidecar-resolve | /llm/resolve returns resolved_entities | PASS | entities=1, confidence=0.95 |
| sidecar-resolve | Go client routes resolved_entities to orchestrator | PASS | R2_intent=clarify, db_focus=True |
| multi-turn | follow-up '还有几瓶/多少钱' inherits product | PASS | R2_intent=clarify, R3_intent=clarify |
| persistence | context_state persisted | PASS | message_id=41, context_state=product_focus |
| persistence | focus_entity_ids persisted | PASS | message_id=41, focus_entity_ids=NULL |
| persistence | context_stack persisted | PASS | message_id=41, context_stack=[{"turn": 1, "intent": "product_location", "system_action": "tool", "system_summ |
