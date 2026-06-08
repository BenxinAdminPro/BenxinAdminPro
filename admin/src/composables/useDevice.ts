/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   响应式设备检测 — 按窗口宽度判定 desktop/tablet/mobile，联动侧边栏
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 14:00:00
 * +----------------------------------------------------------------------
 */
import { watch } from 'vue'
import { useWindowSize } from '@vueuse/core'
import { useAppStore, type DeviceType } from '@/store/app'

const TABLET_MAX = 992 // < 992 视为平板及以下
const MOBILE_MAX = 768 // < 768 视为手机

/** 监听窗口宽度，更新 app store 的 device，并在窄屏自动折叠侧边栏。 */
export function useDevice(): void {
  const app = useAppStore()
  const { width } = useWindowSize()

  function resolve(w: number): DeviceType {
    if (w < MOBILE_MAX) return 'mobile'
    if (w < TABLET_MAX) return 'tablet'
    return 'desktop'
  }

  watch(
    width,
    (w) => {
      const device = resolve(w)
      app.setDevice(device)
      // 手机/平板默认收起侧边栏，桌面展开
      app.setSidebar(device !== 'desktop')
    },
    { immediate: true },
  )
}
