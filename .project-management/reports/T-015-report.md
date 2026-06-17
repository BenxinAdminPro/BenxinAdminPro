# 完成报告：T-015 展示格式化工具收敛（dateText + statusText 抽 utils/format）

## 1. 完成状态
执行端编码 + 自测**全部完成，等 PM 评审 + daxing 零回归验收 + PM 放行后双推**。纯前端 refactor，openapi 不变、零后端。

## 2. 改动文件清单（11 文件：2 新建 + 9 页 vue）
> 注：PM 评审后追加收编 menu（见偏差②），由 10 文件→11 文件。menu 行见下表末。

| 路径 | 说明 | 类型 |
|---|---|---|
| `admin/src/utils/format.ts` | 新建：导出 `dateText`/`statusText` 纯函数（逐字搬现有实现）+ 五项头注释 | **新增** |
| `admin/src/utils/format.spec.ts` | 新建：镜像 tree.spec 范式，锁 dateText/statusText 边界（8 断言） | **新增** |
| `admin/src/views/sys/user/index.vue` | 删本地 dateText+statusText const → import | 修改 |
| `admin/src/views/sys/dept/index.vue` | 删 dateText const → import；**内联 status 模板 `row.status===0?...` → `statusText(row.status)`** | 修改 |
| `admin/src/views/sys/post/index.vue` | 删 dateText+statusText const → import | 修改 |
| `admin/src/views/sys/dict/index.vue` | 删 dateText+statusText const → import | 修改 |
| `admin/src/views/sys/role/index.vue` | 删 statusText const → import（**任务书漏列，见偏差①**） | 修改 |
| `admin/src/views/sys/file/index.vue` | 删 dateText const → import | 修改 |
| `admin/src/views/sys/operlog/index.vue` | 删 dateText const → import | 修改 |
| `admin/src/views/sys/loginlog/index.vue` | 删 dateText const → import | 修改 |
| `admin/src/views/sys/menu/index.vue` | **内联 status `row.status===0?...` → `statusText(row.status)`** + import statusText（PM 评审后追加收编，偏差②） | 修改 |

> 精确 stage 范围 = 上述 **11 文件**。`.project-management/PROJECT_STATUS.md` 当前工作树有改动，但**是 PM 填实 T-014 占位符的 KB 步骤、非本片，不入 stage**。无 config.local.yaml 混入。

## 3. 实现情况
- `format.ts`：`dateText`（ISO→YYYY-MM-DD HH:mm:ss，非字符串→空串）+ `statusText`（Number(v)===0?正常:停用），逐字搬自各页现有实现，纯函数无副作用，五项头注释到秒。
- 各页改为 `import { ... } from '@/utils/format'`（镜像 `@/utils/tree` 引法，并入 import 块），删本地 const。
- dept 状态列由内联三元表达式改为 `statusText(row.status)`（dept 在任务书 statusText 列表内）。
- `format.spec.ts`：dateText（典型 ISO / '' / null / undefined+数字）+ statusText（0→正常 / 1→停用 / 非0→停用 / 字符串'0'→正常）共 8 断言。

## 4. 自验结果
- ✅ **抽 statusText 前逐页核语义**：user/dict/post/role/dept 的 status 均 **0=正常 / 非0=停用**（与各页 el-radio 选项 `value:0=正常 / value:1=停用` 一致），全部统一无语义差异。loginlog 用的是 `success`（1=成功/0=失败）**语义不同→正确未纳入 statusText**（仍各自实现）；operlog 的 `resultText`（0=成功）亦不同→未纳入。
- ✅ **grep 无残留本地 const**：`grep -rn "const dateText\|const statusText" src/views/sys` 空；8 页均 `from '@/utils/format'`。
- ✅ `pnpm build` exit 0，类型干净。
- ✅ `pnpm test`：**2 文件 25 PASS**（tree.spec 既有 17 + format.spec 新增 8）。

## 5. git 提交记录
**尚未提交、尚未双推**（完成判定/双推/改 PROJECT_STATUS 权限仅 PM）。
- 待 stage（精确 10 文件，禁 `git add -A`）：见 §2。
- 拟提交信息：`refactor(admin): T-015 展示格式化工具收敛（dateText+statusText 抽 utils/format）`
- 双仓 Gitee 主 + GitHub 镜像。

## 6. 安全自查
- 纯展示层 refactor，逐字搬迁实现、行为零变化；无新出参、无鉴权/请求改动；不碰后端/openapi/DB。

## 7. 需 daxing 真人验收（纯前端热更，无需重启 demo）
- [ ] 扫一眼各页：user/dept/dict/post/file/operlog/loginlog 的**创建时间列**仍 `YYYY-MM-DD HH:mm:ss`；user/dept/dict/post/**role** 的**状态列**仍「正常/停用」，无变空/乱码/Invalid Date。
- [ ] 顺带 menu 页状态列仍「正常/停用」（本片未改其内联实现，应原样）。

## 8. 偏差与待办
- **偏差①（任务书页面枚举与源码不符·source-first 修正）**：任务书写「statusText 4 页：user/dept/dict/post，净改 7 页/9 文件」。source-first 核实——`const statusText` 实际在 **user/dict/post/role** 四页（任务书把 **role 误记成 dept**），而 **dept 用的是内联表达式**（非 const）。
  - 自测铁律「grep 无残留 const statusText」**强制收 role**（任务漏列）→ 故 statusText import 落在 user/dict/post/**role**；dept（任务列入）内联→statusText。
  - 净改 **8 页 vue + 2 新 = 10 文件**（非任务书估的 7 页/9 文件），多出的是 role。
- **偏差②（menu 内联 status·PM 评审后已收编）**：`menu/index.vue:234` 有与 dept 逐字相同的内联 `row.status===0?'正常':'停用'`（非 const，初版为守 scope 留置）。**PM 评审通过后指示顺手收编**（守同主题、statusText 收敛无遗漏）→ 改为 `statusText(row.status)` + import，menu 纳入 stage。故本片由 10 文件→**11 文件**（9 vue + 2 新）。全站复证：无内联 `status === 0 ? '正常'` 残留、无 `const dateText/statusText` 残留、9 页 import format。
- 未碰债池其余两条（enc==nil ancestors struct tag 彻底化 / m4a 上传失败待定性）。

## 9. 下一步建议
- PM 评审（确认偏差① role 纳入合理、偏差② menu 留置/收编决策）+ daxing 各页时间/状态列零回归验收通过后放行双推（10 文件精确 stage）。
- menu 内联 status 可作为本批尾巴搭车收编，或留小片。
