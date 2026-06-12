/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   文件管理 API — sys_file 列表/上传/鉴权下载/删除（/sys/files，hashid 透传）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-10 17:19:28
 * +----------------------------------------------------------------------
 */
import { http } from '@/request'
import type { PageResult } from '@/request/types'

/**
 * 文件元信息行（id 为 hashid 字符串）。
 * storage_key 为相对存储 key（yyyy/MM/dd/uuid.ext），随列表返回但页面不展示（最小暴露）。
 * uploader 为后端采集的 JWT subject（内部用户 ID 字符串），与 operlog.operator 同源缺口
 * （已上报 T-005b），后端改存用户名后前端零改动生效。
 */
export interface SysFileRow extends Record<string, unknown> {
  id: string
  original_name: string
  storage_key: string
  storage_type: string
  size: number
  mime: string
  ext: string
  uploader: string
  status: number
  created_at: string
  updated_at: string
}

/** 列表。后端支持 page/page_size/uploader（精确匹配）；其余参数透传（后端暂不消费，降级）。 */
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
 * 鉴权流式下载：经 axios 带 Authorization 头取 blob（下载端点挂 RequirePerm，
 * 裸 <a href> 带不上 JWT 会 401，禁用）。文件名取行内 original_name（与后端
 * Content-Disposition 同源）。
 */
export function downloadFile(id: string) {
  return http.get<Blob>(`/sys/files/${id}/download`, { responseType: 'blob' })
}

/** 删除：按 id 软删元信息 + 后端异步物理清理（单条语义，后端无批量端点）。 */
export function removeFile(id: string) {
  return http.delete(`/sys/files/${id}`)
}
