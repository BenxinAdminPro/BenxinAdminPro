# 完成报告：T-007h dept 树选择器 + post 选择器 + 回填（复用 buildTree）+ 岗位页补 sys:post:* 种子

> 状态：**待 PM 评审**（未 commit、未双推、未改 PROJECT_STATUS——按流程铁律）

## 0. 源码核实结论（前置门禁 8 项，逐项带出处）

| # | 核实项 | 结论 | 出处 |
|---|---|---|---|
| 1 | dept 表字段 + 树接口形态 | SysDept：id/parent_id/ancestors/name/sort/leader/status/created_at/updated_at/deleted_at + 非持久化 Children。**`GET /sys/depts/tree` 返回【已嵌套树】**（服务端 buildTree，sort ASC, id ASC），**无任何查询参数**；出参经编码器：id/parent_id 为 hashid，根 parent_id=null（encodeOrZero），children 仅非空时出现 | `rbac/model.go:60-74`、`rbac/dept_service.go:83-89,193-212`、`rbac/handler_dept.go:45,51-55`、`rbac/response.go:68-97` |
| 2 | **buildTree 复用可行性（关键）** | **后端已返嵌套树 → 本片选择器直接消费、buildTree/flattenTree/subtreeIds 实际零调用**（详见下「buildTree 复用实证」节，含参数化够用性验证与如实回报） | 同上 + `admin/src/utils/tree.ts` |
| 3 | post 表字段 + 列表接口 | SysPost：id/code/name/sort/status + 软删。`GET /sys/posts` **扁平分页列表**，仅 page/page_size（page_size 上限 100），**无 search/filter/sort**；出参 id hashid。code 全局唯一（uniqueIndex + 应用层预检 + T-004e 1062 兜底 → ErrPostCodeExists，offset+34，demo 段=11034/409） | `rbac/model.go:81-90`、`rbac/post_service.go:39-51,65-85,101-118`、`rbac/handler_post.go:44,50-56`、`errcode/errcode.go:41,98` |
| 4 | **选择器嵌入点（关键）** | 用户页是 XTable 配置化通用表单，**fields 完全没有 dept_id/post_ids 字段**；XTable 表单字段类型仅 input/password/textarea/number/select，**无自定义控件扩展点**；编辑回填只取**列表行数据**（openEdit 不拉详情），而列表出参不含 posts（List 的 Select 列不含、不加载关联）→ 嵌入+回填**必须**给 XTable 加中立扩展点（见改动说明#1，必要改动非擅扩） | `admin/src/views/sys/user/index.vue`(改前)、`admin/src/components/x-table/types.ts:37-50`、`XTable.vue:157-162`(改前)、`rbac/user_service.go:190-196` |
| 5 | **回填语义（T-003e 兑现·头号正确性点）** | 详情 `GET /sys/users/:id`（sys:user:get）经 svc.Get 手动加载关联岗位；出参 ResponseEncoder.User：**dept_id 为 hashid 或 null；posts 为完整岗位对象数组（id 为 hashid），且仅在非空时出现（空岗位无该字段）**。入参（T-003e 已收口）：update `dept_id` hashid 字符串，**空串=清空部门**（decodeOptionalID）；`post_ids` hashid 数组，**缺省=不变、[]=清空、数组=全量覆写**（decodeIDSlice + Update 先删后插）→ 回填=出参塞回（null→''、posts→map id），提交=hashid 原样回传 | `rbac/user_service.go:146-165,205-235`、`rbac/response.go:26-52`、`rbac/handler_user.go:132-161`、`rbac/decode.go:36-45,75-88` |
| 6 | enforce 权限码逐字 | dept 树：`sys:dept:tree`；post：`sys:post:list/create/update/delete`；用户详情（api.get 回填用）：`sys:user:get`、更新 `sys:user:update`——全部既有路由 RequirePerm，**后端零改动** | `rbac/handler_dept.go:22-27,45`、`rbac/handler_post.go:21-26,43-48`、`rbac/handler_user.go:24-33,59-68` |
| 7 | 岗位 seed 现状 | seed.go **完全没有** 岗位 C 菜单和 sys:post:* F 码（grep "post" 零命中），sys_post 表 0 行——账本「B 部分种子最后一块」属实。另核实：seed 早有 `/sys/dept` C 菜单（`sys/dept/index`）但 `admin/src/views/sys/dept/` **不存在**（T-007b 起的占位降级，非本片引入、部门管理页不在本片 scope，挂观察） | `examples/demo/seed.go:107-164`(改前)、`admin/src/views/sys/` 目录实查 |
| 8 | 用户↔岗位关联 | **多对多**，junction `sys_user_post`（复合 PK user_id+post_id，无软删无时间戳）；提交 post_ids=hashid 数组（见#5）。**岗位 Delete 不检查用户关联**：软删放行、junction 行残留但不再生效（详情按活跃岗位 `id IN` 查询，软删岗位自然消失）→ 删除确认措辞据此写 | `rbac/model.go:97-100`、`rbac/post_service.go:147-153`、`rbac/user_service.go:156-161` |

