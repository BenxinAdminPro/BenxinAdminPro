-- +----------------------------------------------------------------------
-- | @project   本心通用管理后台 / BenxinAdminPro
-- | @mission   T-003b 菜单/权限点表 DDL — menu_type 区分 M/C/F
-- | @author    仗键天涯(daxing)
-- | @email     3442535897@qq.com
-- | @date      2026-06-07 21:06:00
-- +----------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS `{{TABLE_PREFIX}}sys_menu` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `parent_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `ancestors`  VARCHAR(255)    NOT NULL DEFAULT '0',
  `menu_type`  CHAR(1)         NOT NULL COMMENT 'M=目录 C=菜单 F=按钮',
  `name`       VARCHAR(64)     NOT NULL,
  `perm_code`  VARCHAR(128)    NOT NULL DEFAULT '' COMMENT '权限码（F 类型必填）',
  `path`       VARCHAR(255)    NOT NULL DEFAULT '',
  `component`  VARCHAR(255)    NOT NULL DEFAULT '',
  `icon`       VARCHAR(64)     NOT NULL DEFAULT '',
  `sort`       INT             NOT NULL DEFAULT 0,
  `visible`    TINYINT         NOT NULL DEFAULT 1 COMMENT '1=显示 0=隐藏',
  `status`     TINYINT         NOT NULL DEFAULT 0,
  `created_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME            NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_sys_menu_parent_id` (`parent_id`),
  KEY `idx_sys_menu_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
