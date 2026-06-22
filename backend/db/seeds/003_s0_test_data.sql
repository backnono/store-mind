-- ============================================================
-- S0 测试种子数据：更新库存可信度字段 + 新增促销活动 + 补全 FAQ
-- ============================================================

-- 1. 更新 inventory 表的 last_verified_at 和 update_source（四级可信度数据）
--    🟢 高可信 (≤30min)：刚刚人工盘点
--    🟡 中可信 (≤2h)：2 小时前支付扣减
--    🟠 低可信 (≤24h)：今天早上盘点
--    🔴 仅供参考 (>24h)：几天前手动盘点
UPDATE inventory SET
  last_verified_at = CASE
    WHEN sku_id = 1001 THEN NOW() - INTERVAL 10 MINUTE    -- 🟢 10 分前盘点，高可信
    WHEN sku_id = 1002 THEN NOW() - INTERVAL 25 MINUTE    -- 🟢 25 分前，高可信
    WHEN sku_id = 1003 THEN NOW() - INTERVAL 1 HOUR       -- 🟡 1 小时前，中可信
    WHEN sku_id = 1004 THEN NOW() - INTERVAL 90 MINUTE    -- 🟡 1.5 小时前，中可信
    WHEN sku_id = 1005 THEN NOW() - INTERVAL 3 HOUR       -- 🟠 3 小时前，低可信
    WHEN sku_id = 1006 THEN NOW() - INTERVAL 5 HOUR       -- 🟠 5 小时前，低可信
    WHEN sku_id = 1007 THEN NOW() - INTERVAL 8 HOUR       -- 🟠 8 小时前，低可信
    WHEN sku_id = 1008 THEN NOW() - INTERVAL 30 HOUR      -- 🔴 30 小时前，仅供参考
    WHEN sku_id = 1009 THEN NOW() - INTERVAL 2 HOUR       -- 🟡 2 小时前，中可信
    WHEN sku_id = 1010 THEN NOW() - INTERVAL 48 HOUR      -- 🔴 48 小时前，仅供参考
  END,
  update_source = CASE
    WHEN sku_id IN (1001, 1002, 1003, 1004) THEN 'manual_count'
    WHEN sku_id IN (1005, 1006, 1007, 1008) THEN 'payment_deduct'
    WHEN sku_id = 1009 THEN 'manual_count'
    WHEN sku_id = 1010 THEN 'payment_deduct'
  END
WHERE store_id = 1 AND sku_id IN (1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010);

-- 2. 新增一个缺货/低库存商品用于测试缺货引导
INSERT INTO product (id, name, brand, category, aliases, tags, status)
VALUES
  (111, '脉动青柠味', '达能', '饮料', JSON_ARRAY('维生素饮料', '功能饮料'), JSON_ARRAY('运动饮料'), 'active'),
  (112, '德芙牛奶巧克力', '德芙', '零食', JSON_ARRAY('巧克力', '糖果'), JSON_ARRAY('糖果'), 'active'),
  (113, '椰树椰汁', '椰树', '饮料', JSON_ARRAY('椰子水', '椰汁'), JSON_ARRAY('植物饮料'), 'active')
ON DUPLICATE KEY UPDATE
  name = VALUES(name), brand = VALUES(brand), category = VALUES(category),
  aliases = VALUES(aliases), tags = VALUES(tags), status = VALUES(status);

INSERT INTO sku (id, product_id, barcode, spec, price, status)
VALUES
  (1011, 111, '690000000011', '600ml', 4.50, 'active'),
  (1012, 112, '690000000012', '43g', 9.00, 'active'),
  (1013, 113, '690000000013', '245ml', 5.50, 'active')
ON DUPLICATE KEY UPDATE
  product_id = VALUES(product_id), barcode = VALUES(barcode),
  spec = VALUES(spec), price = VALUES(price), status = VALUES(status);

