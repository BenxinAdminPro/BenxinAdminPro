-- +----------------------------------------------------------------------
-- | @project   本心通用管理后台 / BenxinAdminPro
-- | @mission   T-003b 用户-角色关联表 DDL
-- | @author    仗键天涯(daxing)
-- | @email     3442535897@qq.com
-- | @date      2026-06-07 21:06:00
-- +----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `{{TABLE_PREFIX}}sys_user_role` (
  `user_id` BIGINT UNSIGNED NOT NULL,
  `role_id` BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (`user_id`, `role_id`),
  KEY `idx_sys_user_role_role_id` (`role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
