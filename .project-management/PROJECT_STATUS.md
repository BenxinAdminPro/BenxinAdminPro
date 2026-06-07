# PROJECT_STATUS — BenxinAdminPro 底座（进度账本 · 跨对话接续权威来源）

> 每个新对话开始先读本文件；每完成一个切片后更新本文件。本文件优先于"记忆"。

## 项目元信息
- 项目：BenxinAdminPro（本心通用管理后台）。前后端同仓：`server/`（Go）+ `admin/`（Element Plus）
- 定位：开箱即用的完整后台系统 + 可复用底座（脚手架 + 模块库）
- 架构文档：BenxinAdminPro-架构与规划文档 v1.1
- 仓库：Gitee 主仓 https://gitee.com/benxin-admin-pro/benxin-admin-pro.git ；GitHub 镜像 https://github.com/BenxinAdminPro/BenxinAdminPro.git（独立于 BenxinKP）
- 本地：/Users/daxing/projects/BenxinAdminPro（与 BenxinKP 平级，多根工作区聚合）
- 最后更新：2026-06-08（T-006 阶段一收官）

## 核心原则（铁律）
- 业务中立：不进任何业务概念。
- 参数化：禁硬编码 kp_/okp:/BenxinKP；表前缀、Redis 前缀、错误码段、@project 全由配置注入。
- 表前缀随实例走，禁任何包级/全局可变前缀（gorm NamingStrategy{TablePrefix} 绑 *gorm.DB）。
- 纯 Go + 语言中立 spec（供 PHP 等实现 parity）。
- SemVer 库复用，绝不为单应用改底座源码。

## 协作约定
- Cursor / Claude Code：编码 + API/单测/契约测试 + git 提交（双仓库）。
- daxing：用 examples/demo 真人验收 + 关键决策 + git remote/凭证 + 双推确认。
- 节奏：纯 Go 垂直切片；每个能力须在 demo 跑通。
- 本机：docker-compose 仅起依赖，应用原生跑；生产不用 Docker。
- PM 评审硬规矩：真后端正确性由 `-tags=integration` 集成测试兜底；单测用内存假实现/SQLite 保 CI 快。

## 代码头注释（五项，到秒）
项目名中英文 / 本文件功能 / 作者 / 邮箱 / 生成日期时间。改既有文件追加 @updated。模板见 .cursor/rules/benxinadminpro.mdc §6。

## 环境就绪状态
- ✅ 本地目录骨架、go mod init、admin Vite+Vue3+TS、根文件 + .gitignore（含 *.local.yaml / config.yaml / .claude/）、多根工作区、deploy/docker-compose.dev.yml（MySQL 8.0 + Valkey 8.x）
- ✅ git init + 双仓推送链路（Gitee origin/main + GitHub github/main）；注：近期推送 SSL_ERROR_SYSCALL 抖动转频，必要时给 remote 走代理或换 SSH
- ⏳ 待办：放置 .cursor/rules/benxinadminpro.mdc（如尚未提交）

## 里程碑
- 🏁 RBAC 整块完成（T-001 安全地基 → T-002 认证 → T-003 组织+RBAC a/b/c）。认证→功能权限→数据权限闭环。HEAD 当时 e167098。
- 当前 HEAD = 303bf1e（T-005 配置中心收尾）。
- 🏁 **Go 后端底座五大块全部完成（T-001 安全 / T-002 认证 / T-003 RBAC / T-004 系统管理 / T-005 配置中心）+ 渲染债清。阶段一后端收官，剩 admin 前端 + demo 跑通。**

## 切片进度

### 已完成
| 任务编号 | 切片 | 备注 |
|---|---|---|
| T-001 | 安全地基（crypto + JWT + Casbin + 跨语言向量） | ✅ |
| T-002 | 认证授权（登录/刷新/登出 + 验证码 + 失败锁定 + Argon2id） | ✅ 至 8b8e7da |
| T-003a | 组织架构（用户 + 部门 + 岗位 + GormUserProvider） | ✅ 至 3faf158 |
| T-003b | RBAC 核心（角色/权限/菜单 + Casbin 联动 + Hashid 收口） | ✅ 至 bd990a7 |
| T-003c | 数据权限（B 档三档 + 通用 DataScope 解析器） | ✅ 至 e167098（RBAC 收官） |
| T-004a | 系统管理（response Registry 接管 + 字典 + 参数 + 操作日志 + 登录日志） | ✅ 至 f1594d0。详见下方记录。 |
| T-004b | 文件管理 + 存储驱动（StorageDriver + LocalDriver + 鉴权下载 + 上传安全） | ✅ 至 448671f。T-004 整块完成。详见下方记录。 |
| T-004c | 渲染收敛（handler 统一 response.Render + errcode 降级纯常量） | ✅ 至 589f9e1。纯重构，对外零行为变化。 |
| T-005 | 配置中心（动态参数缓存 + 热加载 Pub/Sub + GCM 加密 + 迁移执行器） | ✅ 至 303bf1e |
| T-006 | examples/demo 装配（五大块全链路 + 种子数据 + 迁移建表）| ✅ 阶段一收官 |

