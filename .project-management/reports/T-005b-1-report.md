# 完成报告：T-005b-1 菜单 CUD 触发 Casbin policy 重载（修 T-003b 安全缺陷·自动联动）+ T-005b-2 搭车修 uploader 恒空

## 1. 完成状态
- **T-005b-1（★安全）**：✅ 菜单写路径 create/update/delete 接 Casbin policy 全量重载联动。改/删 menu_type=F 节点的 perm_code 后 casbin_rule 真跟着变；删/改 F 后 dept_mgr enforce 立即 200→403（权限真收回，修 T-003b「以为关了门其实没关」缺陷）。
- **T-005b-2（★装配静默失败）**：✅ auth.Claims 补 GetSubject() 方法，修 system/handler_file.go 鸭子类型断言静默失败 → sys_file.uploader 真落值、过滤可用。
- 自验：`go build` / `go vet` / `go test ./...` 全绿；新增两支集成测试端到端真跑 ALL PASSED；既有 demo e2e 冒烟 + rbac/system 单测零回归。
- **待 PM 评审 + daxing 验收 + PM 放行**。未 commit、未双推、未改 PROJECT_STATUS.md（铁律遵守）。

## 0. 源码核实结论（第 0 节硬性门禁，7 项带出处；#4/#5 明确）
1. **现有 ReloadAll 实现**（`rbac/policy_sync.go:84`）：全量重建——`enforcer.ClearPolicy()` → 从 `role_menu JOIN role JOIN menu WHERE menu.perm_code != ''` 重建 p 规则（`{role.code, menu.perm_code, "access"}`）→ 从 `user_role JOIN role` 重建 g 规则 → `SavePolicy()`。读 `s.db`（非 tx）。开销 = O(role_menu 行)，菜单 CUD 场景为后台配置态、极小。
2. **现有 ReloadAll 调用点**（grep 全量）：① `role_service.go:184` AssignMenus（SyncRolePerms 失败兜底）② `user_service.go:307` AssignRoles（SyncUserRoles 失败兜底）③ `examples/demo/seed.go:197` 种子末尾。**role AssignMenus 范式**（`role_service.go:151-190`）：提前算 permCodes（读 menu 表，不依赖 tx 内写）→ DB 事务写 role_menu → `SyncRolePerms`（操作 casbin 自有 adapter 连接，**先于** `tx.Commit()`）→ casbin 失败则 `tx.Rollback()` + `ReloadAll()` 恢复内存。
3. **MenuService 结构**（`rbac/menu_service.go`）：原 `{db, errs}` **无 policySync 字段**（坐实 T-007g 观察）。Create 无事务（裸 `db.Create`）；Update/Delete 已各自 `db.Transaction(...)`。装配在 `examples/demo/main.go:271` `NewMenuService(db, ecReg)`。
4. **menu_type=F 与 policy 的精确关系**（`menu_service.go:199 validateMenuType`）：仅 F 持 perm_code 进 policy（M/C 强制空 `ErrMenuPermForbidden`，F 必填 + 全局唯一）。**真需触发重载**：改 F 的 perm_code（旧码 p 滞留/新码失联）、删 F（perm_code 的 p 滞留→收不回）。create 未授角色时实为 no-op；M/C 增删改不碰 policy。
5. **#4 全量 vs 增量决策（明确）**：**选全量 ReloadAll**。依据——(a) 菜单 CUD 后台配置态、极低频；(b) ReloadAll 从 role_menu JOIN menu 真相重建，天然正确、**不会误删其它角色/其它 perm_code 的 policy**（增量须手动算受影响角色集、易引新 bug）；(c) 与 role AssignMenus/user AssignRoles 共用同一 PolicySync 范式，可维护性一致。优化（仅 F-affecting 写触发 / 精准增量）留注释，非必要不做（PM 倾向已采纳）。
6. **#5 事务一致性（明确）**：菜单写在事务内提交后调 `syncPolicy`→`ReloadAll`。**顺序为「先提交菜单、后重载」**——因 ReloadAll 读 `s.db` 不可见未提交 tx，若在 tx 内重载会读到旧 menu 态、重建错误。此顺序的一致性优于 AssignMenus 的 sync-before-commit：ReloadAll 读已提交真相，成功即收敛到与菜单一致，**不存在 casbin 领先于未提交菜单写的窗口**。残留口径同既有 AssignMenus——重载失败则菜单已提交、policy 滞留，返回 error，由下次写或重启 `NewEnforcer→ReloadAll` 幂等恢复（ReloadAll 幂等）。真原子回滚需改 PolicySync 接口签名让其读 tx，超本片 scope，未做。
7. **uploader 缺陷确认**（T-005b-2）：`system/handler_file.go:131-142 getClaimsFromContext` 对 `c.Get("jwt_claims")` 取出的值断言 `interface{ GetSubject() string }`；context 实存 `*auth.Claims`（`rbac/jwt_middleware.go:45` `c.Set(CtxKeyClaims, claims)`，claims 来自 `tokenSvc.Verify` 返回 `*auth.Claims`），而 `auth/claims.go` 的 Claims **只有 Subject 字段、无 GetSubject() 方法** → 断言静默失败返 ""。**最小改法**：Claims 加 `GetSubject()` 方法（指针接收者，满足 `*Claims` 方法集）。全仓仅此一处依赖该断言（grep 实证），零破坏。

