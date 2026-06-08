/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   路由实例 + 守卫 — 登录态校验 + 动态路由按 menus 重建（刷新不白屏）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 14:00:00
 * | @updated   2026-06-08 16:00:00  T-007b：守卫拉 menus 动态 addRoute，刷新后重建避免白屏
 * +----------------------------------------------------------------------
 */
import { createRouter, createWebHistory } from 'vue-router'
import { constantRoutes } from './routes'
import { buildRoutesFromMenus } from './dynamic'
import { getAccessToken } from '@/utils/auth'
import { useUserStore } from '@/store/user'
import i18n from '@/i18n'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: constantRoutes,
  scrollBehavior: () => ({ top: 0 }),
})

const APP_TITLE = import.meta.env.VITE_APP_TITLE || 'BenxinAdminPro'

// 已注册的动态路由移除器：会话重建时先清旧（如登出后换用户登录），避免上个用户路由残留
let dynamicRemovers: Array<() => void> = []

router.beforeEach(async (to) => {
  const loggedIn = !!getAccessToken()
  const isPublic = to.meta.public === true

  if (!loggedIn) {
    return isPublic ? true : { path: '/login', query: { redirect: to.fullPath } }
  }
  if (to.path === '/login') {
    return { path: '/' }
  }

  // 已登录但动态路由未注册（首次进入 / 刷新）：拉 menus 重建路由后再放行
  const userStore = useUserStore()
  if (!userStore.routesReady) {
    try {
      const menus = await userStore.loadAuthInfo()
      // 先清掉上一次会话残留的动态路由，再按当前用户菜单重建
      dynamicRemovers.forEach((remove) => remove())
      dynamicRemovers = []
      for (const r of buildRoutesFromMenus(menus)) {
        if (r.name && router.hasRoute(r.name)) router.removeRoute(r.name)
        dynamicRemovers.push(router.addRoute('layout', r))
      }
      userStore.routesReady = true
      // 重新进入当前地址（新路由已注册，按 fullPath 重新解析）
      return to.fullPath
    } catch (e) {
      // 令牌已失效（拦截器刷新失败会清令牌）→ 回登录；否则（菜单拉取异常但仍登录）
      // 放行到基础页 dashboard，保证任何登录用户都能进入系统、不卡死。
      if (!getAccessToken()) {
        return { path: '/login', query: { redirect: to.fullPath } }
      }
      console.error('[router] load menus failed, fallback to dashboard:', e)
      userStore.routesReady = true
      return { path: '/dashboard' }
    }
  }
  return true
})

router.afterEach((to) => {
  const raw = to.meta.title as string | undefined
  // 静态路由 title 是 i18n key；动态路由 title 是菜单名（直接用）
  const title = raw ? (i18n.global.te(raw) ? i18n.global.t(raw) : raw) : ''
  document.title = title ? `${title} - ${APP_TITLE}` : APP_TITLE
})

export default router
