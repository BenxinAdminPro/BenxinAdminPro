# 完成报告：T-007e 操作/登录日志管理页（x-table 只读模式首个真消费页 + 日志菜单/F 码种子补齐）

> 状态：执行端实现完毕 + 自测全绿，**待 PM 评审 + daxing 浏览器验收 + PM 放行后才双推**（T-006 流程铁律）。**尚未 git commit**——diff 已随报告回交待核。

## 0. 源码核实结论（第 0 节 8 项逐条带出处）

| # | 核实项 | 结论 | 出处 |
|---|---|---|---|
| 1 | 两表真实字段 | `SysOperLog`：id/operator/method/path/perm_code/ip/user_agent/req_summary/result_code/latency_ms/created_at（**无软删、无 updated_at、无 oper_ 前缀**）。`SysLoginLog`：id/username/ip/user_agent/success(int8)/reason/created_at。DDL 与 model 逐列一致 | `server/system/model.go:77-89、96-104`；`server/spec/migrations/T004a_sys_oper_log.sql`、`T004a_sys_login_log.sql` |
| 2 | 列表接口与查询参数 | `GET /sys/logs/oper`：仅 `page`/`page_size`/`operator`（**精确匹配 `operator = ?`**）。`GET /sys/logs/login`：仅 `page`/`page_size`/`username`（精确匹配）。**无 DTO 结构体（直接 c.Query）、无排序参数**（固定 `ORDER BY id DESC`）。注意：service 签名有 startTime/endTime 形参但 handler 传 `nil, nil`——**时间范围过滤未暴露**（见 §8-3） | `server/system/handler.go:73,75,179-185,193-199`；`server/system/operlog.go:170-188,195-207` |
| 3 | 列表出参 | 标准 `Page` 包络 `{list,total,page,page_size}`；**全部字段含长字段 req_summary/user_agent 均在列表行内**（真跑 API 实证）。无裸数组（不需要 dict_data 式适配层） | `server/system/handler.go:184,198`；`server/system/response.go:56-58`；真跑见 §4 |
| 4 | 详情接口 | **不存在 `GET …/:id` 详情端点**（日志仅 4 条路由：两 GET 列表 + 两 DELETE 清理）→ 详情用列表行内数据弹窗，零额外请求 | `server/system/handler.go:72-76`（日志路由全集） |
| 5 | 清理接口形态 | `DELETE /sys/logs/oper`、`DELETE /sys/logs/login`：**无任何入参**，handler 硬编码 `time.Now().AddDate(0,-3,0)` 固定删 3 个月前，返回 `{deleted: 行数}`。**无按 id 删除/无多选语义 → 不触发「x-table 多选回 PM」条款** | `server/system/handler.go:187-191,201-205` |
| 6 | RequirePerm 权限码逐字 | `sys:operlog:list` / `sys:operlog:clean` / `sys:loginlog:list` / `sys:loginlog:clean`（清理是独立码非复用 list） | `server/system/handler.go:73-76` |
| 7 | 出参 ID hashid | 已收口：两列表均走 `h.enc.Page(...)` → `Items` → `Item` 反射编码 ID 为 hashid。真跑实证 `"id": "vVjMNYo4"` 等 | `server/system/handler.go:184,198`；`server/system/response.go:35-58`；真跑见 §4 |
| 8 | 现有 seed 菜单结构 | C 菜单挂 `sysDir.ID` 下（如 dict：`seedMenu(db, sysDir.ID, "C", "字典管理", "", "/sys/dict", "sys/dict/index", "dict", 5)`），F 码为其子节点；幂等键：F 按 perm_code、C 按 name+parent+menu_type FirstOrCreate；**超管/dept_mgr 经 `allMenuIDs` 全量循环授权（在菜单插入之后执行）→ 新菜单自动获授**；editor 仅 `sys:user:list` → 天然 403 实证角色 | `server/examples/demo/seed.go:105-167,178-196` |

