/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   登录日志 API — sys_login_log 列表/清理（/sys/logs/login，hashid 透传；只读消费，无增改删）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-10 12:02:21
 * +----------------------------------------------------------------------
 */
import { http } from '@/request'
import type { PageResult } from '@/request/types'

/** 登录日志行（id 为 hashid 字符串）。success：1=成功 0=失败；失败原因在 reason。 */
export interface LoginLogRow extends Record<string, unknown> {
  id: string
  username: string
  ip: string
  user_agent: string
  success: number
  reason: string
  created_at: string
}

/** 列表。后端支持 page/page_size/username（精确匹配）；其余参数透传（后端暂不消费，降级）。 */
export function listLoginLogs(params: Record<string, unknown>) {
  return http.get<PageResult<LoginLogRow>>('/sys/logs/login', { params })
}

/** 清理：后端固定删除 3 个月前的日志（无入参），返回 {deleted: 删除行数}。 */
export function cleanLoginLogs() {
  return http.delete<{ deleted: number }>('/sys/logs/login')
}
