<!--
  +----------------------------------------------------------------------
  | @project   本心通用管理后台 / BenxinAdminPro
  | @mission   文件管理 — sys_file 列表 + 工具栏上传 + 行内鉴权下载/删除（x-table 只读基底消费）
  | @author    仗键天涯(daxing)
  | @email     3442535897@qq.com
  | @date      2026-06-10 17:19:28
  | @updated   2026-06-14 11:55:00  T-005b-4：uploader 列改绑可读化字段 uploader_name；列筛选/排序后端已就绪
  +----------------------------------------------------------------------
  说明：x-table readonly 基底 + 工具栏插槽放上传（x-table 无上传能力、不为本片扩展）；
       上传 multipart 单文件字段名 file；客户端预校验（大小/扩展名）与 demo 后端默认值对齐，
       仅 UX 提前拦截，真安全由后端四件套把关（大小/白名单/文件名消毒/Content-Type 一致）。
       下载为鉴权流式（端点挂 RequirePerm sys:file:download）：axios blob 带 JWT 取流后
       客户端落盘，禁用裸 <a href>（带不上 Authorization）。
       删除：按 id 软删 + 后端异步物理清理（单条语义，后端无批量端点，不造多选）。
       storage_key 随列表返回但不展示（最小暴露，T-004b 待办归后端加固，前端先规避）。
       T-005b-4：uploader 出参已可读化（uploader_name=用户名，已注销→「已注销」）；列筛选 uploader_name
       按用户名模糊、original_name 模糊、created_at 排序后端均已就绪，filterable/sortable 真生效。
-->
<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage, type UploadRawFile, type UploadRequestOptions } from 'element-plus'
import { Upload, Download, Delete } from '@element-plus/icons-vue'
import XTable from '@/components/x-table/XTable.vue'
import type { XTableConfig, XRow } from '@/components/x-table/types'
import { listFiles, uploadFile, downloadFile, removeFile, type SysFileRow } from '@/api/file'

const tableRef = ref<InstanceType<typeof XTable>>()

// 客户端预校验阈值 —— 与 demo 后端默认值对齐（config.example.yaml file.max_size_bytes /
// file.allowed_exts）。仅 UX 提前拦截，不构成安全边界；后端改配置后此处同步调整。
const MAX_SIZE_BYTES = 10 * 1024 * 1024
const ALLOWED_EXTS = ['jpg', 'jpeg', 'png', 'gif', 'pdf', 'docx', 'xlsx', 'zip', 'txt']

const dateText = (v: unknown) => (typeof v === 'string' ? v.slice(0, 19).replace('T', ' ') : '')

// 字节数 → 可读大小
function sizeText(v: unknown): string {
  const n = Number(v)
  if (!Number.isFinite(n) || n < 0) return ''
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  return `${(n / 1024 / 1024).toFixed(2)} MB`
}

const config: XTableConfig = {
  readonly: true,
  api: { list: listFiles },
  columns: [
    // T-005b-4：original_name 模糊过滤后端已就绪，filterable 真生效
    { prop: 'original_name', label: '文件名', minWidth: 200, filterable: true },
    { prop: 'ext', label: '类型', width: 80 },
    { prop: 'mime', label: 'MIME', minWidth: 150 },
    { prop: 'size', label: '大小', width: 100, formatter: (_r, v) => sizeText(v) },
    // T-005b-4：列显示 uploader_name（后端 JOIN sys_user 解析的用户名，已注销→「已注销」、空→「匿名」）；
    // 列筛选 uploader_name 后端按用户名模糊真生效（filter 参数名=prop=uploader_name）
    { prop: 'uploader_name', label: '上传人', width: 100, filterable: true },
    // T-005b-4：created_at 排序后端已就绪，sortable 真生效
    { prop: 'created_at', label: '上传时间', minWidth: 165, sortable: true, formatter: (_r, v) => dateText(v) },
  ],
  actions: [
    { label: '下载', perm: 'sys:file:download', icon: Download, handler: onDownload },
    {
      label: '删除',
      perm: 'sys:file:delete',
      type: 'danger',
      icon: Delete,
      confirm: '删除后文件将不可再下载（软删元信息 + 物理文件异步清理，不可恢复），是否继续？',
      handler: onRemove,
      refresh: true,
    },
  ],
  actionsWidth: 150,
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
    ElMessage.error('文件大小不能超过 10MB')
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
async function onDownload(row: XRow): Promise<void> {
  const r = row as SysFileRow
  let blob: Blob
  try {
    blob = await downloadFile(r.id)
  } catch {
    return // 请求层已 toast（404 等；blob 错误体已解析友好 message）
  }
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = r.original_name || 'download'
  a.click()
  URL.revokeObjectURL(url)
}

// ---- 删除（软删；二次确认由 XAction.confirm 承担，成功后 refresh 自动刷新）----
async function onRemove(row: XRow): Promise<void> {
  try {
    await removeFile(String(row.id))
  } catch {
    return // 请求层已 toast
  }
  ElMessage.success('删除成功')
}
</script>

<template>
  <el-card shadow="never">
    <XTable ref="tableRef" :config="config">
      <!-- 工具栏：上传挂独立上传权限码（非 list 码）；单文件、自动上传、不显内置文件列表 -->
      <template #toolbar>
        <el-upload
          v-permission="'sys:file:upload'"
          :show-file-list="false"
          :before-upload="beforeUpload"
          :http-request="doUpload"
        >
          <el-button type="primary" :icon="Upload">上传文件</el-button>
        </el-upload>
        <el-tag type="info">单文件 ≤ 10MB；支持 {{ ALLOWED_EXTS.join(' / ') }}</el-tag>
      </template>
    </XTable>
  </el-card>
</template>
