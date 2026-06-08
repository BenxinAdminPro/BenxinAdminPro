/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   v-permission 指令 — 无对应权限码则隐藏元素（仅 UX，后端才是边界）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 16:00:00
 * +----------------------------------------------------------------------
 *
 * 用法：v-permission="'sys:user:create'" 或 v-permission="['sys:user:update','sys:user:delete']"
 * 语义：当前用户拥有其中任一码即显示，否则从 DOM 移除该元素。
 * 注意：这是体验层；真正鉴权在后端（T-003d 已 enforce），无权用户即使手拼请求也会 403。
 */
import type { Directive, DirectiveBinding } from 'vue'
import { useUserStore } from '@/store/user'

function check(el: HTMLElement, binding: DirectiveBinding): void {
  const value = binding.value
  const codes = Array.isArray(value) ? value : [value]
  const user = useUserStore()
  const allowed = codes.some((c) => typeof c === 'string' && user.hasPerm(c))
  if (!allowed) {
    el.parentNode?.removeChild(el)
  }
}

export const permission: Directive = {
  mounted: check,
}
