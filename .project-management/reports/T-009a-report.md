# 完成报告：T-009a 后台禁搜索引擎收录（noindex 三件套）

> 状态：执行端自验全绿，**待 PM 评审 + daxing 验收**。未自标完成、未双推、未改 PROJECT_STATUS（守 T-006 铁律）。

## 1. 完成状态
- ✅ 件①后端 X-Robots-Tag 全局中间件（参数化权威控制点，默认安全）。
- ✅ 件②admin `public/robots.txt`（`Disallow: /`，构建期固定）。
- ✅ 件③admin `index.html` 加 `<meta name="robots" content="noindex,nofollow">`（构建期固定兜底）。
- ✅ 默认安全：配置缺省（未声明）即「开 + noindex, nofollow」（viper SetDefault 兜底）。
- 范围外项零触碰：前端不做运行时/构建期 env 联动（A1 裁决）；无 WAF/IP 白名单；无 DDL/错误码/权限码/openapi 端点。

## 2. 改动文件清单

**后端（Go）**
| 文件 | 说明 | 类型 |
|---|---|---|
| `server/httpmw/robots.go` | 新建 httpmw 包（通用 HTTP 中间件之家）+ `XRobotsTag(RobotsConfig)` 全局中间件 + `DefaultXRobotsTag` 常量（默认值单一定义） | 新增 |
| `server/httpmw/robots_test.go` | 中间件四态单测（开默认/关/自定义/开但内容空回退） + 常量值锁定 | 新增 |
| `server/examples/demo/main.go` | 全局注册中间件（Recovery 之后，公共路由之前）；demoConfig 加 XRobotsEnabled/Content；loadConfig 加 SetDefault 默认安全 + 读取 | 修改 |
| `server/examples/demo/config.example.yaml` | 加 `security.x_robots_tag` 段（committed 示例 + 默认安全/私有化关闭说明） | 修改 |
| `server/config/config.go` | 库参考结构体 AppConfig 加 SecuritySection/XRobotsTagSection（保持 canonical 结构完整 + 默认安全责任注释） | 修改 |
| `server/config/config.example.yaml` | 库参考示例加 `security.x_robots_tag` 段 | 修改 |
| `server/examples/demo/robots_integration_test.go` | 全栈 e2e：经 buildApp 坐实全局注入(含 401/404)+内容回退+自定义+toggle 关无头 | 新增 |

**前端（admin）**
| 文件 | 说明 | 类型 |
|---|---|---|
| `admin/public/robots.txt` | 纯 ASCII：一行英文注释 + `User-agent: * / Disallow: /`，Vite 拷到站点根（退回修复后去中文代码头，见 §10） | 新增 |
| `admin/index.html` | head 加 robots meta（noindex,nofollow） | 修改 |

> `server/examples/demo/config.local.yaml`（gitignored，未提交）也加了 security 段，便于 daxing 本地运行 + 可选 toggle 验证。
> `.project-management/PROJECT_STATUS.md` 本会话 `M` 状态系**外部（PM）编辑**，执行端未触碰。

## 3. §0 源码核实门禁（逐项过）
- ✅ **中间件注册**：全局 `r.Use(...)` 在 engine 级（main.go:238 Recovery 之后）。本片中间件挂此处、**先于公共路由与 protected 组**，对所有响应（含鉴权失败 401/未匹配 404）无差别注入。
- ✅ **静态配置范式**：demo 用自有 `demoConfig` + viper `loadConfig`；库 canonical 为 `config.AppConfig`。新配置走 `security.x_robots_tag.{enabled,content}` viper 键 + struct 字段，沿用既有范式。**默认安全经 `v.SetDefault` 兜底**（viper GetBool 缺省返 false=不安全 → SetDefault enabled=true + content=DefaultXRobotsTag）。
- ✅ **后端服务对象**：后端纯 API（admin 是独立 Vite 站点），X-Robots-Tag 注入 JSON/错误响应均无副作用（爬到 API 也不该被索引）；C 端 echo 等同样覆盖。
- ✅ **admin 静态目录**：`admin/public/`（原有 favicon.svg）Vite 原样拷站点根 → 构建产物根含 `/robots.txt` 已实测；`admin/index.html` head 加 meta。

## 4. 接口契约
- 不新增端点。X-Robots-Tag 为响应头、非 API 契约。openapi 不升版（未触碰 spec，按任务书"可提可不提，不强制"——本片未改 openapi.yaml）。

## 5. 自验结果

**默认闸门全绿**
```
go build ./...  OK
go vet ./...    OK
go vet -tags=integration ./...  OK
gofmt -l（本片文件）  clean
go test ./...   ALL ok（含 httpmw 新包）
admin: pnpm build OK + pnpm test 17 passed
```

**中间件单测（httpmw/robots_test.go，进默认闸门）**
- ① 开 + 默认内容 → 头 == `noindex, nofollow`。
- ② 关 → **无该头**。
- ③ 自定义内容 → 注入自定义值。
- ④ 开但内容空 → 回退 `DefaultXRobotsTag`（坐实「开了不会注入空头」= 默认内容安全）。
- ⑤ `DefaultXRobotsTag` 常量值锁定 == `noindex, nofollow`。

