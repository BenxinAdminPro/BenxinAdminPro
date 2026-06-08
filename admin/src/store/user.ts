/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   用户状态 — 令牌 / 用户名 / 菜单 / 权限码 / 登录登出（令牌持久化 localStorage）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 14:00:00
 * | @updated   2026-06-08 16:00:00  T-007b：加菜单树/权限码/routesReady + loadAuthInfo + hasPerm
 * +----------------------------------------------------------------------
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as loginApi, logout as logoutApi, type LoginPayload } from '@/api/auth'
import { getMenus, getPerms, type MenuNode } from '@/api/authInfo'
import { getAccessToken, setTokens, clearTokens } from '@/utils/auth'

const USERNAME_KEY = 'bxap_username'

export const useUserStore = defineStore('user', () => {
  const accessToken = ref(getAccessToken())
  const username = ref(localStorage.getItem(USERNAME_KEY) || '')
  const menuTree = ref<MenuNode[]>([])
  const permCodes = ref<string[]>([])
  // 动态路由是否已注册（刷新后由路由守卫据此重建，避免白屏）
  const routesReady = ref(false)

  const isLoggedIn = computed(() => !!accessToken.value)

  /** 是否拥有某权限码（v-permission 用）。 */
  function hasPerm(code: string): boolean {
    return permCodes.value.includes(code)
  }

  /** 登录：取令牌持久化 + 记录用户名。成功 resolve，失败抛出交登录页处理错误码。 */
  async function loginAction(payload: LoginPayload): Promise<void> {
    const pair = await loginApi(payload)
    setTokens(pair.access_token, pair.refresh_token)
    accessToken.value = pair.access_token
    username.value = payload.username
    localStorage.setItem(USERNAME_KEY, payload.username)
  }

  /** 拉取菜单 + 权限码（登录后 / 刷新重建路由前）。后端空集返回 JSON null，统一兜成 []。 */
  async function loadAuthInfo(): Promise<MenuNode[]> {
    const [menus, perms] = await Promise.all([getMenus(), getPerms()])
    menuTree.value = menus || []
    permCodes.value = perms || []
    return menuTree.value
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
    menuTree.value = []
    permCodes.value = []
    routesReady.value = false
  }

  return {
    accessToken, username, menuTree, permCodes, routesReady,
    isLoggedIn, hasPerm, loginAction, loadAuthInfo, logoutAction, reset,
  }
})
