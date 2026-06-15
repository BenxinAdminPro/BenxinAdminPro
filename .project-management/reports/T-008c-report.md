# 完成报告：T-008c 角色授权树

## 1. 完成状态
**执行端自验全绿，待 PM 评审 + daxing 真人验收。** 角色管理页新增「分配菜单」el-tree 授权树弹窗（check-strictly 独立勾选 + 回填全量已授 + 全量覆写提交），后端搭车给 `GET /sys/roles/:id` 补 `menu_ids` 回填来源。**写端点 `PUT /sys/roles/:id/menus` + Casbin `SyncRolePerms` 联动自 T-003b 已完整存在，本片零改动**（区别于 T-005b-1 的菜单 CUD 需补联动）。「用户挂角色（T-008b）+ 角色挂菜单（T-008c）」权限体系两段闭环成型。demo 已重启编译、冒烟确认 `GET /:id` 返 `menu_ids`。**未双推、未改 PROJECT_STATUS（守 T-006 铁律）。**

## 2. 改动文件清单

| 文件 | 说明 | 新增/修改 |
|---|---|---|
| server/rbac/model_role.go | SysRole 加非持久化 `MenuIDs []uint64 gorm:"-" json:"menu_ids,omitempty"`（详情预载，授权树回填来源） | 修改 |
| server/rbac/role_service.go | `Get` 在 First 后 `Pluck("menu_id")` from SysRoleMenu 回填全量 menu_ids（M/C/F 三层，不过滤类型）；List 不载 | 修改 |
| server/rbac/response.go | `enc.Role` 在 MenuIDs 非空时输出 `menu_ids` hashid 数组（List omitempty 零污染，同 User.roles 范式） | 修改 |
| server/spec/openapi/openapi.yaml | version 0.12.0→0.13.0；SysRole schema 加 menu_ids（hashid 数组、详情 only、含 M/C/F 说明） | 修改 |
| server/examples/demo/role_assign_menus_integration_test.go | T-008c e2e：回填全量+覆写往返+enforce 正向+policy 生命周期全链路+无敏感字段 | 新增 |
| admin/src/api/role.ts | +`getRole`（详情含 menu_ids）+`assignRoleMenus`（PUT :id/menus 全量覆写）+RoleDetail 类型 | 修改 |
| admin/src/views/sys/role/index.vue | +「分配菜单」行操作 + el-tree 授权树弹窗（check-strictly/回填/全量覆写/防误清） | 修改 |

## 3. 接口实现情况
- **GET /sys/roles/:id（搭车补 menu_ids）**：复用既有 `sys:role:list` perm，出参加 `menu_ids`=当前全量已授（M/C/F、hashid 数组）。唯一调用点 `handler_role.go:64`，List 走独立 `enc.RoleList` 不预载 → A 方案零污染。
- **PUT /sys/roles/:id/menus（写端点，零改动）**：`sys:role:assign`，入参 `menu_ids` hashid 数组，service 全量覆写（先删后建）+ `SyncRolePerms` 联动 Casbin p 规则（`role_service.go:151-190` 自 T-003b 已有）。
- **GET /sys/menus/tree（树来源，零改动）**：`sys:menu:list`，全量 M/C/F 嵌套树，前端复用 T-007g 的 `getMenuTree`。
- **policy_sync 零改动**：角色授菜单自 T-003b 已联动（非 T-005b-1 菜单 CUD 缺口）。

## 4. 自验结果
- **后端** `go build ./... && go vet ./... && go test ./...` 全绿（rbac 2.34s，全包 ok）。
- **e2e 集成测试**（`-tags=integration`，真 MySQL+Valkey）`TestRoleAssignMenusE2E` **PASS**：
  - ① 回填全量(M/C/F)：GET /:id `menu_ids`=实存全集、均 hashid。
  - ② 全量覆写往返不丢/无残留：assign A→Get=A→A'(换一节点)→Get=A'→A→Get=A，每步 GET 出参与 DB `role_menu` 行数精确。
  - ③ enforce 正向：editor PUT menus / GET 详情 **403** ↔ dept_mgr（非超管真 Casbin）**200**。
  - ④ policy 生命周期 + 全链路 enforce：授含 F(sys:user:list) 的集 → `casbin_rule` p=1、绑该角色探针登录 GET /sys/users **200**；覆写去该 F → p=0、探针重登录同端点 **403**（SyncRolePerms 真驱动、可收回）。
  - ⑤ enc.Role 加 menu_ids 后无 `password_hash`、menu_ids 均 hashid。
  - 同跑 `TestAssignRolesE2E`、`TestMenuPolicyReloadE2E` 仍 PASS（无回归）。
