/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   请求层类型 — 统一响应包络 + 分页结构（与后端 response.Registry 对齐）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 14:00:00
 * +----------------------------------------------------------------------
 */

/** 后端统一响应包络：成功 code==0 取 data；非 0 为业务错误（message 提示）。 */
export interface ApiEnvelope<T = unknown> {
  code: number
  message: string
  data: T
}

/** 分页返回结构（与后端 {list,total,page,page_size} 一致）。 */
export interface PageResult<T = unknown> {
  list: T[]
  total: number
  page: number
  page_size: number
}
