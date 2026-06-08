# 完成报告：T-003d RBAC 鉴权接线修复（+ T-006 e2e 转绿）

## 1. 完成状态
✅ 已完成（待 daxing 确认后双推）。所有受保护路由改走真 Casbin enforce，假 RequirePerm 彻底消除；
T-006 demo e2e 8 步整条转绿（含此前为 200 的第 7 步无权 403）。代码层自测全绿，未推、未改 PROJECT_STATUS。

## 2. 改动文件清单
| 文件 | 说明 | 新增/修改 |
|---|---|---|
| server/rbac/middleware.go | 删包级假 `RequirePerm` + 无人用的 `Authz` 全局中间件；`AuthzEnforcer.RequirePerm` 顺带写 perm code 入 context 供 OperLog 审计（enforce 仍用闭包捕获 code，非旧破损模式） | 修改 |
| server/rbac/handler_user.go | RegisterRoutes 注入 `*AuthzEnforcer`，8 路由走 `authz.RequirePerm` | 修改 |
| server/rbac/handler_dept.go | 同上，4 路由 | 修改 |
| server/rbac/handler_post.go | 同上，4 路由 | 修改 |
| server/rbac/handler_role.go | 同上，6 路由 | 修改 |
| server/rbac/handler_menu.go | 同上，4 路由 | 修改 |
| server/system/handler.go | 删本地假 `requirePerm`；新增 `PermGuard` 接口；RegisterRoutes 注入 guard，16 路由走 `guard.RequirePerm` | 修改 |
| server/system/handler_file.go | RegisterRoutes 注入 `PermGuard`，4 路由走 `guard.RequirePerm` | 修改 |
| server/examples/demo/main.go | 不再 `_ =` 丢弃 authzEnforcer，注入各 RegisterRoutes；auth_info 保持 JWT-only | 修改 |
| server/rbac/rbac_test.go | 新增 `TestRegisterRoutesEnforcesPerm`：经真 handler.RegisterRoutes 验无权 403 | 修改 |
| server/system/handler_authz_test.go | 记录式假 guard 验 system/file 每路由都挂上 guard + 403 拦在 handler 前 | 新增 |

> 注：本次工作区同时含 T-006 收尾改动（seed 去硬编码 / e2e / 迁移器 #5 / casbin #4），见各自切片报告；提交时分 commit。

## 3. 修复情况
| 项 | 状态 | 备注 |
|---|---|---|
| 消除假 RequirePerm | ✅ 删除式（方案 a） | 包级 `RequirePerm` 与 `Authz` 全局中间件均删除；system 本地 `requirePerm` 删除。grep 确认全仓无定义、无 `required_perm_code` 写入残留（仅 OperLog 读取，由真 enforcer 写入） |
| RegisterRoutes 注入 enforcer（全模块） | ✅ | rbac 用 `*AuthzEnforcer`（同包）；system 用 `PermGuard` 接口（结构性满足，system 不 import rbac，零耦合无环） |
| 路由级真 enforce + 中间件顺序 | ✅ | 路由级闭包捕获 code，不依赖全局中间件读 context value（旧 bug 根源）。链：JWTAuth→authz.RequirePerm(enforce)→OperLog→handler；enforce 在 handler 前生效 |
| 超管短路保留 | ✅ | `AuthzEnforcer.RequirePerm` 内 GetRolesForUser + superSet 短路；超管角色配置注入（cfg.SuperAdminRoles） |
| demo main.go 注入 authzEnforcer | ✅ | 6 处 RBAC + 2 处 system 注入；auth_info 自身菜单/权限接口 JWT-only by design |
| demo e2e 第 7 步 403 转绿 | ✅ | editor(仅 sys:user:list) POST /sys/users → 403（此前 200 建出用户）；其余步骤仍绿 |

## 4. 自验结果
- 构建/静态检查：`go build ./...` ✅ `go vet ./...` ✅ `go vet -tags=integration ./...` ✅（所有集成测试文件随签名变更仍编译）
- 假 RequirePerm 已消除（grep 证据）：
  - `func RequirePerm(` → 无；`func Authz(` → 无；`func requirePerm(` → 无
  - `required_perm_code` → 仅 operlog.go 读取（审计），由真 enforcer 写入
  - 受保护路由真 enforce 调用：`authz.RequirePerm` ×26 + `guard.RequirePerm` ×20 = **46 条全覆盖**；无裸 `RequirePerm(` 调用
