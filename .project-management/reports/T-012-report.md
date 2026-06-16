# 完成报告：T-012 部门管理页（树表 CRUD + 移动节点 / buildTree 首个真消费者）

## 1. 完成状态
执行端编码 + 自测**全部完成，等 PM 评审 + daxing 真人验收 + PM 放行后双推**。
- 纯前端切片，**零后端 Go / 零 seed / 零 openapi（仍 v0.15.0）/ 零 errcode / 零 DDL**，仅消费既有 dept 端点。
- 新增部门管理页，让 seed 早已挂好的 `/sys/dept` 菜单从占位降级页转为真实树表 CRUD + 移动节点页；`buildTree/subtreeIds` 获得 dept 侧首个真实调用者。

## 2. 改动文件清单
| 路径 | 说明 | 类型 |
|---|---|---|
| `admin/src/views/sys/dept/index.vue` | 部门树表 + 工具栏 + 行操作 + 编辑弹窗 + 父选择器 + 双 409 删除确认；强镜像菜单页**砍掉 M/C/F 多态**，静态字段表单 | **新增** |
| `admin/src/api/dept.ts` | 在既有 `getDeptTree()` 上新增 `createDept/updateDept/removeDept` + `DeptPayload` 类型；头注释追加 `@updated` | 修改 |
| `admin/src/layout/components/menuIcon.ts` | **无需改动**——`tree → Share` 映射早已存在（menuIcon.ts:34），任务书"缺则补"前提不成立 | 未动 |

> 精确 stage 范围：`admin/src/views/sys/dept/index.vue` + `admin/src/api/dept.ts`（共 2 文件）。`.project-management/PROJECT_STATUS.md` 为既有改动、属 PM，**不入 stage**；无 config.local.yaml 混入。

## 3. 接口实现情况
`api/dept.ts` 三函数与既有契约逐字一致（对标 menu.ts 范式）：
- `createDept(data)` → `POST /sys/depts`（`sys:dept:create`）
- `updateDept(id, data)` → `PUT /sys/depts/${id}`（`sys:dept:update`）
- `removeDept(id)` → `DELETE /sys/depts/${id}`（`sys:dept:delete`）
- `DeptPayload` 字段集 = `parent_id / name / sort / leader / status`，与后端 `createDeptReq`/`updateDeptReq` 对齐（无 code、无 menu_type）。

页面行为对接命门点：
- **R1 命门（全字段回填/全量提交）**：`openEdit` 回填 `name/leader/sort/status` 四字段全部正确回填，`buildPayload` 全量提交——杜绝漏 leader 静默清空（updateDeptReq.Name 无 binding:required，全量覆写）。
- **R2 移动三态**：`openEdit` 用 `form.parent_id = row.parent_id ?? ''`（null→根、hashid→对应父），不移动即回传当前父=后端 no-op。
- **父选择器 = buildTree 真消费**：编辑态 `subtreeIds(tree, editingId)` → `flattenTree().filter(排除)` → `buildTree()`，逐行镜像菜单页，**默认键零配**。
- **R3 删除双 409**：确认措辞覆盖"子部门 / 归属用户"两种拒删（非只抄菜单页单条）；软删用"移除"不谎称物理销毁；catch return 保留请求层友好 toast。
- **静态表单**：删掉 menu_type radio-group 与 show* 动态显隐及 perm_code/path/component/icon/visible 字段，表单仅余 父级/名称/负责人/排序/状态。
- **安全**：hashid 全程透传不解码；`v-permission` 挂真实码（新增/新增子级=create、编辑=update、删除=delete，与 handler_dept.go:22-26 逐字一致）；前端选择器排除自孙仅 UX，后端 ancestors 校验权威；不消费 ancestors。

## 4. 自验结果
**前端类型 / 构建 / 单测（执行端自动化）：**
- ✅ `pnpm build`（vue-tsc -b && vite build）**exit 0**，类型干净（仅 node_modules vueuse pure-annotation / chunk-size 既有告警，与本片无关）。
- ✅ `pnpm test`：**17 passed (17)**，tree.spec 零改动。

**curl enforce / 后端契约实证（执行端前置自带，吃 T-007e blocker 教训；已 `unset *_PROXY` + `--noproxy '*'` 规避 Clash）：**

enforce 矩阵（dept_mgr ↔ editor）：
| endpoint | dept_mgr | editor |
|---|---|---|
| `GET /sys/depts/tree` | 200 | 403 |
| `POST /sys/depts` | 200 | 403 |
| `PUT /sys/depts/:id` | 200 | 403 |
| `DELETE /sys/depts/:id` | (清理叶子 200) | 403 |

