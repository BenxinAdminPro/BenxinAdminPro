# 完成报告：T-011c 媒体管理消费片

> 状态口径：执行端编码 + 自测完成，**待 PM 评审 → daxing 浏览器验收 → PM 放行 → 双推**。完成判定/双推/改 PROJECT_STATUS 权限仅 PM（T-006），本报告不自标完成、未自行双推。

## 0. 接手前置确认回答
- **T-011a 报告归档 HEAD 确认：`5858470` 属实**。`git log --oneline 5a79b47..5858470` 实测两 commit：`a5224bb`（T-011a 账本，含自引用占位）→ `5858470`（报告归档）；`git show --stat 5858470` = 单 commit 仅 `+ T-011a-report.md`（64 行，1 file changed）= 纯报告归档 commit。链路 `5a79b47`(feature) → `a5224bb`(账本) → `5858470`(报告归档=最终 HEAD) 与账本推定逐字吻合。
- **T-011b 账本/报告归档两 hash：暂无法回报（尚未提交）**。当前工作树仍为 `M .project-management/PROJECT_STATUS.md`（PM 的 T-011b 账本定稿）+ `?? .project-management/reports/T-011b-report.md`（未跟踪），PM 尚未给落账放行指令，执行端**未自行提交/双推**（守 T-006）。待 PM 放行后我将：① 独立账本 commit（精确 stage `PROJECT_STATUS.md`，禁 `git add -A`）② 报告归档 commit（精确 stage `T-011b-report.md`，= 最终 HEAD）→ 双推 → 回报两真实 hash 供 PM 知识库填实。**T-011c 账本同此约定。**

## 1. 完成状态
编码 + 前端自测（typecheck/build/既有单测）+ 后端默认闸门 sanity 全绿，待评审与真人验收。XTable 核心零触碰（types.ts/XTable.vue 未改）。

## 2. 改动文件清单
| 文件 | 说明 | 新增/修改 |
|---|---|---|
| `admin/src/api/file.ts` | 加 `fetchFileBlob`（鉴权 blob 取流，预览/下载共用）；`downloadFile` 改为其语义别名（行为零回归）；加 `batchDeleteFiles(ids)`（POST /sys/files/batch-delete）；`listFiles` 注明 `mime_category` 透传 | 修改 |
| `admin/src/views/sys/file/index.vue` | 文件页升级为媒体管理页：mime 大类 tab（el-radio-group→`:key` 重挂复位）+ `selectable` 多选 + `#batch-actions` 批量删 + `#row-actions`（预览/下载/删除）+ 预览弹窗（el-image/video/audio）+ 降级 + object URL 三处吊销 + `mimeCategoryOf` 镜像后端 | 修改 |
| `server/examples/demo/config.example.yaml` | `allowed_exts` 加 mp4/webm/mov/mp3/wav/ogg/m4a；`max_size_bytes` 10MB→100MB（模板，进仓） | 修改 |
| `server/examples/demo/config.local.yaml` | 同上（daxing 实跑读取，被 `.gitignore` `*.local.yaml` 覆盖**不进仓**，仅本机生效；改后**需重启 demo**） | 修改（本地，不进仓） |
| `.project-management/reports/T-011c-report.md` | 本报告（策略 A 落文件） | 新增 |

## 3. 接口实现情况
| 接口 | 方法/路径 | 状态 | 备注 |
|---|---|---|---|
| 列表 | `GET /sys/files` | 消费 | 页级 `listApi` 包装并入 `mime_category=mimeTab||undefined`，与既有 search/sort/page 同级透传 |
| 批量软删 | `POST /sys/files/batch-delete` | 消费 | `batchDeleteFiles(ids)` → `{deleted_count}`；幂等；空/超 100/非法 hashid 由后端 400 |
| 下载/取流 | `GET /sys/files/:id/download` | 消费 | `fetchFileBlob` axios `responseType:'blob'` + JWT 拦截器；预览与下载共用 |

零后端 Go / 零 openapi（仍 v0.15.0）/ 零新错误码 / 零新权限码（复用 `sys:file:download`/`sys:file:delete`）。

## 4. 自验结果
- `pnpm typecheck`（vue-tsc -b --noEmit）：**净，0 error**。
- `pnpm build`：**exit 0**（@vueuse/core PURE 注解 + chunk size 为既有库告警，非本片）。
- `pnpm test`：**17 PASS**（tree.spec 未动；本片 jsdom 组件测试同 T-011a 豁免）。
- 后端 `go build ./...` / `go vet ./...` / `go test ./...`（默认闸门）：**全绿**（无 Go 改动，YAML 运行时 viper 读取不影响编译）。
- **吊销三处自检**：① 切换预览前 `revokeCurrent()`（openPreview 取流后）② el-dialog `@closed`→`onPreviewClosed`→`revokeCurrent()` ③ `onBeforeUnmount`→`revokeCurrent()`。另 `onDownload` 内 createObjectURL 即时 revoke（自包含无泄漏）。全部 createObjectURL 均有对应吊销。
- **mimeCategoryOf 对齐**：`image/`→image、`video/`→video、`audio/`→audio、其余→other，逐字镜像后端 `system/file_service.go` `mimeCategoryPrefix` 白名单 + `applyMimeCategory`（注释标镜像来源）；后端 LIKE 默认 collation 大小写不敏感 → 前端先 `toLowerCase` 保 parity。
- **批量删链路**：confirm（取消 catch return = no-op）→ `batchDeleteFiles` → 用真实 `deleted_count` toast → `clear()`(=clearSelection) → `tableRef.refresh()`。

