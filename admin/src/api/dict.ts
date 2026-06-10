/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   字典管理 API — 类型 CRUD（/sys/dict/types）+ 数据 CRUD（/sys/dict/data，hashid 透传）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-10 09:38:01
 * +----------------------------------------------------------------------
 */
import { http } from '@/request'
import type { PageResult } from '@/request/types'

/** 字典类型行（id 为 hashid 字符串；dict_type 为业务唯一键）。 */
export interface DictTypeRow extends Record<string, unknown> {
  id: string
  dict_type: string
  name: string
  status: number
  remark: string
  created_at: string
  updated_at: string
}

/** 字典数据行（id 为 hashid 字符串；按 dict_type 归属类型）。 */
export interface DictDataRow extends Record<string, unknown> {
  id: string
  dict_type: string
  label: string
  value: string
  sort: number
  status: number
  created_at: string
  updated_at: string
}

// ---- 字典类型 ----

export function listDictTypes(params: Record<string, unknown>) {
  return http.get<PageResult<DictTypeRow>>('/sys/dict/types', { params })
}

export function createDictType(data: Record<string, unknown>) {
  return http.post('/sys/dict/types', data)
}

export function updateDictType(id: string, data: Record<string, unknown>) {
  return http.put(`/sys/dict/types/${id}`, data)
}

export function removeDictType(id: string) {
  return http.delete(`/sys/dict/types/${id}`)
}

// ---- 字典数据 ----

/**
 * 字典数据列表 — 后端按 dict_type 过滤返回全量数组（无分页包络/无分页参数），
 * 由页面适配层包装为分页结构。禁止以空 dict_type 调用（未选类型时页面不发请求）。
 */
export function listDictData(dictType: string) {
  return http.get<DictDataRow[]>('/sys/dict/data', { params: { dict_type: dictType } })
}

export function createDictData(data: Record<string, unknown>) {
  return http.post('/sys/dict/data', data)
}

export function updateDictData(id: string, data: Record<string, unknown>) {
  return http.put(`/sys/dict/data/${id}`, data)
}

export function removeDictData(id: string) {
  return http.delete(`/sys/dict/data/${id}`)
}
