# PROJECT_STATUS — BenxinAdminPro 底座（进度账本 · 跨对话接续权威来源）

> 每个新对话开始先读本文件；每完成一个切片后更新本文件。本文件优先于"记忆"。

## 项目元信息
- 项目：BenxinAdminPro（本心通用管理后台）。前后端同仓：`server/`（Go）+ `admin/`（Element Plus）
- 定位：开箱即用的完整后台系统 + 可复用底座（脚手架 + 模块库）
- 架构文档：BenxinAdminPro-架构与规划文档 v1.1
- 仓库：Gitee（主） + GitHub（镜像），独立于 BenxinKP
- 本地：/Users/daxing/projects/BenxinAdminPro（与 BenxinKP 平级，多根工作区聚合）
- 最后更新：2026-06-07（T-001 收尾项全部补齐）

## 核心原则（铁律）
- 业务中立：不进任何业务概念。
- 参数化：禁硬编码 kp_/okp:/BenxinKP；表前缀、Redis 前缀、错误码段、@project 全由配置注入。
- 纯 Go + 语言中立 spec（供 PHP 等实现 parity）。
- SemVer 库复用，绝不为单应用改底座源码。

## 协作约定
- Cursor / Claude Code：编码 + API/单测/契约测试 + git 提交（双仓库）。
- daxing：用 examples/demo 真人验收 + 关键决策。
- 节奏：纯 Go 垂直切片；每个能力须在 demo 跑通。
- 本机：docker-compose 仅起依赖，应用原生跑；生产不用 Docker。

## 代码头注释（五项，到秒）
项目名中英文 / 本文件功能 / 作者 / 邮箱 / 生成日期时间。模板见 .cursor/rules/benxinadminpro.mdc §6。

## 环境就绪状态
- ✅ 本地目录骨架已建：server/（含 spec、examples/demo、各模块目录）、admin/、.cursor/rules、.project-management
- ✅ server/ 已 go mod init（github.com/benxin_dev/benxinadminpro-server）
- ✅ admin/ 已 Vite + Vue3 + TS 脚手架
- ✅ 根文件 + .gitignore 就位（T-001 补充 *.local.yaml / config.yaml 忽略）
- ✅ 多根工作区 projects/benxin.code-workspace 已建
- ✅ deploy/docker-compose.dev.yml 就位（MySQL 8.0 + Valkey 8.x）
- ⏳ 待办：放置 .cursor/rules/benxinadminpro.mdc；确认 git init + 首次提交 + 双仓推送（T-001 完成报告未含 git 记录，待执行端确认）

## 切片进度

### 已完成
| 任务编号 | 切片 | 备注 |
|---|---|---|
| — | — | 尚无完全收尾切片 |

### 进行中 / 待收尾
| 任务编号 | 切片 | 状态 | 待收尾项 |
|---|---|---|---|
| T-001 | 安全地基（crypto + JWT + Casbin + 跨语言向量） | 🟢 代码完成，待 git init + 双推 | ①~③ 已全部补齐（Redis 适配器+集成测试、openapi lint、env/config 对齐、SQL 替换说明）；仅剩 git init + 首次提交 + 双仓推送，需 daxing 提供 remote URL |

**T-001 已验收通过的部分（PM 评审）**
- 四项关键设计调整全部落实：errcode segmentBase 注入（非 const）；crypto-vectors 由独立 `cmd/genvectors` 冻结为 golden、test 仅加载断言、含 RFC 4231 + NIST SP 800-38A 标准 KAT；crypto/auth/rbac 不 import config（DI）；ReplayStore/BlacklistStore 接口化、单测内存假实现、`go test ./...` 不依赖 docker。
- 安全自查项齐全：Encrypt-then-MAC + 先验签名后解密、hmac.Equal、nonce TTL>2×window、密钥配置注入 + fail-fast、JWT 校验 + refresh 轮换拉黑旧 jti、Enforce 403、TurnOffAutoMigrate、头注释五项。
- 构建/静态检查/单测全绿（crypto 17 / auth 8 / rbac 6）。

**T-001 待 daxing 真人验收（补齐收尾项后一并做）**
- 评审 spec/crypto-vectors/*.json（尤其 expected_signing_string）作为 PHP parity 基准。
- 评审 spec/rbac/model.conf 与 T001_casbin_rule.sql。
- 本机 `docker compose -f deploy/docker-compose.dev.yml up -d` 后跑 `go test -tags=integration ./...` 全绿（含真 Redis/MySQL 路径）。
- 抽查文件头注释五项到秒、@date 为真实生成时间。
- 双仓同步与 .gitignore 检查。

### 下一步（计划）
1. **T-001 最后一步**：daxing 提供 Gitee/GitHub remote URL → git init + 首次提交 + 双推 + 集成测试真人验收。
2. T-002 认证授权（登录/刷新/验证码/锁定）。登录直接调 auth.TokenService.IssuePair；密码校验 Argon2id（新增 auth/password.go 或独立包）。
3. T-003 RBAC（用户/角色/权限/菜单/部门/岗位）。复用 rbac.NewEnforcer 实例。
4. T-004 系统管理（字典/参数/日志/文件）。response 完整包络模块在此接管 errcode 注册表。
5. T-005 配置中心（驱动化/加密/热加载）。替换 T-001 的静态 viper 加载。
6. admin 前端并行（布局/动态路由/权限/x-table）。
7. examples/demo 跑通（登录+RBAC+配置+日志）。

### 阶段二（底座可用后）
- BenxinKP 引入 BenxinAdminPro，只写业务；backend-php 照 spec 实现 parity。

## 待决策（F 系列，见底座文档 §14）
| # | 决策项 | 状态 |
|---|---|---|
| F1 | 命名 BenxinAdminPro/server/admin | ✅ 已决 |
| F2 | 单独 Claude Project 管底座 | 建议是 |
| F3 | admin 复用形态 | npm 包 + starter |
| F4 | 通用驱动归属 | 通用实现放底座 |
| F5 | 本地位置 | ✅ 已决 |

## 备注
- 宪法级：安全第一、仅开源素材、配置驱动化、参数化复用、统一代码头注释（中英文+到秒）。
- T-001 评审定档：单测用内存假实现（CI 快），真后端正确性由 `-tags=integration` 集成测试兜底；二者分层，不混用。
