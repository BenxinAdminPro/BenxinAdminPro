/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   用户状态 — 令牌 / 当前用户名 / 登录登出（令牌持久化 localStorage）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 14:00:00
 * +----------------------------------------------------------------------
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as loginApi, logout as logoutApi, type LoginPayload } from '@/api/auth'
import { getAccessToken, setTokens, clearTokens } from '@/utils/auth'

const USERNAME_KEY = 'bxap_username'

export const useUserStore = defineStore('user', () => {
  const accessToken = ref(getAccessToken())
  const username = ref(localStorage.getItem(USERNAME_KEY) || '')

  const isLoggedIn = computed(() => !!accessToken.value)

  /** 登录：取令牌持久化 + 记录用户名。成功 resolve，失败抛出交登录页处理错误码。 */
  async function loginAction(payload: LoginPayload): Promise<void> {
    const pair = await loginApi(payload)
    setTokens(pair.access_token, pair.refresh_token)
    accessToken.value = pair.access_token
    username.value = payload.username
    localStorage.setItem(USERNAME_KEY, payload.username)
  }

  /** 登出：尽力通知后端拉黑令牌，然后本地彻底清理。 */
  async function logoutAction(): Promise<void> {
    try {
      await logoutApi()
    } catch {
      // 后端登出失败不阻塞本地清理（令牌可能已失效）
    }
    reset()
  }

  function reset(): void {
    clearTokens()
    localStorage.removeItem(USERNAME_KEY)
    accessToken.value = ''
    username.value = ''
  }

  return { accessToken, username, isLoggedIn, loginAction, logoutAction, reset }
})
