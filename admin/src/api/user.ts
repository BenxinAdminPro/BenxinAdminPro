/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   用户管理 API — 列表/新增/编辑/删除（对接 /sys/users，hashid 透传）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 16:00:00
 * | @updated   2026-06-12 14:24:57  T-007h：+getUser 详情（dept_id/posts 编辑回填来源）
 * | @updated   2026-06-14 14:30:00  T-008a：+resetUserPassword（PUT :id/password）+setUserStatus（PUT :id/status）
 * +----------------------------------------------------------------------
 */
import { http } from '@/request'
import type { PageResult } from '@/request/types'
import type { PostRow } from './post'

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

/**
 * 用户详情（GET /sys/users/:id，权限码 sys:user:get）。
 * 后端 ResponseEncoder.User：posts 仅在用户有岗位时出现（空岗位无该字段）；
 * dept_id 无部门为 null。编辑回填的适配（null→''、posts→post_ids）在消费方做。
 */
export interface UserDetail extends UserRow {
  posts?: PostRow[]
}

export function listUsers(params: Record<string, unknown>) {
  return http.get<PageResult<UserRow>>('/sys/users', { params })
}

export function getUser(id: string) {
  return http.get<UserDetail>(`/sys/users/${id}`)
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

/**
 * 重置用户密码（PUT /sys/users/:id/password，权限码 sys:user:password）。
 * 后端入参仅 { password }（binding:"required" 非空，无长度/强度校验）；
 * 前端预校验（非空 + 二次确认一致）仅 UX、不伪造后端不存在的强度规则。
 */
export function resetUserPassword(id: string, password: string) {
  return http.put(`/sys/users/${id}/password`, { password })
}

/**
 * 设置用户状态（PUT /sys/users/:id/status，权限码 sys:user:status）。
 * status 语义以后端为准：0=正常（启用），非0=停用（auth 登录 Status!=0 → 账号停用拒登）。
 */
export function setUserStatus(id: string, status: number) {
  return http.put(`/sys/users/${id}/status`, { status })
}
