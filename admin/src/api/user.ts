/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   用户管理 API — 列表/新增/编辑/删除（对接 /sys/users，hashid 透传）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 16:00:00
 * +----------------------------------------------------------------------
 */
import { http } from '@/request'
import type { PageResult } from '@/request/types'

/** 用户行（响应字段，id/dept_id 为 hashid 字符串）。索引签名以兼容 x-table 的通用行类型。 */
export interface UserRow extends Record<string, unknown> {
  id: string
  username: string
  nickname: string
  email: string
  mobile: string
  status: number
  remark: string
  dept_id: string | null
  created_at: string
  updated_at: string
}

export function listUsers(params: Record<string, unknown>) {
  return http.get<PageResult<UserRow>>('/sys/users', { params })
}

export function createUser(data: Record<string, unknown>) {
  return http.post('/sys/users', data)
}

export function updateUser(id: string, data: Record<string, unknown>) {
  return http.put(`/sys/users/${id}`, data)
}

export function removeUser(id: string) {
  return http.delete(`/sys/users/${id}`)
}
