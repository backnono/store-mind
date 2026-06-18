-- S0 Schema 变更: 基础设施就绪
-- 1. inventory 表新增 last_verified_at + update_source
-- 2. agent_message 表新增 context_state + focus_entity_ids + context_stack
-- 3. 新建 agent_feedback 表
-- 4. 新建 agent_decision_log 表

-- ========== inventory: 新增可信度字段 ==========
-- last_verified_at: 最后盘点/核实时间，用于计算库存可信度
SET @inv_la_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'inventory' AND COLUMN_NAME = 'last_verified_at'
);
SET @inv_la_sql := IF(
  @inv_la_exists = 0,
  'ALTER TABLE inventory ADD COLUMN last_verified_at DATETIME NULL AFTER safety_stock',
  'SELECT 1'
);
PREPARE inv_la_stmt FROM @inv_la_sql;
EXECUTE inv_la_stmt;
DEALLOCATE PREPARE inv_la_stmt;

-- update_source: 更新来源 (manual_count / payment_deduct / refund_add)
SET @inv_us_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'inventory' AND COLUMN_NAME = 'update_source'
);
SET @inv_us_sql := IF(
  @inv_us_exists = 0,
  'ALTER TABLE inventory ADD COLUMN update_source VARCHAR(32) NULL AFTER last_verified_at',
  'SELECT 1'
);
PREPARE inv_us_stmt FROM @inv_us_sql;
EXECUTE inv_us_stmt;
DEALLOCATE PREPARE inv_us_stmt;

-- ========== agent_message: 新增多轮对话上下文字段 ==========
-- context_state: 当前对话状态 (idle / product_focus / list_browse / transaction / handoff)
SET @msg_cs_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'agent_message' AND COLUMN_NAME = 'context_state'
);
SET @msg_cs_sql := IF(
  @msg_cs_exists = 0,
  'ALTER TABLE agent_message ADD COLUMN context_state VARCHAR(32) NULL AFTER confidence',
  'SELECT 1'
);
PREPARE msg_cs_stmt FROM @msg_cs_sql;
EXECUTE msg_cs_stmt;
DEALLOCATE PREPARE msg_cs_stmt;

-- focus_entity_ids: 当前锁定的实体 ID (JSON, e.g. {"product_ids": [1,2]})
SET @msg_fe_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'agent_message' AND COLUMN_NAME = 'focus_entity_ids'
);
SET @msg_fe_sql := IF(
  @msg_fe_exists = 0,
  'ALTER TABLE agent_message ADD COLUMN focus_entity_ids JSON NULL AFTER context_state',
  'SELECT 1'
);
PREPARE msg_fe_stmt FROM @msg_fe_sql;
EXECUTE msg_fe_stmt;
DEALLOCATE PREPARE msg_fe_stmt;

-- context_stack: 最近 N 轮对话的结构化摘要 (JSON 数组)
SET @msg_css_exists := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'agent_message' AND COLUMN_NAME = 'context_stack'
);
SET @msg_css_sql := IF(
  @msg_css_exists = 0,
  'ALTER TABLE agent_message ADD COLUMN context_stack JSON NULL AFTER focus_entity_ids',
  'SELECT 1'
);
PREPARE msg_css_stmt FROM @msg_css_sql;
EXECUTE msg_css_stmt;
DEALLOCATE PREPARE msg_css_stmt;

-- ========== 新建 agent_feedback 表 ==========
CREATE TABLE IF NOT EXISTS agent_feedback (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  message_id BIGINT NOT NULL,
  session_id BIGINT NOT NULL,
  feedback_value TINYINT NOT NULL COMMENT '1=👍 / 0=👎',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_agent_feedback_message_id (message_id),
  INDEX idx_agent_feedback_session_id (session_id)
);

-- ========== 新建 agent_decision_log 表 ==========
CREATE TABLE IF NOT EXISTS agent_decision_log (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  session_id BIGINT NOT NULL,
  message_id BIGINT NOT NULL,
  intent VARCHAR(64) NOT NULL,
  route VARCHAR(32) NOT NULL,
  rewrite_query VARCHAR(512) NULL,
  confidence DECIMAL(5,4) NOT NULL DEFAULT 0,
  fallback_used TINYINT(1) NOT NULL DEFAULT 0,
  handoff_required TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_agent_decision_log_session_id (session_id),
  INDEX idx_agent_decision_log_message_id (message_id)
);
