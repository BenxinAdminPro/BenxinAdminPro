# 完成报告：T-014 出参纪律收口批 II（uploader/operator 收口 + createDictData openapi 补漏）

## 1. 完成状态
执行端编码 + 自测**全部完成，等 PM 评审（openapi diff + 负向双向锁断言 + 收口 curl）+ daxing 零回归验收 + PM 放行后双推**。
- 续 T-013 出参纪律；Go 仅 2 处 json tag + 测试；含 openapi 补漏。
- openapi v0.16.0 → **v0.17.0**（真物化，已落 openapi.yaml）。
- demo 已用新代码重启（nohup `/tmp/benxin-demo`），curl 收口/补漏实证在新二进制跑通。

## 2. 改动文件清单（6 文件 +77/-17）
| 路径 | 说明 | 类型 |
|---|---|---|
| `server/system/model_file.go` | ③ `Uploader` json tag `uploader`→`"-"`；头注释 +@updated | 修改 |
| `server/system/model.go` | ③ SysOperLog `Operator` json tag `operator`→`"-"`；头注释 +@updated | 修改 |
| `server/system/response_test.go` | anti-fake-green **双向锁**：File 测试扩 uploader（drop + uploader_name 保留）；新增 `TestResponseEncoderOperLogNoOperator`（operator drop + operator_name 保留） | 修改 |
| `server/spec/openapi/openapi.yaml` | ③ 删 SysFile.uploader / SysOperLog.operator（保留 *_name）；① 补 `POST /sys/dict/data`（createDictData，data=SysDictData）；version v0.17.0 + changelog | 修改 |
| `admin/src/api/file.ts` | 删 `SysFileRow.uploader` + 改注释；头注释 +@updated | 修改 |
| `admin/src/api/operlog.ts` | 删 `OperLogRow.operator` + 改注释；头注释 +@updated | 修改 |

> 精确 stage 范围 = 上述 **6 文件**。`.project-management/PROJECT_STATUS.md` 当前工作树有改动，但**是 PM 填实 T-013 占位符的独立编辑、非本片产出，不入 stage**（见 §8 提醒）。无 config.local.yaml 混入。

## 3. 接口实现情况（openapi v0.17.0）
**③ uploader/operator drop**：model_file.go `Uploader` + model.go `Operator` json tag→`"-"`。system ResponseEncoder 走 `json.Marshal(struct)→map`，**model-tag drop 一处覆盖全部出参路径**（list / uploadFile / 任何 detail），无需逐端点改。openapi SysFile 删 uploader、SysOperLog 删 operator，**保留 uploader_name/operator_name**。内部 JOIN 解析走 DB 列（`.Table` 原生查询 + Go 字段），json tag 不影响——*_name 展示路径完好。

**① createDictData 补漏**：`/sys/dict/data` 加 `post:` 块（operationId createDictData，权限 sys:dict:create，入参 dict_type/label/value 必填 + sort/status，data=SysDictData），逐字镜像 /sys/dict/types post 结构。handler.go:64 早注册返 `enc.Item(dd)`，本片仅补 spec、不动 Go。`/sys/dict/data` 现 get+post+put(/{id})+delete(/{id}) 齐全。

## 4. 自验结果
- ✅ `go build ./...` + `go vet ./...` 净。
- ✅ `go test ./...`（默认闸门）全绿（含双向锁负向断言）。
- ✅ `go test -tags=integration ./rbac/ ./system/ ./examples/demo/`（真 MySQL+Valkey）全绿（system 6.0s / demo 11.9s）。
- ✅ **anti-fake-green 自检**：临时把两 json tag 改回（uploader/operator）→ 双向锁「不应含」分支**真 FAIL**（`TestResponseEncoderFileNoStorageKey` line 92 / `TestResponseEncoderOperLogNoOperator` line 114，FAIL 输出含泄漏字段）→ 还原 → PASS。
- ✅ `pnpm build` exit 0 类型干净（删 TS 字段后无残留 `.uploader`/`.operator` 读取致编译错）；`pnpm test` 17 PASS。
- ✅ **收口实证（curl，新 demo）**：
  - `/sys/files` list：**uploader 0 次**、**uploader_name 3 次**；上传响应 uploader 0 次 / uploader_name 1 次。
  - `/sys/logs/oper` list：**operator 0 次**、**operator_name 3 次**（operator_name="admin" 在）。
  - 双向锁达成：raw 字段 drop + 展示名保留。
