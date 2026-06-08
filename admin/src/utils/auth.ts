/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   令牌存储工具 — access/refresh 持久化（localStorage）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 14:00:00
 * +----------------------------------------------------------------------
 *
 * 存储方案：access + refresh 均存 localStorage。
 *   理由：后端用 Bearer 令牌（非 Cookie 会话），不存在 CSRF 自动携带面，故无需 httpOnly Cookie；
 *         localStorage 便于 SPA 跨标签读取与刷新逻辑。
 *   XSS 权衡：localStorage 可被 XSS 读取——以"不 v-html 不可信内容 + Vue 默认转义 + 不打印令牌到
 *         console（生产）"缓解；令牌绝不进 URL。若将来需更强隔离，可改后端下发 httpOnly Cookie 方案。
 */

const ACCESS_KEY = 'bxap_access_token'
const REFRESH_KEY = 'bxap_refresh_token'

export function getAccessToken(): string {
  return localStorage.getItem(ACCESS_KEY) || ''
}

export function getRefreshToken(): string {
  return localStorage.getItem(REFRESH_KEY) || ''
}

export function setTokens(access: string, refresh?: string): void {
  localStorage.setItem(ACCESS_KEY, access)
  if (refresh) localStorage.setItem(REFRESH_KEY, refresh)
}

export function clearTokens(): void {
  localStorage.removeItem(ACCESS_KEY)
  localStorage.removeItem(REFRESH_KEY)
}
