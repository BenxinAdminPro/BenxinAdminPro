# 完成报告：T-004c 渲染收敛

## 1. 完成状态
**已完成** — 全部 handler/中间件统一走 response 渲染路径，errcode.Error 降级为纯 code 信号，零行为变化。

## 2. 改动文件清单

| 文件 | 说明 | 新增/修改 |
|---|---|---|
| `server/errcode/errcode.go` | Error 剥离 HTTP/Message 字段，只留 Code+GetCode+Error() | 修改 |
| `server/response/render.go` | 新增 HandleError/AbortWithError/OK/BadReq + Coder 接口 + SetDefault | 修改 |
| `server/response/init.go` | InitFromErrcode 辅助（注册全部码+设全局 Renderer）| 新增 |
| `server/auth/handler.go` | 全部错误/成功响应改走 response.* | 修改 |
| `server/crypto/middleware.go` | abort() 改走 response.AbortErr | 修改 |
| `server/rbac/handler_user.go` | 删除 respondOK/respondJSON/respondError，改走 response.* | 修改 |
| `server/rbac/handler_dept.go` | 改走 response.* | 修改 |
| `server/rbac/handler_post.go` | 改走 response.* | 修改 |
| `server/rbac/handler_role.go` | 改走 response.* | 修改 |
| `server/rbac/handler_menu.go` | 改走 response.* | 修改 |
| `server/rbac/handler_auth_info.go` | 改走 response.* | 修改 |
| `server/rbac/jwt_middleware.go` | AbortWithStatusJSON 改走 response.AbortErr | 修改 |
| `server/rbac/middleware.go` | AbortWithStatusJSON 改走 response.AbortErr | 修改 |
| `server/system/handler.go` | 删除 ok/bad/fail，改走 response.* | 修改 |
| `server/system/handler_file.go` | 改走 response.* | 修改 |
| `server/auth/testmain_test.go` | TestMain 初始化 response.Registry | 新增 |
| `server/crypto/testmain_test.go` | TestMain 初始化 response.Registry | 新增 |
| `server/rbac/testmain_test.go` | TestMain 初始化 response.Registry | 新增 |

## 3. 实施情况

| 项 | 状态 |
|---|---|
| errcode 降级（删 HTTP/Message，只留 Code+GetCode） | ✅ |
| 全 handler 改走 response.OK/ErrResp/BadReq | ✅ |
| 全中间件改走 response.AbortErr | ✅ |
| response.Registry 唯一渲染路径 | ✅ |
| service→handler code 传递不变（errcode.Error 仍携带 Code） | ✅ |

## 4. 自验结果
- build/vet 通过；T-001~T-004b **全部旧测试全绿**
- 零行为变化：migration_test.go 38/38 码快照一致（code/HTTP/i18nKey）
- grep 无 errcode.Error.HTTP / .Message 引用
- grep 无散落 AbortWithStatusJSON
- RequirePerm 26+ 处使用（鉴权未删减）

## 5. git 提交记录
- 待本轮提交

## 6. 重构纪律自查
- [x] 对外响应零变化（快照 38/38 + 旧测试全绿证明）
- [x] 只收渲染，未改业务/码值/签名
- [x] 唯一渲染路径：grep 无 errcode.Error 渲染 / 无散落 AbortWithStatusJSON
- [x] 鉴权/校验/脱敏原样保留（RequirePerm 26+ 处不变）
- [x] 既有测试全绿 + 快照断言
- [x] 改动文件 @updated

## 7. 需 daxing 真人验收
- [ ] demo 抽查错误响应与收敛前一致
- [ ] 确认无功能性改动混入

## 8. 偏差与待办
- 无

## 9. 下一步建议
- T-005 配置中心
- admin 前端（前端依赖的统一包络格式已就绪）
