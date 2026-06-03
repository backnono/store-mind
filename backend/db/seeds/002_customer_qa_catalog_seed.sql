INSERT INTO store (id, name, address, status)
VALUES
  (1, '演示无人超市', '上海市浦东新区示例路 100 号', 'active')
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  address = VALUES(address),
  status = VALUES(status);

INSERT INTO zone (id, store_id, code, name, description)
VALUES
  (1, 1, 'A', '零食区', '进门后右手边'),
  (2, 1, 'B', '饮料区', '进门后左手边'),
  (3, 1, 'C', '日用品区', '收银区旁')
ON DUPLICATE KEY UPDATE
  store_id = VALUES(store_id),
  code = VALUES(code),
  name = VALUES(name),
  description = VALUES(description);

INSERT INTO shelf (id, store_id, zone_id, code, name, description)
VALUES
  (1, 1, 1, 'A-01', '零食货架 1', '靠近入口'),
  (2, 1, 1, 'A-03', '零食货架 3', '零食区中段'),
  (3, 1, 2, 'B-01', '饮料货架 1', '冷藏柜旁'),
  (4, 1, 2, 'B-02', '饮料货架 2', '饮料区中段'),
  (5, 1, 2, 'B-03', '饮料货架 3', '饮料区最内侧'),
  (6, 1, 3, 'C-01', '日用品货架 1', '靠近客服屏')
ON DUPLICATE KEY UPDATE
  store_id = VALUES(store_id),
  zone_id = VALUES(zone_id),
  code = VALUES(code),
  name = VALUES(name),
  description = VALUES(description);

INSERT INTO product (id, name, brand, category, aliases, tags, status)
VALUES
  (101, '可口可乐', '可口可乐', '饮料', JSON_ARRAY('可乐', '汽水'), JSON_ARRAY('含糖', '碳酸'), 'active'),
  (102, '无糖可乐', '可口可乐', '饮料', JSON_ARRAY('零度可乐', '无糖汽水'), JSON_ARRAY('无糖', '碳酸'), 'active'),
  (103, '雪碧', '可口可乐', '饮料', JSON_ARRAY('柠檬汽水'), JSON_ARRAY('碳酸'), 'active'),
  (104, '元气森林白桃味', '元气森林', '饮料', JSON_ARRAY('白桃气泡水'), JSON_ARRAY('低糖', '气泡水'), 'active'),
  (105, '东方树叶乌龙茶', '农夫山泉', '饮料', JSON_ARRAY('乌龙茶'), JSON_ARRAY('无糖', '茶饮'), 'active'),
  (106, '乐事原味薯片', '乐事', '零食', JSON_ARRAY('薯片'), JSON_ARRAY('膨化食品'), 'active'),
  (107, '奥利奥夹心饼干', '奥利奥', '零食', JSON_ARRAY('夹心饼干'), JSON_ARRAY('饼干'), 'active'),
  (108, '维达抽纸', '维达', '日用品', JSON_ARRAY('纸巾', '抽纸'), JSON_ARRAY('家清'), 'active'),
  (109, '农夫山泉矿泉水', '农夫山泉', '饮料', JSON_ARRAY('矿泉水'), JSON_ARRAY('水'), 'active'),
  (110, '统一冰红茶', '统一', '饮料', JSON_ARRAY('红茶饮料'), JSON_ARRAY('茶饮'), 'active')
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  brand = VALUES(brand),
  category = VALUES(category),
  aliases = VALUES(aliases),
  tags = VALUES(tags),
  status = VALUES(status);

INSERT INTO sku (id, product_id, barcode, spec, price, status)
VALUES
  (1001, 101, '690000000001', '500ml', 3.50, 'active'),
  (1002, 102, '690000000002', '500ml', 3.80, 'active'),
  (1003, 103, '690000000003', '500ml', 3.50, 'active'),
  (1004, 104, '690000000004', '480ml', 5.00, 'active'),
  (1005, 105, '690000000005', '500ml', 5.50, 'active'),
  (1006, 106, '690000000006', '70g', 6.00, 'active'),
  (1007, 107, '690000000007', '97g', 7.50, 'active'),
  (1008, 108, '690000000008', '3 层 100 抽', 12.90, 'active'),
  (1009, 109, '690000000009', '550ml', 2.00, 'active'),
  (1010, 110, '690000000010', '500ml', 4.00, 'active')
