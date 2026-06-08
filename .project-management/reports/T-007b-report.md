# 完成报告：T-007b 前端权限与 CRUD

## 0. 前置确认（§4.0）
逐项核对后端源码（handler + encoder + DTO），契约清晰：
- **/sys/auth/menus**：嵌套树，每节点 `id(hashid) / parent_id / menu_type(M目录/C菜单/F按钮) / name / perm_code / path / **component(如 "sys/user/index")** / icon / sort / visible / status / children`。**含组件路径字段 → 动态路由可行，无 PM 阻塞、无需后端补字段。**
- **/sys/auth/perms**：`string[]` 权限码数组（response.OK(codes)）。
- **/sys/users**：List `?username=&status=&dept_id=&page=&page_size=` → `{list:[User], total, page, page_size}`；User 含 `id(hashid)/username/nickname/email/mobile/status/remark/dept_id(hashid|null)`；`:id` 路径走 **hashid 解码（parseHID）**→ CRUD 透传成立；CreateUserInput 需 username+password；UpdateUserInput 无 username。
- **/sys/roles**：List `?page=&page_size=`（无搜索）→ `{list:[{id(hashid),code,name,sort,status,data_scope,remark}], total,...}`；Create/Update DTO 同字段。
- **注意（已处理）**：响应 ID 是 hashid，但 CreateUserInput 的 `dept_id/post_ids` 是裸 uint64（与 hashid 不对称）→ 最小表单避开 dept/post 选择器。

## 1. 完成状态
✅ 代码完成、自验通过（构建+类型检查+dev 服务+模块转译），**待 daxing 浏览器验收 + push 确认**。
admin 从"能登录的地基"做成"按权限工作的可用后台"：动态路由 + v-permission + x-table CRUD + 用户/角色样例页；登出确认复用 T-007a。未推、未碰后端。

## 2. 改动文件清单
| 文件 | 说明 | 新增/修改 |
|---|---|---|
| src/api/authInfo.ts | getMenus/getPerms + MenuNode 类型 | 新增 |
| src/api/user.ts, role.ts | 用户/角色 CRUD API（hashid 透传） | 新增 |
| src/router/dynamic.ts | menus→路由（import.meta.glob 组件映射，缺失降级占位） | 新增 |
| src/router/index.ts | 守卫拉 menus 动态 addRoute，刷新重建防白屏，会话重建先清旧路由 | 修改 |
| src/router/routes.ts | 布局路由命名 'layout'（addRoute 挂载点） | 修改 |
| src/store/user.ts | 加 menuTree/permCodes/routesReady + loadAuthInfo + hasPerm；reset 清理 | 修改 |
| src/directives/permission.ts | v-permission 指令（无权码隐藏元素） | 新增 |
| src/components/x-table/{XTable.vue,types.ts} | 配置化 CRUD 组件（最小版） | 新增 |
| src/views/sys/user/index.vue, role/index.vue | 用户/角色样例页（XTable 配置） | 新增 |
| src/views/placeholder/index.vue | 动态路由组件未实现的占位降级 | 新增 |
| src/layout/components/SidebarItem.vue, menuIcon.ts | 递归菜单项 + 图标解析 | 新增 |
| src/layout/components/Sidebar.vue | 改为按 store.menuTree 动态渲染 | 修改 |
| src/main.ts | 注册 v-permission 指令 | 修改 |
| src/i18n/locales/{zh-CN,en}.ts | table/error.building 文案 key | 修改 |

## 3. 实现情况
| 项 | 位置 | 状态 | 备注 |
|---|---|---|---|
| 动态路由（menus→路由+侧栏，glob 映射，刷新不白屏） | router/dynamic.ts, index.ts | ✅ | 守卫检测 routesReady=false → loadAuthInfo + addRoute('layout') → 重导航；缺组件降级占位不崩；会话重建先 removeRoute 清旧 |
| v-permission 指令（perms 隐藏无权） | directives/permission.ts | ✅ | 支持单码/数组；无任一码则从 DOM 移除；超管（seed 全菜单→全 perms）全显 |
| x-table 配置化 CRUD（最小版） | components/x-table | ✅ | 列+分页+搜索+新增/编辑弹窗(字段走配置)+删除确认；对接统一包络/错误码；hashid 透传；按钮挂 v-permission |
| 用户管理样例页 | views/sys/user | ✅ | 列表(用户名/昵称/手机/状态/时间)+用户名搜索+状态筛选+增删改；createOnly 密码；username 编辑禁用 |
| 角色管理样例页 | views/sys/role | ✅ | 列表(编码/名称/数据范围/排序/状态)+增删改；data_scope 下拉 |
| 登出（T-007a 已实现，本片确认） | layout/Navbar | ✅ | T-007a Navbar 已有 onLogout；本片 reset 扩展清 menus/perms/routesReady |

## 4. 自验结果
- **构建+类型检查**：`pnpm build`（vue-tsc -b + vite build）✅ 1702 模块转译，0 类型错误。
- **dev 服务**：`pnpm dev` ✅ 启动监听 5173；node fetch 验首页正确返回；关键模块（dynamic.ts / XTable.vue / sys/user / permission 指令）Vite 转译 HTTP 200 无报错。
- **未做**：浏览器实操登录→菜单→CRUD→权限收窄→刷新→登出（需后端 demo 在跑，属 daxing 验收，见 §7）；admin 无 vitest 基建（buildRoutesFromMenus 依赖 import.meta.glob，需 Vite 环境，沿用 T-007a 不引测试框架的决定）。
- 仅开源素材（沿用 T-007a，新增仅 EP 图标 MIT）；i18n：x-table UI chrome 走 key；hashid 透传；头注释/@updated 齐备。