**核实后的关键设计落点**：无详情端点→行内数据弹窗；清理无入参→工具栏按钮+二次确认明示「删 3 个月前」；无排序支持→created_at sortable 诚实降级透传；operator/username 过滤真生效（精确匹配）。

## 1. 完成状态
全部范围完成：操作日志页（只读列表+分页+详情弹窗+清理）、登录日志页（只读列表+分页+清理）、seed 增量（2 C 菜单 + 4 F 码，幂等）、菜单图标映射补齐。自测全绿（admin build+类型 / go build+vet+test / seed 幂等真验 / API 端到端含 editor 403）。未 commit、未双推、未动 PROJECT_STATUS。

## 2. 改动文件清单
| 文件 | 说明 | 新增/修改 |
|---|---|---|
| `admin/src/api/operlog.ts` | 操作日志 API（list/clean，hashid 透传，OperLogRow 类型） | 新增 |
| `admin/src/api/loginlog.ts` | 登录日志 API（list/clean，LoginLogRow 类型） | 新增 |
| `admin/src/views/sys/operlog/index.vue` | 操作日志页：x-table readonly + 详情弹窗（行内数据）+ 工具栏清理按钮 | 新增 |
| `admin/src/views/sys/loginlog/index.vue` | 登录日志页:x-table readonly + 工具栏清理按钮（无行操作列） | 新增 |
| `server/examples/demo/seed.go` | +2 C 菜单（/sys/operlog sort7、/sys/loginlog sort8）+4 F 码；@updated 已追加 | 修改（+10） |
| `admin/src/layout/components/menuIcon.ts` | iconMap 补 `document`/`monitor` 映射（此前未知名兜底 Document）；@updated 已追加 | 修改（+4） |

x-table 本片**零改动**（readonly/插槽/筛选排序均 T-007c 已有，本片纯消费）；前端路由零手改（glob 自动映射新 view）。

## 3. 接口实现情况（消费侧）
| 用途 | 方法/路径 | 参数 | 权限码 | 实现 |
|---|---|---|---|---|
| 操作日志列表 | GET /sys/logs/oper | page/page_size/operator(精确) | sys:operlog:list | XTable api.list；operator 列 filterable（真生效）、path 列 filterable（降级透传）、created_at sortable（降级透传） |
| 操作日志详情 | （无端点） | — | —（列表数据已在手） | 行操作「详情」弹窗直显行内数据（req_summary pre-wrap 全文 + UA + 耗时/结果码） |
| 操作日志清理 | DELETE /sys/logs/oper | 无（后端固定删 3 个月前） | sys:operlog:clean | 工具栏按钮 + 二次确认明示范围 + 成功显示删除条数 + 刷新；取消/失败各自 return 兜底 |
| 登录日志列表 | GET /sys/logs/login | page/page_size/username(精确) | sys:loginlog:list | XTable api.list；username filterable（真生效）、created_at sortable（降级） |
| 登录日志清理 | DELETE /sys/logs/login | 无（同上） | sys:loginlog:clean | 同操作日志清理模式 |

## 4. 自验结果
- **admin**：`pnpm build`（含 `vue-tsc -b` 类型检查）✅（警告均为既有 chunk-size/三方注解提示）
- **server**：`go build ./... && go vet ./... && go test ./...` 全绿 ✅（demo 包测试 3.3s ok）
- **seed 幂等真验（真库连跑两遍）**：基线 37 菜单/75 授权 → 第一遍 43/87（+6 菜单 = 2C+4F；+12 授权 = 6 菜单 × 超管/dept_mgr）→ 第二遍 **43/87 不变** ✅；中文菜单名落库正确（操作日志/登录日志）
- **权限码双向 grep 比对**：后端 `RequirePerm` 4 码 ↔ seed 4 码 ↔ 前端消费码（clean×2 挂 v-permission）**逐字一致** ✅（list 码由 C 菜单授权控制菜单可见性，前端不重复挂）
- **API 端到端（demo 真跑）**：
  - admin 登录 → GET /sys/logs/login：**真数据 38 条**、id 为 hashid（`vVjMNYo4`）、Page 包络 ✅
  - GET /sys/logs/oper：**真数据 41 条**（中间件历史采集，含 T-007d 验收期间的 409 记录——req_summary/result_code/latency_ms 全字段在列表行内）✅
  - `operator=` 过滤参数后端真消费 ✅（但语义见 §8-1）
  - **editor 4 端点全 403** ✅（GET/DELETE × oper/login，后端真 enforce）
  - admin 清理两端点 200 `{deleted:0}` ✅（当前无 3 个月前数据，端点通、现有数据安全）
