# 完成报告：T-007d 字典/参数管理页

## 1. 完成状态
已完成（纯前端，未改后端一行；发现后端缺口 4 项已按 T-007c 模式上报，见 §8）。
构建 + 类型检查 0 错误；API 级手测全通过（含两条 409 友好码端到端回证）；浏览器真人验收待 daxing。

## 2. 前置·后端源码核实结果
（依据 `server/system/handler.go` / `model.go` / `dict_service.go` / `examples/demo/seed.go`，以源码为准）

| 项 | 预期 | 源码实际 | 差异 |
|---|---|---|---|
| dict 端点路径 | /sys/dict/types、/sys/dict/data | GET/POST `/sys/dict/types`、PUT/DELETE `/sys/dict/types/:id`、GET/POST `/sys/dict/data`、PUT/DELETE `/sys/dict/data/:id` | ✅ 一致；但 **types 列表仅收 page/page_size，无搜索/筛选/排序参数**；**data 列表无分页**（按 dict_type 过滤返回全量数组，非 Page 包络） |
| config 端点路径 | /sys/configs | GET/POST `/sys/configs`、PUT/DELETE `/sys/configs/:id` | ✅ 一致；列表同样**仅收 page/page_size，无搜索参数** |
| dict/config 字段 | dict_name / dict_label / dict_value / dict_sort / config_name 等 | sys_dict_type：`dict_type, name, status, remark`；sys_dict_data：`dict_type, label, value, sort, status`（**无 remark / css_class / list_class**）；sys_config：`config_key, config_value, name, remark, is_encrypted`（**无 config_type**） | 字段名以源码为准：`name`（非 dict_name/config_name）、`label/value/sort`（无 dict_ 前缀）；前端已按实际字段实现 |
| perm 码（dict） | sys:dict:* | `sys:dict:list / create / update / delete`（类型与数据**共用**同一组码） | ✅ 一致 |
| perm 码（config） | sys:config:* | `sys:config:list / create / update / delete` | ✅ 一致 |
| F 码 seed 完整性 | list/create/update/delete 齐 | dict 4 码 + config 4 码**全部已 seed**（无 T-007c 式缺口）；另 seed 了 `sys:secret:view`（敏感值查看）但**后端无任何端点消费该码** | ✅ 齐；`sys:secret:view` 为悬空码（见 §8-5） |

**加密参数写链路核实（关键）**：`CreateConfigInput` 仅 `config_key/config_value/name/remark`，**无 is_encrypted**；`ConfigService.Update` 用 map 全量覆写 4 字段且**无再加密**。结论：经 `/sys/configs` API ①无法创建加密参数；②对加密行无论"重填"（明文+flag=1 落库 → GetConfig 解密必败）还是"留空"（密文被清空）都会破坏数据。加密写路径只有程序侧 `ConfigCenter.EncryptValue`（seed/e2e 用）。详见 §8-1。

## 3. 改动文件清单
| 文件 | 说明 | 新增/修改 |
|---|---|---|
| `admin/src/api/dict.ts` | 字典类型/数据 API（hashid 透传；data 列表标注数组响应语义） | 新增 |
| `admin/src/api/config.ts` | 参数 API（注释明确加密行 config_value 恒为后端脱敏 ******） | 新增 |
| `admin/src/views/sys/dict/index.vue` | 字典管理页：类型(左)↔数据(右)双表联动；类型 CRUD + 行操作 XAction「字典数据」选中联动；数据 CRUD（dict_type 由适配层注入不进表单）；未选类型空态不发请求；切换类型经 `:key` 重建刷新；#toolbar 显示当前类型；dict_type 列 filterable / created_at 列 sortable（后端未就绪降级，已标注） | 新增 |
| `admin/src/views/sys/config/index.vue` | 参数管理页：列表/新增走 x-table 内置；**#row-actions 插槽**自定义行操作（编辑/删除挂真实 perm 码）；加密行编辑按钮禁用 + tooltip 说明（安全降级，见 §8-1）；明文行编辑走自有弹窗（config_key 禁改）；#toolbar 显示脱敏提示 | 新增 |
| `admin/src/components/x-table/XTable.vue` | 加固：create/update/remove/list 失败 try-catch 兜底（请求层已 toast，失败保留弹窗供修正、不再冒未处理 rejection——重复键 409 场景 Console 干净）；删除确认取消不再抛异常。成功路径零变化，用户/角色页零回归 | 修改（@updated） |