## 1. 完成状态

全部范围项完成：dept 树选择器组件 + post 多选器组件 + 嵌入用户表单含编辑回填（T-003e 入参 hashid 收口的消费验证闭环）+ 岗位管理页（x-table CRUD）+ seed 增量（岗位 C 菜单 + 4 F 码，幂等）。后端 Go 业务零改动、openapi 不升版（仍 v0.10.0）、错误码零新增、**tree.ts 零触碰**。

## 2. 改动文件清单（10 文件，+97/-8 改动 + 4 新增文件约 +260）

| 文件 | 类型 | 说明 |
|---|---|---|
| `admin/src/components/x-table/types.ts` | 修改 | ① XField.type 增 `'slot'`（业务页经 `#field-<prop>` 作用域插槽提供控件）② XApi 增可选 `get`（编辑回填详情来源）③ XTableConfig 增可选 `delConfirm`（删除确认文案按资源真实行为覆写）。三者全可选、缺省=现状 |
| `admin/src/components/x-table/XTable.vue` | 修改 | ① 表单渲染插槽分支（`f.type==='slot'` → `<slot :name="field-{prop}" :form :disabled>`）② openEdit 异步化：有 api.get 先拉详情填表，**拉失败不开弹窗**（避免残缺回填被全量覆写）③ 数组默认值/回填值拷贝（防多选 v-model 原地改写配置共享引用）④ 删除确认取 config.delConfirm ?? i18n。缺省路径零行为变化 |
| `admin/src/api/dept.ts` | 新增 | DeptNode 类型 + getDeptTree()（嵌套树/hashid 语义注释带后端出处） |
| `admin/src/api/post.ts` | 新增 | PostRow + listPosts/createPost/updatePost/removePost + **listAllPosts()**（选择器用：循环翻页取齐 total，不静默截断；单页上限 100 为后端硬限制） |
| `admin/src/api/user.ts` | 修改 | + getUser(id) 详情 + UserDetail 类型（posts 仅非空出现的语义注释） |
| `admin/src/components/selectors/DeptTreeSelect.vue` | 新增 | el-tree-select 直接消费后端嵌套树；check-strictly（可选非叶子）、clearable；v-model=''/hashid 对齐 decodeOptionalID；停用部门如实标注不隐藏（后端无状态校验，不伪造约束） |
| `admin/src/components/selectors/PostSelect.vue` | 新增 | el-select multiple 消费 listAllPosts；v-model 恒为数组（=全量覆写语义，配合回填自洽）；停用岗位标注 |
| `admin/src/views/sys/user/index.vue` | 修改 | fields 增 dept_id/post_ids 两个 slot 字段 + 两个选择器经 `#field-*` 插槽嵌入；api.get 做出参→表单适配（`dept_id: u.dept_id ?? ''`、`post_ids: (u.posts ?? []).map(p => p.id)`），hashid 全程透传 |
| `admin/src/views/sys/post/index.vue` | 新增 | 岗位管理页：x-table CRUD（code/name/sort/status），permPrefix `sys:post`；列表无 search/sort 诚实降级不伪造；delConfirm 按#8 真实删除行为措辞 |
| `admin/src/layout/components/menuIcon.ts` | 修改 | + postcard → Postcard 映射（岗位菜单图标，沿 T-007e 补映射先例） |
| `server/examples/demo/seed.go` | 修改 | + 岗位管理 C 菜单（/sys/post，sys/post/index，sort 10）+ 4 F 码 sys:post:list/create/update/delete（与 handler_post.go RequirePerm **逐字一致**），幂等 upsert，置于授权循环之前 |

所有新文件带五项头注释（到秒），改既有文件追加 @updated。

## 3. buildTree 复用实证（如实回报）

**结论：本片对 utils/tree.ts 三函数实际零调用——这不是 T-007g 参数化缺口，而是本片场景不需要。**

