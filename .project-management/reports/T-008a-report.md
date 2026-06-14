# 完成报告：T-008a 用户改密码 + status 假能力修复

> 状态：**待 PM 评审 + daxing 真人验收**。未 commit、未双推、未改 PROJECT_STATUS（T-006 铁律）。
> **分层说明**：本片纯前端，叠在尚未双推的 T-005b-4 工作区之上；只碰 `user/index.vue` + `api/user.ts`，与 T-005b-4 文件零重叠，diff 可干净区分。

## 0. 第 0 节源码核实（带出处）

| # | 核实项 | 结论（出处） |
|---|---|---|
| 0-1 | 改密码端点 | `PUT /sys/users/:id/password`，perm `sys:user:password`（`handler_user.go:65` + 常量 `:30`）。入参 `{ password }` **仅 `binding:"required"`（非空），无长度/强度校验**（`handler_user.go:277` + service `ResetPassword` 直接 `hasher.Hash` 落库 `user_service.go`）。→ 前端预校验只能做「非空 + 二次确认」，**不伪造后端不存在的强度规则**。 |
| 0-2 | status 端点 | `PUT /sys/users/:id/status`，perm `sys:user:status`（`handler_user.go:66` + 常量 `:31`）。入参 `{ status: int8 }`（`handler_user.go:296`），service 无取值校验（`user_service.go SetStatus`）。**status 语义：0=正常(启用)，非0=停用**（`model.go:45` 注释 `0=正常 1=禁用`；auth `service_auth.go:136` `Status != 0 → ErrAccountDisabled` 停用拒登）。 |
| 0-3 | updateUserReq 现状 | **确认无 Status 字段**——仅 Nickname/Avatar/Email/Mobile/DeptID/Remark/PostIDs（`handler_user.go:312` updateUserReq struct）。→ 编辑弹窗改 status 提交被全量覆写**静默吞**（T-007h §8-3 假能力根因坐实）。 |
| 0-4 | 用户页现状 | `user/index.vue`：status 同时在 `search`(行46)、`columns`(行60 仅展示)、**`fields`(行71-80 编辑/新增表单，无 createOnly → 编辑时显示=假能力)**；无重置密码入口、无状态独立控件。 |
| 0-5 | 密码控件范式 | 登录页 `el-input type="password" show-password`（`views/login/index.vue:147-150`）；XTable 表单 password 字段亦 `show-password`（`XTable.vue:460`）。本片复用同范式（el-input password + show-password + autocomplete="new-password"）。 |

## 1. 完成状态
✅ 已完成并真跑验证。用户页加「重置密码」行操作（页级弹窗调既有 `PUT :id/password`）；status 假能力根除（编辑表单去 status，状态变更改走行操作切换调既有 `PUT :id/status`）。零后端 Go 改动、openapi 不升版、零新增错误码。

## 2. 改动文件清单
| 文件 | 说明 | 类型 |
|---|---|---|
| `admin/src/api/user.ts` | +`resetUserPassword(id,password)`（PUT :id/password）+`setUserStatus(id,status)`（PUT :id/status） | 修改 |
| `admin/src/views/sys/user/index.vue` | 重置密码弹窗（页级）+ 状态切换行操作 + status 字段标 `createOnly`（编辑不显示）+ 头注释 | 修改 |

## 3. 接口契约
消费既有端点，**不新增/不升版**：`PUT /sys/users/:id/password {password}`、`PUT /sys/users/:id/status {status}`。hashid 透传不解码。

## 4. 自验结果

**构建/测试**：`pnpm build`（vue-tsc + vite）exit 0 ✓；`pnpm test`（vitest）**17 passed**（不破坏既有基建）✓。

**enforce 正向证据（curl 真跑，dept_mgr 全量授权含两码 / editor 仅 sys:user:list）**：
| 动作 | editor | dept_mgr |
|---|---|---|
| PUT :id/password | **403** | **200** |
| PUT :id/status（真值变更 0→1/1→0） | **403** | **200 / 200** |

（超管 200 不计为证据，T-007e 口径；dept_mgr 非超管/不短路/走真 Casbin。）

**功能真跑（curl）**：
- **改密码**：探针用户改密后 → 旧密码登录 **401**、新密码登录 **200** ✓
- **status**：停用(status=1)后该用户登录 **403**（账号停用拒登）、启用(status=0)后登录 **200** ✓

