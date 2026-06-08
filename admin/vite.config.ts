/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   Vite 配置 — 路径别名 + UnoCSS + 开发代理（转发 /auth /sys 到后端，免 CORS）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 14:00:00
 * +----------------------------------------------------------------------
 */
import type { IncomingMessage } from 'node:http'
import { fileURLToPath, URL } from 'node:url'
import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import UnoCSS from 'unocss/vite'

// 代理 bypass：浏览器导航(Accept 含 text/html)的请求不代理、交 Vite 服 SPA 入口；
// 仅 XHR/fetch(API 调用，Accept 非 text/html)走代理转后端。修"刷新前端路由被代理成 404"。
function spaBypass(req: IncomingMessage): string | undefined {
  if (req.headers.accept?.includes('text/html')) {
    return '/index.html'
  }
  return undefined
}

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
        // 后端公共/受保护路由前缀；baseURL 留空时由这里转发，开发免 CORS。
        // bypass：浏览器导航(text/html)请求走 SPA(index.html)，不代理——否则刷新
        // /sys/user 等"前端路由"会被转发到后端导致 404；只有 API 的 XHR 才代理。
        '/auth': { target: proxyTarget, changeOrigin: true, bypass: spaBypass },
        '/sys': { target: proxyTarget, changeOrigin: true, bypass: spaBypass },
        '/api': { target: proxyTarget, changeOrigin: true, bypass: spaBypass },
      },
    },
  }
})
