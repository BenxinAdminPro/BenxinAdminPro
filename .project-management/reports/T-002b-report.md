# 完成报告：T-002b 验证码修复

## 1. 完成状态
✅ 代码完成、自验通过（后端 build/vet/test 全绿 + 前端构建 + 验证码样图目视可读），**待 daxing 浏览器验收 + push 确认**。
两件事都修：① 验证码换开源 Go 字体渲染清晰字符；② 前后端触发对齐（后端 precheck 信号 + 前端按需显示）。未推。

## 2. 改动文件清单
| 文件 | 说明 | 新增/修改 |
|---|---|---|
| server/auth/captcha.go | opentype + Go Mono Bold 渲染清晰字符（替换伪随机像素块）；字符集可配（默认排除易混） | 修改 |
| server/auth/captcha_test.go | 渲染可读性断言 + 字符集排除易混 + 样图落盘(CAPTCHA_DUMP) | 新增 |
| server/auth/service_auth.go | AuthService 加 Precheck(username)→是否需要验证码 | 修改 |
| server/auth/service_auth_test.go | TestPrecheckCaptchaRequiredSignal（阈值前 false/后 true） | 修改 |
| server/auth/handler.go | 新增 POST /auth/precheck | 修改 |
| server/auth/handler_test.go | TestHandlerPrecheck | 修改 |
| server/spec/openapi/openapi.yaml | v0.8.1 + /auth/precheck 文档 | 修改 |
| server/CREDITS.md | Go 字体（BSD-3-Clause）许可记录 | 新增 |
| server/go.mod, go.sum | golang.org/x/image 直接依赖 | 修改 |
| admin/src/api/auth.ts | precheck(username) API | 修改 |
| admin/src/views/login/index.vue | 按 precheck 信号按需显示验证码（不再总要求） | 修改 |

## 3. 实现情况
| 项 | 位置 | 状态 | 备注 |
|---|---|---|---|
| 验证码换开源字体渲染（可读） | server/auth/captcha.go | ✅ | Go Mono Bold(opentype)；字符集排除 0/O/1/l/I/o |
| 干扰强度增强（防 OCR） | server/auth/captcha.go | ✅ | 逐字符 ±16° 旋转 + 整体正弦波浪扭曲 + 间距收紧略粘连 + 5 条交叉干扰线(半数深色) + 噪点~3%；调到"能读出但不一目了然"（6 张样图目视确认） |
| 验证码填错/用过即刷新 | admin 登录页 | ✅ | 一次性消费，登录失败时已显示的验证码必刷新换新 |
| 触发信号 captcha_required（选 a） | server/auth + openapi | ✅ | **方案(a)：新增 POST /auth/precheck** {username}→{captcha_required}，不污染错误包络、可独立测 |
| 后端独立判定+强制校验（防绕过） | server/auth Login | ✅ | **未改既有逻辑——T-002 已服务端权威**：Login 起始查 NeedsCaptcha，达阈值则强制校验，前端不传/乱传一律拒（TestLoginCaptchaThreshold/Invalid 已断言） |
| 前端按需显示验证码 | admin 登录页 | ✅ | 用户名 blur 调 precheck；登录失败后重查（跨阈值则显示+刷新码）；需要时才校验/提交 |
| openapi v0.8.1 + spec 触发语义 | spec/openapi | ✅ | redocly 0 error（49 warnings 均既有未用组件） |

**触发对齐方案说明（选 a precheck 接口，未选 b）**：
- 选 (a) 的理由：option (b)"在 login 失败响应里带 captcha_required"需把字段塞进**错误响应**，而本项目错误响应经 response.Registry 统一渲染为 {code,message}（无 data），塞字段会破坏统一包络/特例化 handler。precheck 独立接口干净、服务端权威、易测、易文档化。
- 前端流程：① 用户名失焦 precheck（达阈值则proactive显示）；② 登录失败后再 precheck（跨过阈值时显示验证码并加载）；③ 需要时提交 captcha_id+code，后端校验。
- 与后端阈值语义一致：失败 3 次（captcha_threshold）后第 4 次需验证码——前端在第 3 次失败后 precheck 转 true 显示。

