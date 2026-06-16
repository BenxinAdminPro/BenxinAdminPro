# 完成报告：T-011b sys_file 后端能力片（mime 大类筛 + 批量软删端点）

> 性质：纯后端 Go，零前端改动。openapi v0.14.0 → **v0.15.0**（见 §8 偏差①：源码实为 0.14.0 非账本所记 0.14.1）。零 DDL、零迁移、零新增错误码段、零新增权限码（复用 sys:file:delete）。
> HEAD 基线 5a79b47（与 T-011a 无依赖）。
> **未自标完成、未双推、未改 PROJECT_STATUS.md（PM 权限）** —— 待 PM 评审 + daxing 验收 + PM 放行后双推。

## 1. 完成状态
✅ 编码 + 自测完成，待评审/验收。给文件列表加 mime 大类（image/video/audio/other）筛选 + 新增批量软删端点 `POST /sys/files/batch-delete`，二者均为 sys_file 通用能力增强，为 T-011c 媒体页消费打底。

## 2. 改动文件清单（8 文件，+275/-3 + 2 新增测试）
| 文件 | 说明 | 类型 |
|---|---|---|
| `server/system/decode.go` | 加 `decodeHashSlice`（批量 hashid→uint64，fail-fast；nil hasher 退化裸十进制保对称）；头追加 @updated | 修改 |
| `server/system/file_service.go` | `FileListQuery` 加 `MimeCategory`；`mimeCategoryPrefix` 白名单 + `applyMimeCategory`（token→后端常量前缀，用户输入永不进 SQL 片段）；`List` 接入；新增 `BatchDelete`（IN bulk 幂等 + 软删 scope + per-file 物理异步清理）；`maxBatchDeleteIDs=100`；头 @updated | 修改 |
| `server/system/handler_file.go` | `List` 读 `mime_category`；新增 `BatchDelete` handler（空/超限→BadReq 400、非法 hashid→ErrInvalidID 400，写前 fail-fast）；`RegisterRoutes` 加 `POST /sys/files/batch-delete`（复用 sys:file:delete）；头 @updated | 修改 |
| `server/system/handler_authz_test.go` | 文件路由数 4→5；加 `sys:file:delete` 被申领两次（单条 + 批量）断言 | 修改 |
| `server/system/file_integration_test.go` | 加 `TestFileIntegration_MimeCategory`（真库大类筛）+ `TestFileIntegration_BatchDelete`（IN bulk 幂等 + scope 排除 + 物理清理轮询）；头 @updated | 修改 |
| `server/spec/openapi/openapi.yaml` | v0.15.0：`mime_category` 查询参数（枚举）+ batch-delete 端点（ids 数组入参 / deleted_count 出参 / 400）+ changelog；头 @updated | 修改 |
| `server/system/file_service_test.go` | **新增** 默认闸门（SQLite）：mime 大类映射 7 例（含注入尝试不报错）/ 批量软删幂等 / decodeHashSlice fail-fast / handler 入参校验 400 | 新增 |
| `server/examples/demo/file_batch_integration_test.go` | **新增** HTTP e2e：enforce(dept_mgr 200↔editor 403) + 上传 3 文件批量删 deleted_count=3 + 幂等 0 + mime 大类筛 + 负例(空/非法/超100) | 新增 |

## 3. 接口实现情况
- `GET /sys/files` 加查询参数 `mime_category`（可选，枚举 image/video/audio/other；未命中/空不筛、不报错）。
- `POST /sys/files/batch-delete`（挂 `RequirePerm("sys:file:delete")`）：req `{ids:[]hashid}` → resp `{deleted_count:int}`。
  - 校验顺序（写前 fail-fast）：空数组/超 100 → 400(BadReq)；任一非法 hashid → 400(ErrInvalidID)。
  - 幂等：已删/不存在 id 不入命中集，`deleted_count` = 实际软删行数（= IN bulk 命中集大小 = RowsAffected）。
- openapi v0.15.0；零新增错误码段、零新增权限码、零 DDL。

## 4. 自验结果
| 项 | 命令 | 结果 |
|---|---|---|
| 构建 | `go build ./...` | ✅ 净 |
| vet 默认 | `go vet ./...` | ✅ 净 |
| vet integration | `go vet -tags=integration ./...` | ✅ 净 |
| 默认闸门 | `go test ./...` | ✅ 全 ok（system / demo 含新单测） |
| 集成（system file） | `go test -tags=integration ./system/... -run TestFileIntegration` | ✅ ok（mime 大类 + 批量删 + 物理清理） |
| 集成（demo batch e2e） | `go test -tags=integration ./examples/demo/... -run TestFileBatch -v` | ✅ PASS（enforce + 幂等 + mime + 负例，日志四段全过） |
| 集成全量回归 | `go test -tags=integration ./system/... ./examples/demo/...` | ✅ 两包均 ok，无连带红 |

**关键正确性证据**：
- **软删 scope 生效（未踩 .Table 陷阱）**：`BatchDelete` 用模型 `Find`/`Delete`（非 `.Table()`），集成测试坐实批量软删后 `List` 不再返回这些行（SQLite 单测 + 真 MySQL 双证）。
- **幂等**：重复删已删/不存在 id → `deleted_count=0` 不报错（单测 + e2e 双证）。
- **物理清理**：集成测试轮询坐实软删行盘上文件被异步清理、未删行文件仍在。
- **mime 注入防护**：`mime_category=image/%';DROP--` 等白名单未命中 → 不筛、不报错（单测断言），用户输入永不进 SQL 片段。
- **enforce**：demo e2e 坐实 editor POST batch-delete 403（文件未动）↔ dept_mgr 200。

