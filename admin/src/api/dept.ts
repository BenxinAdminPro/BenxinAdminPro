/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   部门 API — 树查询（/sys/depts/tree，hashid 透传）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-12 14:24:57
 * +----------------------------------------------------------------------
 *
 * 后端（rbac/handler_dept.go + dept_service.go）：
 *  - GET /sys/depts/tree 返回【已嵌套的树】（服务端 buildTree，sort ASC, id ASC），无查询参数；
 *  - 出参经 ResponseEncoder.Dept：id 为 hashid；parent_id 根为 null、其余 hashid（encodeOrZero）；
 *    children 仅在非空时出现（叶子节点无该字段）。
 */
import { http } from '@/request'

/** 部门树节点（后端嵌套树，字段以 rbac/model.go SysDept + response.go Dept 为准）。 */
export interface DeptNode extends Record<string, unknown> {
  id: string
  parent_id: string | null
  ancestors: string
  name: string
  sort: number
  leader: string
  status: number
  created_at: string
  updated_at: string
  children?: DeptNode[]
}

/** 拉取部门树（后端已嵌套，权限码 sys:dept:tree）。 */
export function getDeptTree() {
  return http.get<DeptNode[]>('/sys/depts/tree')
}
