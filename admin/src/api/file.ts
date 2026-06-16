/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   文件管理 API — sys_file 列表/上传/鉴权下载/删除（/sys/files，hashid 透传）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-10 17:19:28
 * | @updated   2026-06-16 15:40:00  T-011c：fetchFileBlob 鉴权取流（预览/下载共用）+ batchDeleteFiles 批量软删 + listFiles 透传 mime_category
 * +----------------------------------------------------------------------
 */
import { http } from '@/request'
import type { PageResult } from '@/request/types'

/**
 * 文件元信息行（id 为 hashid 字符串）。
 * storage_key 为相对存储 key（yyyy/MM/dd/uuid.ext），随列表返回但页面不展示（最小暴露）。
 * uploader 为后端采集的 JWT subject（内部用户 ID 字符串，不直接展示）；
 * T-005b-4 起后端随列表返回 uploader_name（JOIN 解析的用户名），前端显示该字段。
 */
export interface SysFileRow extends Record<string, unknown> {
  id: string
  original_name: string
  storage_key: string
  storage_type: string
  size: number
  mime: string
  ext: string
  uploader: string // 内部用户 ID 字符串（采集原值，不直接展示）
  uploader_name: string // T-005b-4：后端 JOIN 解析的用户名（已注销→「已注销」、空→「匿名」）
  status: number
  created_at: string
  updated_at: string
}

/**
 * 列表。T-005b-4：后端支持 uploader_name（用户名模糊）/original_name（模糊）/sort/order。
 * T-011c：mime_category（image/video/audio/other/空）随 params 透传 → 后端按常量前缀白名单
 * 大类筛（前端只传 token、不参与 SQL，注入面在后端杜绝）。
 */
export function listFiles(params: Record<string, unknown>) {
  return http.get<PageResult<SysFileRow>>('/sys/files', { params })
}

/** 上传：multipart 单文件，字段名 file（与后端 c.FormFile("file") 对齐）；返回新建文件元信息。 */
export function uploadFile(file: File) {
  const fd = new FormData()
  fd.append('file', file)
  return http.post<SysFileRow>('/sys/files', fd)
}

/**
 * 鉴权流式取字节：经 axios 带 Authorization 头取 blob（下载端点挂 RequirePerm sys:file:download，
 * 裸 <a href>/<img src>/<video src> 带不上 JWT 会 401 且暴露端点形态，一律禁用）。
 * 下载落盘（createObjectURL→a.click）与预览（createObjectURL→el-image/video/audio）共用此取流，
 * 区别仅消费语义。T-011c 起预览复用本函数；downloadFile 为其语义别名（行为与既有逐字一致、零回归）。
 */
export function fetchFileBlob(id: string) {
  return http.get<Blob>(`/sys/files/${id}/download`, { responseType: 'blob' })
}

/** 下载落盘：复用 fetchFileBlob 鉴权取流（语义别名，与 T-007f 既有下载行为一致）。 */
export const downloadFile = fetchFileBlob

/** 删除：按 id 软删元信息 + 后端异步物理清理（单条语义）。 */
export function removeFile(id: string) {
  return http.delete(`/sys/files/${id}`)
}

/**
 * 批量软删（T-011b POST /sys/files/batch-delete，复用 sys:file:delete）：body {ids:[hashid…]}，
 * 返 {deleted_count}；幂等（已删/不存在 id 不计入）；空/超 100/非法 hashid 由后端 400 拦。
 */
export function batchDeleteFiles(ids: string[]) {
  return http.post<{ deleted_count: number }>('/sys/files/batch-delete', { ids })
}