## 5. git 提交记录
- **未提交、未双推**（待 PM 放行）。建议 commit message：`feat(system): T-011b sys_file mime 大类筛 + 批量软删端点（IN bulk 幂等 + 复用 sys:file:delete）`。
- 双推前置：PM 评审 + daxing 验收 + PM 放行 → 推前查 .gitignore（纯 .go/.yaml 源文件，无密钥/产物）→ Gitee origin + GitHub github 双推 → ls-remote 三方一致。

## 6. 安全自查
- **mime_category 强白名单**：客户端只传 token，image/video/audio 前缀由后端常量拼接（参数化绑定），other 走纯常量 SQL；用户输入永不进 WHERE LIKE/ORDER BY 片段（同 T-005b-4 sort 防注入）。负例已断言被忽略不报错。
- **批量删鉴权**：端点挂 sys:file:delete，editor 403 ↔ dept_mgr 200（e2e 坐实，guard 拦在 handler 前、文件未动）。
- **软删 scope 正确**：模型 Delete 让软删 scope 生效（未 .Table()），软删行后续 list 不再出现。
- **解码 fail-fast**：非法 hashid 先于任何写返 400(ErrInvalidID)，防伪造/探测；不暴露内部 uint64。
- **物理清理仅对真实软删的 storage_key 触发**（取自命中集），不误删盘上其他文件。
- 物理清理沿用单条 Delete 的 per-file fire-and-forget（失败仅 slog、不回滚、无法纳事务），与现状一致——不退化。

## 7. 需 daxing 真人验收（demo 验证项）
> **改后端 Go 必须重启 demo 重新编译才生效**（T-005b-1 教训）：`lsof -ti :8080 | xargs kill -9` 后 `cd server && go run ./examples/demo`。
> curl 本机记得 `unset HTTP_PROXY HTTPS_PROXY ALL_PROXY` 或 `--noproxy '*'`（Clash 会代理 loopback 返 000）。

1. **mime 大类筛**（本片用 image vs pdf/txt 验大类逻辑）：上传 jpg/png + pdf/txt → `GET /sys/files?mime_category=image` 仅回 jpg/png、不回 pdf/txt；`?mime_category=other` 仅回 pdf/txt。`video`/`audio` 大类当前返空=正确（demo 尚无此类文件，待 T-011c 扩 allowed_exts 后补验）。注：前端 mime tab UI 属 T-011c，本片经 curl 验后端筛选。
2. **批量删**：curl `POST /sys/files/batch-delete {"ids":[...]}` 传多 id → `deleted_count` 对 + 列表对应行消失；重复传已删 id → 幂等 `deleted_count:0` 不报错；空数组/非法 hashid/超 100 → 400。

## 8. 偏差与待办
- **偏差①（源码 vs 账本，以源码为准并回报）+ PM 指令搭车补回（已执行）**：
  - **核实证据**：① version 起点确为 **0.14.0**（openapi.yaml 最近真实改动 = 1d21cf4/T-005b-3）。② T-009b（c9db3a9）**从未触碰 openapi.yaml**——该 commit 改 8 文件（errcode.go/menu_service.go/user_service.go/org_integration_test.go/service_test.go/migration_test.go/render.go/render_message_test.go），`git show --stat c9db3a9 | grep -c openapi.yaml = 0`；commit message 里"openapi v0.14.0→v0.14.1（错误码枚举 +1）"是报告文本、diff 未落文件。③ **openapi.yaml 根本没有「逐个错误码枚举」**——错误码仅在 `SecurityError.code.description` 以粗粒度 offset 段范围采样说明（原仅列 T-001/T-002），`ErrInvalidParentMenu` 不是"枚举缺一项"而是"整个 T-009b openapi 增量未物化、且无逐码枚举可补"。
  - **搭车补回（PM 指令②·仅文档层零 Go 改动）**：本版 0.14.0 → **0.15.0**（跳过未物化的 0.14.1）。在 `SecurityError.code.description` 采样说明补 RBAC 段菜单专属父节点码 +47（sys.invalid_parent_menu），并在 changelog v0.15.0 显式注明「搭车补回 T-009b 漏物化的 ErrInvalidParentMenu（原计划 0.14.1）」。**Go 错误码常量未碰**（T-009b 已落并在用）；纯文档补全、无端点/schema/行为变更（YAML 解析通过、build/vet 仍净）。
  - 请 PM 据此校正账本/KB：openapi 真实版本线为 …→0.14.0(T-005b-3)→**0.15.0(T-011b，含 T-009b 错误码追补物化)**，从无 0.14.1 实体。
- **设计决策（任务书已锁，记录）**：批量删物理清理为 per-file fire-and-forget，未纳入事务（与单条 Delete 一致，任务书 §2 明确不包含）；部分失败语义 = 命中集软删原子（单 UPDATE）、物理清理各自异步不回滚。
- 无其他偏差；§2 不包含项（前端 / 扩 allowed_exts / 媒体元信息·解码库 / Range）均未碰。

## 9. 下一步建议
- T-011c（媒体管理消费片）：消费本片 `mime_category` 大类 tab + `batch-delete`（叠加 T-011a 多选）+ 图片预览（el-image/blob）+ 视频/音频内联超阈值降级 + 扩 demo allowed_exts 加 video/audio（届时可补验 video/audio 大类筛）。
- 媒体元信息（缩略图/宽高/时长）本批未引解码库、留扩展点（守中立），如未来需要作独立评估（图片宽高可走标准库 image 低成本，视频时长需 ffmpeg 类重依赖）。
</content>
