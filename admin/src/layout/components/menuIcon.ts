/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   菜单图标解析 — 后端图标名（字符串）→ Element Plus 图标组件，缺省兜底
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 16:00:00
 * | @updated   2026-06-10 12:02:21  T-007e：补 document/monitor 图标映射（操作/登录日志菜单）
 * | @updated   2026-06-12 14:24:57  T-007h：补 postcard 图标映射（岗位管理菜单）
 * +----------------------------------------------------------------------
 */
import type { Component } from 'vue'
import {
  Setting,
  User,
  UserFilled,
  Share,
  Grid,
  Collection,
  Edit,
  Document,
  Menu as MenuIcon,
  Files,
  Folder,
  Monitor,
  Postcard,
} from '@element-plus/icons-vue'

// 后端 seed 用的是一套小写图标名，这里映射到 Element Plus 图标；未知名兜底 Document。
const iconMap: Record<string, Component> = {
  setting: Setting,
  user: User,
  peoples: UserFilled,
  tree: Share,
  'tree-table': Grid,
  dict: Collection,
  edit: Edit,
  menu: MenuIcon,
  file: Files,
  folder: Folder,
  document: Document,
  monitor: Monitor,
  postcard: Postcard,
}

export function resolveIcon(name: string): Component {
  return iconMap[name] || Document
}
