-- +----------------------------------------------------------------------
-- | @project   本心通用管理后台 / BenxinAdminPro
-- | @mission   T-005 sys_config 增列 is_encrypted — 敏感参数 GCM 加密标记
-- | @author    仗键天涯(daxing)
-- | @email     3442535897@qq.com
-- | @date      2026-06-08 03:02:00
-- +----------------------------------------------------------------------

ALTER TABLE `{{TABLE_PREFIX}}sys_config`
  ADD COLUMN `is_encrypted` TINYINT NOT NULL DEFAULT 0
  COMMENT '是否加密存储(0=明文,1=GCM加密)'
  AFTER `remark`;
