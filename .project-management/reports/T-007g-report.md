# 完成报告：T-007g 菜单树形管理页（buildTree 树形能力 + M/C/F 三类型树 CRUD）

> 状态：**待 PM 评审**（未 commit、未双推、未改 PROJECT_STATUS——按铁律，diff 在工作区待审）

## 1. 完成状态

全部范围内目标已实现：
- ✅ **buildTree 可复用纯函数**落 `admin/src/utils/tree.ts` 独立工具模块（含 flattenTree / subtreeIds），id/parent_id/children 字段名可参数化，不绑 menu 专属字段，防环/防孤儿兜底 + dev 告警，21 项脏数据自测全过（见附录 B）。
- ✅ **菜单管理页** `admin/src/views/sys/menu/index.vue`：el-table 内建 tree 树表 + 工具栏新增/展开折叠 + 行操作（新增子级/编辑/删除）+ M/C/F 三类型动态表单 + 删除二次确认（措辞按后端"拒删有子节点"行为）。
- ✅ **API 模块** `admin/src/api/menu.ts`：tree/create/update/remove，hashid 透传。
- ✅ seed **零改动**（核实#7：菜单管理 C 页 + 4 个 F 码 seed 早已存在）；后端 Go **零改动**；openapi **不升版**（仍 v0.10.0）；错误码零新增。
- ✅ enforce 正向证据**前置自带**（见附录 C）：dept_mgr 四端点含写动作全 200 ↔ editor 全 403/11009；policy 80→80→80 对照。
- ⛔ 拖拽排序**未做**（核实#6：后端无批量重排接口，按任务书不擅造；sort 经表单单节点修改）。

## 2. 改动文件清单

| 文件 | 类型 | 说明 |
|---|---|---|
| `admin/src/utils/tree.ts` | **新增** | 通用树工具：buildTree（扁平→嵌套，防环/防孤儿降级+dev 告警）+ flattenTree（嵌套→扁平，环状输入防护）+ subtreeIds（自身+子孙 id 集）。纯函数、键名参数化，T-007h dept 选择器直接复用 |
| `admin/src/api/menu.ts` | **新增** | 菜单 API：GET /sys/menus/tree（后端已嵌套）+ POST/PUT/DELETE；MenuRow/MenuPayload 类型；parent_id 三态语义注释入文件头 |
| `admin/src/views/sys/menu/index.vue` | **新增** | 菜单管理页：树表直接消费后端嵌套树（零再组装）；三类型动态表单（F 显 perm_code 必填、M/C 隐 perm_code 强制空串提交、M 隐组件、F 隐路由/组件/图标/可见）；父级树选择器编辑时排除自己及子孙（buildTree 真消费点）；删除确认明示"拒删有子节点" |

无修改既有文件；`server/` 全程未动。动态路由经 import.meta.glob 自动映射 `sys/menu/index` → 新 view，路由/菜单/图标（tree-table→Grid 已映射 menuIcon.ts:33）全部零手改。

## 3. 接口实现情况（消费侧）

| 用途 | 方法/路径 | 入参要点 | 权限码 |
|---|---|---|---|
| 菜单树 | GET /sys/menus/tree | 无任何查询参数；返回**已嵌套树** | sys:menu:list |
| 新增 | POST /sys/menus | parent_id hashid（空串=挂根） | sys:menu:create |
| 编辑 | PUT /sys/menus/:id | parent_id 三态：缺省=不移/空串=移根/hashid=移动；**全量覆写** | sys:menu:update |
| 删除 | DELETE /sys/menus/:id | 有子节点 409/11043 拒删；事务内连删 role_menu | sys:menu:delete |

查询能力诚实降级：树接口无 search/filter/sort（svc.Tree 仅 ctx 入参），本页**不挂假搜索**；展示顺序即后端 `sort ASC, id ASC`。

## 4. 自验结果

- `cd admin && pnpm build`（= vue-tsc -b + vite build）✅ 通过。
- `cd server && go build ./... && go vet ./...` ✅（server 零改动，卫生性复验）。
- buildTree 防环自测 21 项 ALL PASSED（附录 B）。
- 后端约束行为真跑实证（dept_mgr，全部友好码非 500）：
  - 删有子节点 → **409/11043「菜单下有子节点」**
  - 建 F 缺 perm_code → **400/11041「按钮权限码必填」**
  - 建 M 带 perm_code → **400/11042「目录/菜单不能设权限码」**
  - 建 F 撞 sys:user:list → **409/11046「权限码已存在」**
  - 伪造 parent hashid → **400/11045「无效的 ID」**
  - 移动成环（A→B→A）与自挂自 → **400/11035「无效的父部门」**（文案"部门"系后端复用 ErrInvalidParentDept，见 §8-2）
