/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   角色管理 API — 列表/新增/编辑/删除（对接 /sys/roles，hashid 透传）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 16:00:00
 * | @updated   2026-06-14 15:55:00  T-008b：+listAllRoles（分配角色弹窗全量选项，循环翻页取齐）
 * | @updated   2026-06-14 18:25:00  T-008c：+getRole（详情预载 menu_ids 授权树回填）+assignRoleMenus（PUT :id/menus 全量覆写）
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

/** 角色详情（GET /sys/roles/:id）。menu_ids 仅详情预载：当前全量已授菜单 hashid（含 M/C/F 三层）。 */
export interface RoleDetail extends RoleRow {
  menu_ids?: string[]
}

export function listRoles(params: Record<string, unknown>) {
  return http.get<PageResult<RoleRow>>('/sys/roles', { params })
}

/** 角色详情（授权树弹窗回填来源）：menu_ids = 当前全量已授菜单 hashid（含 M/C/F），hashid 透传不解码。 */
export function getRole(id: string) {
  return http.get<RoleDetail>(`/sys/roles/${id}`)
}

/**
 * 分配菜单/授权（PUT /sys/roles/:id/menus，权限码 sys:role:assign）。
 * 入参 menu_ids 为 hashid 数组，**全量覆写**该角色授权（service 先删后建 + 联动 Casbin p 规则）。
 * 故提交前须以「当前已授全量」（详情 menu_ids 回填）为基准改动，避免覆写丢权限。
 */
export function assignRoleMenus(id: string, menuIds: string[]) {
  return http.put(`/sys/roles/${id}/menus`, { menu_ids: menuIds })
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
