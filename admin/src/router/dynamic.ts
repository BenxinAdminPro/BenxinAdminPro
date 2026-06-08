/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   动态路由 — 后端菜单树 → Vue Router 路由（组件路径 glob 映射）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 16:00:00
 * +----------------------------------------------------------------------
 */
import type { RouteRecordRaw } from 'vue-router'
import type { MenuNode } from '@/api/authInfo'

// 后端给组件路径字符串（如 "sys/user/index"），前端 glob 映射到实际组件。
const viewModules = import.meta.glob('../views/**/*.vue')

/** 组件路径 → 懒加载组件；空/找不到映射一律降级到占位页（不崩，始终有组件）。 */
function resolveComponent(component: string) {
  const key = `../views/${component}.vue`
  return viewModules[key] || (() => import('../views/placeholder/index.vue'))
}

/** 由 path 生成唯一路由名（'/sys/user' → 'sys-user'）。 */
function routeName(path: string): string {
  return path.replace(/^\//, '').replace(/\//g, '-') || 'home'
}

/**
 * buildRoutesFromMenus 把菜单树（M 目录 / C 菜单 / F 按钮）转成挂在布局容器下的路由。
 * 仅 C 类型生成路由（F 供按钮权限，不生成路由）；children 递归。
 */
export function buildRoutesFromMenus(menus: MenuNode[] | null | undefined): RouteRecordRaw[] {
  const routes: RouteRecordRaw[] = []
  // 后端空菜单返回 JSON null（权限窄的用户如 editor）；null 安全，避免 for...of 抛异常
  const walk = (nodes: MenuNode[] | null | undefined): void => {
    for (const n of nodes || []) {
      if (n.menu_type === 'C' && n.path) {
        routes.push({
          path: n.path.replace(/^\//, ''), // 作为布局容器子路由
          name: routeName(n.path),
          component: resolveComponent(n.component),
          meta: { title: n.name, icon: n.icon, permCode: n.perm_code },
        })
      }
      if (n.children?.length) walk(n.children)
    }
  }
  walk(menus)
  return routes
}