## 4. 自验结果
- **后端**：`go build ./... && go vet ./... && go test ./...` 全绿（auth/crypto/storage/rbac/response/system）。
- **验证码可读 + 抗自动化**：captcha_test 断言渲染产出有效 PNG、有字形深色像素、非纯色块、背景占多数；字符集排除易混。**6 张不同字符样图目视确认**："看一眼能读出但不一目了然"——字符经旋转/波浪扭曲/穿插干扰线后人眼需稍辨认仍可读，通用 OCR 不易直接识别（CAPTCHA_DUMP=1 复现 /tmp/captcha_sample_*.png）。字符颜色压暗保证最坏随机仍清晰。
- **触发对齐**：TestPrecheckCaptchaRequiredSignal（阈值前 false/后 true）；TestHandlerPrecheck（新用户 false + 缺 username 400）。
- **后端独立判定防绕过**：TestLoginCaptchaThreshold（3 失败后第 4 次不带验证码→ErrCaptchaRequired）+ TestLoginCaptchaInvalid（错验证码→ErrCaptchaInvalid）——**证明后端不依赖前端是否传验证码，达阈值即强制校验**。
- **一次性消费/阈值锁定不变**：TestCaptchaOneTimeConsume + TestLoginLockout 等 T-002 既有测试全绿，锁定语义未动。
- **前端**：`pnpm build`（vue-tsc 类型检查 + Vite 构建）✅。
- openapi redocly：0 error。

## 5. 字体许可说明（BSD-3-Clause + CREDITS）
- 字体：**Go Mono Bold**（`golang.org/x/image/font/gofont/gomonobold`），Bigelow & Holmes 为 Go 项目制作。
- 许可：**BSD-3-Clause**（与 Go 同），公认宽松开源许可，符合"仅开源素材"（任务的 OFL/Apache 是举例，BSD 同属开源；当初"零字体文件"理解过严）。
- 引入方式：经 Go module 依赖、构建期编译进二进制（等同 go:embed），无单独二进制文件维护；许可全文记入 server/CREDITS.md。

## 6. git 提交记录
待 daxing 确认后双推（commit 带 T-002b，跨后端+前端）。当前未提交、未推。

## 7. 安全自查
- [x] 可读但保留干扰防 OCR（每字符位移/颜色抖动 + 干扰线 + 稀疏噪点，强度以不牺牲可读性为度）
- [x] 后端独立判定是否需要验证码并强制校验（不依赖前端；precheck 仅 UX 信号）
- [x] 一次性消费（GetDel）、阈值/窗口/锁定语义不变（T-002 既有）
- [x] 仅开源字体（BSD-3-Clause）、许可入 CREDITS
- [x] 头注释 @updated（captcha.go / handler.go / 登录页）

## 8. 需 daxing 真人验收（浏览器）
- 验证码图**能看清字符**（核心）。
- 首次登录不显示/不强制验证码即可登录（未达阈值）；连续输错到阈值（默认 3）后验证码框出现且必须填对才能登录。
- 故意前端绕过（不填验证码）在"需要"状态下仍被后端拒（可 curl /auth/login 不带 captcha 验证→ErrCaptchaRequired）。
- 确认字体开源许可（CREDITS）。

## 9. 偏差与待办
- 触发对齐选 (a) precheck 接口而非任务推荐的 (b)——因 (b) 会污染统一错误包络，(a) 更干净（已在 §3 说明）。
- 集成测试：触发/防绕过逻辑由 service 级单测（MemLockoutStore）覆盖（与真 Redis 行为一致，锁定 store 本身 T-002 已有 Redis 集成测试）；未额外加 Redis 集成测试（逻辑未改，避免冗余）。
- 前端登录页：precheck 在用户名失焦触发；若用户不失焦直接提交，登录失败后会补 precheck 并按需显示——两路径都覆盖。
- **干扰强度为调参+目视结果**（非可断言的硬指标）：当前档位"能读出但不一目了然"；若实测仍偏易/偏难，参数（旋转角/波幅/线数/噪点密度/字符色深）集中在 captcha.go 的 render/warpWave/addLines/addNoiseDots，易再调。
- **待办（不在本片，PM 已记）**：图形验证码非主力抗自动化防线，真实防护仍靠失败锁定（T-002 已有）。将来如需强防护，正路是行为验证码（滑块 / Cloudflare Turnstile 等），而非把自绘验证码画更花；但涉及第三方依赖与底座中立性，需单独评估（可能违背"仅开源/底座中立"，需 PM 决策）。

## 10. 下一步建议（T-007b）
- 动态路由（/sys/auth/menus）+ v-permission（/sys/auth/perms）+ x-table 配置化 CRUD + sys 模块样例页，在 T-007a 地基上接入。
