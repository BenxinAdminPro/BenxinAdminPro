# 完成报告：T-005b-4 system/日志列表查询能力 + 操作人可读化

> 状态：**待 PM 评审 + daxing 真人验收**。未 commit、未双推、未改 PROJECT_STATUS（遵 T-006 铁律）。
> 基线 HEAD = f42dc80。本片纯后端 Go 改动 + 最小前端配套 + openapi v0.11.0。

## 0. 第 0 节源码核实清单（全部带出处，以源码为准）

| # | 核实项 | 结论（出处） |
|---|---|---|
| 0-1 | dict_type 列表 | 原仅 `page/page_size`，固定 `Order("id ASC")`；字段 dict_type/name/status/sort/remark（`dict_service.go:57` 原 ListTypes、`model.go:23`）。**无** name/code 模糊、**无** status 过滤。 |
| 0-2 | dict_data 列表 | **确认属实：无分页、返全量数组** `ListDataByType` 返 `[]SysDictData`、`Order("sort ASC, id ASC")`（`dict_service.go:106` 原）；handler `enc.Items` 返裸数组（`handler.go:114` 原）。前端适配层本地切页（`dict/index.vue` 原 list）。 |
| 0-3 | config 列表 | 原仅 `page/page_size`、`Order("id ASC")`；字段 config_key/config_value/name/is_encrypted/remark（`dict_service.go:173` 原 List、`model.go:53`）。加密行 `maskEncrypted` 脱敏（`dict_service.go:188`）。**无** is_encrypted 过滤。 |
| 0-4 | oper_log 列表 | **确认属实：`ListOperLogs` 已有 `startTime,endTime *time.Time` 形参，handler 传 `nil,nil` 未暴露**（`operlog.go:170` 原 / `handler.go:183` 原）。operator 过滤为**精确等值** `Where("operator = ?", operator)`；排序固定 `Order("id DESC")`。字段与任务书一致（`model.go:77`）。 |
| 0-5 | login_log 列表 | 原仅 `username` **精确等值** + `page/page_size`，`Order("id DESC")`（`operlog.go:195` 原）。**无**时间范围、**无** success 过滤。**确认 login_log 直接存 username**（`loginlog.go:34` `Username: event.Username`），无需可读化。 |
| 0-6 | operator/uploader 存储真相 | **operator/uploader 存内部自增 ID 字符串**：demo `subjectFn` 返 `claims.Subject`（`demo/main.go:239-245`），Subject = user.ID 字符串（uploader 集成测试 `mgrSubject = strconv.FormatUint(mgr.ID,10)` 实证 `demo/uploader_integration_test.go:85`，落值 `"3"`）。与 `sys_user.id`（BIGINT UNSIGNED）JOIN 干净：operator 是十进制 ID 串，`ParseUint` 后按 id IN 查询。**非用户主体**：写动作均经 JWTAuth（claims 必在），GET 被 oper_log 中间件 `ExcludeMethods:["GET"]` 排除（`demo/main.go:253`）→ 极少空 operator，仍以「匿名」兜底。 |
| 0-7 | sys_user 软删与去重 | 软删 = `DeletedAt gorm.DeletedAt`（`rbac/model.go:49`），DDL `deleted_at DATETIME NULL`（`T003a_sys_user.sql:23`）。**JOIN 仅 `SELECT id, username`，绝不取 password_hash**（`query.go:resolveUserNames`）；用 `.Table()` 原生查询并显式 `deleted_at IS NULL` → 软删用户未命中 → 回落「已注销」。 |
| 0-8 | 分页包络 | `enc.Page(list,total,page,page_size)` 返 `{list,total,page,page_size}`（`response.go:56`）。dict_data 新分页复用同一 `enc.Page`，与其他列表同构。 |
| 0-9 | 排序/过滤范式 | rbac 已有 Query 结构 + `form` 标签 + `ShouldBindQuery` 范式（`rbac/user_service.go:50` UserListQuery）；跨表 JOIN 用 `resolveTable(db,model)`/`db.Table(prefixed)` 范式（`rbac/policy_sync.go:100,138`）。本片**沿用 Query 结构范式**，新建白名单排序工具（无现成 sortable 白名单可复用）。 |
| 0-10 | 操作人可读化选型 | 见下方专节，**采用 PM 默认推荐 B**。 |

