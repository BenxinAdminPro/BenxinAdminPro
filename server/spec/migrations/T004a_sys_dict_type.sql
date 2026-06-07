-- +----------------------------------------------------------------------
-- | @project   本心通用管理后台 / BenxinAdminPro
-- | @mission   T-004a 字典类型表 DDL
-- | @author    仗键天涯(daxing)
-- | @email     3442535897@qq.com
-- | @date      2026-06-08 00:12:00
-- +----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `{{TABLE_PREFIX}}sys_dict_type` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `dict_type`  VARCHAR(64)     NOT NULL,
  `name`       VARCHAR(64)     NOT NULL,
  `status`     TINYINT         NOT NULL DEFAULT 0,
  `remark`     VARCHAR(255)    NOT NULL DEFAULT '',
  `created_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME            NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_sys_dict_type_type` (`dict_type`),
  KEY `idx_sys_dict_type_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
