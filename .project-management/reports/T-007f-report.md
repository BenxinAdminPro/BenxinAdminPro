# 完成报告：T-007f 文件管理页（x-table 消费 + 鉴权上传/下载 + 软删 + seed 补 sys:file:* 菜单与权限码）

> 状态：执行端实现完毕 + 自测全绿，**待 PM 评审 + daxing 浏览器验收 + PM 放行后才双推**（T-006 流程铁律）。**尚未 git commit**——diff 已随报告回交待核。

## 0. 源码核实结论（第 0 节 9 项逐条带出处）

| # | 核实项 | 结论 | 出处 |
|---|---|---|---|
| 1 | sys_file 真实字段 | `SysFile`：id/original_name/storage_key/storage_type/size/**mime**/ext/uploader/status(0=正常 1=待清理)/created_at/updated_at/deleted_at(`json:"-"` 不出参)。**字段名是 `mime` 非 content_type、无 file_ 前缀**；DDL 与 model 逐列一致 | `server/system/model_file.go:18-31`；`server/spec/migrations/T004b_sys_file.sql` |
| 2 | 列表接口 | `GET /sys/files`：仅 `page`/`page_size`/`uploader`（**精确匹配 `uploader = ?`**）。无 DTO（直接 c.Query/DefaultQuery）、无排序参数（固定 `ORDER BY id DESC`）、无按文件名/类型过滤 | `server/system/handler_file.go:110-116`；`server/system/file_service.go:115-128` |
| 3 | 列表出参 + storage_key | 标准 `Page` 包络 `{list,total,page,page_size}`；**storage_key 在出参中**（真跑实证），为相对 key `yyyy/MM/dd/uuid.ext`（如 `2026/06/10/019eb149-….txt`），**非绝对路径，无需回 PM** → 按 §5 PM 立场前端列表不展示该列 | `handler_file.go:115`；`response.go:56-58`；真跑见 §4 |
| 4 | 上传接口 | `POST /sys/files`：**单文件** multipart，**字段名 `file`**（`c.FormFile("file")`）。demo 默认值：max_size_bytes=**10485760（10MB）**、白名单 **jpg/jpeg/png/gif/pdf/docx/xlsx/zip/txt**（config.example.yaml 与 config.local.yaml 一致）；Content-Type 校验=声明 MIME 与扩展名期望 MIME 主类型一致（未知扩展名跳过）。**响应体=新建文件完整元信息（id 已 hashid）**。→ 上传 UI 放工具栏插槽 el-upload，**x-table 零扩展、未触发回 PM 条款** | `handler_file.go:41,48-81`；`drivers/storage/upload.go:20-58`；`demo/config.example.yaml:61-66`；`demo/main.go:189-191` |
| 5 | 下载接口 | `GET /sys/files/:id/download`：**挂 RequirePerm("sys:file:download")，在 protected 组内必须带 Authorization 头**；流式返回（c.Stream + Content-Disposition attachment + 原名）。**无 tokenized URL 机制** → 前端唯一正解：axios blob 带 JWT 取流 + 客户端落盘，禁裸 `<a href>`。LocalDriver.URL() 未在任何出参使用，不泄漏文件系统路径 | `handler_file.go:42,84-107`；`demo/main.go:235,279` |
| 6 | 删除接口 | `DELETE /sys/files/:id`：**单条 id（hashid）**，软删元信息 + status=1 + **异步物理清理**（goroutine driver.Delete，失败仅记日志）。**无批量/多选语义 → 未触发「x-table 多选回 PM」条款** | `handler_file.go:44,119-129`；`file_service.go:131-155` |
| 7 | RequirePerm 权限码逐字 | `sys:file:upload` / `sys:file:download` / `sys:file:list` / `sys:file:delete`（**upload/download 是独立码**，非复用 list） | `handler_file.go:41-44` |
| 8 | 出参 ID hashid | 已收口（T-004d）：Upload 走 `enc.Item`、List 走 `enc.Page` 反射编码；:id 入参走 `decodePathID`；demo 装配注入 hashidHasher。真跑实证 `"id": "ldD7gDWQ"` | `handler_file.go:35-36,80,85,115,120`；`demo/main.go:279`；真跑见 §4 |
| 9 | 现有 seed 菜单结构 | C 菜单挂 `sysDir.ID` 下、component 形如 `sys/xxx/index`；loginlog sort=8 → 文件管理 sort=9；F 码按 perm_code FirstOrCreate 幂等；超管/dept_mgr 经 allMenuIDs 全量授权（授权循环在菜单插入后、ReloadAll 在授权后）。图标用 `folder`——**menuIcon.ts 已有映射，本片无需改它** | `demo/seed.go:106-156(原)`；`admin/src/layout/components/menuIcon.ts:38` |

