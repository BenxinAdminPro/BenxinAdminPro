# 完成报告：T-009b 文案/低优债批收尾（file 文案缺键 + 菜单父节点错误码 + SetStatus 同值 404 修复）

## 1. 完成状态
执行端自测全绿（默认闸门 + `-tags=integration` MySQL 真库），三项内聚低优债清零，零行为外扩。等待 PM 评审 / daxing（可选）验收 / PM 放行后双推。

三项落地（PM 评审通过后裁定 A：文案补全从 file 6 键扩为 **8 键 = file 6 + config 2**）：
1. **文案补 8 键**：response 渲染层 `defaultMessages` 补全 6 个 file 错误文案（修 T-007f §8-2）+ 搭车补 config 中心 2 键 `sys.config_decrypt_failed`(11080)/`sys.migration_failed`(11081)（原 §8 范围外发现，PM 裁定 A 并入本片），均消除返裸 i18n key。
2. **菜单父节点专属错误码**：新增 `ErrInvalidParentMenu`（offset 47，段尾纯追加），`menu_service.go` 三处由复用 `ErrInvalidParentDept` 改用新码（修 T-007g §8-2）。
3. **SetStatus 同值 404 修复**：`RowsAffected==0` 时显式探测存在性，区分"无变更（幂等成功）"vs"记录不存在（仍 404）"（修 T-007h 观察）。

## 2. 改动文件清单（路径 + 说明 + 新增/修改）
| 文件 | 说明 | 类型 |
|---|---|---|
| `server/errcode/errcode.go` | 新增 `OffsetInvalidParentMenu=47` + httpStatus(400) + i18nKeys(`sys.invalid_parent_menu`) + Registry 字段 `ErrInvalidParentMenu` + NewRegistry/allOffsets/allErrors 同步追加（段内顺序一致） | 修改 |
| `server/response/render.go` | `defaultMessages` 补 6 file 键 + `sys.invalid_parent_menu` + config 2 键（`config_decrypt_failed`/`migration_failed`）文案 | 修改 |
| `server/rbac/menu_service.go` | Create 父不存在(:102)、Update 新父不存在(:156)、Update 成环(:161) 三处 `ErrInvalidParentDept`→`ErrInvalidParentMenu` | 修改 |
| `server/rbac/user_service.go` | `SetStatus` 修复：`RowsAffected==0` 时 Count 探测存在性（带软删 scope），存在=幂等成功、不存在=`ErrUserNotFound` | 修改 |
| `server/response/migration_test.go` | 回归快照 +1：`{11047, 400, "sys.invalid_parent_menu"}`；计数 40→41 | 修改 |
| `server/response/render_message_test.go` | **新增**：8 键（6 file + menu + 2 config）经完整渲染路径返中文非裸 key 断言 + `defaultMessageLookup` 直测 | 新增 |
| `server/rbac/service_test.go` | **新增 4 测试**：SetStatus 不存在→404（负例）/真变更→成功/同值→成功（SQLite）；菜单父节点三场景返新码不返 dept 码 | 修改 |
| `server/rbac/org_integration_test.go` | **新增** `TestOrgSetStatusSameValue_MySQL`：MySQL 真库三态（同值→成功命门 / 不存在→404 / 真变更→成功） | 修改 |

> 注：`.project-management/PROJECT_STATUS.md` 在本会话开始前即为 `M`（git status 快照），本片**未触碰**（守 T-006 铁律，PROJECT_STATUS 改动权属 PM）。

## 3. 接口实现情况
- **不新增/不改端点**。
- **错误码 +1**：`ErrInvalidParentMenu` = `11047`（段基址 11000 + offset 47），归属 **T-003b RBAC 核心段（40~49）**。该段既用 offset 40~46，**47 为段内下一个空码、无冲突**，48/49 仍留空。既有码值零变更（migration 回归快照 41/41 逐项对照通过）。HTTP=400、i18nKey=`sys.invalid_parent_menu`、文案"无效的父菜单"。
- **8 键均为文案补全非新增码**：file 6 码（11070~11075）+ config 2 码（11080/11081）早已注册，缺的只是 `defaultMessages` 文案 → **migration 回归快照计数不变（仍 41）**，已核实确认（`TestErrcodeRegressionViaRegistry` 的 `len(specs)==len(expectedCodes)==41` 断言通过）。
- **openapi**：错误码枚举 +1（无端点/schema 变更），patch 级 v0.14.0 → v0.14.1。账本记 spec envelope 未逐端点枚举错误码（409 观察），故按既有 spec 粒度处理，未强制铺全（见 §8）。

