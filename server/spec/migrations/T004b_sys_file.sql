-- +----------------------------------------------------------------------
-- | @project   本心通用管理后台 / BenxinAdminPro
-- | @mission   T-004b 文件元信息表 DDL
-- | @author    仗键天涯(daxing)
-- | @email     3442535897@qq.com
-- | @date      2026-06-08 01:19:00
-- +----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `{{TABLE_PREFIX}}sys_file` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `original_name` VARCHAR(255)    NOT NULL DEFAULT '',
  `storage_key`   VARCHAR(512)    NOT NULL DEFAULT '',
  `storage_type`  VARCHAR(32)     NOT NULL DEFAULT 'local',
  `size`          BIGINT          NOT NULL DEFAULT 0,
  `mime`          VARCHAR(128)    NOT NULL DEFAULT '',
  `ext`           VARCHAR(32)     NOT NULL DEFAULT '',
  `uploader`      VARCHAR(64)     NOT NULL DEFAULT '',
  `status`        TINYINT         NOT NULL DEFAULT 0 COMMENT '0=正常 1=待清理',
  `created_at`    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at`    DATETIME            NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_sys_file_uploader` (`uploader`),
  KEY `idx_sys_file_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