- 树 CRUD 真跑：建临时 C 挂根 → 改名/sort → 删除 → 树复原 48 节点；环测 A/B 节点用毕清理，`demo_sys_menu` 活跃行数复原 48，DB 干净。

## 5. git 提交记录

**未提交**（按任务书第 8 节：diff + 报告回交 → PM 评审 → daxing 浏览器验收 → PM 放行后才 commit + 双推）。工作区仅 3 个新增 untracked 文件，与既有文件零冲突。

## 6. 安全自查

- **hashid 透传不解码**：id/parent_id/树选择器值全程 hashid 字符串原样回传；伪造 hashid 后端统一 400/11045（实测）。
- **建树防环**：buildTree 孤儿/自引用/互引环/三元环入链全部降级为根 + dev 告警，不死循环不白屏（附录 B）；flattenTree/subtreeIds 对环状嵌套输入带 visited 防护。后端 ancestors 校验仍是移动防环权威（实测 400），前端选择器排除自己及子孙仅是预拦 UX。
- **v-permission 全真实码无空值**：sys:menu:create（工具栏+行内新增子级）/ sys:menu:update / sys:menu:delete，与 handler_menu.go RequirePerm 逐字一致。
- **删除破坏性**：二次确认明示"角色授权关联一并移除、不可恢复、有子节点后端拒删"；取消/失败各自 return，无未处理 rejection。
- **menu_type 约束不做伪安全**：F 必填 perm_code/M·C 强制空串仅镜像后端 validateMenuType；父子类型后端无约束，前端选择器如实镜像**未擅自加限制**；editor 403 实证后端真把关。
- 密钥卫生：无任何密钥/IP 入新文件；登录测试密码经 shell 变量传递未回显（demo 本地种子密码，config.local.yaml 在 .gitignore 内）。

## 7. 需 daxing 真人验收（demo 验证项）

demo server 已在跑（:8080，本片后端零改动无需重启）；`cd admin && pnpm dev` 后超管登录：
1. 「菜单管理」页树形展示：系统管理(M) > 9 个 C 页 > 各 F 按钮，缩进/展开/折叠全部正常，共 48 节点与 seed 一致、无错位丢失重复。
2. **真点新增**：某目录下加节点 → 出现在正确父级下；表单切 M/C/F 时字段动态显隐（F 显权限码隐路由/组件/图标/可见；M 隐组件；C 全显）。
3. **真点编辑**：改 name/sort/icon → 生效；试把节点移到自己子孙下 → 选择器里根本选不到（已排除）。
4. **真点删除**：删叶子 → 行消失；删有子节点的目录 → 友好提示「菜单下有子节点」非 500。
5. editor 登录：菜单管理不可见；（可选）直连 API 403 已由执行端 curl 实证。
6. ⚠️ 提醒：改/删**现役**菜单（如改 sys:user:list 的 perm_code）会影响所有人且 **policy 不会自动重载**（§8-1），验收请用自建临时节点试增删改。

## 8. 偏差与待办（上报 PM 裁定，本片未擅改）

1. **【观察·后端】菜单 CUD 不触发 Casbin policy 重载**：MenuService 无 policySync（grep 实证；ReloadAll 仅在 role AssignMenus 失败兜底 / user 角色分配 / seed 调用）。policy 行数对照实测 80→80→80。后果：改/删 F 节点的 perm_code 后，casbin p 规则滞留至下次角色授权变更或重启。属 T-003b 既有行为非本片引入；建议并入 T-005b 篮子评估（菜单写路径挂 policy 联动或文档化）。
2. **【观察·文案】菜单父节点错误复用 ErrInvalidParentDept**：移动成环/父不存在时报「无效的**父部门**」（11035），语境是菜单（menu_service.go:67,118,123 复用部门错误）。+1 个菜单专属码或改通用文案即愈，低优。
3. **【确认·设计】父子类型后端无约束**：C 可挂 F 下等组合后端均放行（validateMenuType 只管 perm_code）；越界组合的副作用是 GetUserMenuTree 过滤 F 后子节点孤儿化提根（后端 buildMenuTree 行为）。前端按"不做伪安全"未擅自加限制，PM 若想收紧应加后端校验（前端零改动跟进）。
4. **【说明·实现选择】本页未套 XTable**：XTable 是分页列表形（list 分页包络/内置弹窗扁平表单），与全量树+三类型动态表单形态不合；任务书 §前置依赖 标 T-007c"可选复用"，故按任务书 §2"树表（el-table tree-props）"直用 el-table，XTable 零改动零回归。若 PM 想让 x-table 长出 tree 模式作为底座能力，建议单列小片。
5. **【待办·T-007i】** tree.ts 自测脚本本片为一次性 node 脚本（vitest 基建未建），脚本未入仓；T-007i 落地后收编为正式单测。

