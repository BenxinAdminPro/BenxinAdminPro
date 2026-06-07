# PROJECT_STATUS — BenxinAdminPro 底座（进度账本 · 跨对话接续权威来源）

> 每个新对话开始先读本文件；每完成一个切片后更新本文件。本文件优先于"记忆"。

## 项目元信息
- 项目：BenxinAdminPro（本心通用管理后台）。前后端同仓：`server/`（Go）+ `admin/`（Element Plus）
- 定位：开箱即用的完整后台系统 + 可复用底座（脚手架 + 模块库）
- 架构文档：BenxinAdminPro-架构与规划文档 v1.1
- 仓库：Gitee 主仓 https://gitee.com/benxin-admin-pro/benxin-admin-pro.git ；GitHub 镜像 https://github.com/BenxinAdminPro/BenxinAdminPro.git（独立于 BenxinKP）
- 本地：/Users/daxing/projects/BenxinAdminPro（与 BenxinKP 平级，多根工作区聚合）
- 最后更新：2026-06-08（T-004a 完成）

## 核心原则（铁律）
- 业务中立：不进任何业务概念。
- 参数化：禁硬编码 kp_/okp:/BenxinKP；表前缀、Redis 前缀、错误码段、@project 全由配置注入。
- 纯 Go + 语言中立 spec（供 PHP 等实现 parity）。
- SemVer 库复用，绝不为单应用改底座源码。

## 协作约定
- Cursor / Claude Code：编码 + API/单测/契约测试 + git 提交（双仓库）。
- daxing：用 examples/demo 真人验收 + 关键决策 + git remote/凭证 + 双推确认。
- 节奏：纯 Go 垂直切片；每个能力须在 demo 跑通。
- 本机：docker-compose 仅起依赖，应用原生跑；生产不用 Docker。
- PM 评审硬规矩：真后端正确性由 `-tags=integration` 集成测试兜底，不靠真人代替；单测用内存假实现保 CI 快。

## 代码头注释（五项，到秒）
项目名中英文 / 本文件功能 / 作者 / 邮箱 / 生成日期时间。改既有文件追加 @updated。模板见 .cursor/rules/benxinadminpro.mdc §6。

## 环境就绪状态
- ✅ 本地目录骨架、go mod init、admin Vite+Vue3+TS、根文件 + .gitignore（含 *.local.yaml / config.yaml / .claude/）、多根工作区、deploy/docker-compose.dev.yml（MySQL 8.0 + Valkey 8.x）
- ✅ git init + 双仓推送链路稳定（Gitee origin/main + GitHub github/main）
- ⏳ 待办：放置 .cursor/rules/benxinadminpro.mdc（如尚未提交）

## 切片进度

### 已完成
| 任务编号 | 切片 | 备注 |
|---|---|---|
| T-001 | 安全地基（crypto + JWT + Casbin + 跨语言向量） | ✅ 实现 + 集成测试 + 双推完成。 |
| T-002 | 认证授权（登录/刷新/登出 + 图形验证码 + 失败锁定 + Argon2id） | ✅ 详见下方收尾记录 |
| T-003a | 组织架构（用户+部门+岗位 CRUD + GormUserProvider） | ✅ 详见 `reports/T-003a-report.md` |
| T-003b | RBAC 核心（角色/菜单/权限 + Casbin 联动 + Hashid） | ✅ 详见 `reports/T-003b-report.md` |
| T-003c | 数据权限（角色 data_scope 三档 + DataScope 解析器 + ApplyScope） | ✅ RBAC 收官。详见 `reports/T-003c-report.md` |
| T-004a | 系统管理（response Registry + 字典/参数 + 操作日志/登录日志） | ✅ 详见 `reports/T-004a-report.md` |

**T-001 收尾记录**
- 三核心包：crypto（AES-256-CBC+PKCS7、HMAC-SHA256、5 步防重放）、auth（HS256 TokenService、UUIDv7 jti、双密钥、refresh 轮换拉黑旧 jti）、rbac（NewEnforcer + TurnOffAutoMigrate + Authz 骨架）。
- errcode segmentBase 注入（非 const）；crypto-vectors 独立 cmd/genvectors 冻结 golden、含 RFC 4231 + NIST SP 800-38A KAT；crypto/auth/rbac 不 import config（DI）。
- ReplayStore/BlacklistStore 接口化；go-redis 实现（RedisReplayStore=SET NX EX；RedisBlacklistStore=jti TTL=剩余寿命）。集成测试 //go:build integration 对真 Valkey/MySQL 实测。
- spec：openapi v0.1.0（redocly 0 error）；model.conf（RBAC+keyMatch2+act 通配）；T001_casbin_rule.sql（{{TABLE_PREFIX}} 占位 + 替换路径三选一）。

