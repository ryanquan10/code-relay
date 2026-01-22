-- 添加 used_balance 字段到 account 表
-- 执行日期: 2026-01-22

-- 添加字段
ALTER TABLE `account` ADD COLUMN `used_balance` DECIMAL(10,2) NOT NULL DEFAULT 0.00 COMMENT '已使用余额' AFTER `balance`;

-- 为现有账号初始化 used_balance（可选：根据历史 usage 数据初始化）
-- UPDATE `account` a
-- SET a.used_balance = (
--     SELECT IFNULL(SUM(u.consume), 0)
--     FROM `usage` u
--     WHERE u.account_id = a.id
-- );
