/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   v-permission 指令 — 无对应权限码则隐藏元素（仅 UX，后端才是边界）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 16:00:00
 * | @updated   2026-06-09 15:30:00  T-007c：空绑定值=笔误防护（隐藏 + dev 告警），公开应"不挂指令"
 * +----------------------------------------------------------------------
 *
 * 用法：v-permission="'sys:user:create'" 或 v-permission="['sys:user:update','sys:user:delete']"
 * 语义：当前用户拥有其中任一码即显示，否则从 DOM 移除该元素。
 * 防笔误（关键）：绑定值为空（undefined / '' / 空数组，很可能是笔误或未定义变量）一律【隐藏 +
 *      dev 模式 console.warn】，绝不静默"所有人可见"——让"忘记传码"不会悄悄变成权限漏洞。
 *      若确实要"所有人可见"，正确表达是【根本不写 v-permission 指令】，而非给指令喂空值。
 * 注意：这是体验层；真正鉴权在后端（T-003d 已 enforce），无权用户即使手拼请求也会 403。
 */
import type { Directive, DirectiveBinding } from 'vue'
import { useUserStore } from '@/store/user'

function check(el: HTMLElement, binding: DirectiveBinding): void {
  const value = binding.value
  const codes = (Array.isArray(value) ? value : [value]).filter(
    (c): c is string => typeof c === 'string' && c !== '',
  )
  // 空绑定值 = 疑似笔误：隐藏元素并在 dev 告警；公开元素应"不挂指令"而非喂空值
  if (codes.length === 0) {
    if (import.meta.env.DEV) {
      console.warn(
        '[v-permission] 绑定值为空（undefined/空字符串/空数组），疑似笔误或未定义变量，元素已隐藏。' +
          '若意在公开，请勿挂 v-permission 指令。',
        el,
      )
    }
    el.parentNode?.removeChild(el)
    return
  }
  const user = useUserStore()
  const allowed = codes.some((c) => user.hasPerm(c))
  if (!allowed) {
    el.parentNode?.removeChild(el)
  }
}

export const permission: Directive = {
  mounted: check,
}