**T-002 收尾记录（PM 评审定档）**
- Argon2id（password.go）：PHC 串参数随存（$argon2id$v=19$m,t,p$salt$hash），subtle.ConstantTimeCompare 恒定时间；OWASP 基线 m=19456KiB/t=2/p=1，配置注入；DummyVerify 防时序枚举。
- 用户表隔离：UserProvider 接口 + 业务中立 AuthUser（id/username/password_hash/status）+ MemUserProvider（demo/测试用）；真实现留 T-003 注入；StatusChecker 可注入避免状态语义写死底座。
- 图形验证码（captcha.go）：**纯 Go 标准库（image/png）+ 像素块自绘字符，零第三方库、零字体文件**（合规最彻底，不沾仅开源素材红线）；answer 存 Redis，CaptchaRedisStore 用 GetDel 原子一次性消费。
- 失败锁定（lockout.go）：三级（失败计数→达 captcha_threshold 要验证码→达 lock_threshold 锁定）；按 username。**关键修复：LockoutRedisStore.IncrFail 改为仅 count==1 设 Expire（固定窗口，不随每次失败续命）——此 bug 由补加的 IncrFixedWindow 集成测试逼出**。
- 编排（service_auth.go）登录顺序：锁定检查→验证码→UserProvider→密码→状态→IssuePair；防枚举：用户不存在与密码错统一 ERR_BAD_CREDENTIALS + dummy 校验拉平时序。
- 令牌复用 T-001：登录调 TokenService.IssuePair；refresh 轮换拉黑、logout 拉黑、Verify 查黑名单。
- handler.go：4 路由（/auth/captcha /login /refresh /logout）+ 统一包络 {code,message,data} + RegisterRoutes；编排层/handler 不 import config、走 Store 接口。
- errcode 增量 offset +20~+24（既有文件改动，追加 @updated）。openapi 升 v0.2.0（redocly 0 error）。
- 测试：单测 docker-free 全绿（Argon2id 5 + 编排 10 + handler 8 + lockout 状态机）；TestLoginLockout 全链路补回不 skip（MemCaptchaStore.Peek 取答案打通验证码门槛→锁定）；集成测试 auth_integration_test.go 对真 Valkey 验 GetDel 一次性/TTL、lockout 固定窗口/锁定 TTL。
- git：双 commit（2d6f6d0 主体 + 8b8e7da 补集成测试&修固定窗口），Gitee+GitHub 同步至 8b8e7da。

**待 daxing 真人验收（用到时补，不阻塞）**
- T-001：docker compose up 后 go test -tags=integration ./... 全绿；评审 crypto-vectors / model.conf / casbin_rule.sql。
- T-002：demo 装配 MemUserProvider 后 curl 跑通 captcha→login→受保护路由→refresh→logout（旧 access 被拒）；触发验证码与锁定阈值确认；评审 UserProvider/AuthUser 边界是否够 T-003 注入真实现。
- 抽查日志无密码/验证码答案/token 明文；文件头注释五项到秒、@date 真实。

### 进行中 / 待收尾
| 任务编号 | 切片 | 状态 | 待收尾项 |
|---|---|---|---|
| — | — | — | 无 |

### 下一步（计划）
1. **T-003 RBAC**（用户/角色/权限/菜单/部门/岗位 + 数据权限）。注入真实 UserProvider（GORM DB 实现，替换 MemUserProvider）；复用 rbac.NewEnforcer 实例；用户表 DDL 走 spec/migrations。
2. T-004 系统管理（字典/参数/操作日志/登录日志/文件）。response 完整包络模块在此接管 errcode 注册表；登录成功/失败日志落库。
3. T-005 配置中心（驱动化/加密/热加载）。替换 T-001/T-002 静态 viper 加载；提供 SQL 迁移执行器（替换 {{TABLE_PREFIX}}）。
4. admin 前端并行（布局/动态路由/权限/x-table）。
5. examples/demo 跑通（登录+RBAC+配置+日志）。

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
- git 经验：镜像仓建仓勿勾任何初始化文件，否则首推需 force；token 走交互式/钥匙串，勿写进 remote url；GitHub 偶发 SSL_ERROR_SYSCALL 网络抖动，重试即可。