- 无权 403 / 有权 200 / 超管放行 单测：rbac 全绿——TestRegisterRoutesEnforcesPerm、TestAuthzMiddlewarePermCode(403)、TestAuthzSuperAdminBypass、TestAuthzNoPermCodePassThrough、TestAuthzDoesNotLeakPolicyDetails、TestEnforcer*；system 新增 guard 接线测试全绿
- 全量非集成单测：auth/crypto/storage/rbac/response/system 全 ok
- demo e2e 整条转绿（真依赖 MySQL3307+Redis）：迁移16表→种子→captcha→login→200→401→**403**→热加载，ALL STEPS PASSED
- migrator 集成测试：仍 ok（建表16、ALTER 列验、版本数18）

## 5. 逐路由核对清单（无遗漏裸奔）
- RBAC 受保护（真 enforce）：user 8 + dept 4 + post 4 + role 6 + menu 4 = **26**
- system 受保护（真 enforce）：dict/config/log 16 + file 4 = **20**
- 合计 **46 条全部真 enforce**
- 按设计无 perm code（非裸奔）：`/sys/auth/menus`、`/sys/auth/perms`（返回当前用户自身菜单/权限，受 protected 组 JWTAuth 保护）
- 公开（设计公开）：`/auth/captcha`、`/auth/login`、`/auth/refresh`、`/auth/logout`
- 结论：无遗漏裸奔受保护路由。

## 6. RegisterRoutes 签名变更对消费方影响说明
- rbac 各 handler：`RegisterRoutes(rg *gin.RouterGroup)` → `RegisterRoutes(rg *gin.RouterGroup, authz *AuthzEnforcer)`。
- system Handler / FileHandler：`RegisterRoutes(rg)` → `RegisterRoutes(rg, guard PermGuard)`（`PermGuard` = `interface{ RequirePerm(code string) gin.HandlerFunc }`，`*rbac.AuthzEnforcer` 结构性满足）。
- **对外契约变更**：消费方（如 BenxinKP）装配时须 `NewAuthzEnforcer(...)` 并注入各 RegisterRoutes。属"修复使鉴权真正生效"的必要变更。
- model.conf / perm code / 错误码 / openapi 均未变；有权限路径行为不变，唯一行为变化＝"原本错误放行的现在正确 403"。
- SemVer：未发版；记录为下次 minor 的对外接线变更（消费方需改装配代码）。

## 7. 安全自查
- [x] 真鉴权生效、无权 403、假 RequirePerm 消除（grep + 单测 + e2e 三重证据）
- [x] 超管短路服务端可信（角色配置注入，非硬编码）
- [x] JWT/数据权限(T-003c)/脱敏/参数校验未放松（仅改"功能权限如何挂鉴权"，OperLog perm code 审计行为保留）
- [x] 逐路由无遗漏裸奔（46 真 enforce + auth_info JWT-only + auth 公开）
- [x] DI/前缀随实例/头注释 @updated 齐备

## 8. 需 daxing 真人验收
- demo：普通用户(editor/biz_user)访问无权接口被 403；有权接口正常；超管(admin)全放行。
- 确认无遗漏裸奔受保护路由；评审 auth_info 的 JWT-only 设计是否符合预期。
- 评审 RegisterRoutes 注入式签名（rbac 用具体类型 / system 用 PermGuard 接口）对 BenxinKP 接入是否清晰。

## 9. 偏差与待办
- auth_info(`/sys/auth/menus`,`/sys/auth/perms`)按设计 JWT-only、未注入 authz（返回当前用户自身数据），未改其签名——如 PM 要求统一签名风格可再议。
- rbac/system 既有 *集成* 测试硬编码 `localhost:3306`（非本片改动），当前本机宿主 mysqld 占 3306 拒 root:root 故未在此环境运行；它们不触达本次改动的 HTTP 接线路径，且 `-tags=integration` 编译通过。env 可覆盖的 e2e/migrator 已用 3307 跑绿。

## 10. 下一步建议（T-006 整体收尾 + 双推 → admin 前端）
1. PM 确认后：分 commit 双推——`T-006`（seed去硬编码 + e2e + 迁移器#5 + casbin#4）与 `T-003d`（鉴权接线）。注：main.go 同含两片改动，提交切分方式请 PM 定（建议 main.go 归 T-003d 或合并提交）。
2. PM 更新 PROJECT_STATUS（执行端不自标完成）。
3. 之后进 admin 前端：在已 e2e 验证的可信后端上联调。
4. 一次性 MySQL benxin-e2e-mysql(3307) 收尾后可删。