## 2. 改动文件清单（路径 + 说明 + 新增/修改）
| 文件 | 说明 | 类型 |
|---|---|---|
| `server/rbac/menu_service.go` | 加 `policySync` 字段 + `SetPolicySync`（同 UserService 范式）+ `syncPolicy` 助手（全量 ReloadAll + 设计/一致性注释）；Create/Update/Delete 在变更提交后调 `syncPolicy`。`NewMenuService` 签名不变。 | 修改 |
| `server/auth/claims.go` | 加 `func (c *Claims) GetSubject() string`（修 uploader 鸭子断言静默失败）。 | 修改 |
| `server/examples/demo/main.go` | `menuSvc.SetPolicySync(policySync)` 注入一行。 | 修改 |
| `server/examples/demo/menu_policy_integration_test.go` | policy 生命周期对照集成测试（建/改/删 F + enforce 200→403），隔离前缀 demomp_/RedisDB8。 | 新增 |
| `server/examples/demo/uploader_integration_test.go` | uploader 落值 + 过滤可用集成测试，隔离前缀 demoup_/RedisDB7。 | 新增 |

> 注：`.project-management/PROJECT_STATUS.md` 在本会话开始前即为 `M`（存量未提交），本片**未触碰**（铁律）。

## 3. 接口实现情况
- **无新端点、不升 openapi（仍 v0.10.0）**；菜单 CUD 路径/权限码不变，行为变化=CUD 后 policy 自动同步。
- **构造签名选择**：MenuService 走 `SetPolicySync` 可选 setter（与 `UserService.SetPolicySync` 同范式），**而非 RoleService 的构造参注入**。理由：`NewMenuService(db, errs)` 签名零变更 → 5 处既有菜单单测 + AuthInfoHandler 复用零回归、policySync nil 时退化原行为。**非对外契约变更**（构造签名未动），仅 demo 装配新增注入一行。
- 行为一致性：syncPolicy 与 role AssignMenus/user AssignRoles 共用同一 PolicySync 接口与「失败返错」口径。

## 4. 自验结果
- `go build ./...` ✅ / `go vet ./...` ✅ / `go test ./...`（单测）✅ 全绿。
- **菜单 policy 生命周期 e2e 对照**（`-tags=integration -run TestMenuPolicyReload`）✅ ALL PASSED（详见第 ② 节）。
- **uploader 落值 e2e**（`-tags=integration -run TestUploaderFill`）✅ ALL PASSED（详见第 ③ 节）。
- **回归**：`-tags=integration ./examples/demo/`（含 TestDemoE2ESmoke 八步 + selfcheck）✅ ok；`-tags=integration ./rbac/` 除 `TestNewEnforcerMySQL_RoleInheritance` 外全绿。
- **该 1 例失败为 pre-existing red**（账本待办 T-003d-fix：T-003b 改 model.conf 为 perm code 精确匹配时遗漏更新的陈旧 URL keyMatch 断言，非真鉴权 bug，与菜单/uploader 无关）。已 `git stash` 我的改动在**干净 HEAD 上复跑实证同样 FAIL** → 坐实非本片引入。

## 5. git 提交记录
- **未 commit、未双推**（流程：diff + 报告 → PM 评审 → daxing 验收 → PM 放行后双推）。
- T-005b-1 与 T-005b-2 同属后端修复、内聚度高，建议**同一 feature commit**（菜单 policy 重载 + claims GetSubject + 两支集成测试），报告/账本另走 docs 与本片分离。提交信息建议：`fix(rbac): T-005b-1 菜单 CUD 联动 Casbin policy 重载（删/改 F 权限真收回）+ T-005b-2 Claims.GetSubject() 修 uploader 恒空`。

## 6. 安全自查
- **policy 重载正确性**：全量 ReloadAll 从 role_menu JOIN menu 真相重建，**不误删其它角色/其它 perm_code 的 policy**（非手动增量删）；改 F → 旧码 p 消失 + 新码进入、删 F → 该 perm_code p 消失，均经 DB 实查 + enforce 翻转双证。
- **事务一致性**：菜单变更提交后重载、读已提交真相，成功即与菜单一致；无 casbin 领先未提交写窗口（优于既有 AssignMenus 顺序）。残留（重载失败）返 error、幂等可恢复，与既有口径同级（第 0 节 #6）。
- **超管短路不受影响**：本片验证用 dept_mgr（非超管/走真 Casbin）做 enforce 对照，超管短路不过 policy，未触及。
- **uploader 不扩权**：GetSubject() 仅返 Claims.Subject，无任何权限放大；落值为上传者自身 Subject。
- **scope 未外扩**：未做手动重载入口（方向 B）、未做 uploader 可读化（日志·8-1）、未碰前端、未做菜单父子类型校验、未动 T-005b-3/4。

