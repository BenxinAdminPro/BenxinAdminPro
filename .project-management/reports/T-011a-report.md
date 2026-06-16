# 完成报告：T-011a x-table 多选基建（selectable gated 扩展）

> 性质：纯前端、零后端 Go / 零 openapi / 零 DDL / 零错误码·权限码。
> HEAD 基线 131e870 · openapi v0.14.1 不升版。
> **未自标完成、未双推、未改 PROJECT_STATUS.md（PM 权限）** —— 待 PM 评审 + daxing 验收 + PM 放行后双推。

## 1. 完成状态
✅ 编码 + 自测完成，待评审/验收。给 XTable 加 `config.selectable` 开关 gated 的多选能力：选择列 + 选中态暴露（`selectedRows` / `clearSelection`）+ `#batch-actions` 作用域插槽。缺省 `selectable` 未设 → 选择列不渲染、批量槽不启用 = 既有 8 消费页零回归（结构性保证：全部走 `v-if="selectable"`）。本片不接任何消费页（基建片），多选消费方真验证留 T-011c。

## 2. 改动文件清单（3 文件，+85/-6）
| 文件 | 说明 | 类型 |
|---|---|---|
| `admin/src/components/x-table/types.ts` | `XTableConfig` 加 `selectable?: boolean`（含语义/零回归/页内多选/不伪安全注释）；头追加 @updated | 修改 |
| `admin/src/components/x-table/XTable.vue` | `selectable` computed + `elTableRef`/`selectedRows`/`onSelectionChange`/`clearSelection`；`fetchData` 数据落表时同步清选中；`showBatchActions`+并入 `showToolbar`；`defineExpose` 加 `selectedRows`/`clearSelection`；模板加 `type=selection` 列（`v-if=selectable`）+ `@selection-change` + el-table `ref` + `#batch-actions` 槽；头追加 @updated | 修改 |
| `admin/src/views/dev/xtable-harness/index.vue` | 加 `selectableConfig`（selectable=true）+ `onBatchDemo`；模板加「多选」卡（`#batch-actions` 读 selected/clear + 选中数 + 清空），既有「只读增强」卡作 selectable=off 对照；alert/header 文案 + @updated | 修改 |

> `git status` 另含 `.project-management/PROJECT_STATUS.md`（M）—— **会话起始即已修改，非本片产出，未触碰**（PM 账本）。

## 3. 接口实现情况
- 无后端接口改动，openapi 不升版。
- 前端契约新增（XTable 对消费方）：
  - `XTableConfig.selectable?: boolean`（缺省 false）。
  - `#batch-actions` 作用域插槽，slot props `{ selected: XRow[], clear: () => void }`，仅 selectable 时渲染于工具栏区。
  - `defineExpose`：`selectedRows`（当前选中行数组）、`clearSelection()`（清空选中，同步 el-table 内部勾选 + 内部态）；既有 `refresh` 不变。
- 页内多选语义（v1）：翻页/筛选/刷新清空选中（`fetchData` 落表时 `selectedRows.value=[]` + el-table 默认无 reserve-selection）。reserve-selection / 跨页保留留作将来 gated flag（本片不做，符合任务书 §2 不包含）。

## 4. 自验结果
| 项 | 命令 | 结果 |
|---|---|---|
| 构建 | `pnpm build`（vue-tsc -b && vite build） | ✅ exit 0，✓ built in 748ms（warning 均为 @vueuse/core PURE 注解 + chunk size，pre-existing 与本片无关） |
| 类型 | `pnpm typecheck`（vue-tsc -b --noEmit） | ✅ 无输出 = 净 |
| 单测回归 | `pnpm test`（vitest run） | ✅ 1 file / 17 passed（既有 tree.spec.ts 未动） |
| 变更范围 | `git diff --stat` | ✅ 仅 3 个 admin 文件，+85/-6 |