### 0-10 操作人可读化选型结论：采用 **B（出参 JOIN 解析 username）**

- **B 无阻断**（0-6/0-7 核实）：operator/uploader 是干净的内部 ID 串、与 `sys_user.id` 一一对应，无混入非用户主体；soft-delete 可稳妥用 `deleted_at IS NULL` 区分 →「已注销」。
- **B 的优势兑现**：自动修存量行（无需回写历史日志）、始终最新用户名、已删用户显「已注销」、**零写路径改动**（采集链路 operator 仍存 ID，不动 oper_log 中间件/上传 handler）。
- 实现：出参新增非持久化字段 `OperatorName`/`UploaderName`（`gorm:"-"`，仅随 json），service 批量解析（一页一次查询）；过滤入参语义同步改为「按用户名模糊」（先用户名→ID 集，再 `WHERE 列 IN 集`）。**内部 ID 不新增对外暴露字段**（守 T-004d 精神）。
- 否决 A（写时存 username）：不修存量、用户改名后旧日志陈旧。否决 C（前端映射）：多一跳、前端承重、删用户处理复杂。

## 1. 完成状态
✅ 已完成并端到端真跑通过。dict/config/oper_log/login_log 查询能力补齐 + dict_data 真分页 + operator/uploader 可读化；前端降级注释点亮、列改绑 _name；openapi 升 v0.11.0；集成测试 + enforce 正向证据前置自带；零回归。

## 2. 改动文件清单

### 后端 Go（server/）
| 文件 | 说明 | 类型 |
|---|---|---|
| `system/query.go` | **新增**：列表查询公共工具（LIKE 转义/排序白名单/时间解析/用户名解析与过滤/兜底文案常量） | 新增 |
| `system/model.go` | SysOperLog 加非持久化 `OperatorName`（`gorm:"-"`） | 修改 |
| `system/model_file.go` | SysFile 加非持久化 `UploaderName`（`gorm:"-"`） | 修改 |
| `system/dict_service.go` | DictTypeListQuery/DictDataQuery/ConfigListQuery + ListTypes/ListData(真分页)/Config.List 重写（模糊/排序白名单） | 修改 |
| `system/operlog.go` | OperLogQuery/LoginLogQuery + ListOperLogs(用户名过滤/path/时间/排序 + operator 可读化)/ListLoginLogs(用户名/ip/success/时间/排序) | 修改 |
| `system/file_service.go` | FileListQuery + List(文件名/上传人用户名模糊/排序 + uploader 可读化) | 修改 |
| `system/handler.go` | 各 list handler 构建 Query 结构 + 时间范围解析校验(400) + queryInt8Ptr/parseTimeRange 辅助 | 修改 |
| `system/handler_file.go` | List 构建 FileListQuery | 修改 |
| `system/query_integration_test.go` | **新增**：可读化/用户名过滤/排序/时间范围/注入负例/dict_data 分页（真 MySQL + sys_user） | 新增 |
| `examples/demo/query_enforce_integration_test.go` | **新增**：enforce 正向（dept_mgr 200↔editor 403）+ operator 可读化 e2e + 负例 | 新增 |
| `system/{system,config,file}_*_test.go`、`configcenter_test.go`、`demo/uploader_integration_test.go` | 适配新签名/新参数契约 | 修改 |
| `spec/openapi/openapi.yaml` | v0.10.0→v0.11.0：补各列表查询参数 + operator_name/uploader_name 出参说明 + dict_data 分页形态 + 400 | 修改 |

### 前端 admin（最小必要配套）
| 文件 | 说明 |
|---|---|
| `api/dict.ts` | `listDictData` 改收 params 返分页包络（真分页） |
| `views/sys/dict/index.vue` | dict_data 适配层一行切真分页（去本地切全量）+ 注释更新 |
| `api/operlog.ts` / `views/sys/operlog/index.vue` | OperLogRow 加 operator_name；列 `operator`→`operator_name`、详情绑定改 operator_name；降级注释点亮 |
| `api/file.ts` / `views/sys/file/index.vue` | SysFileRow 加 uploader_name；列 `uploader`→`uploader_name`；降级注释点亮 |
| `views/sys/{loginlog,config}/index.vue` | 降级注释点亮（功能本就零改动跟进，仅诚实修正注释 + @updated） |

