# 完成报告：T-003d-fix 清 casbin TestNewEnforcerMySQL_RoleInheritance pre-existing red

## 1. 完成状态
- 已完成（待 PM 评审 → 放行 → 才双推；执行端未自标完成/未双推/未改 PROJECT_STATUS）。
- HEAD = dacc8a1（未提交，本片仅 1 个测试文件改动）。
- 重写 `TestNewEnforcerMySQL_RoleInheritance` 的陈旧 URL/keyMatch 断言为 perm code 形态，加同主体负向对照正向证明 g 继承，pre-existing red 清掉，rbac `-tags=integration` 全绿可重复执行。

## 2. 改动文件清单
| 路径 | 说明 | 类型 |
|---|---|---|
| `server/rbac/mysql_integration_test.go` | 重写 `TestNewEnforcerMySQL_RoleInheritance`：obj/act 由 URL（`/api/admin/*`、`/api/admin/users`）改 perm code（`sys:user:list`/`access`），镜像 `rbac_test.go:TestEnforcerRoleInheritance` 元组；加同主体负向对照（alice 无链 deny→加链 allow）；保留 act 通配子断言（`sys:user:create`/`*`）；保留 bob 基线。头注释追加 `@updated` | 修改 |

- **未改** `spec/rbac/model.conf`/matcher、未改任何业务/鉴权逻辑、未删任何测试、未升 openapi、无 DDL/错误码/权限码（严守 §2 范围）。
- 注：`.project-management/PROJECT_STATUS.md` 的 `M` 状态是本片开工前即存在（git status 快照），非本片产出，未触碰。

## 3. 接口实现情况
无接口变更。openapi v0.14.1 原样。

## 4. 自验结果（实跑输出）

**[1] 重写后 RoleInheritance（integration，含同主体负向对照）**
```
=== RUN   TestNewEnforcerMySQL_RoleInheritance
--- PASS: TestNewEnforcerMySQL_RoleInheritance (0.04s)
ok  github.com/benxin_dev/benxinadminpro-server/rbac  2.219s
```
断言含：alice 无 g 链 → deny（负向对照前态）；加链 → allow（后态翻转）；act 通配经继承命中；bob 无角色 deny 基线。

**[2] 正向继承覆盖现状（默认闸门）**
```
--- PASS: TestEnforcerRoleInheritance (0.00s)      # rbac_test.go:83，同 model.conf，file adapter
--- PASS: TestPolicySyncUserRoles (0.00s)          # service_b_test.go:240，用户→角色→perm 传递
ok  github.com/benxin_dev/benxinadminpro-server/rbac  1.037s
```

**[3] rbac `-tags=integration` 全套**
```
ok  github.com/benxin_dev/benxinadminpro-server/rbac  1.503s
PASS 计数: 94
FAIL 计数: 0
```
标签可重复执行（`-count=1`），无连带红、无隐藏红。

**[4] 默认闸门**
```
$ go test ./rbac/... -count=1
ok  github.com/benxin_dev/benxinadminpro-server/rbac  0.797s
```

**[5] build / vet**
```
$ go build ./...        → BUILD OK
$ go vet -tags=integration ./rbac/...  → VET OK
```

**[6] 防假绿自检（T-007i 范式）** — 临时把负向对照 `if allowed` 改坏成 `if !allowed`：
```
# 改坏后（应 FAIL）：
    mysql_integration_test.go:149: alice should be denied before grouping policy (...)
--- FAIL: TestNewEnforcerMySQL_RoleInheritance (0.04s)
# 还原后（应 PASS）：
--- PASS: TestNewEnforcerMySQL_RoleInheritance (0.04s)
```
证明测试非空跑、真在断言。已还原，工作树为正确版本。

## 5. git 提交记录
- **未提交、未双推**（等 PM 评审放行）。当前 HEAD 仍 dacc8a1，改动在工作树。

## 6. 安全自查
- **正向 enforcement 证据**：未止于"alice allow"，经同主体负向对照证明 allow 由 g 继承链承载（无链即 deny），守"enforce 即时翻转"范式（T-005b-1）。
- **allow 可归因硬前提满足**：alice 无任何直挂 p 规则（admin 才持），allow 唯一来源是继承链。
- **非凑绿**：摸底坐实 stale 断言非 bug；负向对照确保将来 alice 误加直挂策略 / g 持久化在 gorm-adapter 回归时本测试会真红。
- 无密钥/IP/证书/.env 改动。

## 7. 需 daxing 真人验收
- **无浏览器可视面**（纯测试/鉴权正确性，无 UI、无 API、无 demo 行为变化）。
- 验收 = 审 PM 评审 + 本证据包，确认"修红不凑绿、继承经 MySQL 路径正向证明、CI 卫生闭合"。同 T-005b-3 加密往返"对外不可视、靠集成测试证"口径（正确性证明归测试，不强塞浏览器）。

## 8. 偏差与待办
- **act 维度处置（任务书 §2 要求显式交代）**：采取「保留」而非移除。原测试经 `p.act="*"` 覆盖 act 通配路径；重写后主断言用 `act=access` 精确匹配（镜像 `rbac_test.go:83`），**另加一条 act 通配子断言**（admin 持 `(sys:user:create, *)`，alice 经继承以 `anyact` 命中），原 act=* 覆盖未丢失。
- **CI 现状回报（任务书 §2 要求）**：全仓**无任何 CI 流水线配置**——无 `.github/workflows`、无 gitee/woodpecker/drone/gitlab 流水线文件；仅有 `deploy/docker-compose.dev.yml`（起依赖）与 config 样例。故 `rbac -tags=integration` **未接入任何自动执行**，目前靠人工本地跑。**待办：将 `-tags=integration`（连 MySQL/Redis）接入 CI 自动执行**（独立立项，本片不做）。
- **未做（任务书 §2 明列不含）**：`localhost:3306 → env 可覆盖`改造（独立待办，本片 :3306 在线不阻塞）。

## 9. 下一步建议
- PM 评审本证据包 → 放行后双推（Gitee 主 + GitHub 镜像，`git ls-remote` 三方 HEAD 一致校验）→ 由 PM 更新 PROJECT_STATUS（移除 T-009b 账本里"casbin pre-existing red 单独立项"待办，标 T-003d-fix 清零）。
- 建议把"CI 接入 `-tags=integration`"列入需求池/债篮（与既有 `localhost:3306 env 可覆盖`待办同批），让此类真后端红能在 CI 自动拦截而非靠人工跑。
