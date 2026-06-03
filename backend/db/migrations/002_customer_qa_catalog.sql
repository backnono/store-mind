CREATE TABLE IF NOT EXISTS store (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  address VARCHAR(255) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS zone (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  store_id BIGINT NOT NULL,
  code VARCHAR(64) NOT NULL,
  name VARCHAR(255) NOT NULL,
  description VARCHAR(255) NOT NULL,
  INDEX idx_zone_store_id (store_id)
);

CREATE TABLE IF NOT EXISTS shelf (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  store_id BIGINT NOT NULL,
  zone_id BIGINT NOT NULL,
  code VARCHAR(64) NOT NULL,
  name VARCHAR(255) NOT NULL,
  description VARCHAR(255) NOT NULL,
  INDEX idx_shelf_store_id (store_id),
  INDEX idx_shelf_zone_id (zone_id)
);

CREATE TABLE IF NOT EXISTS product (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(255) NOT NULL,
  brand VARCHAR(255) NOT NULL,
  category VARCHAR(128) NOT NULL,
  aliases JSON NULL,
  tags JSON NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  INDEX idx_product_name (name),
  INDEX idx_product_category (category)
);

CREATE TABLE IF NOT EXISTS sku (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  product_id BIGINT NOT NULL,
  barcode VARCHAR(64) NOT NULL,
  spec VARCHAR(255) NOT NULL,
  price DECIMAL(10,2) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  INDEX idx_sku_product_id (product_id)
);

CREATE TABLE IF NOT EXISTS product_location (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  store_id BIGINT NOT NULL,
  product_id BIGINT NOT NULL,
  sku_id BIGINT NULL,
  zone_id BIGINT NOT NULL,
  shelf_id BIGINT NOT NULL,
  layer_no INT NOT NULL,
  position_desc VARCHAR(255) NOT NULL,
  INDEX idx_product_location_store_product (store_id, product_id),
  INDEX idx_product_location_sku_id (sku_id),
  INDEX idx_product_location_zone_id (zone_id),
  INDEX idx_product_location_shelf_id (shelf_id)
);

CREATE TABLE IF NOT EXISTS inventory (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  store_id BIGINT NOT NULL,
  sku_id BIGINT NOT NULL,
  quantity INT NOT NULL,
  safety_stock INT NOT NULL DEFAULT 0,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_inventory_store_sku (store_id, sku_id)
);

CREATE TABLE IF NOT EXISTS promotion (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  store_id BIGINT NOT NULL,
  title VARCHAR(255) NOT NULL,
  description TEXT NOT NULL,
  product_scope JSON NULL,
  start_at DATETIME NOT NULL,
  end_at DATETIME NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  INDEX idx_promotion_store_status_time (store_id, status, start_at, end_at)
);

SET @faq_keywords_exists := (
  SELECT COUNT(*)
  FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'faq' AND COLUMN_NAME = 'keywords'
);
SET @faq_keywords_sql := IF(
  @faq_keywords_exists = 0,
  'ALTER TABLE faq ADD COLUMN keywords JSON NULL AFTER category',
  'SELECT 1'
);
PREPARE faq_keywords_stmt FROM @faq_keywords_sql;
EXECUTE faq_keywords_stmt;
DEALLOCATE PREPARE faq_keywords_stmt;

SET @session_ended_exists := (
  SELECT COUNT(*)
  FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'agent_session' AND COLUMN_NAME = 'ended_at'
);
SET @session_ended_sql := IF(
  @session_ended_exists = 0,
  'ALTER TABLE agent_session ADD COLUMN ended_at DATETIME NULL AFTER started_at',
  'SELECT 1'
);
PREPARE session_ended_stmt FROM @session_ended_sql;
EXECUTE session_ended_stmt;
DEALLOCATE PREPARE session_ended_stmt;

SET @message_confidence_exists := (
  SELECT COUNT(*)
  FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'agent_message' AND COLUMN_NAME = 'confidence'
);
SET @message_confidence_sql := IF(
  @message_confidence_exists = 0,
  'ALTER TABLE agent_message ADD COLUMN confidence DECIMAL(5,4) NULL AFTER intent',
  'SELECT 1'
);
PREPARE message_confidence_stmt FROM @message_confidence_sql;
EXECUTE message_confidence_stmt;
DEALLOCATE PREPARE message_confidence_stmt;

CREATE TABLE IF NOT EXISTS agent_tool_call (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  session_id BIGINT NOT NULL,
  message_id BIGINT NOT NULL,
  tool_name VARCHAR(128) NOT NULL,
  input_json JSON NULL,
  output_json JSON NULL,
  latency_ms INT NOT NULL DEFAULT 0,
  success TINYINT(1) NOT NULL DEFAULT 1,
  error_message TEXT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_agent_tool_call_session_id (session_id),
  INDEX idx_agent_tool_call_message_id (message_id)
);
