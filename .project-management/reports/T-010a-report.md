# 完成报告：T-010a 集成测试连接参数 env 收口

## 1. 完成状态
已完成（待 PM 评审 → 放行 → 才双推；执行端未自标完成/未双推/未改 PROJECT_STATUS）。
全仓集成测试硬编码 localhost MySQL/Redis 连接统一为「读 `BENXIN_TEST_*` env、默认 localhost」，新建 `internal/testsupport` 单点维护默认值与 env 名，退役重复命名 `DEMO_E2E_*`。零回归、env 真生效双向坐实。

## 2. 改动文件清单（11 个：1 新增 + 10 修改）
| 路径 | 说明 | 类型 |
|---|---|---|
| `server/internal/testsupport/conn.go` | **新建**共享 helper 包（`//go:build integration`）：`MySQLDSN()`/`RedisAddr()` 读 env + 默认；`EnvMySQLDSN`/`EnvRedisAddr`/`DefaultMySQLDSN`/`DefaultRedisAddr` 常量单点定义 | 新增 |
| `server/rbac/mysql_integration_test.go` | testDSN const→`var = testsupport.MySQLDSN()`（dup 同包复用自动覆盖）+ import + @updated | 修改 |
| `server/rbac/org_integration_test.go` | orgTestDSN const→var（**§2 清单遗漏、§1/§7 要求补改**，见 §8）+ import + @updated | 修改 |
| `server/system/system_integration_test.go` | intTestDSN const→var + import + @updated | 修改 |
| `server/system/query_integration_test.go` | qeDSN const→var + import + @updated | 修改 |
| `server/system/file_integration_test.go` | fileDSN const→var + import + @updated | 修改 |
| `server/crypto/redis_integration_test.go` | `Addr:` →`testsupport.RedisAddr()` + import + @updated | 修改 |
| `server/auth/auth_integration_test.go` | `Addr:` →`testsupport.RedisAddr()` + import + @updated | 修改 |
| `server/auth/redis_integration_test.go` | `Addr:` →`testsupport.RedisAddr()` + import + @updated | 修改 |
| `server/system/config_integration_test.go` | inline `Addr:` →`testsupport.RedisAddr()` + import + @updated | 修改 |
| `server/examples/demo/e2e_integration_test.go` | 连接键退役 `DEMO_E2E_MYSQL_DSN`/`DEMO_E2E_REDIS_ADDR`→`testsupport.*`；删 `defaultE2EDSN`/`defaultE2ERedis` const + `envOr` func + `os` import；doc 注释改 `BENXIN_TEST_*`；**测试密钥/secret 一律未动** + @updated | 修改 |

- migrator（`system/migrator_integration_test.go`）按 §2 作既有样板**保留**，且它本就读 `BENXIN_TEST_MYSQL_DSN`（与本片统一命名空间天然一致）。
- 未升 openapi、无 DDL/错误码/权限码、未改 app 运行时配置层（`DEMO_*` viper 不动，分层保持）、未改任何测试断言/依赖门控形态（`t.Fatalf` fail-loud 保留）、未动 captcha 合理 `t.Skip`。

## 3. 接口实现情况
无接口变更。

## 4. 自验结果（实跑输出）

**[1] 默认闸门 `go test ./... -count=1`**：全绿（不设 env→默认 localhost，与现状一致）。

**[2] 全仓 `go test -tags=integration ./... -count=1`**：全绿，**266 PASS / 0 FAIL**（与 T-010 A1 基线 266 逐字一致，零回归）。`internal/testsupport` 显示 `[no test files]`（纯 helper，正确）。

**[3] env 真生效正向证据（命门，防"改了但没读 env"假绿）**：
```
[MySQL] export BENXIN_TEST_MYSQL_DSN=...localhost:3399... → 
  --- FAIL: TestNewEnforcerMySQL: connect mysql ...: dial tcp [::1]:3399: connect: connection refused
  （证据：真去连 3399 而非默认 3306 → env 被读取）
[MySQL] unset → ok  rbac  0.724s（PASS）
[Redis] export BENXIN_TEST_REDIS_ADDR=localhost:6399 →
  --- FAIL: TestRedisReplayStore_KeyPrefix: redis ping failed ...: dial tcp [::1]:6399: connect: connection refused
[Redis] unset → ok  crypto  2.090s（PASS）
```
MySQL/Redis 双向坐实：错端口→真 FAIL→unset→PASS，env 真被消费、非写死。

**[4] `go build ./... && go vet -tags=integration ./...`**：均通过（BUILD OK / VET OK）。
- 验证了带 `//go:build integration` 的 helper 包不破 `go build ./...`（无 tag 时被静默跳过，因 `./...` 匹配到其他可构建包，/tmp 实证 + 真仓坐实 exit 0）。

