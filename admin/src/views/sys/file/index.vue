<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   媒体管理 — sys_file 列表 + mime 大类 tab 筛 + 多选批量软删 + 图片/视频/音频鉴权预览（超阈值降级）
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-10 17:19:28
  | @updated   2026-06-14 11:55:00  T-005b-4：uploader 列改绑可读化字段 uploader_name；列筛选/排序后端已就绪
  | @updated   2026-06-16 15:40:00  T-011c：mime 大类 tab + 多选批量删 + 鉴权 blob 预览/降级（XTable 核心零改）
  +----------------------------------------------------------------------
  说明：x-table readonly 基底 + selectable 多选（T-011a）+ #batch-actions/#row-actions 槽消费；
       mime 大类 tab 为页级筛（映射 T-011b mime_category 查询参数），切 tab 经 :key 重挂 XTable
       复位第 1 页 + 清选中（XTable.refresh 无复位页码入参、page 为 XTable 内部态，零核心改动下
       重挂是唯一干净的页级复位手段）。
       预览/降级/下载全经 axios 鉴权 blob 取流（fetchFileBlob，端点挂 RequirePerm sys:file:download）
       → createObjectURL 喂 el-image/video/audio，严禁裸 <img/video src=后端URL>（带不上 JWT 且暴露端点）。
       视频/音频 size > 50MB 降级为仅下载（v1 无 Range/无 tokenized URL，大文件整载无 seek，记后端待办）。
       object URL 不吊销会内存泄漏 → 三处吊销：切换预览前 + el-dialog @closed + onBeforeUnmount 卸载兜底。
       上传 multipart 单文件字段名 file；客户端预校验（大小/扩展名）仅 UX 提前拦，与 demo 后端默认值对齐，
       真安全由后端四件套把关（大小/白名单/文件名消毒/Content-Type 一致）。
-->
<script setup lang="ts">
import { ref, onBeforeUnmount } from 'vue'
import { ElMessage, ElMessageBox, type UploadRawFile, type UploadRequestOptions } from 'element-plus'
import { Upload, Download, Delete, View } from '@element-plus/icons-vue'
import XTable from '@/components/x-table/XTable.vue'
import type { XTableConfig, XRow } from '@/components/x-table/types'
import {
  listFiles,
  uploadFile,
  downloadFile,
  fetchFileBlob,
  removeFile,
  batchDeleteFiles,
  type SysFileRow,
} from '@/api/file'

const tableRef = ref<InstanceType<typeof XTable>>()

// 客户端预校验阈值 —— 与 demo 后端默认值对齐（config.local.yaml file.max_size_bytes /
// file.allowed_exts）。仅 UX 提前拦截，不构成安全边界；后端改配置后此处同步调整。
// T-011c：上限抬到 100MB 并扩 video/audio 扩展名，使 daxing 可上传 >50MB 媒体验收降级路径。
const MAX_SIZE_BYTES = 100 * 1024 * 1024
const ALLOWED_EXTS = [
  'jpg', 'jpeg', 'png', 'gif', 'pdf', 'docx', 'xlsx', 'zip', 'txt',
  'mp4', 'webm', 'mov', 'mp3', 'wav', 'ogg', 'm4a',
]

// 视频/音频内联播放上限（v1 前端常量）。超过则不取流、降级为仅下载——大文件整载无 seek、
// 内联体验差且占内存。将来后端支持 Range（Accept-Ranges/Content-Range）流式 seek 后可撤销本阈值。
const VIDEO_AUDIO_INLINE_MAX_BYTES = 50 * 1024 * 1024

/**
 * mime → 大类 token。**逐字镜像 T-011b 后端 system/file_service.go mimeCategoryPrefix 白名单**：
 * image/ → image、video/ → video、audio/ → audio，其余 → other。这是 tab 筛（后端 LIKE 前缀判）
 * 与单行预览/降级判定（前端判）一致性的命门。后端 LIKE 默认 collation 大小写不敏感，故此处先 toLowerCase。
 */
function mimeCategoryOf(mime: unknown): 'image' | 'video' | 'audio' | 'other' {
  const m = String(mime || '').toLowerCase()
  if (m.startsWith('image/')) return 'image'
  if (m.startsWith('video/')) return 'video'
  if (m.startsWith('audio/')) return 'audio'
  return 'other'
}

// 仅 image/video/audio 行可预览（other 如 txt/pdf 无预览，仅下载）
const canPreview = (row: XRow): boolean => ['image', 'video', 'audio'].includes(mimeCategoryOf(row.mime))

const dateText = (v: unknown) => (typeof v === 'string' ? v.slice(0, 19).replace('T', ' ') : '')

