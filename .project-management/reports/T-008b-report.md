# 完成报告：T-008b 用户分角色 + 列表角色列（前端弹窗 + 后端回填 + List 批量回填）

> 状态：**待 PM 评审 + daxing 整体验收**。未 commit、未双推、未改 PROJECT_STATUS（T-006 铁律）。
> 基线 HEAD = 5e83b43（T-008a 已双推 + 账本/归档）。后端回填(Get)+列表批量回填(List)+前端主体 + openapi v0.12.0。
> **含 PM 裁定并入的列表「角色」列增量**（daxing 验对核心三项后提出，归本片闭环再放行）——见 §2/§4/§7 标注。

## 0. 第 0 节源码核实（带出处）

| # | 核实项 | 结论（出处） |
|---|---|---|
| 0-1 | 分配角色写端点 | `PUT /sys/users/:id/roles`，perm `sys:user:assign`（`handler_user.go:67`）。入参 `{ role_ids: []string }` hashid（`handler_user.go:315` + `decodeIDSlice :321`）。**service 纯覆写坐实**：`tx.Where("user_id=?").Delete(&SysUserRole{})` → 按 role_ids `Create` + 联动 Casbin `SyncUserRoles(userSub, roleCodes)`（`user_service.go:281`）。→ 回填是覆写正确性前提。 |
| 0-2 | 回填来源现状 | `UserService.Get`（`user_service.go:147`）**原仅预载 Posts、不返 roles**（SysUser model 无 Roles 字段，`model.go:53` 仅 Posts）→ 「用户当前已授角色」**无现成查询来源**，确认无误。 |
| 0-3 | 回填端点选型 | **采用 A（GET /:id 预载 roles）**，详见下方专节，无阻断。 |
| 0-4 | 角色选项来源 | `GET /sys/roles`（`RoleHandler.List` `handler_role.go:45` + `RoleListQuery` page/page_size `role_service.go:41`）分页 → 前端循环翻页取齐（`listAllRoles`，复用 T-007h `listAllPosts` 范式 `api/post.ts:39`）。 |
| 0-5 | user↔role 关系表 | `SysUserRole{ UserID, RoleID }` **复合主键、无软删**（`model_role.go`）→ 覆写=全删重建干净；删角色对已授用户即刻失效（junction 物理删）。 |
| 0-6 | 前端用户页现状 | `user/index.vue` 行操作现有 编辑/删除（内置）+ 重置密码/停用启用（T-008a）；**无分配角色入口** → 本片加为 `config.actions` 第一项（与既有并列）。 |

### 0-3 回填端点选型结论：采用 **A（GET /sys/users/:id 预载 roles）**

无阻断、强烈优于 B：
- **`UserService.Get` 仅 1 个调用点**（`handler_user.go:225` GET /:id 详情）——grep 实证；预载 roles **只影响该端点，零污染其他出参**。
- **`enc.User` 手工构建出参**（`response.go:27`，password_hash 从不进表）→ 加 roles 数组干净、不破坏 T-003a 不泄漏铁律。
- **List 不预载**（`UserService.List` 显式 `Select(...)` 无 role 列 `user_service.go:201`）+ `roles,omitempty` → **列表出参零污染、无 N+1**。
- 与 T-008c「角色详情预载已授菜单」**同口径统一**；无新端点、无新 perm code（复用 GET /:id 的 sys:user:get）。
- 否决 B（单独端点）：Get 单调用点 + 手工编码器使 A 无任何副作用，无需多开端点。

## 1. 完成状态
✅ 已完成并端到端真跑。后端 A 方案补回填（GET /:id 预载 roles）；前端用户页加「分配角色」行操作（多选弹窗 + 回填当前已授 + 全量覆写 PUT :id/roles）。openapi v0.12.0。