**假能力真除证据**：
- 后端 `updateUserReq` 无 Status 字段（0-3 坐实，`handler_user.go:312`）→ 编辑全量覆写本就不带 status。
- 前端 status 字段标 `createOnly: true`（`user/index.vue`）→ XTable `fields.filter(f => !(mode==='edit' && f.createOnly))`（`XTable.vue:132`）→ **编辑弹窗不再渲染 status 字段**，假 UI 入口物理消除。新增态仍可设初始状态（CreateUserInput 支持 Status，`user_service.go:34`）。
- 状态变更唯一路径 = 行操作「停用/启用」→ 调独立端点 `PUT :id/status`。

## 5. git 提交记录
**未提交**。等 PM 评审 + daxing 真人验收 + 放行后双推。

## 6. 安全自查（逐项）
- **改密码**：新密码不回显既有（后端也不返）、`destroy-on-close` + 提交成功即清空 `pwdForm`（不缓存明文）；二次确认一致才提交；`autocomplete="new-password"` 防浏览器回填；**前端预校验仅 UX（非空+一致），不伪造后端不存在的强度规则**（0-1）。
- **status 开关**：行操作挂 `v-permission="sys:user:status"`、重置密码挂 `sys:user:password`——无权码隐藏按钮；editor 实测两端点 **403**（前端隐藏仅 UX、后端 enforce 是权威）。
- **假能力根除**：编辑弹窗 status 物理移除（createOnly），不再「能改但静默吞」。
- **hashid 透传不解码**：`String(row.id)` 原样回传；伪造 id 后端统一 400。
- 失败兜底：改密/切换失败请求层已 toast，页面 catch 吞 rejection 保留弹窗供修正、不写脏行（状态切换失败不更新视觉态，靠 `refresh` 重拉真值）。

## 7. 需 daxing 真人验收
> ⚠️ **本片纯前端、Vite 热更新，无需重启 demo**（区别于后端 Go 片）。
- 用户页某行点「重置密码」→ 输新密码 + 确认 → 用新密码能登录、旧密码失效。
- 用户页某行点「停用/启用」→ 二次确认 → 停用后该用户登录被拒。
- 编辑某用户 → 弹窗内**已无 status 字段**（不再误导「改了却不生效」）。
- 二次确认不一致 → 前端 toast 拦截、不提交。

## 8. 偏差与待办
- **偏差①（§2 sanctioned，非 scope 蔓延）**：§2 原意「状态列内 el-switch」**未采用**，改为**行操作「停用/启用」按钮（动态二次确认）**。理由：① 列内单元格放 el-switch 需给 x-table 加单元格插槽 = **改 x-table 核心**，§2 明令禁止；② 覆写 `#row-actions` 插槽会丢内置编辑/删除（插槽仅暴露 `row`、不暴露内置 openEdit/onDelete）。→ 按 **§2 显式回退条款「走行操作插槽」**，用 `config.actions` 行操作按钮实现，与内置编辑/删除并列、零 x-table 核心改动。功能等价（切换+二次确认+刷新），UX 为按钮非开关。**请 PM 确认此落点。**
- **观察（pre-existing 后端 quirk，非本片引入，前端已天然规避）**：`SetStatus` 把 status 设为**当前相同值**时 GORM `RowsAffected==0` → handler 返 `ErrUserNotFound`(404)，语义误导（实为「无变更」非「用户不存在」）。属 T-003 既有行为。**本片前端 `toggleStatus` 永远翻转值（0↔1）→ 必为真变更 → 不会触发此 404**，故不影响本片功能；建议记低优待办（后端可改为 no-op 也返 200，或区分 not-found 与 no-change）。
- **待办（归后续 T-008b/c）**：用户分角色、角色授权树（摸底已坐实后端 AssignRoles/AssignMenus 在、但回填查询端点缺，需各片搭车补后端）。

## 9. 下一步建议
- PM 评审 diff + 报告 → daxing 浏览器验收（重置密码 / 状态切换 / 编辑弹窗无 status）→ 放行后双推。
- 之后接 **T-008b（用户分角色，含后端回填查询小补）**，再 **T-008c（角色授权树，最重）**。