- **前端** `pnpm build` exit 0；`pnpm test` 既有 **17 项全 PASS**（tree.ts 零改动）。
- **demo 重启冒烟**（fresh 二进制，proxy unset/no-proxy opener 绕 Clash）：admin 登录 → `GET /sys/roles/{super_admin}` 返 `menu_ids` **54 条全 hashid**、无 password_hash；**List 行无 menu_ids**（omitempty 零污染坐实）。

## 5. git 提交记录
- 本地 feature commit：`f88d3c7 feat(rbac): T-008c 角色授权树 — el-tree 勾选授菜单 + GET /:id 搭车补 menu_ids 回填`（7 文件 +160/-6）。
- **未双推**：待 PM 评审 → daxing 验收 → PM 放行后再双推（Gitee 主 + GitHub 镜像）。未改 PROJECT_STATUS。

## 6. 安全自查
- **全量覆写不丢权限**：回填返当前全量 menu_ids（M/C/F）；check-strictly 下「回填集≡勾选集≡提交集」恒等，集成测试②坐实覆写往返 DB 行数精确无残留。
- **防误清**（吃 T-007h 教训）：开弹窗 `Promise.all([getRole, getMenuTree])` 任一失败不开弹窗、不提交，杜绝空树/残缺回填被覆写清光。
- **hashid 透传不解码**；menu_ids 出参经 encoder hashid 化（守 T-004d，不裸 uint64）。
- **enforce 边界**：editor 端点 403 实证后端真把关，前端隐藏仅 UX、不做伪安全。
- **无敏感字段**：enc.Role 加 menu_ids 后断言无 password_hash（角色本无密码）。
- 提交前 `git status`+`diff --stat` 自查、`config.local.yaml` 确认 gitignore 覆盖、diff 无密钥/IP。

## 7. 需 daxing 真人验收（demo 已重启冒烟确认 menu_ids）
- [ ] **回填全量→改一处→提交不误删其他权限**（头号项）：选多授权角色 → 开授权弹窗确认当前全部已勾 → 只改一个节点 → 提交 → 重开确认仅那一处变、其余全在。
- [ ] **editor 补验闭环（兑现 T-008b）**：给 editor 角色授「系统管理(M)+用户管理(C)」配对菜单 → editor/temp05 无痕登录 → 侧边栏从空（原只工作台）变出该菜单。
- [ ] **check-strictly 行为可见**：勾页面 C 不自动勾父 M（独立勾选）；按 M+C 整片勾时侧边栏目录嵌套正常。
- [ ] **防误清**：模拟回填/载树失败弹窗不开（不出现空树可提交）。
- [ ] **enforce**：editor 角色页无「分配菜单」按钮或点击 403 ↔ dept_mgr 可正常授权。
> 提示：demo 当前已跑 fresh 二进制（:8080）。改后端 Go 已重启生效（前端热更新无需重启）。

## 8. 偏差与待办
- **check-strictly 已知 UX（非 bug，PM 选型已定）**：勾页面 C 不自动勾父目录 M；只勾 C 不勾 M 时该页在侧边栏平铺到顶层（`GetUserMenuTree` 仅渲染 role_menu 里的 M/C，孤儿 C 落根 `buildMenuTree`）。不丢权限、仅丢目录分组；按目录整片勾即正常嵌套。「授页面隐含授目录」若要做属后端+spec 独立决策，本片前端不偷做（已在页头注释 + openapi 说明）。
- **ancestor 自动补全**：未做（§2 不包含）。若将来要消除上述 UX 代价，作有意识增强独立评估。
- **写路径/policy_sync 零改动**：确认角色授菜单联动自 T-003b 已完整，无新增缺口。
- pre-existing 诊断（getRolePermCodes/userRoleTable 等 unusedfunc 提示）为既有，非本片引入，零触碰。

## 9. 下一步建议
- T-008c 收官后「用户/角色管理补缺片」核心闭环（改密/分角色/授菜单）齐活；建议 PM 评估补缺片是否还有遗留（如部门管理页），或转回 T-005b 后端债批次剩余项。
- x-table「列内自定义单元格控件」诉求已第四次（T-008a/b 的 status/角色列 + 本片若未来要列内授权预览）——可专门排一个 x-table 单元格插槽基建片统一了结。
