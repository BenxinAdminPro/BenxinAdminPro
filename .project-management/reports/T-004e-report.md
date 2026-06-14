# 完成报告：T-004e DB 唯一键冲突 → 友好业务错误码（全 create/update 系统性收口）

## 1. 完成状态
执行端编码 + 单测 + 集成测试（编译校验通过）+ build/vet/test 全绿完成，待 PM 评审 + daxing API 验收。
**重大发现（先行声明）**：5 个"已存在"友好错误码 **早已定义并注册进 response.Registry**（T-003a/b/T-004a），HTTP 全映射 409 + i18n message 齐备。本片**无需新增任何错误码**——真正的缺口是 **1062 冲突的转换链路从未接到这些码**：service 只在「应用层预检 SELECT」路径返友好码，而预检漏判的场景（**软删记录占位 / 并发竞态 / update 改名**）下 1062 直接冒泡 → Create 路径包成 500、Update 路径误吞成 404。本片补的是**转换 backstop**，不是新码。故 **openapi 不升版（仍 v0.10.0，无契约变更）**。

## 2. create/update 唯一键扫描清单（对照 spec/migrations 唯一索引，权威）
全库业务单列唯一索引共 5 个（grep `spec/migrations` UNIQUE）：`sys_user.username` / `sys_role.code` / `sys_post.code` / `sys_dict_type.dict_type` / `sys_config.config_key`。逐写操作判定：

| # | 实体.方法 | 写操作位置 | 唯一索引 | 本片前行为 | 收口 |
|---|---|---|---|---|---|
| 1 | user.Create | user_service.go tx.Create(&user) | idx_sys_user_username | 软删/竞态→**500** | ErrUsernameExists（**语句处**就地映射，避开 sys_user_post 复合 PK 混淆） |
| 2 | user.Update | user_service.go Updates(无 username 列) | — | 不写唯一列→**不可能 1062** | 无需（已标注） |
| 3 | role.Create | role_service.go Create(&role) | idx_sys_role_code | 软删/竞态→**500** | ErrRoleCodeExists |
| 4 | role.Update | role_service.go Updates(含 code) | idx_sys_role_code | RowsAffected==0 误吞 | ErrRoleCodeExists（dup 检查前置于 RowsAffected） |
| 5 | post.Create | post_service.go Create(&post) | idx_sys_post_code | 软删/竞态→**500** | ErrPostCodeExists |
| 6 | post.Update | post_service.go Updates(含 code) | idx_sys_post_code | RowsAffected==0 误吞 | ErrPostCodeExists |
| 7 | dict.CreateType | dict_service.go Create(&dt) | idx_sys_dict_type_type | 软删/竞态→**500** | ErrDictTypeExists |
| 8 | dict.UpdateType | dict_service.go Updates(含 dict_type) | idx_sys_dict_type_type | **无预检** → RowsAffected==0 误返 **404** | ErrDictTypeExists |
| 9 | config.Create | dict_service.go Create(&cfg) | idx_sys_config_key | 软删… config 硬删/竞态→**500** | ErrConfigKeyExists |
| 10 | config.Update | dict_service.go Updates(含 config_key) | idx_sys_config_key | **无预检** → RowsAffected==0 误返 **404** | ErrConfigKeyExists |

**明确排除（对照 DDL，无业务唯一键 / 范围外，不会或不属本片）**：
- `dict_data` Create/Update、`dept` Create/Update、`menu` Create/Update — 对照 DDL **均无 UNIQUE 业务键**（menu 的 perm_code 唯一性是 app 层 `validateMenuType` 预检，DB 无唯一索引）→ 不产生 1062。
- junction 复合主键（sys_user_post / sys_role_menu / sys_user_role）— 非业务唯一键。user.Create 的 post 关联插入若传重复 post_id 可撞 PK 1062，但**已就 user 写语句处精确映射**，post 插入仍走原路径（边缘，标注，不误转成"用户名已存在"）。
- `casbin_rule`(ptype,v0..v5) / `sys_migration`(version) — 内部复合键，非用户 create/update 端点。

