/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   菜单管理 API — 树查询 + CRUD（/sys/menus，hashid 透传）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-12 10:51:19
 * +----------------------------------------------------------------------
 *
 * 后端（rbac/handler_menu.go）：
 *  - GET /sys/menus/tree 返回【已嵌套的树】（服务端 buildMenuTree，sort ASC, id ASC），无查询参数；
 *  - parent_id 出参：根为 null，其余 hashid 字符串（encodeOrZero）；
 *  - create 入参 parent_id：空串=挂根；update 入参 parent_id：缺省=不移动 / 空串=移到根 / hashid=移动；
 *  - update 为全量覆写（所有字段都会写入），提交方须带全所有字段。
 */
import { http } from '@/request'

/** 菜单管理行（含 ancestors；与 /sys/auth/menus 的 MenuNode 形状一致但用途不同——这里是管理全量树）。 */
export interface MenuRow extends Record<string, unknown> {
  id: string
  parent_id: string | null
  ancestors: string
  menu_type: 'M' | 'C' | 'F'
  name: string
  perm_code: string
  path: string
  component: string
  icon: string
  sort: number
  visible: number
  status: number
  created_at: string
  updated_at: string
  children?: MenuRow[]
}

/** 菜单 create/update 提交体（parent_id 语义见文件头注释）。 */
export interface MenuPayload {
  parent_id: string
  menu_type: string
  name: string
  perm_code: string
  path: string
  component: string
  icon: string
  sort: number
  visible: number
  status: number
}

/** 拉取菜单管理全量树（后端已嵌套，children 递归）。 */
export function getMenuTree() {
  return http.get<MenuRow[]>('/sys/menus/tree')
}

export function createMenu(data: MenuPayload) {
  return http.post('/sys/menus', data)
}

export function updateMenu(id: string, data: MenuPayload) {
  return http.put(`/sys/menus/${id}`, data)
}

export function removeMenu(id: string) {
  return http.delete(`/sys/menus/${id}`)
}