- **为何不需要**：① 后端 dept 树**已嵌套**（核实#1，与菜单树同口径），选择器零再组装直接消费——强行 flatten→rebuild 是伪消费；② T-007g 的真消费点（父级选择器 subtreeIds 排除自己及子孙防自挂）对应到 dept 侧是**部门管理页的"移动部门"场景**——该页不在本片 scope（任务书 ✅ 范围仅选择器+用户表单+岗位页）。用户表单选部门不存在自挂问题（用户不是树节点）。
- **参数化够用性验证（T-007g 产出的兑现项，正面结论）**：逐字段比对 dept 树出参 —— idKey 默认 `'id'`（dept✓ hashid string）、parentKey 默认 `'parent_id'`（dept✓）、childrenKey 默认 `'children'`（dept✓）、isRoot 默认含 null（dept 根 parent_id=null ✓，菜单同）。**将来部门管理页的父级选择器可用与菜单页逐行同构的 `subtreeIds→flattenTree→buildTree` 管线、零参数调整**——参数化设计被 dept 形态验证为完全够用，无缺口、无需回补。
- **tree.ts 零改动、签名零触碰**（铁律遵守）。DeptTreeSelect 头注释已写明该结论与将来消费点，防后续误判。

## 4. 接口实现情况（全部消费既有接口，后端零改动）

| 用途 | 方法/路径 | 入出参要点 | 权限码 |
|---|---|---|---|
| dept 树（选择器数据） | GET /sys/depts/tree | 出参嵌套树，id/parent_id hashid，根 null | sys:dept:tree |
| post 列表（选择器+岗位页） | GET /sys/posts | 仅 page/page_size（≤100），id hashid | sys:post:list |
| 用户详情（编辑回填） | GET /sys/users/:id | dept_id hashid\|null；posts 仅非空出现 | sys:user:get |
| 用户改（提交回传） | PUT /sys/users/:id | dept_id ''=清空；post_ids []=清空/数组=覆写 | sys:user:update |
| 岗位 CRUD | POST/PUT/DELETE /sys/posts[/:id] | code 唯一 409/11034；删除软删不查关联 | sys:post:create/update/delete |

## 5. 自验结果

- **构建**：`go build ./... && go vet ./...` ✅；`go test ./...` 全绿 ✅（demo/rbac 包另 `-count=1` 强制重跑 ✅）；admin `pnpm build`（vue-tsc -b + vite build）✅。
- **seed 幂等（真库连跑两遍）**：基线 48 菜单/97 授权/80 policy/0 岗位 → 第一遍 **53/107/88**（+5 菜单、+10 授权、+8 p 规则，与"4 码 × 超管+dept_mgr 两角色"精确吻合）→ 第二遍 **53/107/88 不变** ✅。sys:post:* 四码与 handler_post.go 常量逐字比对 ✅。ReloadAll 调用点 seed.go:197（授权循环 174-193 之后）✅。
- **回填闭环（头号正确性点，API 全链路模拟 XTable 提交）**：dept_mgr 编辑 editor 用户——① GET 详情回填基线：dept_id=技术部 hashid、posts 字段缺失（omitempty 语义实证）② 全量 payload 提交改部门→总公司 + 挂岗位 → 200 ③ **重开（再 GET）回显新值精确一致**（dept_id=总公司 hashid、posts=[测试岗位]）④ DB 实查：sys_user.dept_id 落内部 ID 正确、**sys_user_post junction 行真实存在**（user_id=2, post_id=1）⑤ 改回+清空（post_ids=[] 清空语义）→ 回显复原 ✅ **ALL PASSED**。
- **「删被关联岗位」行为实证（delConfirm 措辞依据）**：挂着关联直接 DELETE 岗位 → 200 放行（不拒删）；删后用户详情 posts 即刻消失（软删岗位不再生效）——与#8 源码结论一致，确认文案如实措辞。
- **重名友好码回证**：POST 撞 code → **409/11034**（ErrPostCodeExists，T-004e 兜底链路在 post 实体端到端再回证）。
- **测试产物清理**：editor 复原技术部/零岗位、测试岗位行硬删（活跃 0/总行 0）、junction 0；policy 稳定 88（本片新基线）。

## 6. enforce 正向证据（执行端前置自带）

- **dept_mgr 正向**（非超管、不短路、走真 Casbin，经 allMenuIDs 全量授权 seed.go:191-193）：GET /sys/depts/tree **200**、GET /sys/posts **200**、POST /sys/posts **200**（写）、PUT /sys/posts/:id **200**（写）——含写动作全通。
- **editor 反向**（仅 sys:user:list）：同五端点（tree/list/create/update/delete）全 **403/11009**。
- **policy 增量证据**：80 → 88（+8 = sys:post:* 4 码 × 2 角色 p 规则），增量与种子精确对应；二遍 seed 后 88 不变。
- 超管 200 不计为证据（T-007e 口径）。

## 7. 安全自查