INSERT INTO product_location (id, store_id, product_id, sku_id, zone_id, shelf_id, layer_no, position_desc)
VALUES
  (11, 1, 111, 1011, 2, 3, 3, '冷藏柜旁上层'),
  (12, 1, 112, 1012, 1, 1, 2, '入口旁第二排'),
  (13, 1, 113, 1013, 2, 5, 2, '茶饮区旁边')
ON DUPLICATE KEY UPDATE
  store_id = VALUES(store_id), product_id = VALUES(product_id),
  sku_id = VALUES(sku_id), zone_id = VALUES(zone_id),
  shelf_id = VALUES(shelf_id), layer_no = VALUES(layer_no),
  position_desc = VALUES(position_desc);

-- 111 脉动：低库存（只剩 1 瓶）
-- 112 德芙：库存 0（缺货状态）
-- 113 椰树椰汁：库存充足
INSERT INTO inventory (id, store_id, sku_id, quantity, safety_stock, last_verified_at, update_source, updated_at)
VALUES
  (11, 1, 1011, 1, 3, NOW() - INTERVAL 5 MINUTE, 'manual_count', NOW()),
  (12, 1, 1012, 0, 4, NOW() - INTERVAL 10 MINUTE, 'payment_deduct', NOW()),
  (13, 1, 1013, 20, 5, NOW() - INTERVAL 30 MINUTE, 'manual_count', NOW())
ON DUPLICATE KEY UPDATE
  quantity = VALUES(quantity), safety_stock = VALUES(safety_stock),
  last_verified_at = VALUES(last_verified_at),
  update_source = VALUES(update_source),
  updated_at = VALUES(updated_at);

-- 3. 更新促销活动时间（使所有活动当前有效）
UPDATE promotion SET
  start_at = NOW() - INTERVAL 3 DAY,
  end_at = NOW() + INTERVAL 3 DAY,
  status = 'active'
WHERE store_id = 1;

-- 新增一条缺货/替代测试用促销
INSERT INTO promotion (id, store_id, title, description, product_scope, start_at, end_at, status)
VALUES
  (4, 1, '巧克力买二送一', '德芙、好时等巧克力品牌买二送一。', JSON_ARRAY('112'), NOW() - INTERVAL 2 DAY, NOW() + INTERVAL 5 DAY, 'active')
ON DUPLICATE KEY UPDATE
  store_id = VALUES(store_id), title = VALUES(title), description = VALUES(description),
  product_scope = VALUES(product_scope), start_at = VALUES(start_at),
  end_at = VALUES(end_at), status = VALUES(status);

-- 4. 补全 FAQ（覆盖更多常见问题）
INSERT INTO faq (store_id, question, answer, category, status) VALUES
(1, '怎么退货？', '在订单详情页点击"申请退款"，填写退款原因并上传商品照片，客服将在 24 小时内处理。', 'refund', 'active'),
(1, '什么时候开门？', '本店营业时间为每天 08:00-23:00，节假日正常营业。', 'store_hours', 'active'),
(1, '能刷卡吗？', '目前支持微信和支付宝扫码支付，暂不支持银行卡刷卡。', 'payment', 'active'),
(1, '忘记密码了怎么办？', '在小程序登录页面点击"忘记密码"，通过手机号验证后即可重置密码。', 'customer_service', 'active'),
(1, '有没有会员卡？', '在小程序个人中心即可开通会员，享受积分和专属折扣。', 'customer_service', 'active'),
(1, '卫生间在哪里？', '卫生间在收银区旁边，请从小程序扫码开门进入。', 'store_hours', 'active'),
(1, '可以开发票吗？', '在订单详情页点击"开发票"，填写抬头信息后，电子发票将在 48 小时内发送到您的邮箱。', 'payment', 'active'),
(1, '商品过期了怎么办？', '如果在店内发现过期商品，请联系客服处理，我们将全额退款并赠送小礼品。', 'refund', 'active')
ON DUPLICATE KEY UPDATE
  answer = VALUES(answer),
  category = VALUES(category),
  status = VALUES(status);
