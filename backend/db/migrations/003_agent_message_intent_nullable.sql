-- intent 由 orchestrator 统一决策，用户消息创建时可为空字符串
ALTER TABLE agent_message
  MODIFY COLUMN intent VARCHAR(64) NOT NULL DEFAULT '';
