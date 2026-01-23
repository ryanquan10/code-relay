-- 为 usage 表添加 tokens 字段（如果不存在）
-- 用于记录每次请求消耗的 token 数量

-- 检查并添加 tokens 列
ALTER TABLE `usage`
ADD COLUMN IF NOT EXISTS `tokens` BIGINT UNSIGNED NOT NULL DEFAULT 0
COMMENT 'Token 数量'
AFTER `account_id`;

-- 如果表中已有数据但 tokens 为 0，可以根据 consume 反推 tokens
-- 假设转换率：每 1000 tokens = 0.01 元，即每个 token = 0.00001 元
-- tokens = consume / 0.00001 = consume * 100000
-- UPDATE `usage` SET `tokens` = ROUND(consume * 100000) WHERE `tokens` = 0 AND consume > 0;