## 3. 接口契约（v0.11.0）
- **统一新增查询参数**：`keyword`/资源专属模糊字段、`sort`(白名单)+`order`(asc/desc)、`start_time`/`end_time`(RFC3339 / `2006-01-02 15:04:05` / 纯日期)、dict_data 补 `page`/`page_size`。
- **过滤参数名 = 前端列 prop**（零改动跟进）：oper=`operator_name`、file=`uploader_name`/`original_name`、login=`username`/`ip`、dict=`dict_type`、config=`config_key`。
- **出参新增**：oper_log `operator_name`、sys_file `uploader_name`（值：username | 「已注销」| 「匿名」）。**不新增暴露内部 ID 字段**。
- **dict_data 响应形态变更**：裸数组 → 分页包络（破坏性，前端配套已同步）。
- 时间范围非法 / start>end → 400（复用 `response.BadReq`，**零新增错误码段**）。

## 4. 自验结果（端到端真跑，非仅单测绿）

```
go build ./...                 ✓   go vet ./...                  ✓
go test ./...                  ✓ (全 9 包 ok)
go build/vet -tags=integration ✓
```

**集成测试（真 MySQL :3306）全绿**：
- `system` `TestQueryEnhance_*`（4 支）：
  - **operator 可读化**：id"1"→`alice`；软删用户 id"3"→`已注销`；空 operator→`匿名` ✓
  - **用户名模糊过滤**：`operator_name=ali`→命中 operator="1" 1 条；`nobody-xyz`→空结果（非不过滤）✓
  - **排序**：`latency_ms asc`→5,10,30,50 ✓
  - **排序注入负例**：`sort="id; DROP TABLE x;--"`→被忽略、回退默认 id DESC、仍返 4 条不报错（**绝不进 ORDER BY**）✓
  - **时间范围**：直插 2020/2030 行，`end=2025`→仅 /old、`start=2025`→仅 /new ✓
  - **uploader 可读化 + 用户名过滤**：alice / 软删→已注销 / 空→匿名；`uploader_name=alice`→1 条 ✓
  - **dict_data 真分页**：25 行 → page1 total=25/len=10、page3 len=5、keyword 无匹配=0 ✓
- `demo` `TestQueryEnforceE2E`：
  - **enforce 正向对照**：6 个带新查询参数的列表端点 **dept_mgr（非超管/真 Casbin）全 200 ↔ editor 全 403** ✓
  - **operator 可读化 e2e**：dept_mgr 写动作产生的 oper_log（异步落库轮询）`operator_name == "dept_mgr"` ✓
  - **HTTP 负例**：非法 `start_time`→400 / `start>end`→400 / 排序注入→被忽略返 200（不 500）✓
- `demo` `TestUploaderFillE2E`（适配新契约）：`uploader_name=dept_mgr` 过滤 total≥1 且出参 `uploader_name=="dept_mgr"` ✓

**前端**：`pnpm build`（vue-tsc + vite）exit 0 ✓；`pnpm test`（vitest）17 passed ✓。

**零回归**：T-001~T-005b-1 全部单测/集成绿；未动 DDL/错误码段（response migration 快照测试随 `go test ./...` 通过）；openapi YAML 解析校验通过（ruby YAML.load_file OK）。

**pre-existing red（T-003d-fix）**：`rbac` `TestNewEnforcerMySQL_RoleInheritance` 在 **git stash 干净 HEAD f42dc80 复跑同样 FAIL** → 坐实非本片引入（本片零触碰 rbac）。归该待办切片，**未删断言凑绿**（处理顺序铁律）。

## 5. git 提交记录
**未提交**。等 PM 评审 + daxing 真人验收 + PM 放行后双推（Gitee 主 + GitHub 镜像）。diff 范围见 §2。

