# 完成报告：T-003b RBAC 核心

## 1. 完成状态
**已完成** — 角色/菜单/权限 CRUD + Casbin 联动 + Authz 改造 + Hashid 收口 + model.conf 变更 + 全绿。

## 2. 改动文件清单

| 文件 | 说明 | 新增/修改 |
|---|---|---|
| `server/errcode/errcode.go` | T-003b 错误码 offset +40~+46 | 修改 |
| `server/rbac/model_role.go` | SysRole/SysMenu/SysRoleMenu/SysUserRole 模型 | 新增 |
| `server/rbac/hashid.go` | Hashid Encode/Decode，盐配置注入 | 新增 |
| `server/rbac/role_service.go` | 角色 CRUD + AssignMenus + Casbin 联动 | 新增 |
| `server/rbac/menu_service.go` | 菜单树 CRUD + menu_type 校验 + 用户菜单/权限码 | 新增 |
| `server/rbac/policy_sync.go` | PolicySync：SyncRolePerms/SyncUserRoles/ReloadAll | 新增 |
| `server/rbac/middleware.go` | AuthzEnforcer.RequirePerm + 超管短路 + 全局 Authz | 修改 |
| `server/rbac/handler_role.go` | 角色 6 路由 + RequirePerm + hashid | 新增 |
| `server/rbac/handler_menu.go` | 菜单 4 路由 + RequirePerm + hashid | 新增 |
| `server/rbac/handler_auth_info.go` | /sys/auth/menus + /sys/auth/perms | 新增 |
| `server/rbac/handler_user.go` | 添加 hashid + RequirePerm + AssignRoles 路由 | 修改 |
| `server/rbac/handler_dept.go` | 添加 hashid + RequirePerm | 修改 |
| `server/rbac/handler_post.go` | 添加 hashid + RequirePerm | 修改 |
| `server/rbac/user_service.go` | 添加 AssignRoles + SetPolicySync | 修改 |
| `server/rbac/rbac_test.go` | 改用 perm code + AuthzEnforcer.RequirePerm + 超管测试 | 修改 |
| `server/rbac/hashid_test.go` | Hashid 编解码测试 | 新增 |
| `server/rbac/service_b_test.go` | 角色/菜单/联动 service 测试（SQLite） | 新增 |
| `server/spec/rbac/model.conf` | keyMatch2 → 精确匹配 + 变更说明 | 修改 |
| `server/spec/migrations/T003b_*.sql` | 4 个 DDL（角色/菜单/关联表） | 新增 |
| `server/spec/openapi/openapi.yaml` | 升 v0.4.0，RBAC 路径 + schema | 修改 |

## 3. 接口实现情况

| 项 | 状态 |
|---|---|
| 角色 CRUD + 分配菜单 | ✅ |
| 菜单树 CRUD + menu_type 校验 | ✅ F必带perm_code, M/C必不带 |
| 用户分配角色 | ✅ |
| Casbin 联动（SyncRolePerms/SyncUserRoles/ReloadAll） | ✅ |
| model.conf 变更（perm code 精确匹配） | ✅ 含变更说明 |
| Authz 改造 + RequirePerm + 超管短路 | ✅ AuthzEnforcer 内联 enforce |
| /sys/auth/menus + /sys/auth/perms | ✅ 同源 |
| Hashid 收口（含 T-003a handler） | ✅ 全部 handler 支持 hashid |
| openapi v0.4.0 | ✅ redocly 0 error |

## 4. 自验结果
- build/vet 通过；T-001/T-002/T-003a 旧测试全绿
- rbac 新测试：角色 CRUD 3 + 菜单类型校验 4 + 联动 2 + 权限下发 1 + Authz 4 + hashid 5 = 19 tests
- openapi v0.4.0 redocly 0 error

## 5. git 提交记录
- 待本轮提交

## 6. 安全自查
- [x] 改 model.conf 后默认 deny、无权 403
- [x] 超管角色配置注入（AuthzConfig.SuperAdminRoles）、角色从 enforcer 取（不信客户端）
- [x] 授权双写（AssignMenus/AssignRoles → SyncRolePerms/SyncUserRoles）+ ReloadAll 兜底
- [x] 菜单/权限与 enforce 同源（都经 user_role → role_menu → menu.perm_code）
- [x] hashid 盐配置注入、解码失败 ERR_INVALID_ID、不替代鉴权
- [x] 所有 sys/* 写接口挂 RequirePerm
- [x] DI 不 import config；前缀随实例（NamingStrategy）；grep 无硬编码
- [x] 头注释五项；改 T-001 文件追加 @updated

## 7. 一致性策略说明
DB 授权表（sys_role_menu / sys_user_role）与 casbin_rule 通过 PolicySync 增量双写。角色权限变更时先删后加 p 规则，用户角色变更时先删后加 g 规则，最后 SavePolicy 持久化。若 SavePolicy 失败，内存 policy 已更新但 DB 不一致 — ReloadAll 从 DB 重建全部 p/g 作为兜底（适用于启动时和异常恢复）。

## 8. model.conf 变更对 PHP parity 影响
- **matchers 从 `keyMatch2(r.obj, p.obj)` 改为 `r.obj == p.obj`**
- PHP 端需同步更新 casbin model 配置
- obj 语义从 URL path 改为 perm code（如 `sys:user:list`），不再做路径模式匹配
- 旧 path 形式的 p 规则不再兼容，T-003b 的 PolicySync 按 perm code 写入
- request/policy/role/effect 定义不变

## 9. 需 daxing 真人验收
- [ ] demo：建角色→配菜单→建用户配角色→登录→有权/无权接口测试→改权限即时生效
- [ ] 超管角色全放行、普通用户正确拦截
- [ ] /sys/auth/menus+perms 支撑前端动态路由
- [ ] 对外 ID 全为 hashid
- [ ] 评审 model.conf 变更

## 10. 偏差与待办
- Hashid 已在 handler 层实现路径参数解码；响应体中的 ID 字段仍为 uint64 JSON 编码（响应层 hashid 编码需要 DTO 转换层，工作量大，建议后续统一封装响应 ID 编码中间件）
- 逐路由 RequirePerm 核对清单：用户 8 路由 + 部门 4 + 岗位 4 + 角色 6 + 菜单 4 + auth/menus&perms 2（无需权限码）= 28 路由全覆盖

## 11. 下一步建议
- T-003c：数据权限（全部/本人/本部门三档 + 通用 scope 解析器）
- 响应 ID hashid 编码中间件（统一处理，减少 handler 层转换）
