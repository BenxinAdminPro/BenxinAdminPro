/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   角色管理 API — 列表/新增/编辑/删除（对接 /sys/roles，hashid 透传）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 16:00:00
 * | @updated   2026-06-14 15:55:00  T-008b：+listAllRoles（分配角色弹窗全量选项，循环翻页取齐）
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

/**
 * 拉取全量角色（分配角色弹窗选项用）：循环翻页直到 total 取齐，不静默截断。
 * 后端单页上限 100；角色属底座权限数据，量级可控（同 listAllPosts 范式）。
 */
export async function listAllRoles(): Promise<RoleRow[]> {
  const all: RoleRow[] = []
  let page = 1
  for (;;) {
    const res = await listRoles({ page, page_size: 100 })
    all.push(...(res.list || []))
    if (all.length >= res.total || (res.list || []).length === 0) break
    page++
  }
  return all
}