- ✅ **① createDictData 补漏实证**：`POST /sys/dict/data`（dict_type=t007d_demo, label/value/sort/status）→ `code=0` + data 含 hashid id + label/value 回显，与新 post 块一致；清理 DELETE 200。
- ✅ 前端 grep 无残留：`grep -rn "\.uploader\b\|\.operator\b" admin/src` 仅命中 *_name + 注释，无原值逻辑读取。
- ✅ 测试产物清理：上传的一次性文件 + dict_data probe 均删净（files total=0、dict_data 删 200）。

## 5. git 提交记录
**尚未提交、尚未双推**（完成判定/双推/改 PROJECT_STATUS 权限仅 PM）。
- 待 stage（精确 6 文件，禁 `git add -A`）：见 §2。
- 拟提交信息：`feat(spec): T-014 出参纪律收口 II（drop uploader/operator 出参 + 补 createDictData openapi；v0.17.0）`
- 双仓 Gitee 主 + GitHub 镜像；CI 跑双闸门。

## 6. 安全自查
- ✅ 收口即加固：减少内部用户 ID 外露面（uploader/operator 的 JWT subject 内部 ID 串），未引入新出参字段。
- ✅ drop 后 uploader/operator 不在任何 JSON 出参（list/upload curl 实证 0 次）；uploader_name/operator_name 仍正常（展示路径未误伤，双向锁断言 + curl 双证）。
- ✅ ① 补漏不放宽鉴权、不暴露新字段（createDictData 行为/权限 sys:dict:create 原样）。
- ✅ 未动 uploader_name/operator_name、DB 列、service 内部使用、任何鉴权/路径/状态码。

## 7. 需 daxing 真人验收（后端切片，demo 已用新代码重启）
- [ ] **零回归命门**：**文件管理页「上传人」列正常显名**（uploader_name，如 admin/匿名/已注销）、**操作日志页「操作人」列正常显名**（operator_name）——证删 raw 字段后 *_name 展示路径完好。
- [ ] 文件上传/下载/删除、operlog 列表/按操作人名筛选正常。
- [ ]（可选·可视收口）Network 瞄 `/sys/files` 无 uploader、`/sys/logs/oper` 无 operator，但 *_name 在。
- [ ] ① createDictData 补漏为 spec-only、无 UI 体现，daxing 不需验，PM diff 复核。

## 8. 偏差与待办
- **⚠️ 提醒 PM（非本片改动）**：工作树 `.project-management/PROJECT_STATUS.md` 有未提交改动 = **PM 填实 T-013 占位符的 KB 步骤**（{{FINAL_HEAD}}→e2530e0、{{T013_LEDGER}}→e6877d3）。但因 `{{FINAL_HEAD}}` 为共享 token，全局替换把 **T-012 切片表行的「报告归档」也填成了 e2530e0**（应为 T-012 自身的 **8f80b07**）。请 PM 修正 T-012 行（line ~139）。执行端不碰 PROJECT_STATUS.md（权限属 PM），仅此提醒。
- **债池清理**：本片偿还 T-013 偏差②（createDictData openapi 补漏）+ 候选债（uploader/operator 裸内部 ID 收口）两条。建议 PM 落账时把这两条标 ✅ 已偿还。
- 未碰债池其余三条（enc==nil struct tag 彻底化 / dateText 抽 util / m4a 待定性）——守 scope。

## 9. 下一步建议
- PM 评审（openapi diff 逐块：SysFile/SysOperLog 删字段保留 *_name、createDictData post 块镜像正确、反漂移补核 `git diff <父> <feature>` 含 openapi.yaml；双向锁断言；收口 curl）+ daxing 零回归验收（*_name 列完好为命门）通过后放行双推（6 文件精确 stage）。
- PM 修正 PROJECT_STATUS.md 的 T-012 行 报告归档 hash（8f80b07）。
