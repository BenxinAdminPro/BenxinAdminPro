# 完成报告：T-003c 数据权限（RBAC 收官）

## 1. 完成状态
**已完成** — DataScope 解析器 + ApplyScope 安全失败 + 多角色合并 + 自测样例 + 全绿。

## 2. 改动文件清单

| 文件 | 说明 | 新增/修改 |
|---|---|---|
| `server/errcode/errcode.go` | ErrInvalidDataScope offset 50 | 修改 |
| `server/rbac/model_role.go` | SysRole 增 DataScope int8 字段 | 修改 |
| `server/rbac/datascope.go` | ScopeType 枚举 + DataScope 结构 + GormScopeResolver | 新增 |
| `server/rbac/datascope_gorm.go` | ApplyScope + ScopeFields（安全失败） | 新增 |
| `server/rbac/datascope_test.go` | 解析器矩阵 8 + ApplyScope 安全 9 + 自测样例 1 = 18 tests | 新增 |
| `server/rbac/role_service.go` | Create/UpdateRoleInput 加 DataScope + 校验 | 修改 |
| `server/rbac/user_service.go` | UserListQuery.Scope + List 应用 ApplyScope | 修改 |
| `server/rbac/handler_user.go` | SetScopeResolver + List 注入 DataScope | 修改 |
| `server/rbac/response.go` | Role() 加 data_scope 字段 | 修改 |
| `server/spec/migrations/T003c_sys_role_data_scope.sql` | ALTER TABLE 增列 DDL | 新增 |
| `server/spec/openapi/openapi.yaml` | 升 v0.5.0，SysRole 加 data_scope | 修改 |

## 3. 接口实现情况

| 项 | 状态 |
|---|---|
| sys_role 增列 data_scope + 迁移 | ✅ DEFAULT 2（本人，最小权限） |
| 角色 CRUD 带 data_scope + 校验 | ✅ 0 默认 2，非法 → ErrInvalidDataScope |
| DataScope 模型 + ScopeResolver | ✅ GormScopeResolver 从 DB 查角色 |
| 多角色合并（取最宽） | ✅ All > Dept > Self；All 短路 |
| ApplyScope GORM 辅助 + 安全失败 | ✅ 5 处 WHERE 1=0 分支 |
| 超管 = All | ✅ superRoles 配置注入，短路 |
| 自测样例（用户列表接入 scope） | ✅ SQLite 验证 Self=1/Dept=2/All=3 行 |
| openapi v0.5.0 | ✅ redocly 0 error |

## 4. 自验结果
- build/vet 通过；T-001~T-003b 旧测试全绿
- datascope_test.go：18 tests 全绿（解析器 8 + ApplyScope 9 + 自测样例 1）
- ApplyScope 安全失败：nil/空列名/空 DeptIDs/零 SelfID/未知类型 — 每条断言通过
- grep 无 import config、无业务表名/字段名

## 5. git 提交记录
- 待本轮提交

## 6. 安全自查
- [x] 失败一律收紧（5 处 WHERE 1=0）— 每个分支有断言
- [x] 超管=All 且来源服务端可信（superRoles 配置注入，从 DB 查角色）
- [x] SelfID/DeptID/角色全来自服务端（Resolve 从 DB 查，handler 从 JWT claims 取 userID）
- [x] 数据权限叠加而非绕过 enforce（handler 先过 RequirePerm 再 ApplyScope）
- [x] 业务中立：无业务表名/字段名；ScopeFields 调用方传入
- [x] DI 不 import config；前缀随实例；grep 干净
- [x] 头注释五项；改动文件 @updated

## 7. T-003 整体回顾（a+b+c）

**认证→功能权限→数据权限闭环完整**：
- T-002 登录 → JWT access/refresh token
- T-003a 用户/部门/岗位 → GormUserProvider 接 T-002
- T-003b 角色/菜单/权限 → Casbin enforce（接口级功能权限）+ RequirePerm 28 路由全覆盖
- T-003c DataScope → ApplyScope（行级数据权限）叠加在 enforce 之上

**遗留/扩展位**：
- ScopeDeptAndChild (4)：本部门及子部门 — 预留枚举值，未实现
- 自定义部门集：预留 DeptIDs []uint64 结构，扩展时填充多部门
- 响应 ID hashid 编码：已在 handler 层完成路径参数解码 + ResponseEncoder 响应编码
- 统一响应 ID 编码中间件：可后续优化（当前 ResponseEncoder 方案已覆盖全部 handler）

## 8. 需 daxing 真人验收
- [ ] demo：给角色设不同 data_scope，跨部门用户登录后确认数据范围
- [ ] 故意制造歧义（无角色/无 dept_id）确认收紧为查不到
- [ ] 超管看到全部；普通受限用户被收窄
- [ ] 评审 DataScope/ScopeFields 通用性
- [ ] T-003 整体回顾：a+b+c 构成完整 RBAC

## 9. 偏差与待办
- 无偏差

## 10. 下一步建议
- T-004 系统管理（字典/参数/操作日志/登录日志/文件）
- response 完整包络模块接管 errcode 注册表