**T-001~T-003 收尾记录**（详见此前版本，要点）
- T-001 crypto/auth/rbac 三核心 + crypto-vectors KAT + DI；集成测试真后端。
- T-002 Argon2id+DummyVerify 防枚举；验证码纯标准库自绘零字体；失败锁定固定窗口；编排防枚举；令牌复用 T-001。
- T-003a 组织三表 + GormUserProvider；去包级前缀改 NamingStrategy（两前缀不串表）；部门树防环；password_hash 不泄漏断言。
- T-003b 角色/菜单(menu_type)/Casbin 联动；model.conf 改 perm code 精确匹配（破坏性，同步 spec）；超管短路服务端可信；授权变更事务内回滚保一致；Hashid 入出参闭环（装配须注入 hasher）；RequirePerm 28 路由全覆盖。
- T-003c data_scope 三档 + 中立 DataScope 解析器 + ApplyScope(字段名调用方传入)；失败一律收紧 5 处 WHERE 1=0 有断言；范围来源服务端；叠加 enforce 不绕过。

**T-004a 收尾记录（PM 评审定档）**
- response 模块：Registry（Register/Lookup/冲突 fail-fast）+ Success/Fail/FailMsg/Page 渲染 + i18n 默认表。
- **错误码迁移现状=渐进中间态：两套渲染并存——老 handler 仍走 errcode.Error 渲染路径，response.Registry 为新建集中表经 errcode.AllSpecs() 桥接导入全部既有码，handler 尚未统一改走 response.Render。** 回归硬断言：migration_test.go 32 个既有码 (值/HTTP/i18nKey) 经 AllSpecs→Register→Lookup 逐项断言 32/32 匹配 + 总数防遗漏。既有码零变更，PHP parity 安全。
- 字典（sys_dict_type/data）+ 参数（sys_config）CRUD，**纯 DB，缓存/热加载留 T-005**；sys_config 敏感参数加密存储留 T-005（本片明文 + 注释标注）。
- 操作日志 sys_oper_log + 中间件自动采集（异步 goroutine、写失败 slog.Error 不阻塞、排除列表配置注入）+ 脱敏（password/captcha_code/token/secret→***，有断言）。
- 登录日志 sys_login_log：LoginLogger 接口定义在 auth 包（auth 不依赖 DB，grep 无新增 GORM import、nil 向后兼容），GormLoginLogger 实现在 system 侧由装配注入；T-002 AuthService 登录成功/失败调用（既有文件 @updated）。
- 日志列表/清理挂 RequirePerm，清理独立权限码。errcode offset +60~+63。openapi v0.6.0（redocly 0 error）。集成测试 system_integration_test.go 真 MySQL 验日志落库+过滤。commit f63c1ae+f1594d0。

**T-004b 收尾记录（PM 评审定档，T-004 整块完成）**
- 存储驱动 server/drivers/storage：StorageDriver 接口（Put/Get/Delete/Exists/URL）+ LocalDriver 本地磁盘实现；云驱动（OSS/S3）为扩展点，消费方实现注入，**底座不引任何云 SDK**；storage_type 字段留位。
- **路径穿越防护（头号）：safePath = filepath.Clean + 根目录前缀校验 + 拒 ../绝对路径/控制字符；单测 5 个恶意用例 + 集成测试真盘 os.Stat 确认根目录外无文件写出（双层闭合）。**
- 上传安全四件套（配置注入）：大小上限（流式判断不全载内存）/扩展名白名单/文件名消毒（去 ../控制字符超长）/Content-Type 与扩展名一致校验，各有断言。
- key 策略：yyyy/MM/dd/uuid.ext（日期分目录 + uuid 防覆盖/防枚举），原名存 sys_file.original_name 供展示下载。
- 鉴权下载：下载经 RequirePerm 接口流式返回，根目录不挂静态路由；URL() 返回下载接口引用，不泄漏真实文件系统路径。
- 删除：软删元信息 + 物理异步清理（限根目录内、失败仅告警不阻塞）。sys_file 元信息表 DDL。
- 错误码 +70~75 注册进 response.Registry，回归快照 38/38 一致。openapi v0.7.0（redocly 0 error）。
- 集成测试 file_integration_test.go（//go:build integration）真 MySQL+真磁盘：上传(含中文名)→落库→下载字节一致→软删；同名不覆盖；列表过滤；真盘穿越被拒+os.Stat 验。commit 8c9bd0a+448671f。

**T-004c 收尾记录（PM 评审定档，渲染债清）**
- 纯重构、对外零行为变化：消除 T-004a 遗留的两套渲染并存。
- **errcode.Error 剥离自渲染/HTTP 语义：删 HTTP/Message 字段，只留 Code + GetCode + Error()（满足 error 接口、作 service→handler 的 code 载体）；grep 无 .HTTP/.Message 引用。** service 层完全不动。
- **response.Registry 成为 code→HTTP→i18n 唯一权威 + 唯一渲染路径**：全 handler 走 response.OK/ErrResp/BadReq，全中间件走 response.AbortErr；grep 无 errcode.Error 渲染、无散落 AbortWithStatusJSON。各包旧 respondOK/respondError/fail/ok 已删。
- 零行为变化双证：migration 快照 38/38（code/HTTP/i18nKey 三维一致）+ T-001~T-004b 全部旧测试全绿。各包补 testmain 初始化 Registry。
- 重构纪律守住：未改业务/码值/接口签名；RequirePerm 26+ 处鉴权、参数校验、日志脱敏原样保留。openapi 未升版（对外契约不变）。commit 至 589f9e1。