- **hashid 透传不解码**：dept_id/post_ids/选择器值/详情出参全程 hashid 字符串，前端零解码；提交原样回传，后端 decode（T-003e 收口的消费验证，本片即其兑现）。
- **回填防误清**：api.get 失败不开编辑弹窗（XTable 改动②）——否则无回填的全量覆写会静默清空 dept/posts。**连带修复既有潜伏行为**：改造前用户编辑表单不带 dept_id 字段，每次编辑提交都会把部门清成 NULL（update 语义空串=清空）；现编辑必先回填再提交。
- **v-permission 真实码无空值**：岗位页 permPrefix `sys:post` → 内置按钮挂 sys:post:create/update/delete；用户页沿用 sys:user:*。选择器数据接口（sys:dept:tree / sys:post:list）由后端 enforce，无码角色打开弹窗仅得空选项 + 403 toast，前端不伪造数据。
- **删除破坏性**：岗位删除二次确认（delConfirm）按后端真实行为措辞（软删不可恢复、已分配用户即刻不再持有）；不谎称"拒删"也不隐瞒关联影响。
- **不伪造约束**：停用部门/岗位如实展示并标注（后端对 dept_id/post_ids 无状态校验）；岗位列表无 search/sort 诚实降级。
- seed 无密钥/明文密码改动；测试经配置读取密码未打印明文。

## 8. 偏差与上报（如实，未擅改）

1. **【必要改动·超"选择器+页面"清单】XTable 三项中立扩展**（types.ts+XTable.vue，+48/-6）：字段插槽/api.get/delConfirm——核实#4 证明不加扩展点则嵌入与回填无法落地（同 T-007f 请求层 blob 先例：第 0 节预期内的必要前置，非擅扩 scope）。全部可选配置、缺省行为=T-007c 现状，字典/参数/日志/文件/角色页零回归面（未传新配置项即旧路径）。
2. **【上报·既有潜伏缺陷，本片连带修复】用户编辑清空部门**：T-007b 起编辑表单无 dept_id 字段 → 每次编辑静默清空用户部门（update 空串=清空语义）。demo 未暴露因从未用界面编辑过有部门用户。本片回填机制顺势闭合，无需单列切片。
3. **【上报·观察·后端】用户编辑弹窗 status 字段是假能力**：updateUserReq 无 Status 字段（状态变更走独立 PUT /:id/status + sys:user:status），编辑弹窗里改状态提交被静默忽略——T-007b 既有 UI 行为，非本片引入亦未修（修法：前端编辑态禁用 status 或接 status 端点，归后续）。
4. **【上报·观察】seed 既有 /sys/dept C 菜单指向不存在的 `sys/dept/index`**（占位降级显示空白页）——部门管理页缺位，T-007b 起既有。若 PM 排期部门管理页（树表 CRUD+移动），buildTree/subtreeIds 管线与菜单页同构可直接复用（见第 3 节验证）。
5. **【备忘】跨实体 hashid 同值**：内部 id 相同的不同实体 hashid 相同（测试岗位 id=1 与总公司 dept id=1 同为 1wR9wYV8）——单 salt 单 hasher 既有设计（T-003b），无泄漏增量，仅记录。

## 9. git 提交记录

**未提交**（按任务书：diff + 报告回交，待 PM 评审 + daxing 验收 + 放行后才 commit/双推）。工作树：6 修改 + 4 新增（见第 2 节），feature 与账本/任务书文件分离就绪。Clash/gitee DIRECT 规则事项双推时再确认。

## 10. 需 daxing 真人验收（demo 已起在 :8080，新 seed 已生效）

1. **用户编辑回填（重点）**：admin 登录 → 用户管理 → 编辑 editor（技术部）→ 部门选择器**回显"技术部"**、岗位回显其岗位（先到岗位页建 1-2 个岗位再验）；改选部门+岗位 → 保存 → 重开仍是新值。
2. dept 树选择器：层级正确（总公司>技术部/业务部）、能选子部门、清空=无部门。
3. post 选择器：可多选；停用岗位带"（停用）"标注。
4. 岗位管理页：菜单可见（postcard 图标）→ 真点新增/编辑/删除闭环；重名 code 409 中文提示弹窗保留；删除二次确认措辞核对。
5. 新建用户：部门/岗位可选可存，保存后编辑回显一致。
6. editor 登录：岗位菜单不可见；直连 API 403（已自动化实证，可抽查）。dept_mgr 正向可操作。

## 11. 下一步建议

- 放行后双推；账本 T-007h 翻 ✅、待办 B 部分种子条目销项（post 为最后一块，B 部分全清）。
- T-007i vitest 基建（收编 tree.ts 自测脚本）或切 T-005b 后端批次（含两★优先项），由 PM 裁定。
- 部门管理页若排期（§8-4），buildTree 管线零成本复用。
