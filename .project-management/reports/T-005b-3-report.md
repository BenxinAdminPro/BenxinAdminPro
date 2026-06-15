# 完成报告：T-005b-3 配置中心加密参数写链路

> 状态：执行端自验全绿，**待 PM 评审 + daxing 浏览器验收**。未自标完成、未双推、未改 PROJECT_STATUS（守 T-006 铁律）。

## 1. 完成状态
- ✅ 加密参数能经管理 API **新建**（CreateConfigInput 补 is_encrypted，=1 走 EncryptGCM 落密文）。
- ✅ 加密参数能经管理 API **编辑且不破坏密文**（UpdateConfigInput 值字段指针三态：省略=保持、传入=按现有 is_encrypted 加密/明文覆写）。
- ✅ 前端解除加密行编辑禁用 + 新建弹窗加 is_encrypted 选择 + 编辑值框「留空保持/重填替换」语义落地。
- ✅ seed 搭车补明文 + 真加密样例（免手工 SQL）。
- ✅ openapi v0.13.0 → v0.14.0（搭车补漏掉的 v0.13.0 changelog 行）。
- 范围外项**零触碰**：未做明文查看/解密回显、未碰 sys:secret:view、未做明↔密切换、未碰短信/邮件/2FA/密钥管理、未改读路径裸 fmt.Errorf。

## 2. 改动文件清单

**后端（Go）**
| 文件 | 说明 | 类型 |
|---|---|---|
| `server/system/dict_service.go` | ConfigService 加 `gcmKey` 字段 + `SetGCMKey` setter + `encryptValue` 助手；CreateConfigInput 补 `is_encrypted` + Create 加密分支；新增 `UpdateConfigInput`（`ConfigValue *string` 指针三态）+ Update 重写（先 First 取现有 is_encrypted 作判据，nil 值不入 map=保持） | 修改 |
| `server/system/handler.go` | UpdateConfig 入参由 `CreateConfigInput` 改绑 `UpdateConfigInput` | 修改 |
| `server/examples/demo/main.go` | `configSvc.SetGCMKey(gcmKey)` 注入；`seed(...)` 传 gcmKey；装配自检加「加密能力就绪（gcmKey 非空）」检查（§4/§6 建议项，第二道 fail-fast） | 修改 |
| `server/examples/demo/seed.go` | 新增 `seedConfig`：幂等补 1 条明文（site.title）+ 1 条真 EncryptGCM 加密样例（demo.secret_token），seed 签名加 gcmKey | 修改 |
| `server/spec/openapi/openapi.yaml` | version 0.13.0→0.14.0；createConfig 入参加 is_encrypted+remark；updateConfig 重写（键/类型锁定 + config_value 指针语义）；补 v0.13.0(T-008c)/v0.14.0 changelog | 修改 |
| `server/system/dup_integration_test.go` | 移除 `TestDupConfigUpdateRename_MySQL`（改名撞键场景随 config_key 编辑锁定按设计消失，非凑绿删；附注说明） | 修改 |
| `server/system/config_crypto_test.go` | 新增：解密往返三态 + 明文零回归 + 密钥缺失降级单测（SQLite，进默认 go test 闸门） | 新增 |
| `server/examples/demo/config_write_integration_test.go` | 新增：写路径 enforce（dept_mgr 200↔editor 403）+ 全栈加密（HTTP 建密文 DB 可解 + 列表脱敏）+ 编辑三态 e2e | 新增 |
| `server/examples/demo/seed_config_integration_test.go` | 新增：seed 加密样例连跑两遍幂等 + 加密样例真密文不被覆盖 | 新增 |

**前端（admin）**
| 文件 | 说明 | 类型 |
|---|---|---|
| `admin/src/views/sys/config/index.vue` | 新建 fields 加 is_encrypted 选择（select，默认明文）；移除加密行编辑 disabled/tooltip；openEdit 加密行不回填值；submitEdit 按加密标志分流（加密行空值省略 config_value、明文行始终提交）；config_key 不再随 update 提交；值框留空提示 | 修改 |

