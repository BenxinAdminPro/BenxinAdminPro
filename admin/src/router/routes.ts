/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   静态路由表 — 公共(登录) + 受保护(布局容器)；动态路由留 T-007b 接入
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 14:00:00
 * +----------------------------------------------------------------------
 */
import type { RouteRecordRaw } from 'vue-router'

/**
 * 常量路由：登录页(公共) + 布局壳 + 首页占位 + 404。
 * T-007b 将基于 /sys/auth/menus 动态 addRoute 业务页到布局容器的 children。
 */
export const constantRoutes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/login/index.vue'),
    meta: { public: true, title: 'login.title' },
  },
  {
    path: '/',
    component: () => import('@/layout/index.vue'),
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'dashboard',
        component: () => import('@/views/home/index.vue'),
        meta: { title: 'layout.dashboard', icon: 'HomeFilled' },
      },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/error/404.vue'),
    meta: { public: true, title: 'error.notFound' },
  },
]