- Console 兜底：清理取消/失败各自 return（T-007d XTable 加固模式），无未处理 rejection 路径

### §4 补证：dept_mgr 正向 enforce + ReloadAll 调用点 + policy 行数变化（PM 评审回执要求，A+B 双做）

**背景（PM 指正成立）**：原 §4 两条 enforce 证据均为「天生该绿」的反向/短路证据——超管清理 200 走服务端短路不过 Casbin（demo `super_admin_roles: ["super_admin"]`），editor 403 是天生无码。核心链路第 3 段「ReloadAll 把新 F 码灌进 policy」此前未证。

**B·ReloadAll 调用点与生成机制（seed.go / policy_sync.go 行号）**：
- 调用顺序：seed.go 菜单/F 码插入（106-156，本片新增 149-156）→ 超管/dept_mgr 经 `allMenuIDs` 全量授权（159-176）→ **`ps.ReloadAll(ctx)` 于 181 行、在全部授权之后执行** ✅
- 生成机制：`rbac/policy_sync.go:84` ReloadAll = `ClearPolicy()` → 从 `role_menu JOIN role JOIN menu WHERE perm_code != ''` 重建 p 规则（role_code, perm_code, access）→ 从 `user_role` 重建 g 规则 → SavePolicy。casbin_rule 表完全由 DB 派生、可再生。

**B·policy 行数变化（真库实测，删除-再生对照）**：
| 步骤 | demo_casbin_rule 总行数 | 新 4 码 p 规则 |
|---|---|---|
| 当前（已含本片）| 72 | 8 条（4 码 × super_admin/dept_mgr）|
| 手工删除 8 条新码行 | **64** | 0 条 |
| 重启 demo（seed → ReloadAll）| **72** | **8 条全部再生** ✅ |

8 条逐字：`p/{super_admin,dept_mgr}/{sys:operlog:list, sys:operlog:clean, sys:loginlog:list, sys:loginlog:clean}/access`。editor 无任何日志码 p 规则（与 403 一致）。

**A·dept_mgr 正向 enforce 实测（非超管、不短路、走真 Casbin）**：
dept_mgr 经 `allMenuIDs` 授到全部 4 个日志码（含两个 clean 码，非仅 list）。API 级实测 4 端点：
| 端点 | 结果 |
|---|---|
| GET /sys/logs/oper | **200**（code:0, total:45）✅ |
| GET /sys/logs/login | **200** ✅ |
| DELETE /sys/logs/oper（clean）| **200** `{deleted:0}` ✅ |
| DELETE /sys/logs/login（clean）| **200** `{deleted:0}` ✅ |

**结论**：与 editor 全 403 形成完整正反对照——「新码真进 policy + 非超管授权角色被 Casbin 正确放行 + 无码角色被拒」三段链路全部坐实。

## 5. git 提交记录
**尚未提交**（按流程：PM 评审 → daxing 验收 → PM 放行 → 双推）。拟提交信息：
`feat(admin): T-007e 操作/登录日志页（x-table 只读真消费 + 详情弹窗 + 清理确认）+ seed 补日志菜单/F 码`
注意:工作区另有 PROJECT_STATUS.md 既有未提交存量（账本卫生待办，非本片所改），提交时不混入。

