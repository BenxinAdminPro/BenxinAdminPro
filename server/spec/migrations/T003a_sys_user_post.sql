-- +----------------------------------------------------------------------
-- | @project   本心通用管理后台 / BenxinAdminPro
-- | @mission   T-003a 用户-岗位关联表 DDL（多对多）
-- | @author    仗键天涯(daxing)
-- | @email     3442535897@qq.com
-- | @date      2026-06-07 19:05:00
-- +----------------------------------------------------------------------
-- 占位符 {{TABLE_PREFIX}} 由迁移执行器替换。

CREATE TABLE IF NOT EXISTS `{{TABLE_PREFIX}}sys_user_post` (
  `user_id` BIGINT UNSIGNED NOT NULL,
  `post_id` BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (`user_id`, `post_id`),
  KEY `idx_sys_user_post_post_id` (`post_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