**全栈 e2e（robots_integration_test.go，真 buildApp+MySQL+Redis）**
- 开 + 内容空 → **401（无 token 的 /sys/users）与 404（未知路径）响应均带** `X-Robots-Tag: noindex, nofollow`（坐实全局无差别注入 + 内容回退全栈生效）。
- 自定义内容 → 全栈注入自定义值。
- 关 → 全栈**无该头**（参数化 toggle 生效）。

**默认安全（§5 命门）**：loadConfig `v.SetDefault("security.x_robots_tag.enabled", true)` + `content=httpmw.DefaultXRobotsTag` → 配置整段省略也走禁收录；中间件单测④坐实内容空回退；不会因 key 未声明静默放开收录。

**前端构建产物实测**
- `admin/dist/robots.txt` 内容 == `User-agent: * / Disallow: /`。
- `admin/dist/index.html` head 含 `<meta name="robots" content="noindex,nofollow">`。

## 6. 安全自查
- **默认安全**：配置缺省 = 注入开 + `noindex, nofollow`（SetDefault + 中间件内容回退双保险），不可因配置缺失静默放开收录。
- **参数化不破坏默认安全**：开关可关（私有化场景 enabled=false），但默认值是禁收录。默认内容常量单一定义（`DefaultXRobotsTag`），逻辑分支不硬编码字面。
- **零副作用**：中间件纯响应头注入，不读 JWT、不查 DB、不依赖请求上下文；对所有响应（含错误/404）无差别注入，全栈测试坐实。
- **底座中立**：开关/内容全配置注入；新建 httpmw 通用包不绑应用，不硬编码应用名/前缀。

## 7. 需 daxing 真人验收
> **后端重启**（本片改 Go 中间件 + config，前端热更新不含后端）：`lsof -ti :8080 | xargs kill -9` 后 `cd server && go run ./examples/demo`。
1. `curl --noproxy '*' -I http://localhost:8080/sys/users`（或任一端点）→ 响应头含 `X-Robots-Tag: noindex, nofollow`（401 也带，证无差别注入）。
2. 浏览器（或 `curl --noproxy '*'`）访问 admin 站点根 `/robots.txt` → 返回 `Disallow: /`；查看 admin 页面源码 head 含 robots meta。
3. （可选）config.local.yaml 里 `security.x_robots_tag.enabled: false` → 重启 → curl 确认响应头消失（参数化生效）。

## 8. 偏差与待办
- **无范围外扩**。严格三件套 + 参数化权威控制点（后端中间件），前端兜底件构建期固定（守 A1 裁决，不引构建变量）。
- **搭车决策（非偏差）**：库 canonical `config.AppConfig` 加了 SecuritySection 保持参考结构体完整（demo 用自有 demoConfig，故此为库消费方参考；附默认安全责任注释）。`config/config.example.yaml` 同步。
- **新建 httpmw 包**：选址为通用 HTTP 中间件之家（区别于 crypto/auth 领域包），为将来其他安全响应头（CSP/X-Frame-Options 等）预留，本片只放 X-Robots-Tag。
- **账本既有「更根本」备忘（非本片）**：管理后台正路是内网/VPN/IP 白名单/不挂公网 DNS，noindex 三件套是「万一暴露也不被收录」的兜底——属部署形态，另记。

## 9. 下一步建议
- daxing 重启 demo + curl/浏览器三项验收 → PM 放行 → 三提交链双推归档（feature → 账本 → 报告归档）。
- 新需求池另一条「媒体管理」体量中大、耦合 x-table 多选基建，真排时先只读摸底再拆（账本已记）。

## 10. 编码修复补充（T-009a 退回修复 · 2026-06-15）
> daxing 验收发现 robots.txt 顶部中文代码头注释乱码。功能两行（`User-agent: *` / `Disallow: /`）一直正确，问题在文件套用了源码五项中文代码头。

- **修法 = 选项 B（PM 推荐）**：robots.txt **去掉中文代码头注释**，只留协议内容 + 一行纯英文注释：
  ```
  # BenxinAdminPro admin - disallow search engine indexing
  User-agent: *
  Disallow: /
  ```
  **规范澄清**：robots.txt 是机器读的标准协议文件、非源码，**不纳入「五项代码头注释」规范**（PM 记账本）。
- **编码自测（执行端）**：
  - `file admin/public/robots.txt` → `ASCII text`（纯 ASCII，从根上无 UTF-8/GBK 解析歧义）。
  - 非 ASCII 字节扫描（`grep -P '[^\x00-\x7F]'`）→ 空（纯 ASCII ✓）。
  - 首字节 `23`（`#`），**无 BOM** ✓。
  - 重新 `pnpm build` → `dist/robots.txt` 同为 ASCII、内容正确。
- **顺带核查 index.html**：首字节 `3c`（`<!doctype`），**无 BOM**；`file` 报 UTF-8；本片新增 robots meta 行 `<meta name="robots" content="noindex,nofollow" />` **纯 ASCII**（同行的说明注释为中文，但 index.html 本就是 `lang="zh-CN"` 的 UTF-8 文档，中文注释合规、非编码问题）。
- **范围**：只改 robots.txt 一处（去中文头改纯 ASCII），其余文件零改动；后端中间件 + 前端 meta 功能验收已过未动。
