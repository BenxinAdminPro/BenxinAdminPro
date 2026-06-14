# 完成报告：T-007c x-table 增强（只读模式 + 工具栏/行操作插槽 + 列筛选/排序）

> 状态声明：**新能力 harness 目视通过，端到端待 T-007d 起首个消费方页落地时回证**（见任务书第 8 节测试铁律）。本片不据此自标前端迭代整块完成。完成判定权在 PM。

## 1. 完成状态
执行端自测完成，待 PM 评审 + daxing 浏览器验收（既有页零回归）。纯前端（admin/），未碰后端、未改契约（仍 openapi v0.8.1 消费既有接口）。

## 2. 改动文件清单
| 路径 | 说明 | 类型 |
|---|---|---|
| `admin/src/components/x-table/types.ts` | 配置 schema 扩展：XColumn 加 `sortable`/`filterable`；新增 `XAction`；XApi 的 create/update/remove 改可选（只读表只需 list）；XTableConfig 加 `readonly`/`actions`/`actionsWidth`，`fields` 改可选 | 修改 |
| `admin/src/components/x-table/XTable.vue` | 主组件加厚：只读门控、工具栏插槽、行操作（配置 actions + `row-actions` 插槽）、列筛选（表头 popover）、服务端列排序，查询参数合并（搜索+筛选+排序→一个 query） | 修改 |
| `admin/src/directives/permission.ts` | 语义增强：绑定值为空（undefined/''/空数组）= 不限制、始终显示（供行操作不挂权限码）。既有调用均传真码，零影响 | 修改 |
| `admin/src/router/routes.ts` | 挂 harness 静态路由 `/dev/xtable-harness`（布局子路由，不入业务菜单） | 修改 |
| `admin/src/views/dev/xtable-harness/index.vue` | x-table 增强 harness 目视验证页（内存 mock api，真按 query 过滤/排序/分页 + 回显最近 query） | 新增 |

## 3. x-table 最终配置 schema（本片重点交付）

### 表级（XTableConfig）
```ts
{
  rowKey?: string                 // 默认 'id'（hashid 字符串透传）
  columns: XColumn[]
  fields?: XField[]               // 只读表可省略
  search?: XSearchField[]
  api: XApi                       // list 必需；create/update/remove 可选
  permPrefix?: string             // 'sys:user' → 内置增改删按钮挂 permPrefix:create/update/delete
  readonly?: boolean              // 关闭内置增改删，仅留 列表+分页+搜索+自定义行操作
  actions?: XAction[]             // 自定义行操作（与内置并存；只读下为唯一行操作来源）
  actionsWidth?: number | string  // 操作列宽，默认 160
}
```

### 列级（XColumn）
```ts
{ prop, label, width?, minWidth?, formatter?,
  sortable?: boolean,   // el-table sortable='custom'，排序经 query sort=<prop>&order=asc|desc 透传后端（服务端排序）
  filterable?: boolean  // 表头筛选入口，输入值经 query <prop>=<value> 并入请求透传后端
}
```

### 行操作（XAction）
```ts
{ label,
  perm?: string | string[],  // v-permission 码；缺省=不挂码、始终显示
  type?: 'primary'|'success'|'warning'|'danger'|'info',
  icon?: Component,
  confirm?: string,          // 提供则点击先弹二次确认（危险操作）
  handler: (row) => void | Promise<void>,
  refresh?: boolean          // handler 完成后自动刷新列表
}
```
另提供 `#toolbar` 插槽（表上方自定义按钮区）与 `#row-actions`（slotProps `{ row }`，完全自定义行操作，优先于配置）。

### 查询参数装配
`fetchData` 把 **分页 + 搜索栏 + 列筛选 + 列排序** 合并为单一 query 对象传 `api.list`：
`{ page, page_size, ...搜索非空项, ...列筛选非空项, sort?, order? }`。空值不下发。

## 4. 自验结果
- `npm run build`（`vue-tsc -b && vite build`）：**0 错误，✓ built**（产物含 xtable-harness chunk）。
- `npx vue-tsc -b --force`：**TYPECHECK_EXIT=0**。
- 唯一构建告警来自第三方 `node_modules/@vueuse/core` 的 `/* #__PURE__ */` 注释位置（INVALID_ANNOTATION），与本片改动无关、为既有噪声、非错误。
- 本片无 vitest 基建（T-007i 才引），按任务书以「既有页零回归 + harness 目视 + 构建/类型」兜底。

