/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   操作日志 API — sys_oper_log 列表/清理（/sys/logs/oper，hashid 透传；只读消费，无增改删）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-10 12:02:21
 * | @updated   2026-06-17 00:00:00  T-014：删 OperLogRow.operator（后端 json tag 改 "-" 收口，前端只消费 operator_name）
 * +----------------------------------------------------------------------
 */
import { http } from '@/request'
import type { PageResult } from '@/request/types'

/**
 * 操作日志行（id 为 hashid 字符串）。
 * req_summary 为后端采集的请求体摘要，敏感字段（password/token/secret 等）已由
 * 后端 Sanitize 脱敏为 ***（T-004a），前端原样展示、不还原、不打印。
 * result_code：0=成功；>=400 时为 HTTP 状态码（后端仅在失败时记录）。
 */
export interface OperLogRow extends Record<string, unknown> {
  id: string
  operator_name: string // 后端 JOIN 解析的用户名（已注销→「已注销」、空→「匿名」）。T-014：raw operator 已收口，前端仅消费此名
  method: string
  path: string
  perm_code: string
  ip: string
  user_agent: string
  req_summary: string
  result_code: number
  latency_ms: number
  created_at: string
}

/** 列表。T-005b-4：后端支持 operator_name（用户名模糊）/path（模糊）/start_time/end_time/sort/order。 */
export function listOperLogs(params: Record<string, unknown>) {
  return http.get<PageResult<OperLogRow>>('/sys/logs/oper', { params })
}

/** 清理：后端固定删除 3 个月前的日志（无入参），返回 {deleted: 删除行数}。 */
export function cleanOperLogs() {
  return http.delete<{ deleted: number }>('/sys/logs/oper')
}
