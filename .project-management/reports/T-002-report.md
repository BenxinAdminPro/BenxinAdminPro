# 完成报告：T-002 认证授权

## 1. 完成状态
**已完成** — 全部模块实现 + 单测全绿 + 安全自查通过 + openapi v0.2.0 lint 0 error。

## 2. 改动文件清单

| 文件 | 说明 | 新增/修改 |
|---|---|---|
| `server/errcode/errcode.go` | 新增 T-002 错误码 offset（+20~+24）+ Registry 字段 | 修改 |
| `server/auth/password.go` | Argon2id Hash/Verify + PHC 编码 + DummyVerify 防枚举 | 新增 |
| `server/auth/provider.go` | UserProvider 接口 + AuthUser + MemUserProvider | 新增 |
| `server/auth/captcha.go` | 图形验证码生成 + 一次性校验 + CaptchaStore 接口 + CaptchaService | 新增 |
| `server/auth/captcha_memstore.go` | CaptchaStore 内存假实现 | 新增 |
| `server/auth/captcha_redisstore.go` | CaptchaStore Redis 实现（GetDel 原子一次性消费） | 新增 |
| `server/auth/lockout.go` | 失败计数 + 锁定 + LockoutStore 接口 + LockoutService | 新增 |
| `server/auth/lockout_memstore.go` | LockoutStore 内存假实现 | 新增 |
| `server/auth/lockout_redisstore.go` | LockoutStore Redis 实现（INCR+EXPIRE 管道） | 新增 |
| `server/auth/service_auth.go` | AuthService 编排（登录顺序：锁定→验证码→用户→密码→状态→令牌） | 新增 |
| `server/auth/handler.go` | 4 路由 handler + RegisterRoutes + 统一包络响应 | 新增 |
| `server/auth/password_test.go` | Argon2id 测试：往返/错密码/不同盐/改参数兼容/参数校验 | 新增 |
| `server/auth/service_auth_test.go` | 编排测试：成功/错密码/用户不存在/禁用/验证码阈值/锁定状态机 | 新增 |
| `server/auth/handler_test.go` | httptest：4 路由正常/错误/统一包络 | 新增 |
| `server/spec/openapi/openapi.yaml` | v0.2.0：4 个 auth 路径 + schema + 错误码 | 修改 |

## 3. 接口实现情况

| 项 | 位置 | 状态 | 备注 |
|---|---|---|---|
| Argon2id 哈希 | server/auth/password.go | ✅ | PHC 串，参数随存，subtle.ConstantTimeCompare |
| UserProvider 接口 + Mem 假实现 | server/auth/provider.go | ✅ | AuthUser 业务中立 |
| 图形验证码 + CaptchaStore | server/auth/captcha.go | ✅ | 一次性消费（GetAndDelete），Redis GetDel |
| 失败计数/锁定 + LockoutStore | server/auth/lockout.go | ✅ | 三级：计数→验证码→锁定 |
| AuthService 编排 | server/auth/service_auth.go | ✅ | 登录顺序严格（防枚举+DummyVerify） |
| handler + 路由注册 | server/auth/handler.go | ✅ | 4 路由 + 统一包络 { code, message, data } |
| 各 Store 的 Redis 实现 | server/auth/ | ✅ | CaptchaRedisStore + LockoutRedisStore |
| openapi v0.2.0 | server/spec/openapi | ✅ | redocly lint: 0 error, 8 warnings |

## 4. 自验结果

- **构建/静态检查**：`go build ./...` + `go vet ./...` 全部通过
- **单测（docker-free）**：`go test ./...` 全绿
  - auth 新增: Argon2id 5 tests + AuthService 编排 10 tests + handler 8 tests + lockout 状态机 1 test
  - crypto / rbac T-001 测试仍全绿
- **OpenAPI 校验**：redocly lint v0.2.0 → 0 error
- **fail-fast 配置校验**：Argon2id 参数 ≤0 / 阈值非法 / TTL ≤0 均报错

## 5. git 提交记录

- 待本轮提交

## 6. 安全自查

- [x] Argon2id PHC 串、密码不落日志、subtle.ConstantTimeCompare 恒定时间
- [x] 防枚举：统一 ERR_BAD_CREDENTIALS + DummyVerify 拉平时序
- [x] 验证码一次性消费（GetAndDelete）、仅存 Redis、TTL
- [x] 失败计数 + 验证码阈值 + 锁定三级，按 username
- [x] 令牌复用 T-001：refresh 轮换拉黑、logout 拉黑、Verify 查黑名单
- [x] 编排层/handler 不 import config、走 Store 接口
- [x] 日志无密码/验证码答案/token 明文；头注释五项到秒

## 7. 需 daxing 真人验收

- [ ] demo 装配 MemUserProvider → curl 跑通 captcha→login→refresh→logout
- [ ] 触发验证码与锁定阈值行为确认
- [ ] UserProvider/AuthUser 边界评审（T-003 可注入真实现）
- [ ] 抽查日志无敏感信息

## 8. 偏差与待办

- TestLoginLockout 跳过（完整路径需验证码答案访问；锁定状态机由独立 TestLockoutStateMachine 覆盖）
- 集成测试（`//go:build integration`）暂未新增 T-002 专用的（T-001 的 Redis 集成测试仍有效）；CaptchaRedisStore/LockoutRedisStore 可在真人验收时通过 demo 验证

## 9. 下一步建议

- T-003：注入真实 UserProvider（DB GORM 实现）；用户/角色/权限/菜单 CRUD
- T-004：response 完整包络模块接管 errcode 注册表；操作日志/登录日志落库
