# PROJECT_STATUS — BenxinAdminPro 底座（进度账本 · 跨对话接续权威来源）

> 每个新对话开始先读本文件；每完成一个切片后更新本文件。本文件优先于"记忆"。

## 项目元信息
- 项目：BenxinAdminPro（本心通用管理后台）。前后端同仓：`server/`（Go）+ `admin/`（Element Plus）
- 定位：开箱即用的完整后台系统 + 可复用底座（脚手架 + 模块库）
- 架构文档：BenxinAdminPro-架构与规划文档 v1.1
- 仓库：Gitee 主仓 https://gitee.com/benxin-admin-pro/benxin-admin-pro.git ；GitHub 镜像 https://github.com/BenxinAdminPro/BenxinAdminPro.git（独立于 BenxinKP）
- 本地：/Users/daxing/projects/BenxinAdminPro（与 BenxinKP 平级，多根工作区聚合）
- 最后更新：2026-06-07（T-002 完成）

## 核心原则（铁律）
- 业务中立：不进任何业务概念。
- 参数化：禁硬编码 kp_/okp:/BenxinKP；表前缀、Redis 前缀、错误码段、@project 全由配置注入。
- 纯 Go + 语言中立 spec（供 PHP 等实现 parity）。
- SemVer 库复用，绝不为单应用改底座源码。

## 协作约定
- Cursor / Claude Code：编码 + API/单测/契约测试 + git 提交（双仓库）。
- daxing：用 examples/demo 真人验收 + 关键决策 + git remote/凭证。
- 节奏：纯 Go 垂直切片；每个能力须在 demo 跑通。
- 本机：docker-compose 仅起依赖，应用原生跑；生产不用 Docker。

## 代码头注释（五项，到秒）
项目名中英文 / 本文件功能 / 作者 / 邮箱 / 生成日期时间。模板见 .cursor/rules/benxinadminpro.mdc §6。

## 环境就绪状态
- ✅ 本地目录骨架已建：server/（含 spec、examples/demo、各模块目录）、admin/、.cursor/rules、.project-management
- ✅ server/ 已 go mod init（github.com/benxin_dev/benxinadminpro-server）
- ✅ admin/ 已 Vite + Vue3 + TS 脚手架
- ✅ 根文件 + .gitignore 就位（含 *.local.yaml / config.yaml / .claude/ 忽略）
- ✅ 多根工作区 projects/benxin.code-workspace 已建
- ✅ deploy/docker-compose.dev.yml 就位（MySQL 8.0 + Valkey 8.x）
- ✅ git init + 首次提交 + 双仓推送完成（Gitee origin/main + GitHub github/main 同步；GitHub 首推 force 覆盖了建仓自动初始 commit）
- ⏳ 待办：放置 .cursor/rules/benxinadminpro.mdc（如尚未提交）

## 切片进度

### 已完成
| 任务编号 | 切片 | 备注 |
|---|---|---|
| T-001 | 安全地基（crypto + JWT + Casbin + 跨语言向量） | ✅ 详见下方收尾记录 |
| T-002 | 认证授权（登录/刷新/登出 + 验证码 + 锁定 + Argon2id） | ✅ 完成报告见 `reports/T-002-report.md` |

**T-001 收尾记录（PM 评审定档）**
- 三核心包：crypto（AES-256-CBC+PKCS7、HMAC-SHA256、5 步校验防重放）、auth（HS256 TokenService、UUIDv7 jti、双密钥、refresh 轮换拉黑旧 jti）、rbac（NewEnforcer + TurnOffAutoMigrate + Authz 中间件骨架）。
- 关键设计：errcode segmentBase 注入（非 const）；crypto-vectors 由独立 `cmd/genvectors` 冻结为 golden、test 仅加载断言，含 RFC 4231 + NIST SP 800-38A 标准 KAT；crypto/auth/rbac 不 import config（DI）。
- 存储分层：ReplayStore/BlacklistStore 接口化。单测用内存假实现，`go test ./...` 不依赖 docker。新增 go-redis 具体实现（RedisReplayStore = SET NX EX；RedisBlacklistStore = jti TTL=剩余寿命）。
- 集成测试 `//go:build integration`：对 docker Valkey 实测 SETNX/TTL/前缀；对 MySQL 用 spec SQL 建表（strings.ReplaceAll 替换 {{TABLE_PREFIX}} → db.Exec）+ NewEnforcer + 持久化验证。命令分层：`go test ./...`（docker-free）/ `go test -tags=integration ./...`（需 docker compose up）。
- spec：openapi.yaml 通过 redocly lint（0 error，warning 均为空占位/开发期合理项）；model.conf（RBAC + keyMatch2 + act 通配）；T001_casbin_rule.sql（{{TABLE_PREFIX}} 占位 + 替换路径三选一说明）。
- 配置：config.example.yaml 为主（viper YAML）；.env.example 改为指引文件（BENXIN_ 前缀环境变量可覆盖 YAML 键）。
- 构建/静态检查/单测全绿（crypto 17 / auth 8 / rbac 6）。安全自查项齐全（Encrypt-then-MAC、hmac.Equal、nonce TTL>2×window、密钥注入 fail-fast、Enforce 403、头注释五项、grep 无硬编码专属值）。
- git：`.claude/` 已忽略不提交；首次提交人眼复核确认无密钥/敏感文件混入；Gitee + GitHub 双推同步。

**T-001 可选 daxing 真人验收（用到时补，不阻塞）**
- 本机 `docker compose -f deploy/docker-compose.dev.yml up -d` 后跑 `go test -tags=integration ./...` 全绿（真 Redis/MySQL 路径）。
- 评审 spec/crypto-vectors/*.json（尤其 expected_signing_string）作为 PHP parity 基准。
- 评审 spec/rbac/model.conf 与 T001_casbin_rule.sql。
- 抽查文件头注释五项到秒、@date 为真实生成时间。

### 进行中 / 待收尾
| 任务编号 | 切片 | 状态 | 待收尾项 |
|---|---|---|---|
| — | — | — | 无 |

### 下一步（计划）
1. **T-002 可选 daxing 验收**：demo 装配 MemUserProvider → curl 跑通 captcha→login→refresh→logout + 阈值锁定。
2. T-003 RBAC（用户/角色/权限/菜单/部门/岗位）。复用 rbac.NewEnforcer 实例。
3. T-004 系统管理（字典/参数/日志/文件）。response 完整包络模块在此接管 errcode 注册表。
4. T-005 配置中心（驱动化/加密/热加载）。替换 T-001 的静态 viper 加载；提供 SQL 迁移执行器（替换 {{TABLE_PREFIX}}）。
5. admin 前端并行（布局/动态路由/权限/x-table）。
6. examples/demo 跑通（登录+RBAC+配置+日志）。

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
- git 经验：镜像仓（GitHub）建仓时勿勾任何初始化文件（README/.gitignore/License），否则首推需 force 覆盖；token 走交互式输入或钥匙串，勿写进 remote url。