## 5. 既有用户/角色页零回归证据
两页 config 均未设 `readonly`/`actions`/`sortable`/`filterable`，命中缺省分支：
- `showAddBtn = !readonly(false) && permPrefix truthy = true` → 工具栏与新增按钮照旧（v-permission:create）。
- `showOpColumn = 无 row-actions 插槽 && actions=0 && !readonly = true` → 操作列照旧渲染内置编辑/删除（v-permission:update/delete）。
- 列无 filterable/sortable → 无表头筛选入口、`sortable=false`，表头与排序行为与 T-007b 完全一致。
- `fields` 仍提供 → 新增/编辑弹窗逻辑不变。
- 指令增强仅对「空绑定值」放行；两页全部传真权限码，行为不变。
→ 构建/类型 0 错误 + 上述逐分支推证；**目视复核留 daxing 浏览器验收**。

## 6. 接口实现情况
- 不改后端契约；消费既有 `{code,message,data}` 统一包络与 `{list,total,page,page_size}` 分页。
- 筛选/排序参数命名：列筛选用列 `prop` 作 query key；排序固定用 `sort`(字段)+`order`(asc/desc)。
- **后端现状核实（grep server/ handler）**：list 接口仅认 `page/page_size` 及少量过滤参数（`username`/`status`/`dict_type`/`operator`/`uploader`），**无任何排序参数**。故：排序对当前后端为「降级不可用」（参数下发但被忽略，前端绝不内存伪排序）；列筛选仅对后端已识别字段生效，未识别字段同样被忽略。该约定待 T-007d/e/f 消费方按各自后端能力对齐启用——**本片只提供能力开关，不臆造后端参数**（见第 8 节偏差）。

## 7. 安全自查
- 行操作/工具栏按钮一律可挂 `v-permission`，**前端仅 UX、后端 enforce（T-003d）才是边界**——无权用户手拼仍被 403，不做伪安全。
- 筛选/排序参数前端拼装、后端为过滤/排序权威；前端不假设可信、不内存伪装全量。
- hashid 字符串 ID 原样透传不解码（沿用 T-007a 请求层约定）。
- 不打印/缓存敏感字段；沿用 Vue 转义 XSS 缓解。
- harness 为内存 mock，无真实数据、不触后端、不入业务菜单。

## 8. 偏差与待办
- **偏差1（已声明）**：服务端排序对当前后端降级不可用（后端无排序参数）。`sortable` 开关已就绪，待后端按需补排序参数后即生效；不在本片改后端。
- **偏差2（主动设计决策，请 PM 认可）**：v-permission 指令语义增强为「空绑定值=不限制」。动机：行操作允许不挂权限码（如"详情"查看类）。风险：可能掩盖"忘记传码"的笔误；既有调用均传真码，零回归。
- **偏差3（请 PM 认可）**：XApi 的 create/update/remove 由必需改可选 + XTableConfig.fields 改可选，以支持只读表（如日志页只提供 list）。既有页全字段齐备，不受影响。
- 待办：harness 路由 `/dev/xtable-harness` 为开发目视页，随前端迭代收尾可评估是否移除/gate（当前无害：内存 mock、不入菜单）。
- 列拖拽/列显隐持久化/导出/树形按任务书**明确不做**（树形→T-007g，导出 defer）。

## 9. 需 daxing 真人验收（浏览器）
1. **既有页零回归（硬验收）**：用户/角色页 CRUD、搜索、分页、权限按钮、刷新不白屏 —— 与 T-007b 完全一致。
2. harness 页 `/dev/xtable-harness`（登录后地址栏直达）目视：
   - 只读模式：无内置新增/编辑/删除，仅列表+分页+搜索+自定义行操作。
   - 工具栏插槽：表上方"自定义工具栏按钮"渲染正常。
   - 行操作：超管见"详情"+"清理"；"清理"危险二次确认；"详情"无权码始终显示。
   - 列筛选（名称/分类表头漏斗）+ 列排序（排序值/创建时间表头箭头）→ 下方"最近 query"实时显示条件已并入（sort/order/筛选 key）。
3. （注）只读/工具栏/筛选排序的**真实端到端验证随 T-007d/e/f 首个消费方页回证**。

## 10. 下一步建议
PM 评审 + daxing 浏览器验收（既有页零回归）放行后，执行端贴 `git status + diff` 待核 → 双推（Gitee 主 + GitHub 镜像）。随后进 T-007d（字典/参数页，标准 CRUD），作为 x-table 增强能力的首个端到端回证消费方。