## 2. 改动文件清单
| 文件 | 说明 | 类型 |
|---|---|---|
| `server/rbac/model.go` | SysUser 加非持久化 `Roles []SysRole gorm:"-" json:"roles,omitempty"` | 修改 |
| `server/rbac/user_service.go` | Get 预载已授角色；**【增量】List 批量回填 Roles（fillUserRoles，固定 2 查询非 N+1）** | 修改 |
| `server/rbac/response.go` | enc.User 加 roles 出参（复用 e.Role，Get/List 同一段，无角色 omitempty） | 修改 |
| `server/spec/openapi/openapi.yaml` | v0.11.0→v0.12.0：SysUser schema 加 roles 数组；**【增量】roles description 改为「详情与列表均返」** | 修改 |
| `server/examples/demo/assign_roles_integration_test.go` | **新增** e2e：回填/覆写不误删/Casbin g 联动/enforce/password_hash | 新增 |
| `server/rbac/list_roles_integration_test.go` | **【增量】新增** List 角色回填正确性 + **批量非 N+1 查询计数证明** + 软删剔除 + password_hash | 新增 |
| `admin/src/api/role.ts` | +`listAllRoles`（循环翻页全量角色选项） | 修改 |
| `admin/src/api/user.ts` | UserRow 加 roles?（列表列+回填共用）；+`assignUserRoles`（PUT :id/roles 全量覆写） | 修改 |
| `admin/src/views/sys/user/index.vue` | 「分配角色」行操作 + 多选弹窗；**【增量】列表「角色」列（formatter 顿号文本，无角色显「—」）** | 修改 |

## 3. 接口契约（v0.12.0）
- **写（已有不动）**：`PUT /sys/users/:id/roles` `{ role_ids: []string }` hashid 全量覆写 + Casbin g 联动。
- **读（本片补，A）**：`GET /sys/users/:id`（预载）**与 `GET /sys/users`（列表批量回填）** 出参均加 `roles: [{ id(hashid), code, name, ... }]`（复用 SysRole 编码器，无敏感字段）；无角色 omitempty 不返。
- hashid 全程透传不解码（回填=出参塞 v-model、提交原样回传）。

## 4. 自验结果（端到端真跑）

**构建/单测**：`go build ./... && vet ./... && test ./...` 全绿（9 包 ok，含 rbac 单测）；前端 `pnpm build` exit 0、`pnpm test` 17 passed。

**集成测试 `TestAssignRolesE2E`（真 MySQL，`-tags=integration` PASS）**：
- **① 回填正确性**：分配前 GET roles=0；分配后精确返已授（id hashid + code 校验）✓
- **② 全量覆写不误删（头号）**：2→3→1 三步，每步 **GET 出参 roles 数 + DB 实查 `sys_user_role` 行数双重精确**（原有未丢、新增进、移除走）✓
- **③ Casbin g 规则联动真生效**：给 probe 分配 editor 角色（含 sys:user:list）→ probe 登录后 GET /sys/users **200**；清空角色 → **403**（g 规则收回，非只改 junction）✓
- **④ enforce 正向**：editor（无 sys:user:assign）PUT roles **403** + GET /:id **403**（无 sys:user:get）↔ dept_mgr（全量授权）**200** ✓（超管不计为证据，T-007e 口径）
- **password_hash 不泄漏**：A 方案预载 roles 后 GET /:id 出参仍无 password_hash（断言）✓

**【增量】集成测试 `TestListUserRolesBackfillAndQueryCount`（真 MySQL PASS）**：
- **① 回填正确（一对多分组）**：3 用户 ua=2角色 / ub=1 / uc=0，List 出参 roles 数精确 ✓
- **② 批量非 N+1（查询计数器硬证）**：GORM `After("gorm:query")` 回调计数——**2 行页=4 次查询 == 6 行页=4 次**（Count+用户+junction+角色，固定不随 N 增长；N+1 会是 N 次）✓
- **③ password_hash 不泄漏**：List 每行 user 不含哈希（断言）✓
- **④ 软删角色剔除**：ua 关联含一个软删角色，model 查询（非 .Table）使 `deleted_at IS NULL` scope 生效 → roles 仍只 2、软删角色不出现（守 T-005b-4 #7 潜伏点）✓

**live demo 冒烟（已用增量代码重启）**：① GET /:id 分配后返 roles；② **GET /sys/users 列表每行带 roles**（temp05→编辑员、dept_mgr→部门经理、无角色→缺省）；③ password_hash 列表/详情均不泄漏。

