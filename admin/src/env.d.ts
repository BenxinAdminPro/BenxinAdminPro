/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   类型声明 — Vite 环境变量 + .vue 模块 + UnoCSS 虚拟模块
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 14:00:00
 * +----------------------------------------------------------------------
 */
/// <reference types="vite/client" />

declare module 'virtual:uno.css'

interface ImportMetaEnv {
  /** API 基地址；留空则走 Vite 开发代理（同源 /auth /sys） */
  readonly VITE_API_BASE_URL: string
  /** 应用标题 */
  readonly VITE_APP_TITLE: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<Record<string, unknown>, Record<string, unknown>, unknown>
  export default component
}