**核实后的关键设计落点**：上传→工具栏插槽 el-upload（单文件、字段名 file、预校验对齐 10MB/9 类扩展名）；下载→axios blob 带 JWT（请求层需透传 blob，见 §2/§8）；删除→单条 XAction + confirm；storage_key 不展示；uploader 过滤真生效、original_name 过滤与 created_at 排序诚实降级透传。

## 1. 完成状态
全部范围完成：文件管理页（x-table 只读基底 + 分页 + 工具栏上传 + 行内鉴权下载/删除二次确认）、请求层 blob 透传（鉴权下载前置依赖）、seed 增量（1 C 菜单 + 4 F 码，幂等）。自测全绿（admin build+类型 / go build+vet+test / seed 幂等真验 / dept_mgr 正向四端点 + editor 全 403 / 上传安全负例 / policy 删除-再生）。未 commit、未双推、未动 PROJECT_STATUS。

## 2. 改动文件清单
| 文件 | 说明 | 新增/修改 |
|---|---|---|
| `admin/src/api/file.ts` | 文件 API（list/upload/download/remove，SysFileRow 类型，hashid 透传；下载 responseType:'blob'） | 新增（57 行） |
| `admin/src/views/sys/file/index.vue` | 文件管理页：x-table readonly + 工具栏 el-upload（预校验对齐后端默认值）+ 行操作下载/删除 | 新增（146 行） |
| `admin/src/request/index.ts` | **blob 响应透传**（鉴权下载流不走包络解包）+ blob 错误体解析友好 message；@updated 已追加 | 修改（+17/-1） |
| `server/examples/demo/seed.go` | +1 C 菜单（/sys/file sort9）+4 F 码（sys:file:*）；@updated 已追加 | 修改（+8） |

说明：request/index.ts 的修改超出任务书"1 api + 1 view"清单预期，但为鉴权下载的**必要前置**——现有响应拦截器把一切响应当 JSON 包络解包，blob 流会被解成 `undefined`。改动最小化（成功路径 4 行透传 + 错误路径解析 Blob 错误体取友好 message），普通 JSON 请求路径零行为变化（用户/角色/字典/日志页回归靠 build+类型+既有页面逻辑未触碰）。x-table 本片**零改动**；menuIcon.ts 零改动（folder 已映射）；前端路由零手改（glob 自动映射）。

## 3. 接口实现情况（消费侧）
| 用途 | 方法/路径 | 参数 | 权限码 | 实现 |
|---|---|---|---|---|
| 文件列表 | GET /sys/files | page/page_size/uploader(精确) | sys:file:list | XTable api.list；uploader 列 filterable（后端真生效）、original_name filterable（降级透传）、created_at sortable（降级透传） |
| 文件上传 | POST /sys/files | multipart 字段名 `file`，单文件 | sys:file:upload | 工具栏 el-upload（v-permission 挂整体）：before-upload 预校验（≤10MB + 9 类扩展名，仅 UX）→ http-request 走统一请求层 → 成功 toast+刷新；失败 return（请求层已 toast），无未处理 rejection |
| 文件下载 | GET /sys/files/:id/download | :id hashid | sys:file:download | axios blob 带 Authorization 取流 → URL.createObjectURL + a.download=original_name 落盘 → revoke；**无裸 href** |
| 文件删除 | DELETE /sys/files/:id | :id hashid（单条软删） | sys:file:delete | XAction confirm（明示"软删元信息+物理异步清理不可恢复"）+ handler + refresh:true；取消/失败各自 return 兜底 |

