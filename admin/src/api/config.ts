/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   参数管理 API — sys_config CRUD（/sys/configs，hashid 透传；加密参数值由后端脱敏为 ******）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-10 09:38:01
 * +----------------------------------------------------------------------
 */
import { http } from '@/request'
import type { PageResult } from '@/request/types'

/**
 * 参数行（id 为 hashid 字符串；config_key 为业务唯一键）。
 * is_encrypted=1 的行 config_value 恒为后端脱敏占位 ******（maskEncrypted，T-005），
 * 前端任何路径都拿不到、也绝不展示/打印真实明文。
 */
export interface ConfigRow extends Record<string, unknown> {
  id: string
  config_key: string
  config_value: string
  name: string
  remark: string
  is_encrypted: number
  created_at: string
  updated_at: string
}

export function listConfigs(params: Record<string, unknown>) {
  return http.get<PageResult<ConfigRow>>('/sys/configs', { params })
}

export function createConfig(data: Record<string, unknown>) {
  return http.post('/sys/configs', data)
}

export function updateConfig(id: string, data: Record<string, unknown>) {
  return http.put(`/sys/configs/${id}`, data)
}

export function removeConfig(id: string) {
  return http.delete(`/sys/configs/${id}`)
}
