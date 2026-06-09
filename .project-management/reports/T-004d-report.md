# 完成报告：T-004d system 对外 ID hashid 化（全套）+ 前置抽 idcodec 中立包

> 状态：**待 PM 评审 + daxing API 层真人验收**。执行端未自标完成、未提交、未双推。

## 1. 完成状态
- **前置·抽 idcodec 中立包**：hasher 实现搬至 `server/idcodec`，rbac/system 均引用之，二者无直接 import（解耦红线达成，go list -deps 实证见 §3）。对 rbac 纯搬迁、零行为变化、零回归。
- **system 出参全套 hashid**：dict_type/dict_data/config/file/oper_log/login_log 六实体 `id` 经 encoder 编码为 hashid 字符串输出，不再裸吐 uint64。
- **path :id 收口**：dict/config 的 PUT/DELETE + file download/delete 的 `:id` 改 hashid 解码；非法/伪造/越界 → 400（复用既有 ErrInvalidID，无新码段），防探测语义同 T-003e。
- **system body 入参**：据现状确认无对外 ID 入参（dict/config 靠 string code/key 互引），故只收 path，不动 body（符合任务书明示）。
- **typed-nil 自检盲区修正**：demo 装配自检改用反射 `isNilDep`，能侦测 typed-nil；新旧写法对比单测坐实。
- **openapi v0.10.0**：补全缺失的 dict/config item 路径，file :id 已 string 与实现对齐；redocly 0 error。
- 自验全绿（含 demo e2e 装配真跑）；唯一红为先前已存在的 `TestNewEnforcerMySQL_RoleInheritance`（T-003d-fix 责任，本片未新增红，见 §8）。

## 2. 改动文件清单
**新增**
| 路径 | 说明 |
|---|---|
| `server/idcodec/hashid.go` | 中立包：Hasher/NewHasher/HashidConfig/Encode/Decode/EncodeOptional（自 rbac 搬迁，零业务依赖） |
| `server/idcodec/hashid_test.go` | hasher 单测（自 rbac 搬迁）+ `TestNewHasherNeverNilNil`（坐实永不返回 (nil,nil)） |
| `server/system/response.go` | system 出参编码器：json 重映射 + 反射读 typed ID 覆盖为 hashid；nil hasher 退化裸 id；保留脱敏/`json:"-"` |
| `server/system/decode.go` | path `:id` 解码单一出入口（hashid→uint64，非法 400，nil 退化裸十进制） |
| `server/system/response_test.go` | encoder/decode 单测：6 实体 id hashid、字段保留、脱敏保留、nil 退化、非法 :id→400 |
| `server/system/testmain_test.go` | 初始化 response.Registry（handler 400 路径依赖渲染器） |
| `server/system/hashid_integration_test.go` | 真 MySQL：file 列表 id hashid + download by hashid + 裸/乱码→400；dict 全链路 + PUT/DELETE by hashid |
| `server/examples/demo/selfcheck_test.go` | typed-nil 自检新旧写法对比单测 |
**修改**
| 路径 | 说明 |
|---|---|
| `server/rbac/hashid.go` | 降为类型别名 `type Hasher = idcodec.Hasher`（rbac 业务代码零改动，最强零回归） |
| `server/rbac/{response_test,decode_test,decode_integration_test}.go` | `NewHasher/HashidConfig` 调用迁 `idcodec.*` |
| `server/system/handler.go` | Handler 加 hasher+enc 注入；6 处 `pid`→`decodePathID`；8 处出参经 enc；删旧 pid() |
| `server/system/handler_file.go` | FileHandler 加 hasher+enc；2 处 ParseUint→decodePathID；Upload/List 出参经 enc |
| `server/system/handler_authz_test.go` | NewHandler/NewFileHandler 调用补 hasher 实参 |
| `server/examples/demo/main.go` | 用 idcodec.NewHasher；注入 system handlers；自检改 isNilDep（+reflect import） |
| `server/spec/openapi/openapi.yaml` | v0.9.0→v0.10.0；补 dict/config item 路径（:id hashid）；description + parity 注 |
**删除**
| 路径 | 说明 |
|---|---|
| `server/rbac/hashid_test.go` | 搬迁至 `idcodec/hashid_test.go` |

> **对外契约变更（消费方须跟进）**：`system.NewHandler` / `system.NewFileHandler` 新增 `hasher *idcodec.Hasher` 注入参数；demo 已跟进，BenxinKP 等接入 system 时须注入 idcodec hasher。RegisterRoutes 签名不变。

