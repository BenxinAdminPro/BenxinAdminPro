# 完成报告：T-006 examples/demo 装配 + 收尾（真·跑通）

> 本文件覆盖上个会话误称完成的旧版。T-006 实际收尾经多轮排雷（#4 casbin → #5 迁移器 → #3 鉴权接线，
> 其中 #3 拆为独立切片 T-003d），最终 demo e2e 真依赖整条跑通。无"言过其实"表述。

## 1. 完成状态
✅ 真·完成（HEAD 813dce2，已双推 Gitee+GitHub）。demo 五大块全链路装配 + 真依赖 e2e 冒烟测试
8 步整条 ALL STEPS PASSED。上个会话的"完成"为自评未跑通；本轮以自动化 e2e 实跑证据收尾。

## 2. 改动文件清单（T-006 部分，commit 99716da）
| 文件 | 说明 | 新增/修改 |
|---|---|---|
| server/examples/demo/seed.go | 种子密码全部来自配置、零明文、缺失 fail-fast；删 admin123 兜底 | 修改 |
| server/examples/demo/config.example.yaml | seed 四用户密码占位（空），注释标明 fail-fast + env 注入 | 修改 |
| server/examples/demo/main.go | 装配抽出 buildApp(cfg,db,rdb)；装配错误改返回 error；SetEnvKeyReplacer | 修改 |
| server/examples/demo/e2e_integration_test.go | e2e 冒烟测试（//go:build integration），8 步逐项断言 | 新增 |
| server/system/migrator.go | #5 修复：先逐行剥 -- 注释再按 ; 切分（修建表静默失败） | 修改 |
| server/system/migrator_integration_test.go | 真库断言建表数>0 + 目标表真实存在（堵假阳性） | 新增 |
| server/go.mod / go.sum | #4：gorm-adapter/v3 钉 v3.38.0，移除 casbin/v3 indirect | 修改 |
> 注：main.go 的 authz 注入 + 鉴权接线属 T-003d（commit 813dce2），见 T-003d-report.md。

## 3. 接口/能力实现情况
- 装配：配置加载→MySQL/Redis→迁移执行器建表→各模块 DI→装配自检→种子→路由总装→启动；buildApp 供 e2e 复用。
- 公共路由 /auth/*（captcha/login/refresh/logout）；受保护 /sys/*（JWTAuth + 路由级真 enforce + OperLog）；C 端信封 /api/c/echo。
- 种子：超管/3 角色/3 部门/4 用户/菜单权限 + Casbin ReloadAll，幂等 upsert，密码全配置注入。

## 4. 自验结果（真实）
- go build ./... ✅ go vet ./... ✅ go vet -tags=integration ./... ✅
- 全量非集成单测 ✅（auth/crypto/storage/rbac/response/system）
- migrator 集成测试 ✅：真库建表 16、ALTER 列(is_encrypted/data_scope)验、sys_migration 版本数 18=文件数
- **demo e2e 真依赖（MySQL3307+Redis）整条转绿**：
  step1 迁移16表 → step2 种子(admin密码已hash) → step3 captcha → step4 login拿token
  → step5 带token 200 → step6 无token 401 → step7 无权 403 → step8 改配置热加载读新值
  → ALL STEPS PASSED
- 第 7 步 403 依赖 T-003d 鉴权接线修复（此前为 200，editor 能违规建用户）。

## 5. 排雷链（T-006 真实收尾经过）
| # | 缺陷 | 性质 | 处置 |
|---|---|---|---|
| 假阳性账本 | 上个会话自评"完成"未跑通 | 流程 | 本轮以 e2e 实跑证据纠正 |
| 必补① | 种子密码硬编码明文 | 安全债 | 去硬编码 + fail-fast（commit 99716da） |
| 必补② | 缺 e2e 冒烟测试 | 交付缺失 | 新增 //go:build integration e2e（99716da） |
| #4 | casbin v2/v3 冲突，NewEnforcer panic，demo 无法启动 | 依赖 | 钉 gorm-adapter v3.38.0（99716da） |
| #5 | 迁移器跳过 CREATE，建表静默失败却报 applied | T-005 bug | 修 splitStatements + 回归断言（99716da） |
| #3 | 权限 enforce 未接线，无权用户得 200 | T-003b 接线 | 拆独立切片 T-003d 修复（813dce2） |

## 6. git 提交记录
- commit 99716da `fix(demo): T-006 收尾 — 种子去硬编码 + e2e 冒烟 + 迁移器#5 + casbin#4`
- commit 813dce2 `fix(rbac): T-003d 鉴权接线 …`（使 e2e 第7步转绿）
- 双推：Gitee origin/main + GitHub github/main 同步，最终 HEAD 813dce2（ahead/behind 0 0）。

## 7. 安全自查
- [x] 仓库无真实密钥：config.example 占位为空，真值走 *.local.yaml/env（gitignore）
- [x] 种子零明文密码、缺失 fail-fast
- [x] 受保护路由真 enforce（详见 T-003d）；无权 403 经 e2e 实证
- [x] 迁移器不再假阳性（真库断言表存在）
- [x] 头注释/@updated 齐备

## 8. 需 daxing 真人验收
- demo 实跑：登录/RBAC/配置热加载/日志全链路；普通用户无权 403、超管放行；字典/参数 CRUD；文件上传下载穿越防护。
- 历次积压验收项（T-001~T-005）随 demo 一并核验（README 清单）。

## 9. 偏差与待办
- rbac/system 既有集成测试硬编码 localhost:3306（非本片改动），本机宿主 mysqld 占 3306，未在此环境运行；
  -tags=integration 编译通过；env 可覆盖的 e2e/migrator 已用 3307 跑绿。**待办：将来改 env 可覆盖。**
- 一次性 MySQL 容器 benxin-e2e-mysql(3307) 仅供 e2e，收尾后删除。

## 10. 下一步建议
- PROJECT_STATUS 由 PM 统一更新（执行端不自标完成）。
- 下一步 admin 前端：在已 e2e 验证的可信后端上联调（等 PM 任务书）。