## 3. 1062 捕获/转码单一出入口
**新建中立包 `server/dberr`**（同 idcodec：rbac/system 共用、互不 import，依赖仅 mysql 驱动 + gorm）：
```go
func IsDuplicate(err error) bool  // gorm.ErrDuplicatedKey || *mysql.MySQLError{Number==1062}；nil/其它(外键1452等)→false
```
- **检测逻辑唯一一处**：`Number == mysqlDupEntry` 只在 dberr.go（grep 实证无散落裸 1062 判定）。
- 兼容双路径：未开 TranslateError（当前）→ 查 `*mysql.MySQLError`；将来开启 → 查 `gorm.ErrDuplicatedKey`。`errors.As` 透穿 `fmt.Errorf("%w")` 包裹。
- 各 service 在写边界 `if dberr.IsDuplicate(err) { return s.errs.ErrXxx }`，**返回友好码原样不包裹**——关键：`response.HandleError` 用 `err.(Coder)` 类型断言提码，被 `fmt.Errorf` 包裹会断言失败落 500。user.Create 在 tx 外加 `if _, ok := err.(interface{GetCode()int}); ok { return err }` 守护透传。

## 4. 真 MySQL 集成测试（任务铁律：真库才是终验）
- `rbac/dup_integration_test.go`：user/role/post **软删后重建**（唯一索引不含 deleted_at，预检软删 scope 漏判 → Create 撞 1062 → backstop），断言友好码；user 另测简单重名（预检路径）。
- `system/dup_integration_test.go`：dict_type/config **Update 改名撞键**（无预检，直撞 backstop，验证从误 404→正确 409），另测简单重名。
- 每条均 `assertDupCode`：① 错误实现 GetCode 且 ==预期码（非 500/裸 error）② **中立性=精确等值**：`err.Error()=="errcode:N"`（干净 errcode.Error 形态，任何包裹/裹 DB 原始错误都会令串不等而挂）。
- **真库已跑通（2026-06-09，本地起 Docker Desktop + docker compose MySQL 8.0）**：`go test -tags=integration ./rbac/... ./system/... -run Dup -v -count=1` → **全 PASS**（rbac 5 + system 4 + 既有 1）。
- **复盘（daxing 首轮真跑暴露 2 个测试侧 bug，均已修，非生产 bug）**：
  1. **config 中立性断言误伤**：`ErrConfigKeyExists = 11000+62 = 11062`（合法既有码），旧断言黑名单含子串 `"1062"`，`"errcode:11062"` 文本含 `"1062"` → 误判泄漏。config 生产代码本就裸返正确码（真跑返 11062 即坐实）。修：断言改**精确等值**（更严、不误伤码值数字）。
  2. **role 测试表缺列 1054**：`data_scope` 由 `T003c_sys_role_data_scope.sql` ALTER 添加，原 setup 只载 T003b → role.Create 写 DataScope 撞 1054、遮住 role backstop。修：setup 补载 T003c（与生产迁移序一致），role backstop 真验证 PASS。
- 诚实补充：GORM SQL logger 在 DB 调用处会把原始 `Error 1062 ... idx_xxx` 打到 stderr（服务端日志，service 捕获前），**这不改变客户端/返回值**（exact-equality 断言坐实返回干净 errcode）；调 GORM logger 级别属另事，本片不动。

## 5. 错误码 / Registry / 迁移快照
- **零新增错误码**，零段位变更。5 个友好码（Username/RoleCode/Post Code/DictType/ConfigKey Exists）本就在 errcode 注册表 + response.Registry，HTTP 全 409。
- 迁移回归快照 **N/N 不变**（`response` 测试通过，未动 errcode）。冲突 fail-fast 机制未触（无新码）。

