# 完成报告：T-016 Content-Type 交叉校验确定性收口（m4a 误杀根治）

## 1. 完成状态
✅ 编码 + 自测全绿，已落 feature commit `adbdd2d`。未自标完成 / 未推 / 未改 PROJECT_STATUS.md（账本由 PM 落账指令驱动），等 PM 评审。
单一目标达成：`ValidateContentType` 从「依赖 OS mime db 的全子类型精确相等」改为「底座自持、确定性、顶层大类级」交叉校验，根治 m4a 误杀及跨平台不确定性类级隐患。错误码 / openapi / 调用方签名均未动。

## 2. 改动文件清单
| 路径 | 说明 | 类型 |
|---|---|---|
| `server/drivers/storage/upload.go` | 移除 `mime` import；新增包级 `extCategory map[string]string`（ext→顶层大类，覆盖 demo `allowed_exts` 全集 16 项 + 行内来源注释）；重写 `ValidateContentType` 为大类级确定性比较；头注释追加 `@updated` | 修改 |
| `server/drivers/storage/storage_test.go` | `TestValidateContentType` 改表驱动，新增 9 pass + 3 reject 用例 | 修改 |

`git diff --stat`：
```
 server/drivers/storage/storage_test.go | 32 ++++++++++++++--
 server/drivers/storage/upload.go       | 69 ++++++++++++++++++++++++++++------
 2 files changed, 85 insertions(+), 16 deletions(-)
```

## 3. 接口实现情况
- `ValidateContentType(contentType, ext string) error` —— 签名 / 返回类型不变，上游 `file_service.go` 第 ④ 闸仍 errors-style 映射到 `ErrFileTypeMismatch`(11072)。
- 确定性比较逻辑（不再调 `mime.TypeByExtension`）：
  1. ext 归一化（TrimSpace→ToLower→去点）；
  2. `extCategory[ext]` 未命中（未知扩展名）→ `nil`；
  3. declared = `SplitN(contentType,";",2)[0]` 去空，空串 → `nil`；
  4. declared == `application/octet-stream`（大小写无关）→ `nil`；
  5. `gotCat = declared 顶层大类`；
  6. `gotCat == wantCat` → `nil`，否则 error（保留 declared/gotCat/wantCat/ext 便于排障，英文文案）。
- `extCategory` 覆盖：`jpg/jpeg/png/gif→image`、`mp4/webm/mov→video`、`mp3/wav/ogg/m4a→audio`、`pdf/docx/xlsx/zip→application`、`txt→text`，与 demo `config.example.yaml:71` 白名单逐项对齐。

## 4. 自验结果
- `go build ./...` ✅ OK
- `go vet ./drivers/storage/...` ✅ 干净
  （gopls 残留风格提示 `stringscut`：`SplitN(...,";",2)` 系任务书 §3.2 明确指定写法，保留不改；非 vet 报错。）
- `go test ./drivers/storage/...` ✅ `ok ... 0.800s`
- `go test ./...`（默认闸门）✅ 全绿尾部摘要：
```
ok  	.../auth	(cached)
ok  	.../crypto	(cached)
ok  	.../dberr	(cached)
ok  	.../drivers/storage	(cached)
ok  	.../examples/demo	0.978s
ok  	.../httpmw	(cached)
ok  	.../idcodec	(cached)
ok  	.../rbac	(cached)
ok  	.../response	(cached)
ok  	.../system	1.936s
```

### 防假绿 before/after（§4 要求）
**BEFORE**（故意把第 6 步比较写反 `gotCat != wantCat`）—— pass 与 reject 用例同时真 FAIL，证测试双向有效：
```
--- FAIL: TestValidateContentType (0.00s)
  m4a alias audio/x-m4a: ... should pass, got content-type category mismatch ...   ← 同大类被误拦
  jpg image/jpeg: ... should pass, got ... mismatch ...                            ← 同大类被误拦
  text/plain for jpg (旧用例保持红): ... should error, got nil                     ← 跨大类被误放
  exe-as-jpg 粗错配: ... should error, got nil
  image vs audio 跨大类: ... should error, got nil
FAIL
```
**AFTER**（恢复 `gotCat == wantCat`）：
```
ok  github.com/benxin_dev/benxinadminpro-server/drivers/storage  0.800s
```

## 5. git 提交记录
- `adbdd2d  fix(storage): T-016 Content-Type 交叉校验改确定性大类级（根治 m4a 误杀）`
- 精确 stage（逐文件，未用 `git add -A`），仅含两目标文件；未混入 config.local.yaml / PROJECT_STATUS.md。
- **未双推**（等 PM 放行）。

## 6. 安全自查
- 该校验账本定性为「组织性非安全边界」，本改维持宽容口径（未知/空声明/octet-stream 放行），不收紧亦不放松安全闸——文件大小、扩展名白名单、文件名消毒、路径穿越防护四件套其余三件未动。
- 无硬编码业务前缀；`extCategory` 为通用大类常量，业务中立。
- 未引第三方 mime 库；纯标准库 `strings` + 底座自持常量表。
- 未提交任何密钥 / IP / .env。

## 7. 需 daxing 真人验收（PM 放行后，未替勾）
- [ ] 重启 demo（Go 改动需重编译）后，媒体管理页上传 .m4a → 成功落库 / audio tab 可见 / 内联播放正常。
- [ ] 回归 jpg / mp3 / mp4 仍正常。
- [ ] 负例：构造跨大类 Content-Type 错配（curl `--noproxy '*'`）仍返回 11072。

## 8. 偏差与待办
- **越界自证**（主动贴）：`git show --stat adbdd2d | grep -E 'errcode|openapi|file_service'` → **空**，确认未触碰 file_service.go / errcode.go / openapi.yaml。
- 偏差：无。任务书 §2 边界全数遵守（仅改 upload.go 的 `ValidateContentType` 函数体 + 同包映射 + storage_test.go 补测）。
- gopls `stringscut` 风格提示按 §3.2 指定写法保留，已在 §4 说明，不视为待办。

## 9. 下一步建议
- 等 PM clone 公开仓 `git show --stat adbdd2d` 实证评审 → daxing 验收 → PM 放行 → 双推 → `git ls-remote` 三方一致 → PM 更账本。
- 可选（非本切片，需另立任务书）：若未来白名单新增 `extCategory` 缺项扩展名，当前走「未知放行」兜底，建议在白名单变更流程加一道「新扩展名补 extCategory 表项」检查清单项。
