/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   vitest 测试配置 — 复用 vite.config 的 @ 别名（解析一致）+ node 环境（纯函数无需 DOM）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-13 17:30:57
 * +----------------------------------------------------------------------
 *
 * 与 vite.config.ts 保持单一别名解析口径：'@' -> ./src（两处须一致，勿漂移）。
 * environment 取 node：当前仅测 utils/tree.ts 纯函数，无需 jsdom（最小依赖原则，
 * 未来若测组件再按需引 jsdom + @vue/test-utils）。
 * import.meta.env.DEV 在 vitest 默认 mode='test' 下为 true → tree.ts 的 devWarn 会触发，
 * 测试可对异常输入断言 console.warn 被调用（见 src/utils/tree.spec.ts）。
 */
import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  test: {
    environment: 'node',
    include: ['src/**/*.spec.ts'],
  },
})