**关于「缺省零回归 vitest 守」（任务书 §7 第二项）—— 经评估不引 jsdom，理由如下（任务书明确允许，须说明）**：
- 本片新增逻辑极薄：选择列 `v-if="selectable"`、`@selection-change` 透传、`clearSelection` 调 el-table ref 方法、`defineExpose` 暴露 ref。零回归是**结构性保证**（全部 gated on `selectable`），非分支逻辑可被纯函数单测覆盖。
- el-table 的 `selection-change` 由内部勾选框 DOM 交互驱动，jsdom 下模拟脆弱且主要在验证 el-table 内部而非本片 gating 逻辑；引 `@vue/test-utils + jsdom` 需为单片铺组件测试基建（全局注册 ElTable/指令/i18n 等），与 T-007i 既定 scope（"未来若测组件再按需引 jsdom"、最小依赖）相悖、性价比不足。
- 故零回归改由 **harness（selectable on + off 对照）+ typecheck/build 自动闸门 + daxing 逐页视觉**兜底（与 T-007h field-slot 等 optional-config 扩展同范式：缺省渲染分支不变即零回归）。devDependencies 未新增。

## 5. git 提交记录
- **未提交、未双推**（待 PM 放行）。建议 commit message：`feat(admin): x-table selectable 多选基建（gated 选择列 + 选中态暴露 + 批量槽）`。
- 双推前置：PM 评审报告 + daxing 验收 + PM 放行 → 推前查 .gitignore（仅 3 个 .vue/.ts 源文件，无密钥/产物）→ Gitee origin + GitHub github 双推 → ls-remote 三方一致。

## 6. 安全自查
- 多选纯前端选中能力，**不放宽任何权限**：批量操作的鉴权由后端端点把关（T-011b），本片无任何前端伪批量/伪安全（harness 批量按钮仅 ElMessage 演示）。
- `row-key` 沿用既有 hashid id 透传，未新增/未改 row-key，不解码、不暴露内部 uint64。
- 选中态纯内存（`selectedRows` ref），无持久化、无密钥、无敏感数据；`clearSelection` 不触网。
- 零新增端点 / 错误码 / 权限码 / DDL。

## 7. 需 daxing 真人验收（demo 验证项）
1. **harness `selectable=on`（多选卡）**：勾选若干行 → 「批量操作（已选 N）」按钮 N 实时变化、点击拿到 selected 行（toast 列出名称）→ 「清空选中」点击后选中清零、勾选框全消；**翻页/刷新后选中自动清空**（页内语义如实）。
2. **harness `selectable=off`（只读增强卡，对照）**：无 48px 选择列、无横向布局位移、工具栏/行操作/筛选/排序逐字现状。
3. **逐页扫验 8 个消费页**（user / role / dict / config / operlog / loginlog / file / post）：均未设 selectable → 无多出选择列、布局无位移、`#toolbar`/`#row-actions`/增删改查行为逐字现状（缺省零回归命门）。

## 8. 偏差与待办
- **偏差①（任务书明确允许）**：未引 jsdom 组件测试，以 harness + 视觉兜底，理由见 §4。如 PM 坚持要自动化组件守，可后续单独评估引 `@vue/test-utils + jsdom`（建议作为独立测试基建增量，非塞本片）。
- 无其他偏差；§2 不包含项（cell-slot / reserve-selection / 跨页保留 / 后端 / 媒体页）均未碰。

## 9. 下一步建议
- T-011b（后端能力片）：sys_file mime 大类前缀筛 + 批量软删端点（部分失败语义/事务/权限码待 PM 裁），与本片无依赖、可并行。
- T-011c（媒体管理消费片）：消费本片 `selectable` + `#batch-actions` + `defineExpose(selectedRows/clearSelection)` 接批量删，叠加 T-011b mime 筛与媒体预览。
- 本片暴露的 x-table 后续可选增强（非本片）：cell-slot `#cell-<prop>`（清账本 4 次「列内自定义单元格控件」诉求前 3 个）、reserve-selection 跨页保留——均建议作独立 gated 扩展评估。
</content>
</invoke>