## 3. 解耦证据（go list -deps，最重要的架构红线）
```
① go list -deps ./system/ | grep .../rbac      → 空（system 不依赖 rbac ✅）
② go list -deps ./system/ | grep .../idcodec    → 命中（system→idcodec ✅）
③ go list -deps ./rbac/   | grep .../idcodec    → 命中（rbac→idcodec ✅）
④ go list -deps ./idcodec/| grep benxinadminpro → 仅自身（idcodec 零本仓业务依赖 ✅）
⑤ go list -deps ./idcodec/| grep (rbac|system)  → 空（无环 ✅）
⑥ grep -rn .../rbac system/*.go                 → 空（system 源码零 rbac import ✅）
```
结论：依赖方向为 `rbac → idcodec ← system`，system 与 rbac 互不依赖、无环；`PermGuard` 接口解耦（T-003d）保住，引 hasher 未反向耦合。

## 4. idcodec 搬迁 rbac 零回归证据
- rbac/hashid.go 降为 `type Hasher = idcodec.Hasher` 别名 → rbac 业务代码（response/decode/全 handler 中的 `*Hasher`）**一行未改**（编译期同一类型）。
- rbac 全部既有单测 + 集成（`-tags=integration`，排除先前已红 casbin）全绿；hashid 出参回归（response_test `bareIntID`）、T-003e 入参用例均守门通过。
- 行为零变化：hasher 逻辑逐行搬迁（仅错误文案 `rbac:`→`idcodec:`，内部错误非用户可见、无断言依赖）。

## 5. system 出参/path hashid 化清单（逐实体逐接口）
| 实体 | 出参接口（id→hashid） | path :id 接口（hashid 解码） |
|---|---|---|
| SysDictType | GET /sys/dict/types(list)、POST(create) | PUT/DELETE /sys/dict/types/{id} |
| SysDictData | GET /sys/dict/data(list)、POST(create) | PUT/DELETE /sys/dict/data/{id} |
| SysConfig | GET /sys/configs(list)、POST(create) | PUT/DELETE /sys/configs/{id} |
| SysFile | POST /sys/files(upload)、GET(list) | GET /sys/files/{id}/download、DELETE /sys/files/{id} |
| SysOperLog | GET /sys/logs/oper(list) | —（无 item 路径） |
| SysLoginLog | GET /sys/logs/login(list) | —（无 item 路径） |

- 出参全部经 `h.enc.Item/Items/Page`（grep：handler.go 8 处 + handler_file.go 2 处，无裸 model 直吐）。
- path 全部经 `decodePathID`（grep：system 无 `ParseUint(c.Param` 残留）。
- **全链路零裸 uint64 主键**：出参 + path 均 hashid；持久层 model `id uint64` 保留（DB 层，经 encoder 出参，不裸暴露）。

## 6. typed-nil 自检新旧写法对比单测
- `examples/demo/selfcheck_test.go::TestIsNilDepCatchesTypedNil`：构造 `(*idcodec.Hasher)(nil)` 装入 `any` —— 旧写法闭包 `func(v any) bool { return v==nil }` 返回 false（漏检，坐实盲区），新 `isNilDep` 返回 true（抓得住）；有效实例不误报；untyped nil 判 nil。
- `TestIsNilDepOnVariousKinds`：覆盖 nil map/interface/非 nil 值。
- 诚实定位：自检为第二道；真正兜底是构造器契约（`NewHasher` 永不返回 (nil,nil)，`idcodec` 单测 `TestNewHasherNeverNilNil` 坐实）+ demo main.go:165-166 err fail-fast。

## 7. openapi v0.10.0 diff 摘要
- version 0.9.0→0.10.0；header @updated；description 增 v0.10.0 段 + system PHP parity 注。
- **补全**此前完全缺失的 item 路径：`/sys/dict/types/{id}`、`/sys/dict/data/{id}`、`/sys/configs/{id}`（各 PUT+DELETE，:id 为 `type: string` hashid）。
- file `:id`（download/delete）此前已 `type: string`，实现此前为 ParseUint(int)——本片把实现改 hashid，**spec↔impl 现已一致**（原不一致消除）。
- system 实体响应未在 components/schemas 强类型化（沿用 ApiResponse 泛型包络），无 `id:integer` 出参 schema 需改；id 为 hashid 的约定以 description 为准。
- redocly：**0 error**（55 warnings，含新增 6 operation 沿用既有"仅 200 响应"风格的同类告警，非新错误）。

