/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   应用 UI 状态 — 侧边栏折叠 / 设备类型（响应式）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 14:00:00
 * +----------------------------------------------------------------------
 */
import { defineStore } from 'pinia'
import { ref } from 'vue'

export type DeviceType = 'desktop' | 'tablet' | 'mobile'

export const useAppStore = defineStore('app', () => {
  const sidebarCollapsed = ref(false)
  const device = ref<DeviceType>('desktop')

  function toggleSidebar(): void {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  function setSidebar(collapsed: boolean): void {
    sidebarCollapsed.value = collapsed
  }

  function setDevice(d: DeviceType): void {
    device.value = d
  }

  return { sidebarCollapsed, device, toggleSidebar, setSidebar, setDevice }
})
