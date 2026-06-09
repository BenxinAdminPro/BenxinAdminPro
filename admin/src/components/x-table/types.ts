/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   x-table 配置类型 — 列/表单字段/搜索/接口/权限前缀/只读/行操作
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 16:00:00
 * | @updated   2026-06-09 14:30:00  T-007c：只读模式 + 行操作配置 + 列筛选/排序
 * +----------------------------------------------------------------------
 */
import type { Component } from 'vue'
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
  /**
   * 服务端排序：开启后该列表头出现排序箭头，排序经查询参数 sort=<prop>&order=asc|desc 透传后端。
   * 后端是排序权威；若后端不支持该字段排序，参数被忽略（前端不做内存伪排序）。
   */
  sortable?: boolean
  /**
   * 列筛选：开启后该列表头出现筛选入口，输入值经查询参数 <prop>=<value> 并入请求透传后端。
   * 后端是过滤权威；若后端无对应过滤参数则降级为无效（前端只拼参数、不假设可信）。
   */
  filterable?: boolean
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

/** 自定义行操作（业务页定义；按钮可挂 v-permission，仅 UX，后端 enforce 才是边界）。 */
export interface XAction {
  label: string
  /** 权限码（单码或数组）。缺省=该按钮为公开操作、不挂 v-permission 指令（而非喂空值，见 directives/permission.ts）。 */
  perm?: string | string[]
  /** 按钮类型（默认 primary）。 */
  type?: 'primary' | 'success' | 'warning' | 'danger' | 'info'
  /** Element Plus 图标组件（可选）。 */
  icon?: Component
  /** 提供则点击先弹二次确认（危险操作），文案即此值。 */
  confirm?: string
  /** 点击处理；接收当前行。 */
  handler: (row: XRow) => void | Promise<void>
  /** handler 完成后是否自动刷新列表（如清理/状态变更类操作）。 */
  refresh?: boolean
}

/** 数据接口（对接 T-007a 请求层，已解包统一包络）。只读表只需提供 list。 */
export interface XApi<T extends XRow = XRow> {
  list: (params: Record<string, unknown>) => Promise<PageResult<T>>
  create?: (data: Record<string, unknown>) => Promise<unknown>
  update?: (id: string, data: Record<string, unknown>) => Promise<unknown>
  remove?: (id: string) => Promise<unknown>
}

/** x-table 总配置。 */
export interface XTableConfig<T extends XRow = XRow> {
  /** 行主键字段（hashid 字符串），默认 'id'。 */
  rowKey?: string
  columns: XColumn[]
  /** 表单字段；只读模式可省略（不出现新增/编辑弹窗）。 */
  fields?: XField[]
  search?: XSearchField[]
  api: XApi<T>
  /** 权限码前缀（如 'sys:user'）→ 新增/编辑/删除按钮挂 permPrefix:create/update/delete。 */
  permPrefix?: string
  /** 只读模式：关闭内置新增/编辑/删除，仅保留列表+分页+搜索+自定义行操作。 */
  readonly?: boolean
  /** 自定义行操作列表（与内置编辑/删除并存；只读模式下为唯一行操作来源）。 */
  actions?: XAction[]
  /** 操作列宽度（默认 160；行操作多时调大）。 */
  actionsWidth?: number | string
}