## 6. 安全自查（逐项）
- **排序强白名单**：`applySort` 仅把命中白名单的对外字段映射到代码字面量列名，未命中→回退默认 `Order`，**用户输入永不进 ORDER BY**（注入负例集成 + HTTP 双证）。
- **查询全参数化**：模糊走 GORM 占位符 `LIKE ?`；`escapeLike` 对 `% _ \` 转义，杜绝注入与意外全表匹配。
- **时间范围校验**：解析失败/`start>end`→400，不静默吞（`parseTimeRange`）。
- **分页边界**：`normalize()` page≥1、page_size≤100 封顶（沿既有硬限）。
- **JOIN 不泄漏**：`resolveUserNames`/`userIDsByName` **仅 `SELECT id, username`**，不带 password_hash/敏感字段；用户表不存在/查询失败 → 空映射（纯展示增强，不阻断、不 500）。
- **enforce 不旁路**：列表仍挂原 RequirePerm（perm code 不变）；可读化不扩权（dept_mgr 可见范围不变，仅 ID→用户名）；editor 全 403 实证。日志无数据权限收窄（核实确认），可读化不引跨权限泄漏。
- **内部 ID 不外泄**：仅新增 `operator_name`/`uploader_name`（用户名），不新增内部 ID 出参字段（守 T-004d）。

## 7. 需 daxing 真人验收（demo 验证项）
> **⚠️ 验收前置（后端 Go 片必做）**：先确认 demo 用**新代码重启过**（`lsof -ti :8080 | xargs kill -9` 后 `cd server && go run ./examples/demo`，或让我重启）——前端有热更新无需重启，后端 Go 改动不重启会跑旧二进制看不到效果。
- 操作日志页：operator 列显**用户名**（非内部 ID）；模糊搜索（按用户名）/排序（操作时间、耗时）真生效；删用户后旧日志显「已注销」。
- 登录日志页：username/ip 模糊 + created_at 排序生效。
- 字典页：dict_data 真分页（翻页正确、非本地切全量）；字典/参数页 dict_type/config_key 搜索 + 排序生效。
- 文件页：uploader 列显用户名。
- 注：时间范围为**后端能力 + 集成测试已覆盖**；前端暂无日期范围 UI 控件（x-table 无日期范围控件、本片按「不擅扩 x-table」未加）→ 时间范围 daxing 可经 curl 复核，UI 控件归后续前端小片。

## 8. 偏差与待办
- **偏差①（必要前置非擅扩）**：file 的 uploader 过滤随显示列改为**按用户名模糊**（原任务书 §4 仅明确 oper 改用户名过滤）。理由：列显示已改 uploader_name，若过滤仍按内部 ID 则 UI 过滤框不可用——属正确性而非 scope 蔓延。已在 openapi/前端注释标明，请 PM 知会。
- **偏差②**：login_log 顺带补 `ip` 模糊 + `success` 过滤（任务书 §2 列了 username/ip + success）；前端暂无对应筛选 UI（仅 username 列 filterable），属后端能力先行、零改动待前端按需点亮。
- **待办（仍归后续）**：时间范围前端日期范围 UI 控件（需 x-table 扩展或页面自有搜索栏，本片未做）；config 加密写链路 = T-005b-3；文件文案 6 键 / 菜单父子类型 / dict_type 禁改名 = 文案低优批。
- **待办（已存在，未恶化）**：system 出参未强类型化（operator_name/uploader_name 同走 ApiResponse 泛型包络，靠 description 文字约束）——本片沿用，归既有「openapi system 出参强类型化」待办。
- **待办（低优·PM 评审记，可读化按名过滤边缘）**：username→ID 集解析走 `deleted_at IS NULL`（`query.go:121` userIDsByName），**删用户后其旧日志按用户名过滤不到、仅显「已注销」**。属可接受边缘（已删用户的名义本就不该作为活跃过滤维度），低优待办。
- **待办（低优·PM 评审记，数据展示文案 vs 错误码 i18n）**：「已注销」「匿名」现为 system 包集中常量（`query.go:24-27`，单一出口 displayUserName），**刻意不进 response.Registry 错误码 i18n 表**（数据展示兜底值非错误消息，塞入会污染错误码命名空间）。将来若做数据层多语言，再统一收一处数据展示文案表，低优。

## 9. 下一步建议
- PM 评审本 diff + 报告 → 我重启 demo → daxing 按 §7 真人验收（重点：operator 列显用户名 + 删用户后「已注销」+ dict_data 真翻页）→ 放行后双推。
- 之后按账本序进 **T-005b-3（配置中心加密参数写链路，体量最大）**，再收文案/低优批。
