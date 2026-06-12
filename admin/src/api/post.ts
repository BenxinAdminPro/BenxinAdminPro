/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   岗位 API — 列表/新增/编辑/删除（/sys/posts，hashid 透传）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-12 14:24:57
 * +----------------------------------------------------------------------
 *
 * 后端（rbac/handler_post.go + post_service.go）：
 *  - GET /sys/posts 分页列表，仅 page/page_size（page_size 上限 100，无 search/filter/sort）；
 *  - 出参经 ResponseEncoder.Post：id 为 hashid；字段 code/name/sort/status；
 *  - code 全局唯一，重名 409 ErrPostCodeExists（T-004e 1062 兜底已收口）；
 *  - 删除为软删，【不检查用户关联】：已分配用户的 sys_user_post 关联行残留但不再生效
 *    （用户详情按 id IN 查活跃岗位，软删岗位自然不显示）。
 */
import { http } from '@/request'
import type { PageResult } from '@/request/types'

/** 岗位行（响应字段，id 为 hashid 字符串）。 */
export interface PostRow extends Record<string, unknown> {
  id: string
  code: string
  name: string
  sort: number
  status: number
  created_at: string
  updated_at: string
}

export function listPosts(params: Record<string, unknown>) {
  return http.get<PageResult<PostRow>>('/sys/posts', { params })
}

/**
 * 拉取全量岗位（选择器用）：循环翻页直到 total 取齐，不静默截断。
 * 后端单页上限 100；岗位属底座组织数据，量级可控。
 */
export async function listAllPosts(): Promise<PostRow[]> {
  const all: PostRow[] = []
  let page = 1
  for (;;) {
    const res = await listPosts({ page, page_size: 100 })
    all.push(...(res.list || []))
    if (all.length >= res.total || (res.list || []).length === 0) break
    page++
  }
  return all
}

export function createPost(data: Record<string, unknown>) {
  return http.post('/sys/posts', data)
}

export function updatePost(id: string, data: Record<string, unknown>) {
  return http.put(`/sys/posts/${id}`, data)
}

export function removePost(id: string) {
  return http.delete(`/sys/posts/${id}`)
}
