/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   应用入口 — 装配 UnoCSS / Element Plus / Pinia / Router / i18n / 主题
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 14:00:00
 * +----------------------------------------------------------------------
 */
import { createApp } from 'vue'
import 'virtual:uno.css'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import 'element-plus/theme-chalk/dark/css-vars.css'

import App from './App.vue'
import router from './router'
import pinia from './store'
import i18n from './i18n'
import { initTheme } from './theme'
import { permission } from './directives/permission'
import './styles/index.scss'

initTheme()

const app = createApp(App)
app.use(pinia)
app.use(router)
app.use(i18n)
app.use(ElementPlus)
app.directive('permission', permission)
app.mount('#app')
