# 完成报告：T-001 安全地基

## 1. 完成状态
**已完成** — 全部模块实现 + 单测全绿 + Redis/MySQL 集成测试就绪 + 安全自查通过。

## 2. 改动文件清单

| 路径 | 说明 | 新增/修改 |
|---|---|---|
| `server/errcode/errcode.go` | offset 常量 + HTTP 映射 + Registry（segmentBase 注入） | 新增 |
| `server/crypto/aes.go` | AES-256-CBC + PKCS7 加解密纯函数 | 新增 |
| `server/crypto/hmac.go` | HMAC-SHA256 签名/验签（hmac.Equal 恒定时间） | 新增 |
| `server/crypto/envelope.go` | 信封组装：签名串 + body 编解码 + 签名 | 新增 |
| `server/crypto/store.go` | ReplayStore 接口定义 | 新增 |
| `server/crypto/memstore.go` | ReplayStore 内存假实现（单测用） | 新增 |
| `server/crypto/redisstore.go` | RedisReplayStore — SET NX EX 原子防重放 | 新增 |
| `server/crypto/middleware.go` | Gin 中间件：请求解密 + 响应加密 + 5 步校验 | 新增 |
| `server/crypto/crypto_test.go` | 单测：往返/篡改/过期/重放/缺头/配置校验 | 新增 |
| `server/crypto/vectors_test.go` | 加载 golden JSON 断言（NIST/RFC KAT + 信封） | 新增 |
| `server/crypto/redis_integration_test.go` | `//go:build integration` Valkey SETNX/TTL 实测 | 新增 |
| `server/auth/claims.go` | Claims + TokenPair 结构体 | 新增 |
| `server/auth/service.go` | TokenService 接口 | 新增 |
| `server/auth/store.go` | BlacklistStore 接口定义 | 新增 |
| `server/auth/memstore.go` | BlacklistStore 内存假实现（单测用） | 新增 |
| `server/auth/redisstore.go` | RedisBlacklistStore — jti 黑名单 Redis 实现 | 新增 |
| `server/auth/jwt.go` | HS256 实现：IssuePair/Parse/Verify/Refresh/Revoke | 新增 |
| `server/auth/jwt_test.go` | 单测：签发/过期/错tt/Revoke/Refresh轮换 | 新增 |
| `server/auth/redis_integration_test.go` | `//go:build integration` 黑名单+完整 JWT 流程实测 | 新增 |
| `server/rbac/enforcer.go` | NewEnforcer + TurnOffAutoMigrate | 新增 |
| `server/rbac/middleware.go` | Authz gin.HandlerFunc + subjectFn 注入 | 新增 |
| `server/rbac/rbac_test.go` | 单测：策略/通配/角色继承/keyMatch2/403 | 新增 |
| `server/rbac/mysql_integration_test.go` | `//go:build integration` spec SQL 建表+Enforce 持久化 | 新增 |
| `server/config/config.go` | viper YAML→struct + fail-fast 校验 | 新增 |
| `server/config/config.example.yaml` | 配置示例（无真实密钥） | 新增 |
| `server/cmd/genvectors/main.go` | 向量生成工具（一次性，冻结 golden） | 新增 |
| `server/spec/rbac/model.conf` | Casbin RBAC 模型（keyMatch2 + act 通配） | 修改 |
| `server/spec/migrations/T001_casbin_rule.sql` | DDL + `{{TABLE_PREFIX}}` 占位 + 替换路径说明 | 新增 |
| `server/spec/openapi/openapi.yaml` | OpenAPI 3.0 v0.1.0（安全头/信封/错误码 schema） | 新增 |
| `server/spec/crypto-vectors/aes_cbc.json` | AES 向量（5 条：NIST SP 800-38A KAT ×2 + 自定义 ×3） | 新增 |
| `server/spec/crypto-vectors/hmac_sha256.json` | HMAC 向量（6 条：RFC 4231 KAT ×3 + 自定义 ×3） | 新增 |
| `server/spec/crypto-vectors/envelope.json` | 端到端信封向量（4 条：中文/空体/大小写） | 新增 |
| `deploy/docker-compose.dev.yml` | MySQL 8.0 + Valkey 8.x | 新增 |
| `.project-management/PROJECT_STATUS.md` | 项目状态文档 | 新增 |
| `.env.example` | 改为指引文件，主配置统一到 config.example.yaml | 修改 |
| `.gitignore` | 补充 `*.local.yaml` / `server/config/config.yaml` | 修改 |
| `server/go.mod` / `server/go.sum` | 依赖管理 | 修改 |

## 3. 接口实现情况