路由/菜单：**前端零改动**——seed 已有 C 菜单 `/sys/dict`→`sys/dict/index`、`/sys/config`→`sys/config/index`，动态路由 glob 自动映射到新建的两个 view 文件。

## 4. 页面/接口对接情况
| 页面/功能 | 对接接口 | 状态 | 备注 |
|---|---|---|---|
| 字典类型列表/新增/编辑/删除 | GET/POST /sys/dict/types、PUT/DELETE /sys/dict/types/{id} | ✅ | dict_type 编辑禁改（唯一键 + 后端改名无级联会孤儿化数据，见 §8-4） |
| 字典数据联动列表/CRUD | GET /sys/dict/data?dict_type=xxx、POST、PUT/DELETE /sys/dict/data/{id} | ✅ | 数组响应由适配层包装为分页包络 + 本地切页展示（数据全量未伪造）；dict_type 由适配层注入 |
| 参数列表/新增/删除 | GET/POST /sys/configs、DELETE /sys/configs/{id} | ✅ | 新增仅明文参数（后端无 is_encrypted 入参） |
| 参数编辑（明文行） | PUT /sys/configs/{id} | ✅ | 自有弹窗，config_key 禁改 |
| 参数编辑（加密行） | — | ⛔ 安全降级禁用 | 后端缺更新加密链路，任何提交都会破坏密文（§8-1） |
| 列筛选/排序 | 参数并入 query 透传 | ⚠️ 降级 | 后端 list 暂无过滤/排序参数，UI 就绪、补参后前端零改动生效（§8-2） |

## 5. 自验结果
- **构建/类型检查**：`npm run typecheck` 0 错误；`npm run build` 成功（仅既有 node_modules rolldown 注释告警与 chunk 体积提示，与本片无关）。
- **API 级手测（真后端 demo + 真 MySQL/Redis，admin token）**：
  - 字典类型 create → 200，id 为 hashid（`1wR9wYV8`）；update（PUT hashid）→ 200；列表分页包络正确。
  - **重复 dict_type 新增 → HTTP 409 + code 11060「字典类型已存在」**（T-004e 端到端回证 ✅，非 500）。
  - **重复 config_key 新增 → HTTP 409 + code 11062「参数键已存在」**（✅ 非 500）。前端请求层对 409 走 default 分支 toast 后端 message，弹窗保留供修正（XTable 已加固）。
  - 字典数据 create×2 → 200；`GET /sys/dict/data?dict_type=t007d_demo` 返回**数组**（适配层设计核实）。
  - 非法 hashid `PUT /sys/dict/types/abc123` → **400 + 11045「无效的 ID」**（防探测语义）。
  - editor token 调 `GET /sys/dict/types` → **403 + 11009「无权限」**（后端 enforce 边界实证）。
- **加密参数 UX**：已向 demo 库手工插入加密样例行 `t007d.secret_demo`（is_encrypted=1），列表返回 `config_value:"******"` 实证脱敏 ✅。代码层证据：加密行编辑按钮禁用 → `openEdit` 仅明文行可达，表单回填的 `config_value` 只能是明文行真值或加密行的 ******（且后者不可达）；全工程无 console 打印参数值路径。
- **字典双表联动**：代码完成（行操作选中 → `:key` 重建数据表；未选空态不发请求）；浏览器逐点验证待 daxing。
- **x-table 增强真页面消费**：#toolbar（两页均用）、XAction 配置行操作（字典页「字典数据」）、#row-actions 插槽（参数页编辑/删除）、列筛选/排序开关（已挂、后端未就绪标注）。
- **既有用户/角色页回归**：XTable 改动仅失败路径兜底，成功路径零变化；typecheck/build 全绿。浏览器回归点验待 daxing。
- **验收样例数据已留库**：字典类型 `t007d_demo`（含 2 条数据）、参数 `t007d.demo_key`（明文）、`t007d.secret_demo`（加密，列表显 ******），daxing 可直接点验后删除（删除本身即 CRUD 闭环验证项）。

