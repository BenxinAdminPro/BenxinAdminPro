-- +----------------------------------------------------------------------
-- | @project   本心通用管理后台 / BenxinAdminPro
-- | @mission   T-003a 岗位表 DDL
-- | @author    仗键天涯(daxing)
-- | @email     3442535897@qq.com
-- | @date      2026-06-07 19:05:00
-- +----------------------------------------------------------------------
-- 占位符 {{TABLE_PREFIX}} 由迁移执行器替换。

CREATE TABLE IF NOT EXISTS `{{TABLE_PREFIX}}sys_post` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `code`       VARCHAR(64)     NOT NULL COMMENT '岗位编码',
  `name`       VARCHAR(64)     NOT NULL,
  `sort`       INT             NOT NULL DEFAULT 0,
  `status`     TINYINT         NOT NULL DEFAULT 0 COMMENT '0=正常 1=停用',
  `created_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME            NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_sys_post_code` (`code`),
  KEY `idx_sys_post_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