## 4. 自验结果（go build / vet / test，契约/安全用例）
- `go build ./...` ✅
- `go vet ./...` ✅
- `go test ./...`（默认闸门 SQLite）✅ 全包绿，含：
  - `response`：`TestErrcodeRegressionViaRegistry`（快照 41/41，计数不变）、`TestRenderFileAndMenuMessages`（**8 码**渲染返中文非裸 key）、`TestDefaultMessageLookupNoBareKey`（8 键）全 PASS。
  - `rbac`：`TestUserSetStatusNotFound`/`TestUserSetStatusRealChange`/`TestUserSetStatusSameValueIdempotent`/`TestMenuParentUsesMenuErrorCode` 全 PASS。
- `go test -tags=integration ./...`（真 MySQL :3306）：
  - `response` ✅ / `system` ✅。
  - `rbac`：新增 `TestOrgSetStatusSameValue_MySQL` ✅（同值 1→1 返成功、不存在→`ErrUserNotFound`、真变更 1→0 落值），其余 rbac 集成测试全绿。
  - ⚠ **唯一红：`TestNewEnforcerMySQL_RoleInheritance`（casbin 角色继承）= 既有 pre-existing red，与本片无关**。已 `git stash` 在干净 HEAD 6503cfe 实证同样 FAIL（alice should inherit admin role permissions），非本片引入。`-skip` 该用例后 rbac 集成全绿。

**SetStatus 三态断言（命门）**：
| 场景 | 期望 | SQLite 单测 | MySQL 集成 |
|---|---|---|---|
| status 设相同值 + 记录存在 | 成功（不 404） | ✅（SQLite 同值返 RowsAffected==1，旧实现亦过，仅文档化） | ✅ **命门**（MySQL 同值返 RowsAffected==0，旧实现误返 404，本测试坐实修复） |
| status 设值 + 记录不存在 | `ErrUserNotFound`（不放宽存在性） | ✅ 负例 | ✅ 负例 |
| status 真变更 | 成功 + 落值 | ✅ | ✅ |

> 关键："同值→成功"的真正区分点是 MySQL `RowsAffected==0` 路径（SQLite 不暴露缺陷，吃账本"单测绿≠装配能跑"教训），故由 `-tags=integration` MySQL 测试兜底命门。

## 5. git 提交记录（commit message / 是否双仓同步）
PM 复核通过后放行，三独立提交（顺序：feature → 账本 → 报告归档），双推 Gitee 主 + GitHub 镜像：
1. **feature** `c9db3a9`：`fix(system): 8 键文案补全 + 菜单父节点专属错误码 ErrInvalidParentMenu + SetStatus 同值幂等不误返 404 (T-009b)` — 8 源/测试文件 +323/-4。
2. **账本** `912b93b`：`docs(status): T-009b 文案低优债批收尾归档（并落 T-009a HEAD 对账回填）` — PROJECT_STATUS.md 由 PM 产出原样落，HEAD/feature/账本/报告 hash 留占位（双推后回报 PM 补对账）；同笔落地 T-009a HEAD 对账回填（工作区 leftover）。
3. **报告归档** `本提交`：`docs(task): T-009b 任务书与完成报告归档` — 任务书（含补充指令/放行要点）+ 本报告（8 键口径）= 最终 HEAD。
双推后三方 HEAD 一致（LOCAL==Gitee origin==GitHub github）由 ls-remote 实测核验，最终 HEAD = 报告归档提交 hash。

## 6. 安全自查
- **不碰鉴权/加密/数据权限**：纯文案 + 错误码 + 幂等修复。
- **SetStatus 存在性校验未放宽**（§5 安全要求核心）：`RowsAffected==0` 改为先 Count 探测——记录存在才视为"无变更成功"，**不存在/软删的 id 仍正确返 404**。Count 走 `Model(&SysUser{})` 带 GORM 软删 scope，软删用户不被误判为"存在"。负例（不存在→404）SQLite 单测 + MySQL 集成双证，杜绝"一刀切返成功=不存在 id 也成功"的正确性倒退。
- **错误码段不破坏**：新码纯追加 offset 47，既有 40 码值零变更（回归快照逐项断言）；冲突 fail-fast 由 `response.Registry.Register` 既有机制保证；`response.Registry` 仍是唯一注册/渲染权威。
- 无 SQL 注入面（无新查询入参）；无对外 ID 暴露变化；无新增端点/权限码/DDL。