## 9. 下一步建议

- daxing 浏览器验收 → PM 放行 → commit（建议 `feat(admin): T-007g 菜单树形管理页（el-table 树表 + M/C/F 动态表单 + buildTree 通用树工具防环）`）+ 双推（注意 Clash/gitee DIRECT 规则）。
- T-007h dept/post 选择器开工即可复用 `utils/tree.ts`（dept 树接口同为后端嵌套返回，subtreeIds 防自挂用法与本页一致）；岗位页连带补 sys:post:* 种子（B 部分最后一块）。
- §8-1（policy 滞留）与 §8-2（文案）并入 T-005b 篮子。

---

## 附录 A：源码核实结论（第 0 节 9 项，逐条带出处）

1. **menu 表真实字段**：`SysMenu`（rbac/model_role.go:41-59）+ DDL（spec/migrations/T003b_sys_menu.sql）：id / parent_id（BIGINT UNSIGNED，默认 0）/ ancestors（'0' 起祖链）/ **menu_type CHAR(1) 字符串 'M'/'C'/'F'**（常量 model_role.go:19-23：M=目录 C=菜单 F=按钮）/ name / perm_code / path / component / icon / sort int / visible int8（1=显示 0=隐藏）/ status int8（0=正常）/ created_at / updated_at / deleted_at 软删。
2. **列表/树接口形态（#2 结论：后端已返嵌套树）**：GET /sys/menus/tree（handler_menu.go:37）→ svc.Tree（menu_service.go:83-89）`Order("sort ASC, id ASC")` 全量查出后**服务端 buildMenuTree 建树**（menu_service.go:234-250），encoder 递归带 children（response.go:154-188）。**无任何查询参数**。→ 前端 buildTree 按任务书该分支定位为"类型适配+工具"：表格直接消费后端树（零再组装、不引入顺序分歧）；buildTree 真消费点 = 父级选择器"flatten→排除自己及子孙→重建"管线，且为 T-007h 备好通用扁平→树能力。
3. **parent_id 语义**：内部根=0（DDL 默认 0）；**出参根=null**（encodeOrZero，response.go:195-200），非根 hashid 字符串。入参 create 走 decodeZeroableID（空串→0=挂根，decode.go:47-54）；update 走 decodeMovableID **三态**（缺省=不移动/空串=移根/hashid=移动，decode.go:56-71；handler_menu.go:76-78 注释）。前端建树根判定 null/''/0/'0' 与此对齐。
4. **menu_type 三类型约束（#4 结论明确）**：validateMenuType（menu_service.go:199-220）——**F：perm_code 必填**（11041）**且全局唯一**（11046）；**M/C：perm_code 必须为空**（11042）。**父子类型无任何约束**（Create/Update 均不检查 parent.MenuType）。**类型可互转**（Update menuType 取入参、空=保持，menu_service.go:97-100；不检查是否有子节点）。Update 为**全量覆写**（updates map 含全部字段，menu_service.go:106-110）→ 前端表单带全字段提交；隐藏字段保留原值（隐藏≠清除），唯 perm_code 在 M/C 下强制空串（后端硬性要求）。
5. **CRUD 接口**：POST /sys/menus、PUT /sys/menus/:id、DELETE /sys/menus/:id（handler_menu.go:38-40）。删除**拒删有子节点**（先 count parent_id=id，menu_service.go:142-147 → 11043），无级联；删除事务内连删 sys_role_menu + 软删菜单（148-151）。移动经 update parent_id（支持），后端 ancestors 串校验防环（112-124 → 11035）。**菜单 CUD 不触发 policy 重载**（MenuService 无 policySync 字段；ReloadAll 调用点 grep 仅 role_service.go:184 兜底 / user_service.go:307 / seed.go:189）——前端不假设，已列 §8-1。
6. **排序能力（#6 结论明确）**：sort 仅经 create/update 单节点改；**无批量重排接口**（handler_menu.go 全部 4 条路由即全集）→ 本片**不做拖拽**，树按 sort 展示 + 表单改 sort 值。
7. **RequirePerm 逐字**：sys:menu:list / sys:menu:create / sys:menu:update / sys:menu:delete（handler_menu.go:37-40）。**seed 已有**菜单管理 C 页（/sys/menu，组件 sys/menu/index，图标 tree-table，sort 4，seed.go:131）+ 4 F 码（seed.go:132-135，与 RequirePerm 逐字一致）→ **本片 seed 零改动**；图标 tree-table→Grid 已映射（menuIcon.ts:33）。
8. **出参 ID hashid**：id 经 encode、parent_id 经 encodeOrZero（根→null）、children 递归编码（response.go:154-188，T-004d/T-003b 收口）；前端 children 关联即用 hashid 字符串。
9. **超管短路影响**：dept_mgr 经 allMenuIDs **全量授权**（seed.go:182-185）含 sys:menu:* → 正向 enforce 对照可做（非超管、不短路、走真 Casbin）；editor 仅 sys:user:list（seed.go:175-180）→ 天然 403 反向对照。