## 6. 安全自查（对照任务书第 5 节）
- [x] 加密参数前端无明文展示/打印/回填——后端恒返 ******，前端无任何解密/打印路径；加密行编辑禁用使回填路径不可达
- [x] 行操作 v-permission 传真实码、无空值——`sys:dict:list`（字典数据按钮）、`sys:config:update`、`sys:config:delete`；内置按钮经 permPrefix 自动挂 `sys:dict:*` / `sys:config:*`
- [x] hashid 透传不解码、删除二次确认——id 全程字符串透传；删除均走 ElMessageBox 确认（x-table 内置 + 参数页自有）
- [x] 权限仅 UX，后端 enforce 才是边界——editor 403 已实证；前端不做伪安全

## 7. 需 daxing 真人验收
- [ ] 超管登录 → 字典页双表联动：点「字典数据」加载右表（样例 `t007d_demo` 2 条）；切换类型右表刷新；未选类型空态；类型/数据各真点一次增删改闭环（逐点操作列，不止看按钮）
- [ ] 参数页 CRUD 闭环；`t007d.secret_demo` 显示 ******、其编辑按钮禁用且 tooltip 说明、删除可用；明文行编辑弹窗 config_key 禁改
- [ ] 真触发重复 dict_type / config_key 新增 → toast「字典类型已存在」/「参数键已存在」、弹窗保留、非 500
- [ ] editor 登录 → 当前 seed 下 editor 无 dict/config 菜单（仅 sys:user:list）；如需演示**页面内操作列收窄**，可先用角色管理→给 editor 勾「字典管理」C 菜单 + 仅「字典查看」F 码，再 editor 登录看字典页操作列只剩「字典数据」
- [ ] Console No Issues（含重复键提交场景）；暗色 / i18n 切换 / 响应式（窄屏双卡上下堆叠）不破
- [ ] 验收后可顺手删除 t007d_* 样例数据（即删除闭环）

## 8. 偏差与待办（按 T-007c 模式上报，处置由 PM 裁定，本片未改后端）
1. **【缺陷·后端】加密参数管理 API 写链路缺失**：`CreateConfigInput` 无 `is_encrypted`；`Update` 全量明文覆写、无再加密。任务书「留空=保持原值、重填=替换」语义后端不支持（留空清值、重填存明文配 flag=1 → 解密必败，两路都破坏数据）。前端已安全降级：加密行编辑按钮禁用 + tooltip 说明。需后端补「加密参数更新语义」（如空值=不更新 + 提交值走 EncryptValue 再落库），建议单列后端小切片。
2. **【缺口·后端】dict/config 列表无搜索/筛选/排序参数**：仅 page/page_size。任务书预期"分页+搜索"不成立；本片**未铺假搜索栏**，列筛选/排序开关已挂（参数并入 query 透传，Network 可见），后端补参后前端零改动生效。与 T-007c 排序降级同性质，可同片顺带补。
3. **【缺口·demo】无加密参数种子样例**：seed.go 不建任何 sys_config 行，验收 ****** 无数据可看。本次已手工 SQL 插入 `t007d.secret_demo` 应急；建议 demo seed 补一条加密样例（走 ConfigCenter.EncryptValue 产真密文），挂 demo 侧小改。
4. **【观察·后端】UpdateType 允许改 dict_type 且无级联**：改名会使既有 dict_data 孤儿化。前端已规避（编辑禁改 dict_type），后端是否禁改/级联由 PM 裁定。
5. **【观察】悬空 F 码 `sys:secret:view`**：seed 有"敏感值查看"权限码，后端无任何端点消费。属预留位还是清理，待 PM 裁定。
6. **【既有待办重申】** editor 仅 sys:user:list：页面内操作列收窄演示需临时授权（§7 给了免改种子的操作路径）。
7. dict_data 本地切页展示：后端无分页参数前的过渡（数据全量、仅展示切页，非伪造）；后端补分页后适配层一行切换。

## 9. 下一步建议
- PM 评审本报告 → daxing 浏览器验收 → 放行后贴 git status+diff 待核 → 双推。
- §8-1（加密参数更新语义）+ §8-2（list 搜索/筛选/排序参数）建议合并为一个后端小切片（同属 system list/update 增强），它落地后参数页放开加密行编辑、两页解除筛选排序降级，T-007d 的列筛选才能真正端到端回证。
- 按批次序推进 T-007e 操作/登录日志页（连页带码补 B 部分种子 `sys:operlog:*` / `sys:loginlog:*`，x-table 只读模式首个真消费页）。