## 6. 安全自查
- hashid 透传不解码（行 id 仅展示用；清理无 id 入参）✅
- v-permission 全挂真实码无空值（clean×2）；「详情」按钮不挂码=公开按钮语义（XAction 无 perm 不挂指令，T-007c 机制），数据来源即 list 响应，门槛在后端 list enforce ✅
- 清理破坏性操作：二次确认明示「删除 3 个月之前全部 + 不可恢复」+ 独立清理权限码（非 list 码）✅
- 脱敏数据原样展示不还原不打印：req_summary 后端 Sanitize（T-004a）产物直显;前端无 console 输出 ✅
- enforce 边界实证：editor 4 端点 403（见 §4），前端隐藏仅 UX ✅

## 7. 需 daxing 真人验收（浏览器）
- [ ] 超管登录：「操作日志」「登录日志」出现在系统管理菜单（sort 7/8，图标 Document/Monitor）、列表真数据、分页可翻
- [ ] 操作日志逐行点「详情」：请求摘要/UA/耗时/结果码完整显示；找一条含敏感字段的记录确认显示 ***（如登录相关 POST 的 password）
- [ ] operator 列筛选：输入用户 ID（如 `1`）真过滤（**注意是 ID 不是用户名**，见 §8-1）；path 筛选/时间排序点了不报错（降级）
- [ ] 登录日志:username 列筛选输用户名（如 `admin`）真过滤；成功/失败列显示正确（可故意输错密码制造失败行）
- [ ] **真点一次清理**（两页各一次）:二次确认弹窗 → 确定 → 提示「清理完成，删除 N 条」→ 列表刷新（当前数据均 3 个月内,预期 N=0、数据不丢）
- [ ] editor 登录：两日志菜单不可见；直连 `/sys/logs/oper` API 得 403
- [ ] （PM 评审回执追加）dept_mgr 登录：两日志菜单可见、列表可看、清理可用——真人侧正向对照（主要正向证据以执行端 API 级补证为准，见 §4 补证）

## 8. 偏差与待办（上报 PM 裁定，本片未擅动）
- **§8-1【发现·可用性+暴露】operator 存内部用户 ID 而非用户名**：demo 装配 subjectFn 返回 `claims.Subject`（`main.go:238-244`），操作日志 operator 落库/出参为 `"1"` 这类内部自增 ID 字符串——① 列表可读性差（看不出谁操作）；② 按用户名过滤不可用（精确匹配 ID 才命中）；③ 与「对外不暴露内部自增 ID」（T-004d 精神）相悖。改法在后端/装配（存 username 或出参 hashid 化），归 PM 裁定；前端已挂诚实注释，后端改后零改动生效。
- **§8-2【缺口·同 T-005b §8-2 同类】日志列表无 search/sort**：仅 page/page_size + 单字段精确过滤;无模糊匹配、无时间范围、无排序（固定 id DESC）。前端 filterable/sortable 开关已挂、参数透传，后端补参后零改动生效。可并入 T-005b 打包。
- **§8-3【观察】时间范围过滤后端半成品**：`ListOperLogs` service 签名已有 startTime/endTime 形参，handler 传 `nil, nil` 未暴露（`handler.go:183`）——后端补两个 query 参数即可点亮，属 §8-2 最低成本子项。
- **§8-4【观察】清理粒度固定 3 个月硬编码**：handler 写死 `AddDate(0,-3,0)`,不可配置不可指定天数。够用但将来或需参数化（带入参校验），归 PM 评估。
- **§8-5【低优】登录日志 user_agent 为空**：API 直连(curl)无 UA 时落空串，浏览器登录正常；非缺陷仅备忘。

## 9. 下一步建议
- daxing 浏览器验收（§7 清单,吃 T-007b/c 教训逐项真点）→ PM 放行 → 双推。
- 下一片按既定序 T-007f 文件管理页（连页带码补 sys:file:*）；§8-1/8-2/8-3 可并入 T-005b 后端打包切片。
