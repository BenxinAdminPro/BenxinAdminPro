-- +----------------------------------------------------------------------
-- | @project   本心通用管理后台 / BenxinAdminPro
-- | @mission   T-003a 用户表 DDL — 表前缀占位符参数化
-- | @author    仗键天涯(daxing)
-- | @email     3442535897@qq.com
-- | @date      2026-06-07 19:05:00
-- +----------------------------------------------------------------------
-- 占位符 {{TABLE_PREFIX}} 由迁移执行器替换；禁止写死任何应用前缀。

CREATE TABLE IF NOT EXISTS `{{TABLE_PREFIX}}sys_user` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `username`      VARCHAR(64)     NOT NULL,
  `password_hash` VARCHAR(255)    NOT NULL DEFAULT '',
  `nickname`      VARCHAR(64)     NOT NULL DEFAULT '',
  `avatar`        VARCHAR(255)    NOT NULL DEFAULT '',
  `email`         VARCHAR(128)    NOT NULL DEFAULT '',
  `mobile`        VARCHAR(32)     NOT NULL DEFAULT '',
  `dept_id`       BIGINT UNSIGNED     NULL DEFAULT NULL,
  `status`        TINYINT         NOT NULL DEFAULT 0 COMMENT '0=正常 1=禁用',
  `remark`        VARCHAR(255)    NOT NULL DEFAULT '',
  `created_at`    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at`    DATETIME            NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_sys_user_username` (`username`),
  KEY `idx_sys_user_dept_id` (`dept_id`),
  KEY `idx_sys_user_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
