/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   登录态权限下发 API — 当前用户菜单树 + 权限码集合
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 16:00:00
 * +----------------------------------------------------------------------
 */
import { http } from '@/request'

/** 后端菜单节点（/sys/auth/menus，hashid id；menu_type：M 目录 / C 菜单 / F 按钮）。 */
export interface MenuNode {
  id: string
  parent_id: string | null
  menu_type: 'M' | 'C' | 'F'
  name: string
  perm_code: string
  path: string
  component: string
  icon: string
  sort: number
  visible: number
  status: number
  children?: MenuNode[]
}

/** 拉取当前用户可见菜单树。 */
export function getMenus() {
  return http.get<MenuNode[]>('/sys/auth/menus')
}

/** 拉取当前用户权限码集合。 */
export function getPerms() {
  return http.get<string[]>('/sys/auth/perms')
}
