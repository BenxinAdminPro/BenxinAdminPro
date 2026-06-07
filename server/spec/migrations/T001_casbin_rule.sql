-- +----------------------------------------------------------------------
-- | @project   本心通用管理后台 / BenxinAdminPro
-- | @mission   T-001 Casbin 策略表 DDL — 表前缀占位符参数化
-- | @author    仗键天涯(daxing)
-- | @email     3442535897@qq.com
-- | @date      2026-06-07 14:18:00
-- +----------------------------------------------------------------------
--
-- 占位符 {{TABLE_PREFIX}} 由迁移执行器替换；禁止在本文件写死任何应用前缀。
-- gorm-adapter 已 TurnOffAutoMigrate，建表只走本 SQL。
--
-- 替换路径（三选一）：
--   1. 迁移执行器（推荐）：未来 T-005 配置中心将提供迁移 runner，
--      自动读取 app.table_prefix 配置并对 spec/migrations/*.sql 做文本替换后执行。
--   2. 手动替换：复制本文件，将 {{TABLE_PREFIX}} 全局替换为实际前缀（如 "kp_"），
--      然后在 MySQL 客户端中执行：
--        sed 's/{{TABLE_PREFIX}}/kp_/g' T001_casbin_rule.sql | mysql -u root -p benxinadminpro
--   3. 集成测试路径（参考）：rbac/mysql_integration_test.go 中的 setupMySQL()
--      演示了 Go 代码中 strings.ReplaceAll + db.Exec 的替换方式。

CREATE TABLE IF NOT EXISTS `{{TABLE_PREFIX}}casbin_rule` (
  `id`    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `ptype` VARCHAR(100) NOT NULL DEFAULT '',
  `v0`    VARCHAR(100) NOT NULL DEFAULT '',
  `v1`    VARCHAR(100) NOT NULL DEFAULT '',
  `v2`    VARCHAR(100) NOT NULL DEFAULT '',
  `v3`    VARCHAR(100) NOT NULL DEFAULT '',
  `v4`    VARCHAR(100) NOT NULL DEFAULT '',
  `v5`    VARCHAR(100) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_casbin_rule` (`ptype`,`v0`,`v1`,`v2`,`v3`,`v4`,`v5`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