| 项 | 位置 | 状态 | 备注 |
|---|---|---|---|
| C 端加密信封 | server/crypto | ✅ | AES-256-CBC+PKCS7, HMAC-SHA256, 5步校验顺序正确 |
| TokenService | server/auth | ✅ | HS256, UUIDv7 jti, 双密钥, refresh 轮换 |
| Casbin enforcer + Authz | server/rbac | ✅ | TurnOffAutoMigrate, subjectFn 注入 |
| ReplayStore Redis | server/crypto/redisstore.go | ✅ | SET NX EX 原子操作 |
| BlacklistStore Redis | server/auth/redisstore.go | ✅ | SET + GET + TTL |
| crypto-vectors | server/spec/crypto-vectors | ✅ | NIST SP 800-38A + RFC 4231 KAT + 自定义 |
| casbin_rule SQL | server/spec/migrations | ✅ | {{TABLE_PREFIX}} 占位 + 替换路径三选一 |
| model.conf | server/spec/rbac | ✅ | RBAC + keyMatch2 + act 通配 |
| openapi v0.1.0 | server/spec/openapi | ✅ | redocly lint valid, 7 warnings(均预期) |
| docker-compose.dev | deploy/ | ✅ | MySQL 8 + Valkey 8 |

## 4. 自验结果

- **构建/静态检查**：`go build ./...` + `go vet ./...` 全部通过，无警告
- **单测（docker-free）**：`go test ./...` 全绿
  - crypto: 17 tests（往返/篡改/过期/重放/缺头 + NIST/RFC KAT + 信封向量）
  - auth: 8 tests（签发/解析/过期/错tt/Revoke/Refresh轮换/配置校验）
  - rbac: 6 tests（策略/通配/角色继承/keyMatch2/中间件403/不泄漏）
- **集成测试**：`go test -tags=integration ./...` 需先 `docker compose -f deploy/docker-compose.dev.yml up -d`
  - crypto: Redis SETNX 语义 / TTL 过期 / key 前缀格式
  - auth: 黑名单 Add+Check / TTL 过期 / 完整 JWT 签发→Revoke→Refresh 流程
  - rbac: spec SQL 建表（无 AutoMigrate）/ NewEnforcer / AddPolicy 持久化 / 角色继承 / 索引验证
- **fail-fast 配置校验**：aes_key≠32B / 空 secret / segment_base≤0 均报错
- **OpenAPI 校验**：`redocly lint` → valid, 7 warnings（info-license/localhost/unused-components 均为 paths 空占位所致）

**测试分层**：

| 层级 | 命令 | 依赖 | 覆盖 |
|---|---|---|---|
| 单测 | `go test ./...` | 无（MemoryReplayStore / MemoryBlacklistStore） | 功能正确性、安全负例、向量断言 |
| 集成 | `go test -tags=integration ./...` | docker compose (Valkey + MySQL) | Redis SETNX/TTL、MySQL spec SQL 建表、Enforce 持久化 |

rbac 单测使用 Casbin file adapter + 内存策略，不碰数据库。MySQL spec SQL 建表的完整路径由 `rbac/mysql_integration_test.go` 覆盖。

## 5. git 提交记录

**卡点**：项目目录尚未 `git init`，无 remote 配置。需 daxing 提供 Gitee/GitHub 仓库 URL 后执行首次提交和双推。

## 6. 安全自查

- [x] Encrypt-then-MAC + 先验签名后解密（middleware.go: 缺头→时间→签名→nonce→解密）
- [x] hmac.Equal 恒定时间比较（crypto/hmac.go:27）
- [x] nonce TTL(600s) > 2×window(300s)、先验时间签名再落 nonce
- [x] 密钥全部配置注入、启动 fail-fast、无内置默认密钥
- [x] JWT 校验 exp/nbf/iss/tt + 黑名单 + refresh 轮换拉黑旧 jti
- [x] Enforce 失败 403、不泄漏策略细节
- [x] grep 无硬编码 kp_/okp/BenxinKP/写死密钥（注释中作为示例说明除外）
- [x] 头注释五项到秒齐全（所有 .go 文件已验证）
- [x] crypto/auth/rbac 均不 import config 包（DI 模式）
- [x] ReplayStore/BlacklistStore 接口化 + Redis 真实现 + 内存假实现

## 7. 需 daxing 真人验收

- [ ] 评审 `spec/crypto-vectors/*.json`（尤其 `expected_signing_string`）作为 PHP parity 基准
- [ ] 评审 `spec/rbac/model.conf` 与 `T001_casbin_rule.sql`
- [ ] 本机 `docker compose -f deploy/docker-compose.dev.yml up -d` 后跑 `go test -tags=integration ./...` 全绿
- [ ] 抽查文件头注释五项到秒
- [ ] 提供 Gitee/GitHub 仓库 URL → 执行 git init + 首次提交 + 双推

## 8. 偏差与待办

- `go.mod` 中 Go 版本使用 `go 1.24.4`（当前最新稳定版），CLAUDE.md 标注 Go 1.26（尚未发布）
- `server/.env.example` 原为空文件，已改为指引文件统一到 config.example.yaml

## 9. 下一步建议

- T-002 认证授权：登录直接调 `auth.TokenService.IssuePair`；密码 Argon2id（新增 `auth/password.go`）
- T-003 RBAC CRUD：复用 `rbac.NewEnforcer` 实例
- T-004 系统管理：`response` 完整包络模块接管 `errcode` 注册表