## 4. 自验结果
- **admin**：`pnpm build`（含 `vue-tsc -b` 类型检查）✅（警告均为既有 chunk-size/三方 PURE 注解提示）
- **server**：`go build ./... && go vet ./... && go test ./...` 全绿 ✅
- **seed 幂等真验（真库连跑两遍）**：基线 43 菜单/87 授权/72 policy → 第一遍 **48/97/80**（+5 菜单=1C+4F；+10 授权=5 菜单×超管/dept_mgr；+8 policy=4 码×2 角色）→ 第二遍 **48/97/80 不变** ✅；中文菜单名落库正确（文件管理/文件列表/上传/下载/删除，utf8mb4 实查）
- **权限码双向 grep 比对**：后端 `handler_file.go` RequirePerm 4 码 ↔ seed 4 码 `diff` 比对**逐字一致** ✅（前端消费：upload 挂 v-permission、download/delete 经 XAction.perm 挂、list 由菜单可见性控制）
- **上传真跑（dept_mgr）**：txt 上传 → 200 + 完整元信息（id=hashid `ldD7gDWQ`、storage_key=`2026/06/10/uuid.txt` 日期分目录+uuidv7）✅
- **上传安全负例**：11MB 超限 → **413 / code 11070**；`.sh` 非白名单 → **415 / code 11071** ✅（四件套生效；message 为裸 i18n key，见 §8-2）
- **下载真跑（dept_mgr）**：blob 取流落盘 → `diff` 与原文件**逐字节一致** ✅；Content-Disposition 带原名
- **删除真跑（dept_mgr）**：200 → 列表消失；DB 实查 `status=1 + deleted_at 非空`（软删语义坐实）；**物理文件已被异步清理**（uploads 目录实查为空）✅

## 5. git 提交记录
**无（按任务书要求先不 commit）**。当前工作区：上述 4 文件改动 + 既有存量（PROJECT_STATUS.md 未提交存量、历史任务书/报告 untracked——均非本片产物，未触碰，归档时按账本卫生条款单独走 docs: 提交）。提交前密钥扫描：本片改动无密钥/IP/证书（seed 仅菜单数据；前端常量仅公开的上传限制值）。

## 6. 安全自查
- **hashid 透传不解码**：行 id/下载 id/删除 id 均字符串原样回传；入参伪造由后端 decodePathID 统一 400。
- **鉴权下载无绕过**：经统一 axios 实例带 JWT（401 自动刷新链路同样生效）；无裸 `<a href>`；objectURL 用后即 revoke。
- **storage_key 最小暴露**：随出参返回但列表/详情 UI 不展示（仅 original_name/ext/mime/size/uploader/时间）；其值为相对 key 非绝对路径，后端是否 mask 仍归 T-004b 加固（不阻塞）。
- **上传安全边界在后端**：前端预校验仅 UX（注释明示），负例实证后端四件套独立把关（413/415）；预校验阈值与 demo 默认值对齐并注明来源。
- **v-permission 全真实码无空值**：upload/download/delete 三码 + 无 perm 的按钮根本不挂指令（T-007c 语义）。
- **删除破坏性**：二次确认明示软删+物理清理不可恢复；XAction confirm 取消路径 return。
- **enforce 边界实证**：见下节——前端隐藏仅 UX，后端真把关。

## 7. enforce 正向证据（PM 要求前置自带）
- **三段链路**：① 4 条 F 行入 `demo_sys_menu`（48 条含 sys:file:*×4，实查）② role_menu 授超管/dept_mgr（87→97）③ **ReloadAll 调用点在授权之后**（seed.go:189，授权循环 166-185）从 role_menu(perm_code≠'') 重建 p 规则。
- **policy 删除-再生对照（强证据）**：casbin_rule **80 → 手删 8 条 `p, *, sys:file:*` → 72 → 重启 ReloadAll → 80 全再生**（实查再生行 = dept_mgr/super_admin × 4 码笛卡尔全集），再生后 dept_mgr list 仍 200。
- **dept_mgr 正向（非超管、不短路、走真 Casbin）**：list **200** / upload **200** / download **200**（内容 diff 一致）/ delete **200** —— 四端点全绿。
- **editor 反向**：同四端点（含直连下载/删除存量 id）**全 403** ✅。正反对照完整；超管 200 不计为证据（走短路，T-007e 口径）。

