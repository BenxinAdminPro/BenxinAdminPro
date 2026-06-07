# CLAUDE.md — BenxinAdminPro（本心通用管理后台）

> 本文件是 Claude Code 在本项目的工作宪法，每次会话自动加载。BenxinAdminPro 是开箱即用的完整后台系统，同时是纯 Go 的可复用底座（脚手架 + 模块库）。

## 工作流（每次会话遵守）
1. **动手前先读 `.project-management/PROJECT_STATUS.md`**，了解当前进度、已完成切片、下一步任务，再开始。
2. 按"垂直切片"推进：一次只做一个切片，不要横向铺开。
3. **改完代码必须自验**：`cd server && go build ./... && go vet ./... && go test ./...`，全绿才算完成。
4. **危险操作先征求确认**：删除文件、改 `.gitignore`/`.env`、`git push`、批量重构前，先说明意图等我确认。
5. 完成后输出**可复制 markdown 完成报告**（见末尾模板），并更新 `PROJECT_STATUS.md`。
6. **报告落文件规则（策略 A）**：完成报告**同时**写入 `.project-management/reports/T-{编号}-report.md`。每个切片**一个文件**，文件名带编号；重新生成时**覆盖该文件（禁止追加）**；**禁止把多个切片的报告写进同一文件**。该文件始终只代表本切片当前最新报告。

## 这是底座，不是业务应用
- 只放与具体业务无关、每个后台应用都用得上的能力：认证、RBAC、系统管理、配置中心、加密中间件、统一响应、限流、监控、代码生成、消息中心、驱动接口。
- 判断标准：动手前自问"下一个完全不同的应用还用得上吗？"——用不上不要写。
- **禁止任何业务概念**：内容/文章/音视频/直播/会员/积分/支付/订单/优惠券/评论/客服等一律不进。

## 参数化铁律（最重要）
**禁止硬编码任何应用专属常量**：
- 表前缀用配置项（如 `cfg.TablePrefix`），禁止写死 `kp_`。
- Redis key 前缀用配置项（如 `cfg.RedisNamespace`），禁止写死 `okp:`。
- 错误码：底座占约定段（如 1000–1999），业务段留消费方。
- 加密主密钥经环境变量注入，禁止内置默认密钥。

## 复用机制
- 通过带 SemVer 的 Go module（`github.com/benxin_dev/benxinadminpro-server`）复用，非复制粘贴。
- 应用差异通过接口/扩展点/配置注入，绝不为某应用改底座源码。
- `v1.x` 内向后兼容；破坏性变更升 `v2`。

## 纯 Go + 语言中立 spec
- 底座只写 Go，不写 PHP。
- 同时维护 `server/spec/`（OpenAPI / SQL DDL / casbin model.conf / 加密向量），供其它语言（如 BenxinKP 的 PHP 后端）实现 parity。

## 技术栈
Go 1.26 + Gin；GORM + gorm/gen；Casbin；go-redis；golang-jwt；viper；zap+lumberjack；validator；swaggo/swag；gopsutil；ulule/limiter。

## 安全（宪法级）
- 安全第一，服务端唯一权威，永不信任前端。
- 加密：C 端传输 AES-256-CBC + PKCS7 + 随机 IV 前缀 + HMAC-SHA256 + timestamp/nonce 防重放（窗口 300s，nonce TTL 600s）；配置 at-rest AES-256-GCM（密文 `base64(nonce+ciphertext+tag)`）。向量落 `server/spec/crypto-vectors/`。
- 密码 Argon2id；JWT 访问+刷新+jti 黑名单（Redis）。
- DB：金额存分、统一时间字段、软删除、对外 ID 用 Hashid(加盐)/UUIDv7；DDL 走 `server/spec/migrations/` 纯 SQL，禁 AutoMigrate。
- SQL 参数化、输出转义、配置 URL 防 SSRF；安全响应头固定值。

## 代码头注释（每个新建文件必须带，五项）
①项目名称中英文 ②本文件功能/模块 ③作者 ④邮箱 ⑤生成日期时间（到秒）。
```
// +----------------------------------------------------------------------
// | @project   本心通用管理后台 / BenxinAdminPro
// | @mission   本文件功能或模块说明 — API 控制器附 HTTP 方法与路由
// | @author    仗键天涯(daxing)
// | @email     3442535897@qq.com
// | @date      2026-06-07 11:30:00
// +----------------------------------------------------------------------
```
`@date` 取生成当时 `YYYY-MM-DD HH:mm:ss`；改既有文件追加 `@updated YYYY-MM-DD HH:mm:ss`。

## 职责分工
- **你（Claude Code）负责**：编码 + API/单测/契约测试 + git 提交（双仓库 Gitee 主 + GitHub 镜像）。
- **daxing 负责**：用 `server/examples/demo` 真人验收 + 关键决策。
- 每个能力须能在 `examples/demo` 跑起来、可演示。

## git 规则
- 提交信息格式：`type(scope): 简述`（如 `feat(auth): JWT 签发与校验`）。
- **提交前必查**：禁止把密钥/IP/证书/.env 提交进任一仓库；确认 `.gitignore` 覆盖 `.env`/`*.key`/`*.pem`/`*.crt`/`vendor/`/`node_modules/`。
- **push 前先征求确认**；双仓库推送需两个 remote 都已配置。

## 本机环境
- docker-compose 仅起依赖（MySQL/Redis），server/admin/demo 原生跑；连 `localhost:3306 / localhost:6379`。
- 生产不容器化。

## 完成报告模板（每个任务结束输出）
```markdown
# 完成报告：T-{编号} {切片名}
## 1. 完成状态
## 2. 改动文件清单（路径 + 说明 + 新增/修改）
## 3. 接口实现情况
## 4. 自验结果（go build / vet / test，契约/安全用例）
## 5. git 提交记录（commit message / 是否双仓同步）
## 6. 安全自查
## 7. 需 daxing 真人验收（demo 验证项）
## 8. 偏差与待办
## 9. 下一步建议
```
