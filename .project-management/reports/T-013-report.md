# 完成报告：T-013 对外出参纪律收口批（ancestors / storage_key 收口 + system 出参 openapi 强类型化）

## 1. 完成状态
执行端编码 + 自测**全部完成，等 PM 评审（重点 openapi diff 逐块 + 负向断言 + 收口 curl）+ daxing 零回归验收 + PM 放行后双推**。
- 后端 Go 改动极小（删 2 编码行 + 改 1 json tag + 测试断言）；主体为 spec/openapi。
- openapi v0.15.0 → **v0.16.0**（真物化，已落进 openapi.yaml 实体）。
- demo 已用新代码重启（10:10:05 起，login 200），curl 收口实证已在新二进制上跑通。

## 2. 改动文件清单（8 文件 +210/-21）
| 路径 | 说明 | 类型 |
|---|---|---|
| `server/rbac/response.go` | ① 删 Dept 编码器 `"ancestors": d.Ancestors` + Menu 编码器 `"ancestors": m.Ancestors` 两行；头注释 +@updated | 修改 |
| `server/system/model_file.go` | ② `StorageKey` json tag `storage_key`→`"-"`；头注释 +@updated | 修改 |
| `server/rbac/response_test.go` | anti-fake-green：Dept/Menu 测试加 ancestors **负向断言**（fixture 设 Ancestors，泄漏即 FAIL） | 修改 |
| `server/system/response_test.go` | anti-fake-green：新增 `TestResponseEncoderFileNoStorageKey`（SysFile 出参无 storage_key 负向断言 + 零回归字段保留） | 修改 |
| `server/spec/openapi/openapi.yaml` | ① 删 SysDept/SysMenu schema 的 ancestors；③ 新增 6 schema + 9 端点响应强类型化；info.version v0.16.0 + changelog | 修改 |
| `admin/src/api/dept.ts` | 删 `DeptNode.ancestors`；头注释 +@updated | 修改 |
| `admin/src/api/menu.ts` | 删 `MenuRow.ancestors` + 改注释；头注释 +@updated | 修改 |
| `admin/src/api/file.ts` | 删 `SysFileRow.storage_key` + 改注释；头注释 +@updated | 修改 |

> 精确 stage 范围 = 上述 8 文件；无 PROJECT_STATUS.md / config.local.yaml 混入。

## 3. 接口实现情况（openapi v0.16.0）
**① ancestors 收口**：SysDept / SysMenu schema 各删 `ancestors: { type: string }`。后端删 response.go 两编码行；Menu 编码器一处覆盖 `/sys/menus/tree`（管理）+ `/sys/auth/menus`（侧边栏，最高频）两端点。

**② storage_key 收口**：`model_file.go` StorageKey json tag→`"-"`；SysFile 新 schema 天然无 storage_key；前端接口删字段。内部 Go（file_service.go:82/109/217/261 下载/清理/批删）走 `StorageKey` 字段不经 JSON，零影响。

**③ system 出参强类型化（新增 6 schema + 9 端点 typed，字段逐字对齐 model）**：
- schema：SysDictType / SysDictData / SysConfig（config_value 注明加密行恒 ****** 脱敏）/ SysFile（无 storage_key）/ SysOperLog / SysLoginLog，id 一律 `{type: string, description: "对外 ID(hashid 字符串)"}`。
- list 端点（data=PageResult 且 list 项 typed，镜像 RBAC allOf 写法）：listDictTypes / listDictData / listConfigs / listOperLogs / listLoginLogs / listFiles 共 6 个。
- detail/实体端点（data=Entity）：createDictType / createConfig / uploadFile 共 3 个（这 3 个 handler 返 `enc.Item(实体)`，是源码里实际返回 typed 实体的响应）。
- **不改任何请求/路径/状态码**，仅强类型化响应 data。put/update/delete/clean 等返回 nil 的端点保持裸 ApiResponse（30 处，符合预期）。

引用核对：SysDictType/SysConfig/SysFile 各被 $ref 2 次（list+实体），SysDictData/SysOperLog/SysLoginLog 各 1 次（仅 list）。

## 4. 自验结果
- ✅ `go build ./...` + `go vet ./...` 净。
- ✅ `go test ./...`（默认闸门）全绿（rbac / system 含本批新增负向断言）。
- ✅ `go test -tags=integration ./rbac/ ./system/ ./examples/demo/`（真 MySQL+Valkey）全绿（rbac 4.0s / system 6.9s / demo 13.4s）。
- ✅ **anti-fake-green 自检**：临时把泄漏改回（response.go 重加 ancestors 行 + model_file.go tag 改回 storage_key）→ 两负向断言**真 FAIL**（`TestResponseEncoder_Dept` / `TestResponseEncoderFileNoStorageKey`，FAIL 输出含泄漏字段）→ 还原 → PASS。证测试真在断言、非空跑。
- ✅ `pnpm build` exit 0 类型干净；`pnpm test` 17 PASS（tree.spec 零改动）。
- ✅ **收口实证（curl，已 unset *_PROXY + --noproxy '*'，新 demo 二进制）**：
  - `/sys/depts/tree`、`/sys/menus/tree`、`/sys/auth/menus` 响应体 **ancestors 出现 0 次**，HTTP 200；功能零回归（id 仍 hashid、children 嵌套仍在、技术部 leader "derek" 保留）。
  - `/sys/files` list 响应体 **storage_key 出现 0 次**；列表含 id/original_name/uploader_name 正常。
