/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   x-table 配置类型 — 列/表单字段/搜索/接口/权限前缀
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 16:00:00
 * +----------------------------------------------------------------------
 */
import type { PageResult } from '@/request/types'

export type XRow = Record<string, unknown>

/** 列定义。 */
export interface XColumn {
  prop: string
  label: string
  width?: number | string
  minWidth?: number | string
  /** 单元格格式化（如状态码→文案、时间裁剪）。 */
  formatter?: (row: XRow, value: unknown) => string
}

/** 表单字段（用于新增/编辑弹窗）。 */
export interface XField {
  prop: string
  label: string
  type?: 'input' | 'password' | 'textarea' | 'number' | 'select'
  required?: boolean
  placeholder?: string
  options?: { label: string; value: string | number }[]
  /** 仅新增时出现（如密码）。 */
  createOnly?: boolean
  /** 编辑时是否可改（默认 true）。 */
  editable?: boolean
  /** 新增时的默认值。 */
  default?: unknown
}

/** 搜索字段。 */
export interface XSearchField {
  prop: string
  label: string
  type?: 'input' | 'select'
  placeholder?: string
  options?: { label: string; value: string | number }[]
}

/** 数据接口（对接 T-007a 请求层，已解包统一包络）。 */
export interface XApi<T extends XRow = XRow> {
  list: (params: Record<string, unknown>) => Promise<PageResult<T>>
  create: (data: Record<string, unknown>) => Promise<unknown>
  update: (id: string, data: Record<string, unknown>) => Promise<unknown>
  remove: (id: string) => Promise<unknown>
}

/** x-table 总配置。 */
export interface XTableConfig<T extends XRow = XRow> {
  /** 行主键字段（hashid 字符串），默认 'id'。 */
  rowKey?: string
  columns: XColumn[]
  fields: XField[]
  search?: XSearchField[]
  api: XApi<T>
  /** 权限码前缀（如 'sys:user'）→ 新增/编辑/删除按钮挂 permPrefix:create/update/delete。 */
  permPrefix?: string
}