## 7. 需 daxing 真人验收（demo 验证项）
- [ ] 菜单管理页：用**自建临时 F 节点**改/删其 perm_code，配合执行端 enforce 对照，确认权限变化符合预期（不再「删了门没关」）。
- [ ] 文件管理页：上传文件 → **上传人列不再为空**（显示上传者 Subject/ID）。
- [ ] 回归：菜单 CUD、文件上传/下载/删除整体不坏。

## 8. 偏差与待办
- **create 对 policy 为 no-op-until-assign**（语义诚实）：新建未授角色的 F 节点 total p 不变（真正增量发生在后续 role AssignMenus）。本片仍统一在 create 调 syncPolicy（写路径一致、为将来「建时挂默认角色」预留），不特判——已在 syncPolicy 注释与测试日志诚实说明，未伪造「建即增 p」。
- **真原子回滚未做**：受限于 ReloadAll 读 s.db 不可见 tx（改接口超 scope）。现取「先提交后重载、失败返错、幂等恢复」口径，与既有 AssignMenus 同级。若 PM 要求强原子，另立切片改 PolicySync 接口让其读 tx。
- **优化留注释**：当前菜单 CUD 一律全量 ReloadAll（含 M/C 写、含 create 的 no-op 重载）。可优化为仅 F-affecting 写触发，已在注释标注，非必要不做。
- **pre-existing red**（T-003d-fix，账本既有待办）：`TestNewEnforcerMySQL_RoleInheritance` 非本片引入，处理顺序铁律仍归该待办切片。

## 9. 下一步建议
- 按账本既定序：T-005b-1+2（本片）→ **T-005b-4（system/日志列表查询能力，前端已备好降级，零改动生效）** → T-005b-3（配置加密参数写链路，体量大）→ 文案/低优批收尾。
- 顺手可清 T-003d-fix pre-existing red（先确认角色继承经 perm code 路径有覆盖，再重写/删旧 URL keyMatch 断言，不许删了凑绿）。

---

## ① 源码核实结论
见上「第 0 节」7 项（#4 全量决策、#5 事务一致性均明确）。

## ② policy 生命周期对照 + enforce 即时生效证据（本片核心）
测试 `TestMenuPolicyReloadE2E`（隔离表前缀 demomp_，dept_mgr 为非超管 enforce 主体，admin/super_admin 做菜单 CUD），真跑日志：

```
baseline OK: dept_mgr GET /sys/posts=200, p(dept_mgr,sys:post:list)=1, total p=85
改 F OK:   old p→0, new p→1, dept_mgr GET /sys/posts 200→403（perm_code 改名权限即时失联）
改回 OK:   p→1, dept_mgr GET /sys/posts 恢复 200
建 F OK:   新建未授角色 F 节点，total p 不变=85（create no-op-until-assign，语义诚实）
删 F OK:   p(dept_mgr,sys:post:list)→0, dept_mgr GET /sys/posts 403（删 F 权限真收回）
ALL PASSED: 建/改/删 F 节点 casbin_rule 真跟着变 + 改/删 F 后 dept_mgr enforce 即时 200→403（T-003b 安全缺陷已修）
```

逐项对照（**区别于 T-007g 观察到的 80→80→80 纹丝不动缺陷态**）：
- **建 F**：total p 维持 85 不变（未授角色 → 全量重载不增 p，no-op-until-assign，诚实）。
- **改 F**（perm_code `sys:post:list`→`sys:post:list_renamed`）：casbin_rule 中 `p(dept_mgr, sys:post:list)` 由 1→**0**，`p(dept_mgr, sys:post:list_renamed)` 由 0→**1**；dept_mgr GET /sys/posts（仍由 sys:post:list 守卫）**200→403**（旧码即时失联）。
- **改回**：恢复 `p(dept_mgr, sys:post:list)`=1，enforce **恢复 200**（双向 + 可恢复）。
- **删 F**（删除 sys:post:list 节点）：`p(dept_mgr, sys:post:list)`→**0**；dept_mgr GET /sys/posts **403**（删 F 权限真收回 = 本片头号安全证据）。

## ③ uploader 修复真跑（落值证据）
测试 `TestUploaderFillE2E`（隔离表前缀 demoup_，dept_mgr 持 sys:file:upload，multipart 字段 file + 显式 Content-Type text/plain 过一致校验），真跑日志：

```
uploader OK:    sys_file.uploader="3"（非空、= 上传者 Subject）
uploader 过滤 OK: GET /sys/files?uploader=3 total=1
ALL PASSED: uploader 真落值 + 过滤可用（T-005b-2 GetSubject() 修复生效）
```

- **DB 实查**：上传后 `demoup_sys_file.uploader` = `"3"`（dept_mgr 的 user.ID 字符串形态 = Subject），**非空**（修复前恒空），且精确等于上传者 Subject。
- **过滤可用**：`GET /sys/files?uploader=3` total=1，证 uploader 落值后过滤链路通（修复前 uploader 恒空 → 过滤实际不可用）。
- 注：本片仅修「恒空」使其落值（值为内部 ID/Subject）；「内部 ID→用户名/hashid 可读化」属日志·8-1，**不在本片 scope**。
