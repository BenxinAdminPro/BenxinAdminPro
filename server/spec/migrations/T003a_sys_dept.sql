-- +----------------------------------------------------------------------
-- | @project   本心通用管理后台 / BenxinAdminPro
-- | @mission   T-003a 部门表 DDL — 树形结构（parent_id + ancestors）
-- | @author    仗键天涯(daxing)
-- | @email     3442535897@qq.com
-- | @date      2026-06-07 19:05:00
-- +----------------------------------------------------------------------
-- 占位符 {{TABLE_PREFIX}} 由迁移执行器替换。

CREATE TABLE IF NOT EXISTS `{{TABLE_PREFIX}}sys_dept` (
  `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `parent_id`  BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '父部门 ID，0=根',
  `ancestors`  VARCHAR(255)    NOT NULL DEFAULT '0' COMMENT '祖级路径，如 0,1,3',
  `name`       VARCHAR(64)     NOT NULL,
  `sort`       INT             NOT NULL DEFAULT 0,
  `leader`     VARCHAR(64)     NOT NULL DEFAULT '' COMMENT '负责人',
  `status`     TINYINT         NOT NULL DEFAULT 0 COMMENT '0=正常 1=停用',
  `created_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME            NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_sys_dept_parent_id` (`parent_id`),
  KEY `idx_sys_dept_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