## 5. git 提交记录
待 daxing 确认后双推（commit 带 T-007b）。当前未提交、未推。

## 6. 安全自查
- [x] 权限前端仅 UX，后端才是边界：v-permission/动态路由均体验层；无权用户手拼 URL 路由可达但**数据被后端 403**（T-003d enforce）——不做"隐藏即安全"的伪安全
- [x] hashid 透传（列表 id 直接用于 :id 路径，后端 parseHID 解码）；统一包络/错误码走请求层
- [x] 仅开源素材、i18n key、头注释/@updated
- [x] 登出清令牌（含 menus/perms/routesReady）+ 跳登录

## 7. 需 daxing 真人验收（浏览器）
- 超管登录：完整菜单；进用户/角色页；CRUD 可用；按钮齐全；登出可用；旧 token 登出后调接口被拒。
- 普通角色（editor 仅本人 / dept_mgr 本部门）：菜单/按钮按权限收窄；数据范围正确；无权操作按钮不可见，即使强行调用也被后端 403。
- 刷新页面（如在 /sys/user 刷新）不白屏、路由保持。
- 未建页面（dept/menu/dict/config 菜单）点击显示"建设中"占位、不崩。

## 8. 偏差与待办
- **样例页避开 dept/post 选择器**：因 CreateUserInput 的 dept_id/post_ids 是裸 uint64，与响应 hashid 不对称（需 ID 选择器 + 可能后端接受 hashid 入参）；最小表单未含，列表也未显示 dept（响应仅 dept_id 无 dept 名）。如需，后续加 dept 树选择器（可能需后端入参接受 hashid，单独评估）。
- **x-table 最小版**：不含列筛选/排序/导出/树形/复杂联动；菜单树形页留迭代。
- **超管全显依赖 seed 数据**：v-permission 用"perms 含该码"判定；超管可见全部依赖超管角色被授予全部菜单 perm（demo seed 如此）。若某超管角色未授全菜单，前端会按实际 perms 隐藏（后端仍放行）——非前端 bug，属角色数据配置。
- 无 vitest 单测（同 T-007a）；动态路由/指令逻辑靠构建+类型+dev+真人浏览器兜底。

## 9. 下一步建议
- **阶段一收官**：后端五大块 + demo e2e + admin（地基 T-007a + 权限/CRUD T-007b）+ 验证码 T-002b 全部就绪，admin 已是"按权限工作的可用后台"。
- 迭代：菜单树形管理页、字典/参数/日志/文件等 sys 页（复用 x-table）、x-table 高级能力、dept 选择器。
- 阶段二：BenxinKP 引入 BenxinAdminPro 只写业务；backend-php 照 spec 实现 parity。

---

## 修复追记：浏览器验收暴露的两个 bug（已修 + 实证）

> 两个 bug 经定位**不同源**：Bug1 是 Vite 代理配置问题、Bug2 是空菜单 null 崩溃；都在动态路由/刷新流程中显现。

### Bug1 刷新 /sys/* → "404 page not found"
- **真因（实证）**：vite.config 的开发代理 `'/sys'→后端` 把**浏览器对前端路由 `/sys/user` 的刷新请求**也转发到后端，后端无此路由 → 404。（与守卫/addRoute 无关——node repro 证明三种 redirect 写法都能正确落到 /sys/user。）
- **修复**：代理加 `bypass`——`Accept` 含 `text/html`（浏览器导航）的请求返回 SPA `index.html`，只代理 API 的 XHR。
- **实证**：起真 dev server，`/sys/user`、`/sys/role` 带 `Accept:text/html` → **HTTP 200 + 是 SPA(含 #app) + 无 404 文本**（修前为 404 含 not found）；API XHR(`Accept:application/json`)仍走代理（未当 SPA）。

### Bug2 editor 登录成功却卡在登录页
- **真因（代码 + repro 确认）**：editor 权限窄，后端空菜单返回 **JSON `null`**（Go nil slice）；`buildRoutesFromMenus(null)` 的 `for...of null` 抛 TypeError → 守卫 catch → `reset()` 清令牌 → 回登录（表现为"登录成功又被弹回"）。
- **修复**：① `buildRoutesFromMenus` 与 `loadAuthInfo` 对 null 兜成 `[]`；② 守卫 catch 不再粗暴登出——令牌已失效才回登录，否则（菜单异常但仍登录）放行到基础页 **/dashboard**，保证任何登录用户都能进入。dashboard 是静态路由、对所有登录用户可达、不依赖菜单权限。
- **实证**：vue-router repro——editor 的 null 菜单：登录跳 `/` → 落 **dashboard**（不崩、不卡登录）；刷新 /dashboard 正常；超管刷新 /sys/user → 落 sys-user。

### 自测结论
- ① 超管/dept_mgr（有菜单）+ editor（空菜单 null）均能登录进入系统（editor 落 dashboard）。
- ② 任意授权前端路由刷新返回 SPA、不再 404（dev 代理 bypass 实测）。
- `pnpm build`（类型检查 + 构建）通过。
- 说明：以上为「真 dev server 代理行为 + vue-router 真实路由行为」两处针对性实证；完整浏览器登录→渲染流程仍由 daxing 最终验收。