- ✅ **下载/批删零回归（一次性文件完整闭环）**：上传 57B txt → 响应 data 含 id 无 storage_key → 下载 57B **逐字节 IDENTICAL** → 批删 `deleted_count=1` → 列表已无该 id。坐实 storage_key `json:"-"` 不影响内部 Go 字段路径。
- ✅ 前端 grep 无残留消费方：`grep -rn "storage_key\|\.ancestors\|ancestors:" admin/src` 仅命中本批新增注释/头，无逻辑读取。

## 5. git 提交记录
**尚未提交、尚未双推**（完成判定/双推/改 PROJECT_STATUS 权限仅 PM）。
- 待 stage（精确 8 文件，禁 `git add -A`）：见 §2。
- 拟提交信息：`feat(spec): T-013 对外出参纪律收口（删 dept/menu ancestors + file storage_key 出参 + system 出参 openapi v0.16.0 强类型化）`
- 双仓 Gitee 主 + GitHub 镜像；CI 会跑双闸门。

## 6. 安全自查
- ✅ 收口即安全加固：减少内部信息外露面（ancestors 裸内部 ID 串 + storage_key 相对存储路径），未引入任何新出参字段。
- ✅ storage_key `json:"-"` 后任何 JSON 出参（list/upload）均无 storage_key（curl 实证）；下载/批删走 id 与 Go 字段正常（闭环实证）。
- ✅ ③ 强类型化未放宽鉴权、未暴露新字段；config_value 脱敏行为不变（schema 仅描述既有 ****** 行为）。
- ✅ 未动 ancestors DB 列 / 防环逻辑 / 子树同步；未动 storage_key DB 列 / service 内部使用。

## 7. 需 daxing 真人验收（后端切片，验收前 demo 已用新代码重启）
- [ ] **零回归为主**（删的是无人消费字段，UI 应完全不变）：部门管理页树正常、菜单管理页树正常、**侧边栏菜单正常**（/sys/auth/menus 改了编码器，重点确认侧边栏未崩/层级正常）、文件管理页列表+上传+下载+删除正常。
- [ ]（可选·可视收口）浏览器 Network 瞄 `/sys/auth/menus` 无 ancestors、`/sys/files` 无 storage_key。
- [ ] ③ openapi 强类型化为 spec-only、无 UI 体现，daxing 不需验，由 PM diff 复核。

## 8. 偏差与待办
- **偏差①（PM 源码实证受理）**：任务书 §4 列「dict/types(list+detail)、dict/data(list+detail)、configs(list+detail)、files(list+detail)」期望有 GET 单实体 detail 端点；source-first 核实**实际无任何 GET 单实体 detail 路由**（dict/types、dict/data、configs 的 `{id}` 路径仅 put/delete；files 仅 download/delete）。源码里返回 typed 实体的「detail 形」响应实为 **create POST + upload**（handler 返 `enc.Item`）。故按「以源码 paths 实际端点为准」，detail typed 落在 createDictType/createConfig/uploadFile 三个 POST，list typed 落在 6 个 GET。
- **偏差②（既有 spec 缺口·未补，避免 scope creep）**：`POST /sys/dict/data`（createDictData，handler.go:64 已注册）**openapi 自始缺失**（/sys/dict/data 路径仅 get），非本批引入。本 typing 片不新增缺失端点定义（属 spec 完整性补漏，独立小片）。**记待办**：补 createDictData 的 openapi path 定义（含 typed SysDictData 响应）。
- enc==nil 退化路径理论残留：SysMenu/SysDept 结构体 `Ancestors json:"ancestors"` 标签未改（任务 ① scope = 仅删编码器行、不动 struct/DB）；仅在 hasher 缺失的退化装配（同时已泄漏裸 int id，更严重前置问题）下该 fallback 才序列化 struct。demo 恒有 enc，curl 实证 0 泄漏。如需彻底，另评估（非本批）。

## 9. 下一步建议
- PM 评审（openapi diff 逐块 + 6 schema 字段逐字对齐 model + 负向断言 + 收口 curl）+ daxing 零回归验收通过后，由 PM 放行执行端双推（精确 8 文件 stage）。
- 偏差②的 createDictData openapi 补漏可随下一个 spec 小片或搭车清。
