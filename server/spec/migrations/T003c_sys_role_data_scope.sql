-- +----------------------------------------------------------------------
-- | @project   本心通用管理后台 / BenxinAdminPro
-- | @mission   T-003c sys_role 增列 data_scope — 数据权限范围
-- | @author    仗键天涯(daxing)
-- | @email     3442535897@qq.com
-- | @date      2026-06-07 23:33:00
-- +----------------------------------------------------------------------
-- 占位符 {{TABLE_PREFIX}} 由迁移执行器替换。
-- 增量迁移：为 sys_role 增加 data_scope 列，不重建表。

ALTER TABLE `{{TABLE_PREFIX}}sys_role`
  ADD COLUMN `data_scope` TINYINT NOT NULL DEFAULT 2
  COMMENT '数据权限范围(1=全部,2=本人,3=本部门)'
  AFTER `status`;