**[5] grep 残留核查**：
- 集成测试连接串无残留裸 localhost（默认兜底除外）：仅 `testsupport/conn.go`（单一默认源）+ `migrator`（§2 指定保留样板）+ e2e/migrator 的**文档注释**保留裸串（非真连接，doc 用途）。
- `DEMO_E2E_*` 真引用 = 0（grep 命中的 1 处是本片 e2e 文件 @updated 注释里的「退役 DEMO_E2E_*」描述词，非代码引用）。

## 5. git 提交记录
未提交、未双推，等 PM 评审放行。`PROJECT_STATUS.md` 的工作树改动 = PM 对 T-003d-fix 的 hash 回填（`{{FINAL_HEAD}}`→16bf4d3、`{{LEDGER_HASH}}`→23e2e4e），属 PM territory，本片未碰、不纳入提交。

## 6. 安全自查
- **零回归硬约束满足**：env 缺省默认值与原硬编码逐字一致（MySQL `root:root@tcp(localhost:3306)/benxinadminpro?...`、Redis `localhost:6379`），不设 env 的本地 `go test -tags=integration` 原样跑通（266 PASS 不变）。
- **127.0.0.1→localhost 归一说明**：demo e2e 原默认用 `127.0.0.1`，统一后用 `localhost`（功能等价，均解析到本地回环；这是命名统一的预期结果，非行为回归）。
- 默认值无真实/生产凭证（root/root 是本地 dev，非生产）。
- 测试连接 env（`BENXIN_TEST_*`）与 app 运行时配置 env（`DEMO_*` viper）严格分层：testsupport 仅读 `os.Getenv("BENXIN_TEST_*")`，不读取/不混用应用配置。
- env 名 `BENXIN_TEST_*` 中性（底座名，无 kp_/okp_/应用名）。

## 7. 需 daxing 真人验收
- **无浏览器可视面**（纯测试基建）。验收 = 审 PM 评审 + 证据包，**env 真生效正向证据（§4-[3]）是命门**（证明读 env 非写死）。同 T-003d-fix「基建正确性归测试、不强塞浏览器」口径。

## 8. 偏差与待办
- **helper vs 逐文件选择 + 理由**：采**共享 helper 包 `internal/testsupport`**（任务推荐项）。理由：① 无 import cycle（仅 import `os`，不反向依赖任何被测包）；② 经 /tmp + 真仓实证，带 `//go:build integration` 的包在 `go build ./...`（无 tag）下被静默跳过、不触发「build constraints exclude all Go files」错误（因 `./...` 总匹配到其他可构建包），故不破 §7 的 `go build ./...` 验收；③ 单点维护默认值/env 名，避免 8+ 处重复。**未退化为逐文件内联**（helper 路径无阻断）。
- **偏差①（§2 清单遗漏，已补）**：任务书 §2 MySQL 清单未列 `rbac/org_integration_test.go:31 orgTestDSN`，但 T-010 摸底 B1 已标它、且 §1「全仓统一」+ §7 grep「无残留裸 localhost」要求覆盖。执行端按 §1 目标 + §7 验收补改（同 rbac 包、同机械变更），已纳入本片。PM 如认为应剔除请指示。
- **偏差②（搭车清理，零风险）**：退役 `DEMO_E2E_*` 时删除了随之失效的 `defaultE2EDSN`/`defaultE2ERedis` const、`envOr` func、`os` import（grep 实证仅 e2e 文件使用、删后 build/vet 净），避免遗留死代码引旧命名。
- **act/其他维度**：无。本片不碰断言/逻辑。
- **CI 接入（T-010b，本片前置已铺平）**：env 收口已完成，CI runner 可经 `BENXIN_TEST_MYSQL_DSN`/`BENXIN_TEST_REDIS_ADDR` 注入连接串。CI 流水线本身仍待 T-010b 立项（全仓无 CI 配置的根因待办仍开）。

## 9. 下一步建议
- PM 评审本证据包 → 放行后双推（Gitee 主 + GitHub 镜像，`git ls-remote` 三方 HEAD 一致校验）→ 由 PM 更新 PROJECT_STATUS（连同 T-003d-fix hash 回填一并落、标 T-010a）。
- 建议 T-010b（CI 接入）直接复用本片 `BENXIN_TEST_*` 注入点 + 既有 docker-compose（带 healthcheck）；CI 最小序列 = compose up + 等就绪 + `go test ./...`（默认闸门）+ `go test -tags=integration ./...`（连 deps）。
