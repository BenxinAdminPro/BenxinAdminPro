# 完成报告：T-010b CI 接入 GitHub Actions（默认闸门 + integration 双跑）

## 1. 完成状态
✅ 已完成并双推（PM 评审通过验证半程后放行）。HEAD = `dfc5fc1`，三方一致（LOCAL == Gitee origin == GitHub github == dfc5fc1，`git ls-remote` 实测）。
闭合根因债：**全仓此前无任何 CI 流水线**（首跑前 Actions `total_count=0` 坐实），`-tags=integration` 从未自动跑过 —— 现每次 push(main)/PR(main) 自动跑默认闸门 + integration 双跑。

## 2. 改动文件清单
| 路径 | 说明 | 类型 |
|---|---|---|
| `.github/workflows/ci.yml` | GitHub Actions CI 工作流（build + vet×2 + 默认闸门 + integration 闸门，mysql:8.0 + valkey:8 service container，127.0.0.1 连接 env 注入） | 新增 |
| `README.md` | 加 CI status badge + 项目/CI 简述（原为空文件） | 修改 |

- **未改任何 Go 业务/测试代码**（A1 摸底已证接上即绿，CI 首跑 0 真红，无需夹带修）。
- **未碰 `PROJECT_STATUS.md`**（PM 账本权限；其既有未暂存占位填实编辑全程未触）。

## 3. 接口实现情况
无接口变更。未升 openapi（仍 v0.14.1）、无 DDL / 错误码 / 权限码 / secrets。

## 4. 自验结果
**本地（HEAD dfc5fc1，含 127.0.0.1 CI env 镜像）：**
- `go build ./...` ✅
- `go vet ./...` ✅ / `go vet -tags=integration ./...` ✅
- `go test ./... -count=1`（默认闸门）✅ 全绿
- `go test -tags=integration ./... -count=1`（127.0.0.1 env）✅ **266 PASS / 0 FAIL**（逐字对齐 T-010 A1 基线）
- YAML 语法校验通过（解析出 job=test、8 steps、on=push+pull_request）；`ci.yml` 纯 ASCII（`file` 报 ASCII text，吃 T-009a 中文头乱码教训）。

**CI 首跑绿（run #1，push 触发，dfc5fc1）：**
- https://github.com/BenxinAdminPro/BenxinAdminPro/actions/runs/27555518589 — conclusion=**success**，~2m47s，全 10 步绿。
- 命门步：步骤 8「Wait for MySQL ready」（`mysqladmin ping -h 127.0.0.1 -uroot -proot`）✅ + 步骤 9「Test (default gate)」✅ + 步骤 10「Test (integration gate)」✅ → 默认 + integration 双步真跑且双绿；127.0.0.1 注入真连 service container（无 ::1 抖动）。

**假红验证（同一条 PR #1，pull_request 触发）：**
- 红态 run #2（`8a885e2`）: https://github.com/BenxinAdminPro/BenxinAdminPro/actions/runs/27557023860 — conclusion=**failure**，失败步=「Test (default gate)」（`TestDefaultXRobotsTagValue` 断言 `got "noindex, nofollow", want "FAKE-RED-VERIFY-T010b"`），integration 步被跳过 → 闸门在默认门即拦下，对真红绝不放行。
- 绿态 run #3（`288b07d`）: https://github.com/BenxinAdminPro/BenxinAdminPro/actions/runs/27557483159 — conclusion=**success**，撤回破坏后同一 PR 立即恢复，两 gate 均跑均绿。
- 触发器：`pull_request` 闸门真生效（daxing 浏览器侧确认 + API 坐实）。

## 5. git 提交记录
- feature commit：`dfc5fc1` `ci(workflow): add GitHub Actions CI - default gate + integration gate (T-010b)`
- 假红临时 commit（仅活在已删分支，**未进 main**）：`8a885e2`（改坏）→ `288b07d`（改回）。
- 双推：先仅推 GitHub 镜像触发 Actions → 拿证据回报 PM → PM 放行后补推 Gitee。**三方一致 == dfc5fc1**（ls-remote 实测）。

## 6. 安全自查
- workflow 不含任何真实/生产凭证：mysql root/root + 测试 DSN 均为一次性 service container 本地 dev 凭证（对齐 docker-compose.dev.yml），非生产；GCM/JWT 等业务密钥不出现（test-only CI 自带固定测试密钥，不读业务 secrets）。
- `permissions: contents: read`（最小权限）。第三方 action 钉官方主版本（actions/checkout@v4、actions/setup-go@v5）。
- 127.0.0.1 显式注入规避 IPv6 ::1 回环抖动（T-010a watch-point 闭合）。

## 7. 需 daxing 真人验收（已完成项）
- ✅ GitHub Actions 页面绿勾、commit 上 CI 状态标记可见（首跑 run #1）。
- ✅ PR #1 假红真拦（All checks failed）→ 撤回恢复绿，daxing 浏览器侧确认。
- ✅ PR #1 已关闭（不合并）+ `ci/fake-red-verify` 分支已删，main 零污染（`git ls-remote --heads github ci/fake-red-verify` 空）。
- （可选）run 页「Test (integration gate)」日志可见字面 `266 PASS`（unauth API 取不到原始日志，留浏览器侧核对；本地同 env 已复跑 266 实证）。

## 8. CI 片专项交代
- **Go 版本对齐值**：`go.mod` 仅 `go 1.25.0`（无 `toolchain` 指令）→ setup-go 用 `go-version-file: server/go.mod` 直接读 go.mod 的 `go` 指令钉死（不臆造字面值）。
- **DSN 串对齐情况**：`BENXIN_TEST_MYSQL_DSN` = `root:root@tcp(127.0.0.1:3306)/benxinadminpro?charset=utf8mb4&parseTime=true&loc=Local` —— 与 `testsupport.DefaultMySQLDSN` 逐字一致，**仅 host 由 localhost 换 127.0.0.1**（规避 ::1）；`BENXIN_TEST_REDIS_ADDR` = `127.0.0.1:6379`。
- **假红验证处置**：临时分支 `ci/fake-red-verify`（off dfc5fc1）承载破坏 commit `8a885e2`，**从未进 main**；经 PR #1 触发 pull_request 闸门验红→推 `288b07d` 改回验绿→PR 关闭(不合并)+分支删除。净效果 `git diff main -- server/httpmw/robots_test.go` = 空。**破坏 commit 未遗留进任一仓库发布历史**（main/Gitee/GitHub 三方均无该 commit，分支已删）。
- **首跑绿证据**：run #1 = https://github.com/BenxinAdminPro/BenxinAdminPro/actions/runs/27555518589（success）。
- **push workflow 文件的 PAT scope 插曲**：首推被 GitHub 拒（PAT 缺 `workflow` scope）→ daxing 给 PAT 加 workflow scope 并更新 keychain token 后重推成功（credential 由 daxing 掌管，执行端未碰 token 明文）。

## 9. 下一步建议
- **账本由 PM 落**：PROJECT_STATUS 把 T-010b 记为「质量闸门基建批次 T-010 收官片」，HEAD=dfc5fc1。
- 待办池 CI 卫生根因（T-003d-fix / T-010a 上报）至此**闭合**。
- 可选后续（非本片）：①Gitee 侧 CI（如需，另立片，本片只 GitHub Actions）②CI 加 admin 前端 `pnpm test` + `pnpm build` 闸门（T-007i vitest 基建已就绪，可低成本接）③workflow 缓存/并发优化。均按需另拆，不夹带。
