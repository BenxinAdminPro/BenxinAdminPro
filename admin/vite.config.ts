/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   Vite 配置 — 路径别名 + UnoCSS + 开发代理（转发 /auth /sys 到后端，免 CORS）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 14:00:00
 * +----------------------------------------------------------------------
 */
import { fileURLToPath, URL } from 'node:url'
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import UnoCSS from 'unocss/vite'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  // 开发代理目标：后端 demo 默认 :8080，可经 .env.local 的 VITE_PROXY_TARGET 覆盖
  const proxyTarget = env.VITE_PROXY_TARGET || 'http://localhost:8080'

  return {
    plugins: [vue(), UnoCSS()],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    server: {
      host: true, // 监听局域网，便于移动端真机调试响应式
      proxy: {
        // 后端公共/受保护路由前缀；baseURL 留空时由这里转发，开发免 CORS
        '/auth': { target: proxyTarget, changeOrigin: true },
        '/sys': { target: proxyTarget, changeOrigin: true },
        '/api': { target: proxyTarget, changeOrigin: true },
      },
    },
  }
})
