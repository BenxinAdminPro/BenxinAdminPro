/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   路由实例 + 守卫 — 未登录跳登录，已登录访问登录页回首页
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 14:00:00
 * +----------------------------------------------------------------------
 */
import { createRouter, createWebHistory } from 'vue-router'
import { constantRoutes } from './routes'
import { getAccessToken } from '@/utils/auth'
import i18n from '@/i18n'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: constantRoutes,
  scrollBehavior: () => ({ top: 0 }),
})

const APP_TITLE = import.meta.env.VITE_APP_TITLE || 'BenxinAdminPro'

router.beforeEach((to) => {
  // 令牌直接读 localStorage（守卫早于组件，避免依赖 store 初始化时序）
  const loggedIn = !!getAccessToken()
  const isPublic = to.meta.public === true

  if (!loggedIn && !isPublic) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }
  if (loggedIn && to.path === '/login') {
    return { path: '/' }
  }
  return true
})

router.afterEach((to) => {
  const key = to.meta.title as string | undefined
  const title = key && i18n.global.te(key) ? i18n.global.t(key) : ''
  document.title = title ? `${title} - ${APP_TITLE}` : APP_TITLE
})

export default router
