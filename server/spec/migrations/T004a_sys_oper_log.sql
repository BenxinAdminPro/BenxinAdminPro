-- +----------------------------------------------------------------------
-- | @project   本心通用管理后台 / BenxinAdminPro
-- | @mission   T-004a 操作日志表 DDL
-- | @author    仗键天涯(daxing)
-- | @email     3442535897@qq.com
-- | @date      2026-06-08 00:12:00
-- +----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `{{TABLE_PREFIX}}sys_oper_log` (
  `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `operator`    VARCHAR(64)     NOT NULL DEFAULT '',
  `method`      VARCHAR(8)      NOT NULL DEFAULT '',
  `path`        VARCHAR(255)    NOT NULL DEFAULT '',
  `perm_code`   VARCHAR(128)    NOT NULL DEFAULT '',
  `ip`          VARCHAR(64)     NOT NULL DEFAULT '',
  `user_agent`  VARCHAR(255)    NOT NULL DEFAULT '',
  `req_summary` TEXT,
  `result_code` INT             NOT NULL DEFAULT 0,
  `latency_ms`  INT             NOT NULL DEFAULT 0,
  `created_at`  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_sys_oper_log_operator` (`operator`),
  KEY `idx_sys_oper_log_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