## 5. git 提交记录
- **未提交、未双推**（待 PM 评审 + daxing 验收 + PM 放行）。当前工作树 4 项 tracked 改动（api/file.ts、file/index.vue、config.example.yaml、本报告）+ config.local.yaml 本地改动（不进仓）。
- 放行后双推 Gitee 主 + GitHub 镜像（精确 stage、撞 fake-IP 走 DoH 隧道、推后 CI 双闸门绿 + ls-remote 三方一致）。

## 6. 安全自查（对照任务书第 5 节）
- [x] 预览/降级/下载全走 axios JWT blob（`fetchFileBlob`），el-image/video/audio 喂 `createObjectURL` blob URL，**无裸 `<img/video src=后端URL>`**。
- [x] 预览/下载按钮挂 `v-permission="'sys:file:download'"`；批量删/单删挂 `'sys:file:delete'`（无码隐藏，T-007b 范式）。
- [x] `mime_category` 仅传 token（image/video/audio/other/空），后端常量前缀白名单映射，前端不拼 SQL。
- [x] 批量删二次确认（ElMessageBox.confirm，取消 no-op）+ 删后 `clear()` + `refresh()`，用真实 `deleted_count`（吃 T-007f「成功提示≠状态变更」教训）。

## 7. 需 daxing 真人验收（对照 §7）
- [ ] **（前置）** demo 改 `allowed_exts`/`max_size_bytes` 后**已重启**（`lsof -ti :8080 | xargs kill -9` → `cd server && go run ./examples/demo`）；上传 1 图 + 1 小视频(<50MB) + 1 音频 + 1 大视频(>50MB) + 1 txt。
- [ ] mime tab：全部/图片/视频/音频/其他各筛对；切 tab 复位第 1 页 + 选中清空；与搜索/排序/列筛共存（注：切 tab 经重挂会一并重置列筛/排序，见 §8 偏差②）。
- [ ] 图片预览：弹窗经 blob 显图（Network 见带 JWT 的 download 请求、非 401）；关闭正常。
- [ ] 视频/音频(<50MB)内联播放可放。
- [ ] 大视频(>50MB)降级：无内联播放器，显「下载查看」+ 下载成功。
- [ ] 批量删：选 2~3 行 → confirm（取消验 no-op）→ 确认 → toast 真实 deleted_count → 行真消失 → 选中清空；重删验幂等 count 诚实。
- [ ] other(txt)行无「预览」按钮，下载仍可用。
- [ ] 内存泄漏巡查：反复开关预览 + 切换不同文件，无 console 报错；（可选）DevTools 验 object URL 被吊销。
- [ ] enforce：editor（无 `sys:file:delete`）无批量删/单删按钮，强行调 403；预览/下载权限正确门控。

## 8. 偏差与待办
- **偏差①（demo config 抬上限，§2.5 漏项补齐，需 PM 确认）**：§2.5 仅写「扩 `allowed_exts`」，但 §7 要 daxing 上传 **>50MB 大视频**验收降级路径 → 原 `max_size_bytes=10MB` + 前端预校验 10MB 会先拦死、传不上 >50MB。故连带：demo 两份 config `max_size_bytes` 10MB→**100MB** + 前端 `MAX_SIZE_BYTES`/提示文案同步抬到 100MB。**纯 demo 配置 + UX 预校验，零后端 Go / 零 openapi / 零底座改动**，不违中立（扩展名/上限本就 config 注入，T-004b）。若 PM 认为不应抬上限请回退，但那样 §7 大视频降级项无法验收。
- **偏差②（mime tab 复位实现，§6.2 适配，零核心改动）**：XTable `refresh`(=fetchData) 无复位页码入参、`page` 为 XTable 内部态，外部无法只复位页码。§6.2 设想的「页内自管 query.page=1 后 refresh」在真实 XTable 下不可达（page 不在页内控制）。零核心约束下唯一干净的页级复位手段 = 给 XTable 绑 `:key="mimeTab"` 重挂 → 复位 page=1 + 清选中 + 重新拉取。**代价**：切 tab 会一并重置列筛/排序（同一 tab 内三者共存无回归；跨 tab 视为新上下文重置）。若 PM 要求跨 tab 保留列筛/排序，需 XTable 核心暴露「refresh 复位页码」能力（碰核心，超本片 §6 中立铁律范围）→ 记可选基建待办，本片不擅扩。
- **预览走 `#row-actions` 槽（非 cell-slot，守 T-011a 裁决③）**：因 XAction 无「按行条件显隐」字段（预览仅媒体行显示），用既有 `#row-actions` 作用域插槽自定义整列（预览 v-if `canPreview` + 下载 + 删除），替代 config.actions。零核心改动。
- **后端待办（本片只记不做）**：① 媒体流 + Range（`Accept-Ranges`/`Content-Range` + 可能 tokenized 短时 URL）→ 做了才能撤销前端 50MB 降级阈值，独立后端片。② 后端缩略图 → 网格/缩略图视图前置（无缩略图则网格拉原图=流量灾难），独立片。

## 9. 下一步建议
1. PM 评审本报告（重点：偏差① 抬上限是否接受 / 偏差② 重挂复位是否接受 / 命门 mimeCategoryOf 镜像与吊销三处）。
2. daxing 重启 demo 后按 §7 浏览器验收（重点：大视频降级、批量删幂等 count、内存泄漏巡查、enforce）。
3. PM 放行 → 执行端双推（精确 stage：api/file.ts + file/index.vue + config.example.yaml + 本报告；config.local.yaml 不进仓）→ CI 双闸门绿 + ls-remote 三方一致 → 回报 feature hash。
4. 并行处理 T-011b 账本落账（待 PM 放行指令）。
5. T-011（媒体管理批次）三片 a/b/c 收口后建议 PM 评估「XTable 列内自定义单元格控件（cell-slot）」基建片——el-tag/switch/列内控件诉求已第 N 次（T-008a/b 记录），与本片预览 row-actions 同源。
