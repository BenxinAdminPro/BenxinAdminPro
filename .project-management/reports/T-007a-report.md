# 完成报告：T-007a admin 前端地基

## 1. 完成状态
✅ 代码完成、自验通过（构建+类型检查+dev 服务），**待 daxing 浏览器联调验收 + push 确认**。
admin 从最小脚手架建成可运行地基：请求层（JWT/包络/错误码/自动刷新/hashid 透传）+ 登录页（图形验证码 + 错误码友好提示）+ 路由守卫 + 响应式布局 + Pinia + 主题/i18n 骨架。未推、未碰后端。

## 2. 改动文件清单
| 文件 | 说明 | 新增/修改 |
|---|---|---|
| admin/package.json | 加依赖：element-plus / pinia / vue-router / axios / vue-i18n / @vueuse/core / @element-plus/icons-vue / unocss(66) / sass | 修改 |
| admin/vite.config.ts | 路径别名 @、UnoCSS 插件、dev 代理(/auth /sys /api → 后端，免 CORS) | 修改 |
| admin/uno.config.ts | UnoCSS 预设（presetUno+presetAttributify，MIT） | 新增 |
| admin/tsconfig.app.json | paths 别名 @/* | 修改 |
| admin/index.html | 标题/lang → BenxinAdminPro / zh-CN | 修改 |
| admin/.env | 占位环境变量（无密钥；!.env 反忽略入库） | 新增 |
| admin/.gitignore | `!.env` 反忽略（根 .gitignore 默认忽略 .env）；真值仍走 .env.local | 修改 |
| admin/CREDITS.md | 依赖/素材许可记录（全 MIT/Apache，无商业模板/字体） | 新增 |
| admin/src/main.ts | 装配 UnoCSS/ElementPlus/Pinia/Router/i18n/主题 | 修改 |
| admin/src/App.vue | el-config-provider（EP 组件语言随 i18n）+ router-view | 修改 |
| admin/src/env.d.ts | 环境变量类型 + virtual:uno.css 声明 | 新增 |
| admin/src/request/{index,types}.ts | 请求层核心 | 新增 |
| admin/src/api/auth.ts | captcha/login/refresh/logout | 新增 |
| admin/src/store/{index,user,app}.ts | Pinia + user/app store | 新增 |
| admin/src/router/{index,routes}.ts | 路由实例 + 守卫 + 常量路由 | 新增 |
| admin/src/layout/{index,components/*}.vue | 响应式布局壳（Sidebar/Navbar/AppMain） | 新增 |
| admin/src/views/{login,home,error/404}/*.vue | 登录页 / 工作台占位 / 404 | 新增 |
| admin/src/theme/index.ts | 暗色(useDark)+主题色切换骨架 | 新增 |
| admin/src/i18n/{index,locales/zh-CN,en}.ts | i18n 框架（中文默认+预留 en） | 新增 |
| admin/src/composables/useDevice.ts | 响应式设备检测，联动侧边栏 | 新增 |
| admin/src/utils/auth.ts | 令牌存储工具（localStorage） | 新增 |
| admin/src/styles/index.scss | 全局样式 | 新增 |
| （删除）HelloWorld.vue / style.css / assets 示例图 / public/icons.svg | 清理脚手架样例 | 删除 |

## 3. 实现情况
| 项 | 位置 | 状态 | 备注 |
|---|---|---|---|
| 请求层（JWT/包络/错误码/刷新/hashid 透传） | src/request | ✅ | 见下 |
| 登录页（captcha + 错误码处理） | src/views/login | ✅ | 验证码 base64 显示+点击刷新；失败显后端 message+刷新码 |
| 路由/布局框架 + 守卫 | src/router, src/layout | ✅ | 未登录跳登录；已登录访问登录页回首页；响应式 |
| Pinia user/app store + 令牌持久化 | src/store | ✅ | user(令牌/用户名/登录登出) + app(折叠/设备) |
| 主题 + i18n 骨架 | src/theme, src/i18n | ✅ | 暗色 html.dark + 主题色 css-var 覆盖；文案走 key |
| .env 配置化 | admin/.env | ✅ | baseURL 留空走 dev 代理；真值 .env.local |

**请求层要点（对接已验证后端契约）**：
- 请求拦截注入 `Authorization: Bearer <access>`（认证端点除外）。
- 响应拦截解包 `{code,message,data}`：code==0 返回 data。
- 错误分流按 **HTTP 状态**（code=base+offset、base 可配，不硬编码绝对码）：401→刷新重试一次（去重并发刷新）/失败跳登录；403→提示无权；423→锁定提示；其余→message toast。
- **401 刷新只对非 /auth/* 端点触发**——后端"凭证错误"也是 401 但发生在 /auth/login，已被 isAuthUrl 排除，不会误刷新（这是契约核对出的关键陷阱）。
- 认证端点错误一律交登录页处理（本层不 toast/跳转），避免双重提示。
- hashid：ID 字符串原样透传，不解码。
- CBC 加密信封：admin 不接（注释说明，C 端 uni 才用）。

## 4. 自验结果
- **构建+类型检查**：`pnpm build`（vue-tsc -b + vite build）✅ 通过，1685 模块转译成功，0 类型错误。
- **dev 服务**：`pnpm dev` ✅ 启动（Vite8 ready ~350ms），监听 5173，node fetch 验证首页正确返回（含 `#app`、标题 BenxinAdminPro）。
- 仅开源素材：CREDITS 记录，全 MIT/Apache/ISC；无第三方 admin 模板、无商业字体/图标（仅 @element-plus/icons-vue MIT + 系统字体栈）。
- **未做**：浏览器实操登录流程（需后端 demo 在跑，属 daxing 验收项，见 §8）；前端无 vitest 测试基建（请求层逻辑已结构化便于将来补单测）。
- 已知警告（非错误）：① element-plus 全量引入致主 chunk ~951kB（地基取稳，后续可 auto-import 瘦身）；② @vueuse/core 14（element-plus 自带副本）的 `/* #__PURE__ */` Rolldown 注释位置警告，无害。

## 5. 令牌存储方案与安全权衡说明
- **方案**：access + refresh 均存 **localStorage**（key 前缀 `bxap_`）。
- **理由**：后端用 Bearer 令牌（非 Cookie 会话），不存在 Cookie 自动携带面 → 无 CSRF 暴露；localStorage 便于 SPA 跨标签读取与刷新逻辑。
- **XSS 权衡**：localStorage 可被 XSS 读取——以"不 v-html 不可信内容 + Vue 默认转义 + 生产不打印令牌到 console"缓解；**令牌绝不进 URL**；登出彻底清理（clearTokens + 用户名）。
- **可演进**：若将来需更强隔离，可改后端下发 httpOnly Cookie + 前端不持令牌的方案（需后端配合，超出本片）。

## 6. git 提交记录
待 daxing 确认后双推（commit 带 T-007a）。当前未提交、未推。

## 7. 安全自查
- [x] 仅开源素材、无第三方 admin 模板/商业字体（CREDITS 记录）
- [x] 令牌存储权衡说明、不在 URL 带令牌、登出清理
- [x] console 无令牌/密码打印
- [x] .env 配置化、占位无密钥；真值 .env.local（*.local gitignore）；.gitignore 加注释防误放密钥
- [x] 头注释五项（TS/Vue 块注释）

## 8. 需 daxing 真人验收（浏览器）
- 起本地 demo 后端（go run examples/demo + MySQL/Redis）+ `pnpm dev`，浏览器：打开→登录页→输错密码/验证码错→正确登录→进首页→登出。
- 令牌过期场景（改短 access TTL 或等待）确认自动 refresh 一次或正确跳登录。
- 响应式：PC/平板/手机（窄屏侧栏转抽屉）布局正常；暗色/语言切换。
- 确认无商业素材、console 无令牌泄漏。
- 评审请求层封装与错误码处理是否清晰（为 T-007b CRUD 铺底）。

## 9. 偏差与待办
- 登录页始终要求验证码：后端仅在累计失败≥阈值后才强制校验，首登发送的验证码后端会忽略（无害）；如需"按需显示验证码"可后续优化。
- element-plus 全量引入（bundle 偏大）：地基取构建稳定，后续可切 unplugin-auto-import 按需引入瘦身。
- 前端无 vitest 单测基建：请求层刷新/错误码逻辑已结构化，将来可补；本片以构建+类型+dev+真人浏览器验收兜底。
- 首页用户名取自登录表单（非 /me 接口）：用户信息/菜单/权限消费留 T-007b（/sys/auth/menus、/sys/auth/perms）。

## 10. 下一步建议（T-007b）
- 动态路由（消费 /sys/auth/menus）+ `v-permission` 按钮指令（消费 /sys/auth/perms）+ x-table 配置化 CRUD + sys 模块样例页，在本地基上接入（路由框架已留 addRoute 接入点、布局侧栏已留动态菜单位）。