## 6. 自验结果
- `go build ./...` ✅ | `go vet ./...` ✅ | `go vet -tags=integration ./...` ✅（集成编译）
- `go test ./...` ✅ 全包绿（含新 `dberr` 单测 9 子用例：1062 直接/包裹、ErrDuplicatedKey、外键 1452/1364 不误转、NotFound/普通 error 不误转）
- `go mod tidy`：`go-sql-driver/mysql` 由 indirect 转**直接依赖**（dberr 直接 import），go.mod 更新。
- 先前已红的 `TestNewEnforcerMySQL_RoleInheritance`（T-003d-fix 待办，需真库 -tags=integration）本片**未新增红**（非集成全绿；该用例与本片无关）。
- gofmt：`gofmt -l` 标到 role_service.go / dict_service.go **两处既有单行 if 风格**（`normalize()`、List 拼接，**非本片改动行**）；新增行全部 gofmt-clean。刻意不重格式化既有行以锁定 diff 范围（项目门禁=build/vet/test，非 gofmt；codebase 历史即此风格）。

## 7. 安全自查
- **不泄漏 DB 细节**：友好 message 经 Registry i18n（"用户名已存在"等），`errcode.Error.Error()` 仅返 `errcode:N`，绝不含索引名/表名/原始 SQL；集成测试 assertNeutral 断言。补充：**当前 500 响应体本就是通用 `"internal error"`（HandleError 落 500 不回显原始错误），索引名只可能进日志**——修后 1062 根本不到 500 路径，日志亦不再现 1062 噪声。
- **防枚举权衡（点明）**：“XX 已存在”会泄漏“某 key 是否存在”。但本片所有 create/update 端点**均挂 RequirePerm（T-003d enforce，非公开注册）**，是管理员后台操作；管理员知晓记录存在合理，**不构成对匿名用户的枚举面**。底座是通用后台非公开注册场景，明确提示可接受。
- 未改鉴权、未绕过 enforce、未改 service 对外签名（转码在既有 return error 路径内完成）。

## 8. 改动文件清单
| 文件 | 说明 | 类型 |
|---|---|---|
| server/dberr/dberr.go | 1062/ErrDuplicatedKey 检测单一出入口（中立包） | 新增 |
| server/dberr/dberr_test.go | IsDuplicate 单测（9 子用例，含负例不误转） | 新增 |
| server/rbac/user_service.go | Create 1062→ErrUsernameExists（语句处+透传守护） | 修改 |
| server/rbac/role_service.go | Create/Update 1062→ErrRoleCodeExists | 修改 |
| server/rbac/post_service.go | Create/Update 1062→ErrPostCodeExists | 修改 |
| server/system/dict_service.go | dict_type/config Create/Update 1062→友好码 | 修改 |
| server/rbac/dup_integration_test.go | 真 MySQL：user/role/post 软删重建得友好码 | 新增 |
| server/system/dup_integration_test.go | 真 MySQL：dict_type/config 改名撞键得友好码 | 新增 |
| server/go.mod (+go.sum) | go-sql-driver/mysql 转直接依赖 | 修改 |

## 9. 需 daxing 真人验收（API 层）
- `docker compose up` 起 MySQL 后跑集成测试（命令见 §4），全 PASS。
- demo/curl：重复创建同名用户 → **4xx + "用户名已存在"**（不再 500）；抽验重复 dict_type / config_key 同样返友好码；友好 message 不含索引名/表名/原始 SQL。

## 10. 偏差与待办
- **偏差（认可请 PM 裁）**：任务书假设"新增业务码"，实测**码已存在、只需接转换链路**，故未新增码、openapi 不升版。这是更省的正解（不造冗余码）。
- 待办（不阻塞）：openapi 是通用 envelope 模型，未逐端点枚举 409 错误响应（既有，非本片引入）；将来 openapi 强类型化时可补各 create/update 的 409 响应文档。
- 待办（不阻塞）：junction 表（如 sys_user_post）传重复 ID 的 PK 1062 未专门转友好码（边缘、非 5 业务键），用到再评估是否在入参层去重。

## 11. 下一步
PM 评审 + daxing API 验收（真库集成 + curl 不再 500）放行后，贴 git status + diff --stat 待核 → 双推 Gitee 主 + GitHub 镜像。在放行前不提交、不双推。