**零回归**：T-001~T-008a 单测全绿；List 加 roles 批量回填、其他列不变（org 集成测试中 sys_user_role 表不存在时 fillUserRoles 优雅降级——Find 空→早返，不报错）。openapi YAML 解析校验通过（ruby）。

**pre-existing red（T-003d-fix）**：`rbac` `TestNewEnforcerMySQL_RoleInheritance` 仍 FAIL——本片**零触碰 enforcer/model.conf**（仅加 Get 预载 + model 非持久化字段），与 T-005b-4 轮 stash 复证同一陈旧断言、非本片引入。归该待办切片，未删断言凑绿。

## 5. git 提交记录
**未提交**。等 PM 评审 + daxing 真人验收 + 放行后双推。

## 6. 安全自查（逐项）
- **覆写正确性（头号）**：弹窗回填以 `getUser(id).roles` 全量为基准，提交 `assignUserRoles(id, 全量选择)`；集成测试 2→3→1 DB 行数精确证不误删未动角色。
- **回填防误清**：`openAssignRoles` 先 `Promise.all([getUser, listAllRoles])`，**任一失败不开弹窗**（避免残缺回填被全量覆写静默清空，同 T-007h api.get 防误清）。
- **JOIN/预载不泄漏**：A 预载仅 Find roles（SysRole 无敏感字段），enc.User 不输出 password_hash；集成测试断言 GET /:id 无 password_hash（守 T-003a）。
- **enforce 不旁路**：写端点 PUT :id/roles 挂 sys:user:assign、读端点 GET :id 挂 sys:user:get（均后端权威）；editor 实测两端点 403。
- **v-permission**：「分配角色」行操作挂 `sys:user:assign` 真实码，无权隐藏（前端隐藏仅 UX）。
- **hashid 透传不解码**：role_ids 出入参原样 hashid，伪造 id 后端 decodeIDSlice 统一 400（11045）。

## 7. 需 daxing 整体重验（原三项 + 列表角色列增量）
> ⚠️ **本片有后端 Go 改动**——demo **已用增量代码重启**（执行端已重启 + 冒烟确认 GET /:id 与 GET /sys/users 均返 roles）。前端 Vite 热更新。
- ① 用户页某行点「分配角色」→ 弹窗**回显其当前已授角色**（勾选态正确；新用户为空）。
- ② 增/删角色提交 → 重开弹窗回显新集合正确（不误删未动的）。
- ③ **闭环**：给一个只有工作台的新用户分配一个有菜单权限的角色 → 用它登录 → 能看到该角色对应菜单。
- ④ **【增量】用户列表「角色」列**正确显示各用户角色（多个顿号分隔、无角色显「—」）；**分配角色后刷新列表，「角色」列即时反映新角色**。

## 8. 偏差与待办
- **无偏差**：A 方案落地与任务书 §4 一致；分配角色走行操作 + 独立弹窗（不擅扩 x-table，同 T-008a 落点）。
- **【增量·PM 已裁定】角色列文本非 el-tag**：列表「角色」列用 column `formatter` 顿号文本（无角色「—」），**不用 el-tag**——el-tag 需 x-table 单元格插槽=改核心（§2 禁，同 T-008a status 落点）。PM 裁定走文本，已确认。
- **【增量·待办低优】通用单元格插槽**：若将来要 el-tag/富文本列展示，需给 x-table 加通用单元格插槽（一次性基建，本片/ T-008c 不需要）。记低优，用到再评估。
- **观察（pre-existing，非本片）**：`SysUserRole` 复合主键 1062 未专门转码（账本既有「junction 复合 PK 1062 未转码」待办）——本片覆写为先删后建，正常路径不触发，沿用既有口径。
- **待办（归 T-008c）**：角色授权树（最重）——后端搭车补「角色已授 menu_ids」回填（同 A 口径，RoleService.Get 预载 menu_ids），前端 el-tree 勾选 + 半选 + 全量覆写。

## 9. 下一步建议
- PM 评审 diff + 报告 → daxing 浏览器验收（重点：回显当前角色 + 覆写不误删 + 新用户分角色闭环）→ 放行后双推。
- 之后接 **T-008c 角色授权树**（本批最重，压轴）。
