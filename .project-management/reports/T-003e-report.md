# 完成报告：T-003e 入参对外 ID 收口（hashid 入参解码 + 系统性扫尾）

> 状态：**待 PM 评审 + daxing API 层真人验收**。执行端不自标完成、未双推。

## 1. 完成状态
- 范围决策：经 PM 确认走 **方案 A（仅 rbac 入参收口）**。system 包（字典/参数/文件）进出均为裸 uint64（从未 hashid 化，是 T-003b 只覆盖 rbac 的遗留），作为**独立债**记入待办，不在本片。
- rbac 全部"引用对外 ID 的入参"已在 handler/DTO 边界统一收口为 hashid 字符串解码；service 层签名保持 `uint64` 不变（最小爆破面）。
- 非法/伪造/空/越界/被改一位的 hashid 一律 **400（ErrInvalidID，复用既有码，无新增码段）**，不回落、不暴露内部自增 id。
- openapi 升 **v0.9.0**（redocly 0 error），入出参对称；PHP parity 说明已落 spec。
- 自验：`go build` / `go vet` / 全量单测 / 全量集成（含 demo e2e 8 步）全绿，唯一例外是一条**先前就红**的陈旧用例（见 §8，与本片无关，已证在干净 HEAD 同样失败）。

## 2. 改动文件清单
| 路径 | 说明 | 类型 |
|---|---|---|
| `server/rbac/decode.go` | **新增** 入参对外 ID 解码单一出入口：`decodeID`（nil hasher 退化裸十进制保进出对称）+ `decodeOptionalID`/`decodeZeroableID`/`decodeMovableID`/`decodeIDSlice`（分别承载 空→nil、根=0、移动三态、列表 nil/空语义） | 新增 |
| `server/rbac/handler_user.go` | 引入 `userListReq`/`createUserReq`/`updateUserReq`（string 化对外 ID）+ `toQuery`/`toInput` 解码；List/Create/Update/AssignRoles 改走解码；非法→400 | 修改 |
| `server/rbac/handler_role.go` | AssignMenus `menu_ids` 收 `[]string` 解码 | 修改 |
| `server/rbac/handler_dept.go` | 引入 `createDeptReq`/`updateDeptReq`（parent_id string/*string）+ 解码 | 修改 |
| `server/rbac/handler_menu.go` | 引入 `createMenuReq`/`updateMenuReq`（parent_id string/*string）+ 解码 | 修改 |
| `server/rbac/user_service.go` | service Input 对外 ID 字段 `json/form` 标签降级 `"-"`（内部 uint64，由 handler 解码注入；**闭掉"uint64 字段挂可绑定 JSON 名"潜伏陷阱**），Go 类型/签名不变 | 修改 |
| `server/rbac/dept_service.go` | 同上（parent_id 降级 `json:"-"`） | 修改 |
| `server/rbac/menu_service.go` | 同上（parent_id 降级 `json:"-"`） | 修改 |
| `server/rbac/decode_test.go` | **新增** 解码辅助单测（合法/非法/空/越界/被改一位）+ toInput 合法映射 + handler 非法 hashid→400 接线（user/role） | 新增 |
| `server/rbac/decode_integration_test.go` | **新增** `//go:build integration` 真 MySQL：以 hashid 建用户带 dept_id/post_ids→落库内部 id 正确 + user_post 关联；裸 uint64 入参被 400 拒且不落库；伪造 hashid→400 | 新增 |
| `server/spec/openapi/openapi.yaml` | v0.8.1→**v0.9.0**；rbac 实体 path `:id`、body `dept_id/post_ids/role_ids/menu_ids/parent_id`、query `dept_id` 及出参 schema（SysUser/SysDept/SysPost/SysRole/SysMenu 的 id/parent_id/dept_id）`integer→string(hashid)`；info.description 加 v0.9.0 + PHP parity 约定 | 修改 |

> 注：装配/`RegisterRoutes` 签名 **零变更** —— hasher 自 T-003b 起已是 `NewXxxHandler(svc, errs, hasher)` 的构造入参，本片解码直接复用 `h.hasher`，消费方（demo/将来 BenxinKP）无需任何跟进。

## 3. 入参扫描清单（第 3 节方法 · 逐字段"收口/保留 + 理由"）

**扫描方法**：`grep 'json:"*_id(s)?"|form:"*_id(s)?"'` 全库（排除 `_test` 与纯 gorm 主键），逐项判定。证据见 §4。

### 收口（11 项，全 rbac）
| # | 位置 | 字段 | 原类型 | 现入参 | 解码语义 |
|---|---|---|---|---|---|
| 1 | CreateUserInput | dept_id | `*uint64` | `string` | 空→nil（无部门） |
| 2 | CreateUserInput | post_ids | `[]uint64` | `[]string` | 列表解码 |
| 3 | UpdateUserInput | dept_id | `*uint64` | `string` | 空→nil（清空） |
| 4 | UpdateUserInput | post_ids | `[]uint64` | `[]string` | nil=不变 / []=清空 / [...]=替换 |
| 5 | **UserListQuery（查询过滤）** | dept_id | `*uint64 form` | `string form` | 空→不过滤（**初版漏列，彻底扫描补获**） |
| 6 | AssignRoles body | role_ids | `[]uint64` | `[]string` | 列表解码 |
| 7 | AssignMenus body | menu_ids | `[]uint64` | `[]string` | 列表解码 |
| 8 | CreateDeptInput | parent_id | `uint64` | `string` | 空→0（挂根） |
| 9 | UpdateDeptInput | parent_id | `*uint64` | `*string` | nil=不移动 / 空串=移到根 / 值=移动 |
| 10 | CreateMenuInput | parent_id | `uint64` | `string` | 空→0（挂根） |
| 11 | UpdateMenuInput | parent_id | `*uint64` | `*string` | nil=不移动 / 空串=移到根 / 值=移动 |

### 路径参数（已对外 ID 化，本片复核无需改）
- rbac 全部 `:id` path 参数自 T-003b 已走 `parseHID`（hasher.Decode）→ hashid。复核通过。

### 穷尽复核（PM 评审追问点 — 防别名漏网）
- 拉出 rbac+system 全部 `*Input/*Req/*Query` 结构逐字段，确认**唯一承载对外 ID 的入参就是上述 11 项**，无别名漏网：
  - `SysDept.Leader` = `string`（负责人**姓名**，model.go:66），**非 `leader_id`**，无对外 ID。
  - **无**"给角色直接授单个权限码/权限 id"的端点：角色权限统一经 `AssignMenus`(menu_ids，已收口)；权限码 `menu.PermCode` 是稳定语义串（`sys:user:list`，类比 config_key）非实体主键；`/sys/auth/perms` 为 GET 只读。
  - `CreateRoleInput`/`UpdateRoleInput` **无任何对外 ID 入参**（角色无 parent）；system 入参全 string code/key。

### 保留（不收口 + 理由）
| 位置 | 字段 | 理由 |
|---|---|---|
| `UserListQuery.status` / `menu_type` / `data_scope` / `page` / `page_size` | — | 纯枚举/分页值，非对外实体主键 |
| `auth` 包 `captcha_id` | — | 不透明随机验证码令牌，非实体主键，无内部 id 语义 |
| system 包 `CreateDictTypeInput.dict_type` / `CreateConfigInput.config_key` 等 | — | 系统实体以 string code/key 互相引用，**本就无对外 ID 入参** |
| system 路径 `:id`（dict/config/file）+ 出参 model `id` | — | **整块裸 uint64（进出对称、从未 hashid 化）**，整体 hashid 化是独立债（PM 定方案 A 不在本片，见 §9） |
| rbac DB model（SysUser/SysDept/SysMenu）`id`/`parent_id`/`dept_id` `uint64` | — | 持久层字段；对外**出参契约由 ResponseEncoder 定义**（T-003b 已全 hashid），非裸暴露 |
| rbac service Input（CreateUserInput 等）`uint64` 字段 | — | 任务明定"service 层签名保持 uint64"；已降级 `json:"-"` 杜绝误绑定 |

## 4. 自验结果

```
go build ./...        → OK
go vet ./...          → OK
go test ./...         → ok auth/crypto/storage/rbac/response/system（全绿）
go test -tags=integration ./...
  → ok auth / crypto / storage / examples/demo(e2e 8步) / response / system
  → rbac: 仅 TestNewEnforcerMySQL_RoleInheritance FAIL（先前已红，见 §8）
```

**新增针对性用例（全绿）**：
- `TestDecodeID/OptionalID/ZeroableID/MovableID/IDSlice`：合法解码、非法→error、空/nil/[] 语义、**被改一位**不得仍解出原值、nil hasher 退化。
- `TestCreateUserReqToInput`：合法 hashid → 内部 uint64 正确映射。
- **`TestUpdateUserReqPostIDsThreeState`（decode_test.go:167）**：post_ids 三态经 `updateUserReq.toInput` 独立断言 —— 态①缺省→`in.PostIDs==nil`（不变，:177）/ 态②`[]`→非 nil 空切片（清空，:187）/ 态③`[...]`→替换（:197）。
- **`TestUpdateUserReqJSONNilVsEmpty`（decode_test.go:202）**：坐实 JSON 绑定脚枪本身成立 —— 缺省→nil 切片(:208)、`[]`→非 nil 空切片(:215)。
- **`TestUpdateUserPostIDsThreeState_MySQL`（integration，真跑通过）**：三态经 HTTP `PUT` → DB 端到端 —— 缺省=岗位不动(1) / `[]`=清空(0) / `[...]`=替换为 p2(1)。
- `TestUserCreateRejectsBadHashid` / `TestUserCreateRejectsBadPostID` / `TestRoleAssignMenusRejectsBadHashid`：body 非法 hashid → **400**，不触达 svc。
- `TestDecodeHashidInput_MySQL`（integration，真跑通过）：hashid 建用户→落库内部 id 正确 + user_post 关联；**裸 uint64 dept_id（JSON 数字）→ 400 且不落库**；伪造 hashid→400。

**nil hasher 退化生产不可达（PM 评审追问点）**：
- `decode.go` 的 `decodeID` 在 `h==nil` 时退化裸十进制，**仅服务于不接 hasher 的单测**。
- 生产保证（demo 装配）：`NewHasher`（hashid.go）三个 return 要么 `(nil,err)` 要么 `(&Hasher{},nil)`，**永不 `(nil,nil)`**；demo main.go:164-166 对 err **fail-fast**（`return nil, fmt.Errorf("hashid: %w", err)`）→ 构造后 `hashidHasher` 必非 nil，并直接传入全部 `NewXxxHandler`。
- 第二道：装配自检 main.go:204-210 清单含 `"hashid hasher": hashidHasher`，nil 则 `assembly self-check: hashid hasher is nil` 中止。诚实注脚：Go 中 typed-nil 装入 `any` 后 `v==nil` 为假，故此自检对 *typed* nil 指针不可靠 —— **真正的保证是构造器契约 + err fail-fast（上一条），自检为带式裤带**。

**grep 佐证（全链路对外无裸 uint64 主键）**：
- ① rbac handler 所有 bind 目标均为 string 化 req DTO（`grep ShouldBind` 全部指向新 DTO）。
- ② req DTO 对外 ID 字段类型全为 `string/[]string/*string`。
- ③ 残留 `uint64 + 对外 ID json 名`：仅剩 3 个 gorm DB model 字段（持久层，出参经 ResponseEncoder 已 hashid）；service Input 已降级 `json:"-"`。
- 出参侧：ResponseEncoder（T-003b）全 hashid，`response_test.go` 的 `bareIntID` 回归仍守门。

## 5. git 提交记录
- **未提交、未双推**（遵流程铁律：完成判定权在 PM，执行端不自标完成/不自行双推）。
- 待 PM 评审 + daxing API 层验收通过后，再按 `feat(rbac): T-003e 入参对外 ID 收口 hashid` 提交并双仓推送（确认后）。
- 提交前已自查：无密钥/IP/证书/.env；新增文件均为源码与测试。

## 6. 安全自查
- 非法/伪造/空/越界/被改一位 hashid → 统一 400（ErrInvalidID），**绝不回落为某个 uint64**、不暴露内部 id 空间大小、错误信息中立不回显解码细节。
- 防探测：解码失败不区分"id 不存在 vs 格式非法"，均同一 400。
- hasher 盐/配置经装配注入（沿用 T-003b，禁包级可变全局）；decode helper 为纯函数，hasher 由 handler 实例传入。
- 收口后全链路对外（入 path/body/query + 出参）无裸自增主键（grep 佐证 + 集成断言）。

## 7. 需 daxing 真人验收（API 层，选择器 UI 在 T-007h）
- 用 curl / demo：先 `GET /sys/depts/tree` 取某部门返回的 hashid，作为 `dept_id` `POST /sys/users` 建用户成功、`GET` 回读数据正确。
- 故意传裸数字 `"dept_id": 5`（JSON 数字）或乱码 hashid `"dept_id":"abc"` → 被 **400** 拒。
- 同理验 `PUT /sys/users/{id}/roles`（role_ids hashid）、`PUT /sys/roles/{id}/menus`（menu_ids hashid）、`POST /sys/depts`（parent_id hashid）。

## 8. 偏差与待办
- **【偏差·主动扩面】出参 schema 同步**：openapi 出参 schema（SysUser/SysDept/SysMenu/SysPost/SysRole 的 id/parent_id/dept_id）此前仍标 `integer`，但运行时自 T-003b 起早已返回 hashid 字符串——这是 **T-003b 遗留的文档债**。为让"入参 schema 与出参对称"在契约中真正成立，本片一并将出参 schema 改为 `string`。仍属 rbac-only、且是 spec 文档与现实对齐，未改任何出参运行时行为。**请 PM 确认此顺手修订。**
- **【偏差·service 标签】** 为闭掉"uint64 字段挂可绑定 JSON 名"潜伏陷阱（T-006 教训），将 service Input 对外 ID 字段 `json/form` 标签降级 `"-"`。Go 类型/方法签名未变，不属任务所指"service 签名变更"，但触及 service 文件，特此显式标注。
- **【先前已红·非本片】`TestNewEnforcerMySQL_RoleInheritance` 失败**：已用 `git stash` 在干净 HEAD(78342cf) 复现，**与 T-003e 无关**。诊断：该用例断言 `("admin","/api/admin/*","*")` 的 **URL 通配 keyMatch** 语义，但 T-003b 已把 model.conf 改为 **perm code 精确匹配**（账本明载"破坏性变更"）——此测试是 T-003b 当时**漏更新的陈旧用例**，非真鉴权 bug（真鉴权用精确 perm code，demo e2e 第 7 步 403 已证）。建议单独清理（更新或删除该陈旧断言）。
- **【待办·独立切片】system 整体 hashid 化**：字典/参数/文件实体进出仍裸 uint64，需出参（ResponseEncoder for system / 引 hasher 进 system）+ 入参 + path :id + openapi + 集成测试一整块，建议单开切片（如 T-004d）。注：spec 中 `/sys/files/{id}` path 现为 string 与 handler ParseUint 不一致，属该独立债，本片未碰。

## 9. 下一步建议
1. PM 评审本报告（重点：§8 两处主动偏差是否认可）。
2. daxing 按 §7 做 API 层 curl 验收。
3. 通过后执行端按 §5 提交 + 双推，账本翻 ✅。
4. 前端 T-007h（dept/post 选择器）前置障碍已扫清——选择器可直接用列表/树返回的 hashid 作为 `dept_id`/`post_ids` 提交。