## 8. 偏差与待办（上报 PM 裁定，本片均未动后端）
1. **【缺陷·后端·T-004b 潜伏】uploader 永远落空串**：`handler_file.go:131-142` 用类型断言 `interface{ GetSubject() string }` 取上传人，但 `auth.Claims` 只有 `Subject` 字段**没有 GetSubject() 方法**（`auth/claims.go:15`），断言恒失败 → 真跑实证出参 `"uploader": ""`。后果：上传人列恒空、uploader 过滤（本片唯一真生效过滤）实际不可用。比 operlog operator 存内部 ID 的缺口（T-005b §日志-8-1）更甚——连 ID 都没存上。修法极小（auth.Claims 加 GetSubject() 方法即可，零破坏），建议并入 T-005b；前端列+过滤已挂、注释诚实归因，后端修后零改动生效。
2. **【缺口·后端】response defaultMessages 缺全部 6 个 file 键**（sys.file_too_large/file_ext_not_allowed/file_type_mismatch/file_not_found/file_name_invalid/storage_failed，`response/render.go:25-53` 实查仅到 config 段）→ 这些错误 message 返回**裸 i18n key**（如 "sys.file_too_large"）。浏览器路径不受阻：前端预校验先行中文拦截（超限/非白名单到不了后端）；curl 直连才见裸 key。+6 行文案即点亮，建议并入 T-005b。
3. **【已做·超清单】request/index.ts blob 透传**：见 §2 说明，鉴权下载必要前置，最小化实现。
4. **【观察·仓库卫生】demo uploads 目录不在 .gitignore**：现 .gitignore 仅 `/server/uploads/`，demo 实际落盘 `server/examples/demo/uploads/`（root_dir "./uploads" 相对 demo 目录）。当前无文件无污染，但 daxing 验收上传后会出现 untracked 文件。**建议加一行 `/server/examples/demo/uploads/`——改 .gitignore 属危险操作，待 PM 确认后落**。
5. **【观察·验收预期】存量行 id=`1wR9wYV8`（t004d.txt，06-09 测试产物）物理文件已不在**（uploads 实查为空）→ 验收点它下载会得 **500 / 11075 storage 失败**（已实测），属既有数据不一致非本片缺陷。建议验收时顺手删掉该行（正好真点一次删除），或忽略。
6. **【沿袭】列表查询能力降级**：original_name 过滤/created_at 排序后端未支持，开关已挂参数透传（同 T-005b §8-2 列表查询能力补齐项），后端补参后前端零改动生效。

## 9. 需 daxing 真人验收（demo 验证项）
demo 已重启在跑（新 seed 已生效，:8080）；admin 起 `pnpm dev` 即可。
- [ ] 超管登录：「文件管理」出现在系统管理菜单（folder 图标，排登录日志后）
- [ ] **真点上传**：选 txt/png 上传 → 列表出现新行、文件名/大小/类型/时间正确；试一个 >10MB 或 .sh → 前端预校验中文拦截（不发请求）
- [ ] **真点下载**：下载刚上传的 → 浏览器落文件、打开内容正确（注意：存量行 t004d.txt 物理文件缺失会报错，见 §8-5，可顺手删它）
- [ ] **真点删除**：二次确认 → 列表刷新消失
- [ ] storage_key/MIME 之外无敏感列裸露（storage_key 不在列表）
- [ ] 列筛选：uploader 过滤注意 §8-1（值恒空，过滤暂不可用属后端缺陷）；original_name 过滤点了不报错（降级）
- [ ] dept_mgr 登录：文件菜单可见、上传/下载/删除可用（API 级正向已自测全绿，浏览器侧复核）
- [ ] editor 登录：文件菜单不可见；直连 /sys/files 得 403
- [ ] Console 干净：上传失败/下载失败/删除取消均无未处理 rejection

## 10. 下一步建议
- 本片放行归档后：T-007g 菜单树形页（建 buildTree 能力）→ T-007h dept/post 选择器（复用 buildTree）→ T-007i vitest（可选）。
- T-005b 篮子再 +2 项（§8-1 uploader 断言缺陷、§8-2 file 文案缺失），连同既有 8 项，后端小切片体量已可观，建议排期时机一并裁定。
- §8-4 .gitignore 一行（PM 点头即落，可与本片归档同走）。
