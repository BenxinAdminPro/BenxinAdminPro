-- +----------------------------------------------------------------------
-- | @project   本心通用管理后台 / BenxinAdminPro
-- | @mission   T-005 迁移版本记录表 DDL
-- | @author    仗键天涯(daxing)
-- | @email     3442535897@qq.com
-- | @date      2026-06-08 03:02:00
-- +----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `{{TABLE_PREFIX}}sys_migration` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `version`    VARCHAR(128)    NOT NULL COMMENT '迁移文件标识',
  `applied_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `checksum`   VARCHAR(64)     NOT NULL DEFAULT '' COMMENT '文件内容校验',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_sys_migration_version` (`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