## 8. 自验结果
```
go build / go vet ./...    → OK
go test ./...              → 全绿（idcodec/rbac/system/demo/...）
go test -tags=integration ./... → 除 TestNewEnforcerMySQL_RoleInheritance 外全绿
  新增真 MySQL 集成：TestSystemHashidFileE2E_MySQL（文件 download by hashid + 防枚举 400）
                    TestSystemHashidDictE2E_MySQL（dict 全链路 + PUT/DELETE by hashid）
  demo e2e（装配真跑兜底，跨包 idcodec + system 新注入）→ ALL PASSED
```
- **本片未新增红**：不带 skip 跑全集成，失败恰好仅 `TestNewEnforcerMySQL_RoleInheritance` 一条——已多次证其在 78342cf 干净 HEAD 即红，属 T-003d-fix 责任（T-003b 改 perm code 精确匹配时漏更新的陈旧 URL keyMatch 断言），不在本片。

## 9. git 提交记录
- **未提交、未双推**（流程铁律：判定权在 PM）。拟提交文件见下方"提交清单"，待 PM 评审 + daxing API 验收 + 放行后双推。

## 10. 安全自查
- 非法/伪造/空/越界 :id → 400（ErrInvalidID），不回落、不暴露内部 id、错误中立、不区分"不存在 vs 格式错"（防探测）。
- **file `:id` 防枚举为核心安全收益**：hashid 叠加 RequirePerm（T-004b）纵深防御，攻击者无法枚举 id 遍历他人文件（集成测试以裸数字 1/999999 + 乱码三连验 400）。
- config 脱敏（T-005 maskEncrypted，service 层）未被破坏：encoder 仅改 id 不碰 value，单测 `TestResponseEncoderPreservesMaskedValue` 守门。
- idcodec hasher 盐/配置经装配注入，禁包级可变全局；nil 退化仅存在于不接 hasher 的单测，生产经构造器契约 + fail-fast 不可达。

## 11. 需 daxing 真人验收（API 层，sys 页在 T-007d/e/f）
- `GET /sys/files` 列表 id 为 hashid → 用该 hashid `GET /sys/files/{id}/download` 取到文件；裸数字/乱码 :id → 400。
- 抽验 dict 或 config：列表 id 为 hashid → 用该 hashid `PUT`/`DELETE` 通；裸数字 :id → 400。

## 12. 偏差与待办
- **【需 PM 决策·非我提交】工作区出现 `server/demo`（56MB Mach-O 二进制，go build 产物）**：未被 .gitignore 覆盖。我未暂存、未删除、未改 .gitignore（危险操作待定）。**建议**：删除该产物 + 在 .gitignore 增 `/server/demo` 或编译产物规则。请 PM 指示。
- **【设计选择·已说明】rbac.Hasher 用类型别名**而非全量替换为 idcodec.Hasher：为最强零回归（rbac 业务代码零改动）。decoupling 仅取决于 import（system 不 import rbac），别名不影响。如 PM 要求 rbac 内部也直引 idcodec.Hasher，可后续机械替换（纯改类型名）。
- **【设计选择】system encoder 用反射 + json 重映射**而非逐实体手写 gin.H：6 实体均单一 ID、无跨引用，反射方案 DRY、字段新增自适应、从 typed uint64 取 id 无 float64 风险。reflect 在 admin 低 QPS 端点可接受。
- **【先前已红·非本片】`TestNewEnforcerMySQL_RoleInheritance`**：T-003d-fix 独立切片处理（先确认角色继承经 perm code 有覆盖再重写陈旧断言）。
- **【遗留·非本片】sys_file `path` 字段**是否在出参泄漏真实文件系统路径，属 T-004b 范畴，本片未碰（encoder 仅改 id，保持原字段集）。

## 13. 提交清单（待放行）
**纳入（mine，18 项）**：idcodec/{hashid.go,hashid_test.go}、system/{response.go,decode.go,response_test.go,testmain_test.go,hashid_integration_test.go}、examples/demo/{main.go,selfcheck_test.go}、rbac/{hashid.go,response_test.go,decode_test.go,decode_integration_test.go}、system/{handler.go,handler_file.go,handler_authz_test.go}、spec/openapi/openapi.yaml、删除 rbac/hashid_test.go、本报告。
**排除**：`PROJECT_STATUS.md` 及 daxing 投放的 `BenxinAdminPro-PROJECT_STATUS.md` / 任务书 md（账本/任务书归 PM）；`server/demo`（二进制产物，见 §12）。

## 14. 下一步建议
PM 评审（重点 §3 解耦证据、§12 server/demo 处置）→ daxing §11 API 验收 → 放行 → 双推翻 ✅。后续 T-007c x-table 增强、T-007d/e/f sys 页可直接消费 system 的 hashid 出参/入参。