## 7. 需 daxing 真人验收（demo 验证项，可选——本片纯后端文案/码，自动化已足）
1. 后端重启：`lsof -ti :8080 | xargs kill -9` 后 `cd server && go run ./examples/demo`（本片改 Go，需重启，前端热更新不含后端）。
2.（可选）curl 触发 file 错误（如 >10MB 上传）→ 错误 `message` 为中文（"文件大小超出限制"等）非裸 `sys.file_too_large`（curl 加 `--noproxy '*'`）。
3.（可选）菜单页移动节点成环/选不存在父 → 提示与"父菜单"相关，非"父部门"。
4.（可选）`curl PUT /sys/users/:id/status` 传当前相同 status → 返成功非 404。

## 8. 偏差与待办
- **范围内但需 PM 知会（getByID 第 4 处 ErrInvalidParentDept）**：`menu_service.go` 实有 **4 处** `ErrInvalidParentDept` 引用，本片按任务书"三处"精确处理了语义为"父节点"的 3 处（Create 父不存在 / Update 新父不存在 / Update 成环）。第 4 处在 `getByID`（:239 附近，记录不存在时返 dept 码）：
  - 经 Create 调用时其返回值被 :102 显式覆盖（不外露）；
  - 经 **Update 自身查找**（编辑一个不存在的菜单）时会外露 `ErrInvalidParentDept` —— 语义实为"被编辑的菜单不存在"，却返"无效的父部门"，是**既有轻微语义瑕疵**。本片未动（严守"三处"边界，且无 `ErrMenuNotFound` 码、改用 `ErrInvalidParentMenu` 对"自身不存在"亦不精准）。**建议 PM 裁定**是否单开微调（治本需新增"菜单不存在"码或复用现有 NotFound 语义）。
- **~~范围外发现（render 层另缺 2 键）~~ → 已并入本片（PM 裁定 A）**：`defaultMessages` 此前另缺 T-005 配置中心 2 键 `sys.config_decrypt_failed`(11080) / `sys.migration_failed`(11081)。PM 评审后裁定搭车补全，已补中文文案（"参数解密失败"/"数据库迁移失败"）+ 渲染断言纳入（8 键），码本就注册、快照计数不变（41）。**本项已闭环，无遗留。**
- **既有 pre-existing red**：`TestNewEnforcerMySQL_RoleInheritance`（casbin 角色继承集成测试）在干净 HEAD 6503cfe 即 FAIL，与本片无关（已 stash 实证）。**建议 PM 单独立项排查**（疑似 casbin gorm-adapter 角色继承/策略加载，非文案债）。
- **未触及范围外项**：§2 ⛔ 清单（dict_type 改名 / 悬空 sys:secret:view / storage_key mask / openapi 强类型化 / junction 1062 / GORM logger 级别 / 超管短路 / 菜单父子类型校验 / 部门管理页 / PolicySync 原子性 / 集成测试硬编码端口 / T-003d-fix 等）一律未碰。
- **openapi 落版**：本片未实改 openapi.yaml（账本记 spec 错误码未逐端点枚举），错误码 +1 按既有 spec 粒度。若 PM 要求 spec 显式记 v0.14.1 changelog，可补一行；当前按"不强制铺全"处理。

## 9. 下一步建议
1. PM 评审本报告（重点核：新码段归属 47/段内零变更、SetStatus 三态尤其"不存在仍 404"负例双证、6 file 键文案）。
2. PM 裁定 §8 剩余项：getByID 第 4 处（菜单自身不存在返"父部门"）是否微调（config 2 键已按裁定 A 并入本片闭环）。
3. 放行后双推（Gitee 主 + GitHub 镜像），账本 HEAD/提交链按实际 hash 回填对账。
4. casbin `TestNewEnforcerMySQL_RoleInheritance` pre-existing red 建议单独立项。
