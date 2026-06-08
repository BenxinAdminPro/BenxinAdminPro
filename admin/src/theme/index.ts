/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   主题骨架 — 暗色切换（@vueuse useDark，挂 html.dark）+ 主题色覆盖
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 14:00:00
 * +----------------------------------------------------------------------
 *
 * 暗色：useDark 在 <html> 上挂/去 `dark` 类并持久化；Element Plus 暗色 css-vars 即据此生效
 *       （main.ts 引入 element-plus/theme-chalk/dark/css-vars.css）。
 * 主题色：覆盖 --el-color-primary 及其 light/dark 派生变量。
 */
import { useDark, useToggle } from '@vueuse/core'

export const isDark = useDark()
export const toggleDark = useToggle(isDark)

const PRIMARY_KEY = 'bxap_primary'
export const DEFAULT_PRIMARY = '#409eff'

/** 将颜色与白/黑按权重混合，生成 Element Plus 的 light-N / dark-2 派生色。 */
function mix(color: string, mixWith: string, weight: number): string {
  const c = hexToRgb(color)
  const m = hexToRgb(mixWith)
  const r = Math.round(c.r * (1 - weight) + m.r * weight)
  const g = Math.round(c.g * (1 - weight) + m.g * weight)
  const b = Math.round(c.b * (1 - weight) + m.b * weight)
  return rgbToHex(r, g, b)
}

function hexToRgb(hex: string): { r: number; g: number; b: number } {
  const h = hex.replace('#', '')
  const full = h.length === 3 ? h.split('').map((x) => x + x).join('') : h
  return {
    r: parseInt(full.slice(0, 2), 16),
    g: parseInt(full.slice(2, 4), 16),
    b: parseInt(full.slice(4, 6), 16),
  }
}

function rgbToHex(r: number, g: number, b: number): string {
  const toHex = (n: number) => n.toString(16).padStart(2, '0')
  return `#${toHex(r)}${toHex(g)}${toHex(b)}`
}

/** 应用主题色：写入 --el-color-primary 及派生变量，持久化。 */
export function applyPrimaryColor(color: string): void {
  const el = document.documentElement
  el.style.setProperty('--el-color-primary', color)
  for (let i = 1; i <= 9; i++) {
    el.style.setProperty(`--el-color-primary-light-${i}`, mix(color, '#ffffff', i * 0.1))
  }
  el.style.setProperty('--el-color-primary-dark-2', mix(color, '#000000', 0.2))
  localStorage.setItem(PRIMARY_KEY, color)
}

export function getStoredPrimary(): string {
  return localStorage.getItem(PRIMARY_KEY) || DEFAULT_PRIMARY
}

/** 启动时应用持久化的主题色。 */
export function initTheme(): void {
  applyPrimaryColor(getStoredPrimary())
}
