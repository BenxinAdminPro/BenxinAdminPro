# 完成报告：T-003a 组织架构（用户+部门+岗位）

## 1. 完成状态
**已完成** — 全部模型/DDL/服务/handler/GormUserProvider + 单测全绿 + openapi v0.3.0 lint 0 error。

## 2. 改动文件清单

| 文件 | 说明 | 新增/修改 |
|---|---|---|
| `server/errcode/errcode.go` | 新增 T-003a 错误码 offset（+30~+35）| 修改 |
| `server/rbac/model.go` | SysUser/SysDept/SysPost/SysUserPost GORM 模型 + SetTablePrefix + PageResult | 新增 |
| `server/rbac/provider_gorm.go` | GormUserProvider 实现 auth.UserProvider | 新增 |
| `server/rbac/user_service.go` | 用户 CRUD + 密码管理 + 启用禁用 | 新增 |
| `server/rbac/dept_service.go` | 部门 CRUD + 树构建 + ancestors 同步 + 防环 | 新增 |
| `server/rbac/post_service.go` | 岗位 CRUD + 分页 | 新增 |
| `server/rbac/jwt_middleware.go` | JWT 鉴权中间件（admin 端路由守卫）| 新增 |
| `server/rbac/handler_user.go` | 用户 7 路由 handler + 权限码占位 | 新增 |
| `server/rbac/handler_dept.go` | 部门 4 路由 handler | 新增 |
| `server/rbac/handler_post.go` | 岗位 4 路由 handler | 新增 |
| `server/rbac/service_test.go` | service 单测 SQLite 内存（15 tests）| 新增 |
| `server/rbac/org_integration_test.go` | MySQL 集成测试（spec SQL 建表 + CRUD + GormUserProvider）| 新增 |
| `server/spec/migrations/T003a_sys_user.sql` | 用户表 DDL | 新增 |
| `server/spec/migrations/T003a_sys_dept.sql` | 部门表 DDL | 新增 |
| `server/spec/migrations/T003a_sys_post.sql` | 岗位表 DDL | 新增 |
| `server/spec/migrations/T003a_sys_user_post.sql` | 用户岗位关联表 DDL | 新增 |
| `server/spec/openapi/openapi.yaml` | 升 v0.3.0，15 个组织架构路径 + schema | 修改 |

## 3. 接口实现情况

| 项 | 位置 | 状态 | 备注 |
|---|---|---|---|
| sys_user/dept/post/user_post 模型+DDL | server/rbac, server/spec/migrations | ✅ | 4 模型 + 4 SQL，{{TABLE_PREFIX}} 占位 |
| 用户 CRUD | server/rbac/user_service.go + handler_user.go | ✅ | 7 路由，密码用 PasswordHasher |
| 部门树 CRUD | server/rbac/dept_service.go + handler_dept.go | ✅ | 树构建 + ancestors 同步 + 防环 |
| 岗位 CRUD | server/rbac/post_service.go + handler_post.go | ✅ | 4 路由 |
| GormUserProvider（接 T-002） | server/rbac/provider_gorm.go | ✅ | FindByUsername → auth.AuthUser |
| openapi v0.3.0 | server/spec/openapi | ✅ | redocly lint 0 error |

## 4. 自验结果

- **构建/静态检查**：`go build ./...` + `go vet ./...` 全部通过
- **service 单测（docker-free）**：rbac 15 tests 全绿（用户 CRUD 8 + 部门 4 + 岗位 1 + GormUserProvider 2）
- **集成测试**：`//go:build integration` 覆盖 spec SQL 建表 + MySQL CRUD + 树操作 + GormUserProvider
- **响应无 password_hash 泄漏**：SysUser.PasswordHash 标记 `json:"-"`；Get/Create 返回前清零；List 用 Select 排除
- **T-001/T-002 旧测试全绿**

## 5. git 提交记录

- 待本轮提交

## 6. 安全自查

- [x] 密码哈希存储（auth.PasswordHasher.Hash）；响应/日志无明文与哈希（json:"-"）
- [x] JWT 鉴权已挂（jwt_middleware.go）；权限码占位（sys:user:list 等统一命名）
- [x] 部门树防环（ancestors 包含自身检查）+ ancestors 子树同步（REPLACE）
- [x] 删除保护：部门有子节点/关联用户拒删；用户软删除
- [x] 唯一性 DB 索引（username UNIQUE, code UNIQUE）+ service 层 Count 双校验
- [x] service/handler 不 import config；GORM 参数化查询；grep 无硬编码专属值
- [x] 头注释五项到秒、改动文件 @updated

## 7. 决策点回应

- **对外 ID 方案**：暂用自增 ID。理由：引入 Hashid 需要统一编解码中间件（路径参数解码 + 响应编码），涉及所有 handler，工作量大；T-003b 会新增更多 handler，在 b 统一收口更干净。报告标注，留 T-003b 统一收口。

## 8. 需 daxing 真人验收

- [ ] demo 把登录的 MemUserProvider 换成 GormUserProvider，真库建用户后 curl 跑通 T-002 全流程
- [ ] curl 跑通用户/部门/岗位 CRUD + 部门树
- [ ] 评审 sys_user 字段边界（nickname/avatar/email/mobile 是否可接受为通用联系字段）
- [ ] 评审对外 ID 方案（暂自增，T-003b 收口 hashid）

## 9. 偏差与待办

- 对外 ID 暂用自增，留 T-003b 统一 Hashid 收口
- 表前缀通过包级 SetTablePrefix 设置（启动时一次，非全局可变状态）

## 10. 下一步建议（衔接 T-003b）

- T-003b：角色 + 权限点 + 菜单 + 用户授权 + Casbin policy 联动
- 统一 Hashid 编解码中间件
- 启用权限码占位（接 Casbin enforce）