**T-005 收尾记录（PM 评审定档，后端底座收官）**
- 边界：聚焦动态参数（sys_config/字典）缓存+热加载+加密；静态启动配置仍归 viper，不纳入。
- **GCM 加密：crypto 新增 EncryptGCM/DecryptGCM（AES-256-GCM 认证加密），与 T-001 CBC 信封并存互不影响、C 端向量回归全绿；主密钥配置注入 fail-fast；每次随机 12B nonce + 篡改检测。** 存储格式 base64(nonce12||ct||tag)，自描述、PHP 可解，入 spec。
- sys_config 增列 is_encrypted：写加密落库密文、读自动解密；**加密闭环三道：存密文 + 授权路径(ConfigCenter.GetConfig)解密 + 列表/详情 maskEncrypted() 返回 ******（脱敏断言 docker-free+integration）**。明文不入列表/日志。
- ConfigCenter + 缓存层：ConfigCache 接口（RedisConfigCache 真实现 + 内存假实现）；命中→回源 DB→回填；写后失效。
- **热加载 Redis Pub/Sub：RedisPublisher/RedisSubscriber（协程 + context 优雅退出）；集成测试验跨实例场景（实例A发布→实例B订阅→B缓存失效→读新值）真 Valkey 跑通；单实例无 Redis 退化本地刷新。**
- **SQL 迁移执行器 migrator.go（还历史债）：读 spec/migrations、文件名字典序排序、{{TABLE_PREFIX}} 替换、按序执行、sys_migration 记录版本+校验和、幂等跳过已执行、失败中止报告。** 替代此前集成测试手动 ReplaceAll，demo 可一键建表。
- errcode +80~81，回归快照 40/40。openapi v0.8.0（redocly 0 error）。commit 1c339e2+303bf1e。

**待 daxing 真人验收（用到时补，不阻塞）**
- 各片历史验收项见对应记录。
- T-004a：demo 字典/参数 CRUD + 操作/登录日志查看；确认日志无敏感信息；确认错误码返回与迁移前一致；评审 Registry 业务模块可注册。

### 进行中 / 待收尾
| 任务编号 | 切片 | 状态 | 待收尾项 |
|---|---|---|---|
| — | — | — | 无 |

### 下一步（计划）
1. **examples/demo 跑通**（登录+RBAC+配置+日志全链路）：用迁移执行器一键建表、装配 GormUserProvider/Redis 各 Store/ConfigCenter/hasher，串通五大块；**一次性兑现历次积压的 daxing 真人验收项**。不依赖前端。
2. admin 前端（布局/动态路由/权限/x-table）：在 demo 验证过的可信后端上联调。
> 建议执行顺序：demo 跑通（验证后端协同 + 清积压验收）→ admin 前端。前端联调时后端已是已验证状态。

### 阶段二（底座可用后）
- BenxinKP 引入 BenxinAdminPro，只写业务；backend-php 照 spec 实现 parity。

## 待决策（F 系列，见底座文档 §14）
| # | 决策项 | 状态 |
|---|---|---|
| F1~F5 | 见底座文档 | ✅ 已决（F2 单独 Project 建议是） |

## T-004 子切片拆分
- T-004a response 接管 + 字典 + 参数 + 日志 ✅
- T-004b 文件管理 + 存储驱动 ✅
- T-004c 渲染收敛（handler 统一走 response.Render、errcode 降级常量）✅

## 备注
- 宪法级：安全第一、仅开源素材、配置驱动化、参数化复用、统一代码头注释（中英文+到秒）。
- 设计硬约束：表前缀随实例走禁包级（T-003a）；未完成切片接口至少挂 JWT；对外 ID 入出参 hashid 闭环+装配注入 hasher（T-003b）；授权变更事务内回滚保一致（T-003b）；数据权限失败一律收紧绝不放宽（T-003c）；日志脱敏+异步不阻塞、auth 不因日志依赖 DB（T-004a）。
- 错误码：crypto/auth（T-001/T-002）、sys（T-003 +30~+50）、system（T-004a +60~+63）、file（T-004b +70~+75）；response.Registry 为唯一注册/渲染权威（T-004c 已收敛）；errcode 已降级为纯常量（Code 载体，剥离 HTTP/Message）；段不破坏、冲突 fail-fast。
- Casbin obj=perm code（非 URL），命名 模块:资源:动作（sys:user:list）；底座只放 sys:* 通用权限点。
- git 经验：镜像仓建仓勿勾初始化文件否则首推 force；token 走交互式/钥匙串勿写进 remote url；GitHub/Gitee 偶发 SSL_ERROR_SYSCALL 重试即可（近期转频，必要时走代理/SSH）。