## 附录 B：buildTree 防环自测（esbuild 转译 tree.ts → node 真跑，21 项 ALL PASSED）

| 用例 | 输入 | 兜底行为（实测） |
|---|---|---|
| 正常树 | hashid id、根 parent_id=null（对齐后端出参） | 1 根/嵌套正确/**同层保持输入顺序**/输入数组未被改写（纯函数）|
| 孤儿 | parent 指向不存在节点 | 降级为根 + dev 告警「父节点 GONE 不存在（孤儿）」；孤儿的子正常归位；**节点不丢** |
| 自引用环 | a.parent=a | a 降级为根 + 告警；a 的子正常归位；不死循环 |
| 互引环 | a→b→a + 环外子挂 a | **环成员全部降根**（各自告警）；挂环成员下的子不丢；总数不丢 |
| 三元环入链 | a→b→c→a + x→a | 环三员全降根；入链节点 x 归位降根后的 a 下 |
| 键名参数化 | idKey/parentKey/childrenKey 自定义 + 数值 0 根 | 正确建树（dept/任意实体可复用）|
| 环状嵌套输入 | children 互含的环 | flattenTree/subtreeIds visited 防护，不死循环、去重 + 告警 |
| 空输入 | [] | 空树/空数组 |

运行方式：`esbuild src/utils/tree.ts --define:import.meta.env.DEV=true` → node 断言脚本（/tmp 一次性，未入仓，T-007i 收编 vitest）。控制台输出 21×PASS + 预期 dev 告警逐条出现，`ALL PASSED`。

## 附录 C：enforce 正向证据（执行端前置自带）

种子授权链路：4 个 sys:menu:* F 行 seed 早已入表（本片无 seed 增量，无新授权动作）；dept_mgr 经 allMenuIDs 全量授权（seed.go:182-185）；policy 由历史 ReloadAll 已建（基线 80 条，与 T-007f 归档一致）。

**dept_mgr 正向**（非超管、不短路、走真 Casbin，含写动作）：
| 端点 | 结果 |
|---|---|
| GET /sys/menus/tree | **200**，48 节点（根=系统管理，parent_id=null）|
| POST /sys/menus（建临时 C 挂根） | **200**，返 hashid id，parent_id=null |
| PUT /sys/menus/:id（改名+sort） | **200** |
| DELETE /sys/menus/:id | **200**，树复原 48 节点 |

**editor 反向**（仅 sys:user:list）：同四端点逐个实测全 **403 {"code":11009,"message":"无权限"}**。

**policy 行数对照**：demo_casbin_rule 基线 **80** → dept_mgr 建临时 F（perm_code=t007g:policy:probe）后 **80** → 删后 **80**。三处不变实证：① 本片消费现有码、零 policy 增量诉求；② 菜单 CUD 不触发 policy 重载（§8-1 观察的正面证据）。超管 200 不计为证据（T-007e 口径）。测试产物全部清理，demo_sys_menu 活跃行数复原 48。
