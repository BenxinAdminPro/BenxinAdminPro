/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   角色管理 API — 列表/新增/编辑/删除（对接 /sys/roles，hashid 透传）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 16:00:00
 * +----------------------------------------------------------------------
 */
import { http } from '@/request'
import type { PageResult } from '@/request/types'

/** 角色行（data_scope：1=全部 2=本人 3=本部门）。索引签名以兼容 x-table 的通用行类型。 */
export interface RoleRow extends Record<string, unknown> {
  id: string
  code: string
  name: string
  sort: number
  status: number
  data_scope: number
  remark: string
  created_at: string
  updated_at: string
}

export function listRoles(params: Record<string, unknown>) {
  return http.get<PageResult<RoleRow>>('/sys/roles', { params })
}

export function createRole(data: Record<string, unknown>) {
  return http.post('/sys/roles', data)
}

export function updateRole(id: string, data: Record<string, unknown>) {
  return http.put(`/sys/roles/${id}`, data)
}

export function removeRole(id: string) {
  return http.delete(`/sys/roles/${id}`)
}