> `.project-management/PROJECT_STATUS.md` 在本会话**开始前**即为 modified 状态（非本片改动），执行端未触碰。

## 3. 接口实现情况
- **POST /sys/configs**：入参增 `is_encrypted`(0/1)。=1 → `EncryptGCM(gcmKey, value)` 存密文 + `IsEncrypted=1`；=0 → 明文存（现状）。密钥未注入而 =1 → `ErrConfigDecryptFailed`，不 panic。
- **PUT /sys/configs/:id**：改绑 `UpdateConfigInput{ConfigValue *string, Name, Remark}`。**不含 config_key（禁改）、不含 is_encrypted（类型锁定）**。先 `First` 取该行现有 is_encrypted：`ConfigValue==nil` → updates map 不含 config_value 键（密文/明文原样保留）；非 nil 且 is_encrypted==1 → 重新加密；非 nil 且 ==0 → 明文覆写。
- 不新增端点、不新增权限码（复用 sys:config:create/update）、不新增错误码段（复用已注册 `ErrConfigDecryptFailed`）、无 DDL。
- 对外 List/GetByKey `maskEncrypted` 恒 ****** 不动；读路径 `ConfigCenter.GetConfig` 自动解密不动。

## 4. 自验结果

**默认闸门（go test ./...）全绿**
```
go build ./...  OK
go vet ./...    OK
go vet -tags=integration ./...  OK
go test ./...   ALL ok（system / examples/demo 等全 ok）
```

**集成测试（-tags=integration，docker MySQL+Redis 实跑）全绿**
- `go test -tags=integration ./system/... ./examples/demo/...` → ok。

**头号正确性证据 — 解密往返三态**（`config_crypto_test.go`，SQLite 单测，进默认闸门）
- ① 新建加密 → GetConfig 解出 == 原明文，落库为合法 base64 信封、非明文字面。
- ② Update 带新值 → GetConfig 解出 == 新明文，密文确已更换。
- ③ Update 不带值(nil) → 密文**原样保留**、GetConfig 仍解出原明文、name 已更新（防数据损坏命门坐实）。
- 明文零回归：明文 create 明文存 / update 带值覆写 / update nil 保持，均断言。
- 负例：无 gcmKey + is_encrypted=1 写入 → `ErrConfigDecryptFailed` 不 panic；明文写入无 key 仍正常。

**全栈加密 + 写路径 enforce**（`config_write_integration_test.go`，真 HTTP+MySQL）
- enforce 正向：dept_mgr POST/PUT `/sys/configs` 全 200 ↔ editor 全 403。
- 全栈加密：经 HTTP 建 is_encrypted=1 → DB 落真密文（≠明文、`DecryptGCM` 还原原值）+ 列表恒 `******`。
- 全栈编辑三态：留空 PUT（省 config_value）密文不变仍可解原值；填值 PUT 密文换新且解出新值。

**seed 幂等**（`seed_config_integration_test.go`）
- 连跑两遍 buildApp(migrate+seed)：sys_config 行数稳定 = 2；加密样例为真密文（可解 `s3cr3t-demo-token`）且第二遍不被重新加密覆盖。

## 5. git 提交记录
- **未提交、未双推**（守 T-006：执行端不自标完成、不自行双推、不改 PROJECT_STATUS）。
- 待 PM 评审 + daxing 浏览器验收放行后，再按账本流程双推（Gitee 主 + GitHub 镜像，注意 Clash TUN/fake-IP 对 gitee.com 的劫持）。