// 字节数 → 可读大小
function sizeText(v: unknown): string {
  const n = Number(v)
  if (!Number.isFinite(n) || n < 0) return ''
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(2)} MB`
}

// ---- mime 大类 tab（页级筛，映射 mime_category；XTable 核心零改）----
const mimeTab = ref<'' | 'image' | 'video' | 'audio' | 'other'>('')

// list 包装：把 mimeTab 并进请求参数（与既有 search/sort/page 同级）。空 → 不传（=全部）。
// 切 tab 经模板 :key="mimeTab" 重挂 XTable 触发复位（page=1 + 清选中），故无需手动 refresh。
function listApi(params: Record<string, unknown>) {
  return listFiles({ ...params, mime_category: mimeTab.value || undefined })
}

const config: XTableConfig = {
  readonly: true,
  selectable: true,
  api: { list: listApi },
  columns: [
    { prop: 'original_name', label: '文件名', minWidth: 200, filterable: true },
    { prop: 'ext', label: '类型', width: 80 },
    { prop: 'mime', label: 'MIME', minWidth: 150 },
    { prop: 'size', label: '大小', width: 100, formatter: (_r, v) => sizeText(v) },
    { prop: 'uploader_name', label: '上传人', width: 100, filterable: true },
    { prop: 'created_at', label: '上传时间', minWidth: 165, sortable: true, formatter: (_r, v) => dateText(v) },
  ],
  actionsWidth: 210,
}

// ---- 上传（工具栏；客户端预校验仅 UX，后端四件套才是安全边界）----
function beforeUpload(file: UploadRawFile): boolean {
  const dot = file.name.lastIndexOf('.')
  const ext = dot >= 0 ? file.name.slice(dot + 1).toLowerCase() : ''
  if (!ALLOWED_EXTS.includes(ext)) {
    ElMessage.error(`仅支持上传：${ALLOWED_EXTS.join(' / ')}`)
    return false
  }
  if (file.size > MAX_SIZE_BYTES) {
    ElMessage.error('文件大小不能超过 100MB')
    return false
  }
  return true
}

async function doUpload(opts: UploadRequestOptions): Promise<void> {
  try {
    await uploadFile(opts.file)
  } catch {
    return // 请求层已 toast（413 超限 / 415 白名单·类型不符 等友好码）
  }
  ElMessage.success('上传成功')
  tableRef.value?.refresh()
}

// ---- 鉴权下载：blob 带 JWT 取流 → 客户端落盘 ----
async function onDownload(row: SysFileRow): Promise<void> {
  let blob: Blob
  try {
    blob = await downloadFile(row.id)
  } catch {
    return // 请求层已 toast（404 等；blob 错误体已解析友好 message）
  }
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = row.original_name || 'download'
  a.click()
  URL.revokeObjectURL(url)
}

// ---- 单行删除（软删；二次确认 + 删后刷新）----
async function onRowDelete(row: SysFileRow): Promise<void> {
  try {
    await ElMessageBox.confirm(
      '删除后文件将不可再下载（软删元信息 + 物理文件异步清理，不可恢复），是否继续？',
      '危险操作',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' },
    )
  } catch {
    return // 用户取消 = no-op
  }
  try {
    await removeFile(String(row.id))
  } catch {
    return // 请求层已 toast
  }
  ElMessage.success('删除成功')
  tableRef.value?.refresh()
}

// ---- 批量删除（消费 #batch-actions；二次确认 → 用真实 deleted_count → clear + refresh）----
async function onBatchDelete(selected: XRow[], clear: () => void): Promise<void> {
  if (!selected.length) return
  try {
    await ElMessageBox.confirm(
      `确认删除选中的 ${selected.length} 个文件？删除后将不可再下载（软删 + 物理清理，不可恢复）。`,
      '危险操作',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' },
    )
  } catch {
    return // 用户取消 = no-op
  }
  const ids = selected.map((r) => String(r.id))
  let deleted = 0
  try {
    const res = await batchDeleteFiles(ids)
    deleted = res.deleted_count
  } catch {
    return // 请求层已 toast（空/超 100/非法 hashid → 400）
  }
  // 诚实显示真实命中数（幂等下可能 < 选中数）——吃 T-007f「成功提示≠状态变更」教训
  ElMessage.success(`已删除 ${deleted} 个文件`)
  clear() // = clearSelection
  tableRef.value?.refresh()
}

// ---- 预览（鉴权 blob → el-image/video/audio；超阈值降级；object URL 三处吊销防泄漏）----
const previewUrl = ref('') // 当前 object URL
const previewType = ref<'' | 'image' | 'video' | 'audio' | 'downgrade'>('')
const previewVisible = ref(false)
const previewLoading = ref(false)
const previewFile = ref<SysFileRow | null>(null)

function revokeCurrent(): void {
  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value)
    previewUrl.value = ''
  }
}

async function openPreview(row: SysFileRow): Promise<void> {
  previewFile.value = row
  const cat = mimeCategoryOf(row.mime)
  // 视频/音频超阈值 → 降级，不取流
  if ((cat === 'video' || cat === 'audio') && Number(row.size) > VIDEO_AUDIO_INLINE_MAX_BYTES) {
    previewType.value = 'downgrade'
    previewVisible.value = true
    return
  }
  if (cat === 'other') return // other 无预览（按钮本就不渲染，双保险）
  previewType.value = cat
  previewVisible.value = true
  previewLoading.value = true
  try {
    const blob = await fetchFileBlob(row.id) // 鉴权 blob 取流
    revokeCurrent() // 切换预览前先吊销旧 URL
    previewUrl.value = URL.createObjectURL(blob)
  } catch {
    previewVisible.value = false // 拦截器已 toast，失败关弹窗
  } finally {
    previewLoading.value = false
  }
}

function onPreviewClosed(): void {
  revokeCurrent() // 弹窗关闭（动画结束）吊销
  previewType.value = ''
  previewFile.value = null
}

onBeforeUnmount(() => revokeCurrent()) // 卸载兜底
</script>

<template>
  <el-card shadow="never">
    <!-- mime 大类 tab：页级筛，映射 mime_category；切 tab 经 :key 重挂 XTable 复位第 1 页 + 清选中 -->
    <el-radio-group v-model="mimeTab" class="file-mime-tabs">
      <el-radio-button value="">全部</el-radio-button>
      <el-radio-button value="image">图片</el-radio-button>
      <el-radio-button value="video">视频</el-radio-button>
      <el-radio-button value="audio">音频</el-radio-button>
      <el-radio-button value="other">其他</el-radio-button>
    </el-radio-group>

    <XTable :key="mimeTab" ref="tableRef" :config="config">
      <!-- 工具栏：上传挂独立上传权限码；单文件、自动上传、不显内置文件列表 -->
      <template #toolbar>
        <el-upload
          v-permission="'sys:file:upload'"
          :show-file-list="false"
          :before-upload="beforeUpload"
          :http-request="doUpload"
        >
          <el-button type="primary" :icon="Upload">上传文件</el-button>
        </el-upload>
        <el-tag type="info">单文件 ≤ 100MB；视频/音频 &gt; 50MB 预览降级为下载</el-tag>
      </template>

      <!-- 批量操作槽：批量删除按钮挂 sys:file:delete，显选中数 -->
      <template #batch-actions="{ selected, clear }">
        <el-button
          v-permission="'sys:file:delete'"
          type="danger"
          :icon="Delete"
          :disabled="!selected.length"
          @click="onBatchDelete(selected, clear)"
        >
          批量删除（{{ selected.length }}）
        </el-button>
      </template>

      <!-- 行操作：预览（仅媒体行）+ 下载 + 删除，均挂 v-permission（无码隐藏） -->
      <template #row-actions="{ row }">
        <el-button
          v-if="canPreview(row)"
          v-permission="'sys:file:download'"
          link
          type="primary"
          :icon="View"
          @click="openPreview(row as SysFileRow)"
        >
          预览
        </el-button>
        <el-button
          v-permission="'sys:file:download'"
          link
          type="primary"
          :icon="Download"
          @click="onDownload(row as SysFileRow)"
        >
          下载
        </el-button>
        <el-button
          v-permission="'sys:file:delete'"
          link
          type="danger"
          :icon="Delete"
          @click="onRowDelete(row as SysFileRow)"
        >
          删除
        </el-button>
      </template>
    </XTable>

    <!-- 预览弹窗：blob object URL 喂 el-image/video/audio（非裸后端 URL）；@closed 吊销 -->
    <el-dialog
      v-model="previewVisible"
      :title="previewFile?.original_name || '预览'"
      width="640px"
      destroy-on-close
      @closed="onPreviewClosed"
    >
      <div v-loading="previewLoading" class="file-preview">
        <el-image
          v-if="previewType === 'image' && previewUrl"
          :src="previewUrl"
          fit="contain"
          class="file-preview__media"
        />
        <video
          v-else-if="previewType === 'video' && previewUrl"
          :src="previewUrl"
          controls
          class="file-preview__media"
        />
        <audio
          v-else-if="previewType === 'audio' && previewUrl"
          :src="previewUrl"
          controls
          class="file-preview__audio"
        />
        <div v-else-if="previewType === 'downgrade'" class="file-preview__downgrade">
          <p>文件较大（{{ sizeText(previewFile?.size) }}），暂不支持在线预览，请下载查看。</p>
          <el-button
            v-if="previewFile"
            v-permission="'sys:file:download'"
            type="primary"
            :icon="Download"
            @click="onDownload(previewFile)"
          >
            下载查看
          </el-button>
        </div>
      </div>
    </el-dialog>
  </el-card>
</template>

<style scoped lang="scss">
.file-mime-tabs {
  margin-bottom: 12px;
}
.file-preview {
  min-height: 120px;
  display: flex;
  justify-content: center;
  align-items: center;

  &__media {
    max-width: 100%;
    max-height: 60vh;
  }
  &__audio {
    width: 100%;
  }
  &__downgrade {
    text-align: center;
    p {
      margin-bottom: 12px;
      color: var(--el-text-color-regular);
    }
  }
}
</style>