editor 403 body：`{"code":11009,"message":"无权限"}`（后端真把关实证）。

- ✅ **移动防环**：把"总公司"(root) 移到其子"技术部"下 → **400** `{"code":11035,"message":"无效的父部门"}`（ErrInvalidParentDept，后端 ancestors 校验权威）。
- ✅ **删除双 409**：
  - 删"总公司"(有子部门) → **409** `{"code":11032,"message":"部门下有子部门"}`（ErrDeptHasChildren）
  - 删"技术部"(有归属用户 editor/dept_mgr) → **409** `{"code":11033,"message":"部门下有用户"}`（ErrDeptHasUsers）
- ✅ **叶子可删 + 测试数据清理**：删探针叶子部门 → 200，树残留检查无 `__enf_probe`，**工作区部门树恢复原状**（总公司 + 技术部/业务部）。
- ✅ 树出参形态实证：嵌套 children、id=hashid、根 parent_id=null、其余 hashid。

## 5. git 提交记录
**尚未提交、尚未双推**（完成判定 / 双推 / 改 PROJECT_STATUS 权限仅 PM）。
- 待 stage（精确 2 文件，禁 `git add -A`）：`admin/src/views/sys/dept/index.vue`、`admin/src/api/dept.ts`。
- 拟提交信息：`feat(admin): T-012 部门管理页（树表 CRUD + 移动节点 + 父选择器防环 + 双 409 删除；buildTree dept 侧首个真消费者；纯前端零后端改动）`
- 双仓：Gitee 主 + GitHub 镜像；若撞 Clash fake-IP（gitee→198.18.0.x）按账本既有绕法。

## 6. 安全自查
- ✅ hashid 全程透传不解码（树/回填/提交 id、parent_id 均 hashid 字符串，后端 decode）。
- ✅ `v-permission` 三码与后端常量逐字一致；editor 端点 403/11009 = 后端真把关已实证，前端隐藏仅 UX。
- ✅ 移动防环：前端 subtreeIds 仅 UX 预挡，后端 400/11035 权威已实证，前端不冒充安全。
- ✅ 不消费 ancestors（避免依赖裸内部 ID 串，R6）。
- ✅ 未碰 tree.ts（中立工具铁律）、未碰 DeptTreeSelect.vue / user/index.vue（只读消费方零回归）。
- ✅ 无密钥/IP/.env 入改动；config.local.yaml 未触碰。

## 7. 需 daxing 真人验收（demo 验证项）
- [ ] **（R7 先决）** 新建 `views/sys/dept/index.vue` 后若 `/sys/dept` 仍显占位"building"，**先重启 vite dev server**（import.meta.glob 需重扫新文件）再验。
- [ ] 树层级正确：总公司 > 技术部/业务部 缩进/展开折叠正常，无错位/丢失/重复。
- [ ] 真点 CRUD 闭环：新增根部门 + 新增子级（预填父级）各一次，列表真出现。
- [ ] **R1 命门**：给某部门设 leader→保存→重新编辑改其它字段保存→重开确认 leader **未被清空**。
- [ ] **R2 移动**：编辑子部门改父级→树重挂正确（连子部门一起移）；父选择器**看不到该部门自身及子孙**。
- [ ] 删有子部门→提示"部门下有子部门"非 500；删有用户部门（技术部）→提示"部门下有用户"非 500；删干净叶子→行真消失。
- [ ] editor 登录侧边栏无"部门管理"/ 直接访问被拦（后端 enforce 已兜底）。

## 8. 偏差与待办
- **偏差①（无需改 menuIcon.ts）**：任务书 §2 写"缺则补一行 tree 映射"，核实 `tree → Share` 早已存在（menuIcon.ts:34），故未改动——前提不成立，非遗漏。
- **列表新增"创建时间"列**：菜单页无此列，dept 出参含 created_at 故展示，纯只读展示无副作用；不消费 ancestors（守 R6）。
- 无新增后端缺口上报。

## 9. 下一步建议
- PM 评审 + daxing 真人验收（尤其 R1 命门 / R2 移动 / 双 409 三项）通过后，由 PM 放行执行端双推（精确 2 文件 stage）。
- 后续若要"本部门及子部门"数据权限（datascope.go:25 预留 ScopeDeptAndChild），属独立后端片，移动部门届时需设计快照/事件重算（本片已实证当前移动对 data_scope 零副作用，不阻塞）。
- ancestors 裸内部 ID 串出参收口（R6）仍为独立低优后端片，本片前端未消费。