ON DUPLICATE KEY UPDATE
  product_id = VALUES(product_id),
  barcode = VALUES(barcode),
  spec = VALUES(spec),
  price = VALUES(price),
  status = VALUES(status);

INSERT INTO product_location (id, store_id, product_id, sku_id, zone_id, shelf_id, layer_no, position_desc)
VALUES
  (1, 1, 101, 1001, 2, 4, 2, '进门后左手边中段'),
  (2, 1, 102, 1002, 2, 4, 3, '可口可乐旁边'),
  (3, 1, 103, 1003, 2, 4, 2, '可乐右侧'),
  (4, 1, 104, 1004, 2, 5, 2, '饮料区最内侧中层'),
  (5, 1, 105, 1005, 2, 5, 3, '靠近矿泉水'),
  (6, 1, 106, 1006, 1, 2, 2, '零食区中段'),
  (7, 1, 107, 1007, 1, 1, 3, '入口旁第一排'),
  (8, 1, 108, 1008, 3, 6, 2, '客服屏旁'),
  (9, 1, 109, 1009, 2, 3, 1, '冷藏柜旁底层'),
  (10, 1, 110, 1010, 2, 5, 1, '茶饮区底层')
ON DUPLICATE KEY UPDATE
  store_id = VALUES(store_id),
  product_id = VALUES(product_id),
  sku_id = VALUES(sku_id),
  zone_id = VALUES(zone_id),
  shelf_id = VALUES(shelf_id),
  layer_no = VALUES(layer_no),
  position_desc = VALUES(position_desc);

INSERT INTO inventory (id, store_id, sku_id, quantity, safety_stock, updated_at)
VALUES
  (1, 1, 1001, 12, 4, '2026-05-28 10:00:00'),
  (2, 1, 1002, 8, 4, '2026-05-28 10:00:00'),
  (3, 1, 1003, 9, 4, '2026-05-28 10:00:00'),
  (4, 1, 1004, 6, 3, '2026-05-28 10:00:00'),
  (5, 1, 1005, 10, 3, '2026-05-28 10:00:00'),
  (6, 1, 1006, 15, 5, '2026-05-28 10:00:00'),
  (7, 1, 1007, 11, 4, '2026-05-28 10:00:00'),
  (8, 1, 1008, 20, 6, '2026-05-28 10:00:00'),
  (9, 1, 1009, 25, 8, '2026-05-28 10:00:00'),
  (10, 1, 1010, 14, 5, '2026-05-28 10:00:00')
ON DUPLICATE KEY UPDATE
  store_id = VALUES(store_id),
  sku_id = VALUES(sku_id),
  quantity = VALUES(quantity),
  safety_stock = VALUES(safety_stock),
  updated_at = VALUES(updated_at);

INSERT INTO promotion (id, store_id, title, description, product_scope, start_at, end_at, status)
VALUES
  (1, 1, '饮料第二件半价', '可口可乐、雪碧、无糖可乐参与第二件半价。', JSON_ARRAY('101', '102', '103'), '2026-05-28 00:00:00', '2026-05-28 23:59:59', 'active'),
  (2, 1, '低糖饮料满 2 件减 3 元', '低糖和无糖饮料任选两件立减 3 元。', JSON_ARRAY('102', '104', '105'), '2026-05-28 00:00:00', '2026-05-30 23:59:59', 'active'),
  (3, 1, '纸品专区 95 折', '维达抽纸等纸品享受 95 折优惠。', JSON_ARRAY('108'), '2026-05-28 00:00:00', '2026-06-03 23:59:59', 'active')
ON DUPLICATE KEY UPDATE
  store_id = VALUES(store_id),
  title = VALUES(title),
  description = VALUES(description),
  product_scope = VALUES(product_scope),
  start_at = VALUES(start_at),
  end_at = VALUES(end_at),
  status = VALUES(status);
