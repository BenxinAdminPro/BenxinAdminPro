-- +----------------------------------------------------------------------
-- | @project   本心通用管理后台 / BenxinAdminPro
-- | @mission   T-004a 字典项表 DDL
-- | @author    仗键天涯(daxing)
-- | @email     3442535897@qq.com
-- | @date      2026-06-08 00:12:00
-- +----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `{{TABLE_PREFIX}}sys_dict_data` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `dict_type`  VARCHAR(64)     NOT NULL,
  `label`      VARCHAR(64)     NOT NULL,
  `value`      VARCHAR(64)     NOT NULL,
  `sort`       INT             NOT NULL DEFAULT 0,
  `status`     TINYINT         NOT NULL DEFAULT 0,
  `created_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME            NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_sys_dict_data_type` (`dict_type`),
  KEY `idx_sys_dict_data_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
