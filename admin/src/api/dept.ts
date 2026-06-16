/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   部门 API — 树查询（/sys/depts/tree，hashid 透传）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-12 14:24:57
 * | @updated   2026-06-16 21:20:34  T-012：补 create/update/remove + DeptPayload（部门管理页 CRUD/移动消费）
 * +----------------------------------------------------------------------
 *
 * 后端（rbac/handler_dept.go + dept_service.go）：
 *  - GET /sys/depts/tree 返回【已嵌套的树】（服务端 buildTree，sort ASC, id ASC），无查询参数；
 *  - 出参经 ResponseEncoder.Dept：id 为 hashid；parent_id 根为 null、其余 hashid（encodeOrZero）；
 *    children 仅在非空时出现（叶子节点无该字段）。
 *  - create 入参 parent_id：空串=挂根（decodeZeroableID）；
 *  - update 入参 parent_id：缺省=不移动 / 空串=移到根 / hashid=移动（decodeMovableID 三态；同值后端 no-op）；
 *  - update 为全量覆写 name/sort/leader/status（updateDeptReq.Name 无 binding:required）——
 *    提交方须带全字段并正确回填，漏 leader 即静默清空部门负责人（对标 T-007h §8-3 同型缺陷）。
 *  - delete 软删：有子部门→409 ErrDeptHasChildren；有归属用户→409 ErrDeptHasUsers；不存在→400。
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

/** 部门 create/update 提交体（parent_id 三态语义见文件头注释；全字段全量提交）。 */
export interface DeptPayload {
  parent_id: string // 空串=根（create）/ 移到根（update）；hashid=指定父；update 同值后端 no-op
  name: string
  sort: number
  leader: string
  status: number
}

/** 拉取部门树（后端已嵌套，权限码 sys:dept:tree）。 */
export function getDeptTree() {
  return http.get<DeptNode[]>('/sys/depts/tree')
}

export function createDept(data: DeptPayload) {
  return http.post('/sys/depts', data)
}

export function updateDept(id: string, data: DeptPayload) {
  return http.put(`/sys/depts/${id}`, data)
}

export function removeDept(id: string) {
  return http.delete(`/sys/depts/${id}`)
}
