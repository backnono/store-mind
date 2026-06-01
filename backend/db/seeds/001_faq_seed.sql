INSERT INTO faq (store_id, question, answer, category, status) VALUES
(1, '怎么付款？', '你可以使用小程序扫码结算，支持微信和支付宝。', 'payment', 'active'),
(1, '可以退款吗？', '如商品存在质量问题，可在订单详情提交退款申请。', 'refund', 'active'),
(1, '营业到几点？', '本店营业时间为每天 08:00-23:00。', 'store_hours', 'active'),
(1, '如何联系客服？', '可在小程序右上角点击“联系客服”进入人工服务。', 'customer_service', 'active')
ON DUPLICATE KEY UPDATE
answer = VALUES(answer),
category = VALUES(category),
status = VALUES(status);