## 6. 安全自查
- **对外永不回显明文**：未新增任何返明文的对外路径；maskEncrypted 不动；前端加密行编辑值框「打开即空」，绝不回填 ****** 或明文（openEdit 对 is_encrypted==1 行置空）。
- **防潜伏数据损坏**：指针 nil/present 语义严格落地（map 不含键=保持）；加密判据取 **DB 现有 is_encrypted** 非入参（防类型被偷改）；前端加密行空值省略字段、绝不把 ****** 当新明文加密；单测/集成测试双层坐实「留空不破坏密文」。
- **enforce 服务端把关**：写路径 dept_mgr 200↔editor 403 实证；前端隐藏仅 UX。
- **密文不入日志**：加密在 service 层、value 不经额外日志路径；操作日志中间件既有 secret 脱敏不受影响。
- **密钥注入**：gcmKey 走配置注入（main.go decodeB64 fail-fast 32 字节），ConfigService 未配密钥时运行时降级返错不 panic（底座库对明文参数不强依赖密钥）。

## 7. 需 daxing 真人验收（demo 浏览器）
> **验收前务必确认 demo 用新代码重启过**（本片改后端 Go，前端热更新不含后端）：
> `lsof -ti :8080 | xargs kill -9` 后 `cd server && go run ./examples/demo`。
1. 新建加密参数：弹窗「加密存储」选「是（加密）」→ 列表新行值显 `******`（非明文）。
2. 编辑加密行**留空提交**：成功后列表仍 `******`、参数仍可用（不报解密错）。
3. 编辑加密行**填新值提交**：成功（替换；明文真换由集成测试②坐实）。
4. 编辑加密行：值框**打开即空** + placeholder「留空＝保持原密文不变，填写＝重新加密替换」（不回填明文/不回填 ******）。
5. 明文参数编辑零回归（改 name/value 正常）。
6. enforce 边界：editor 调 create/update 应 403（前端隐藏 + 后端把关）。
> 现成验收数据：seed 已种 `demo.secret_token`（加密样例，列表显 ******）+ `site.title`（明文对照），无需手工 SQL。
> curl 自查时记得 `--noproxy '*'` 或 `unset HTTP_PROXY HTTPS_PROXY ALL_PROXY`（避 Clash 代理 loopback 返 000）。

## 8. 偏差与待办
- **偏差①（按任务书设计，非擅扩）**：移除 `TestDupConfigUpdateRename_MySQL`。原因：UpdateConfigInput 按 §4 不含 config_key（编辑态键锁定，对齐前端 disabled），「改名撞键」场景按设计不可能，该 backstop 路径对 config 不再可达。已在原处留注释说明，Create 侧友好码仍由 `TestDupConfigSimpleCreate_MySQL` 覆盖。Update 内仍保留 `dberr.IsDuplicate` 兜底（与 Create 同范式，无害）。
- **搭车修文档债**：openapi changelog 缺失的 v0.13.0(T-008c menu_ids) 行一并补上（此前升版未补 changelog 的小遗漏）；createConfig 此前漏列的 `remark` 字段一并补入 schema。
- **gofmt 说明**：`dict_service.go`/`handler.go` 用项目既有的紧凑单行 `if {…}` 风格，gofmt -l 会标红——属**既有风格非本片引入**（本片新增的 encryptValue/SetGCMKey/Update 体均 gofmt-clean，仅沿用文件原风格的那一行被标）。未全文件重排以免破坏既有风格 + 制造噪声 diff（守「match surrounding idiom」）。
- **未变动既有待办**：明文查看(sys:secret:view 悬空码)、明↔密切换、短信/邮件、绑手机号+2FA、密钥管理加固、读路径裸 fmt.Errorf 改用 ErrConfigDecryptFailed —— 均按 §2 留在账本待办池，本片不碰。

## 9. 下一步建议
- PM 评审本片选型落地（尤其 Update 指针三态 + 加密判据取 DB 现有标志）后，交 daxing 浏览器验收（重启 demo 跑新二进制）。
- 放行后双推归档。
- T-005b 后端债篮剩余项（按账本）可续推；config 加密链路至此「建/编/读/脱敏/seed」闭环。
