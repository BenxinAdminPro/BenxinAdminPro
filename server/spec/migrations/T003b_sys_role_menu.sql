-- +----------------------------------------------------------------------
-- | @project   本心通用管理后台 / BenxinAdminPro
-- | @mission   T-003b 角色-菜单关联表 DDL
-- | @author    仗键天涯(daxing)
-- | @email     3442535897@qq.com
-- | @date      2026-06-07 21:06:00
-- +----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `{{TABLE_PREFIX}}sys_role_menu` (
  `role_id` BIGINT UNSIGNED NOT NULL,
  `menu_id` BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (`role_id`, `menu_id`),
  KEY `idx_sys_role_menu_menu_id` (`menu_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
