/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   请求层 — Axios 实例 + 拦截器（JWT 注入 / 包络解包 / 错误码 / access 过期自动刷新一次 / hashid 透传）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 14:00:00
 * | @updated   2026-06-10 17:19:28  T-007f：blob 响应透传（鉴权下载流不走包络解包）+ blob 错误体解析友好 message
 * +----------------------------------------------------------------------
 *
 * 约定（与已验证后端契约对齐）：
 *  - 成功：HTTP 2xx + 包络 {code:0, data}；拦截器解包后直接返回 data。
 *  - 业务错误：后端以非 2xx HTTP + 包络 {code,message} 返回，走 axios error 分支。
 *  - 401：access 失效/过期。若非 /auth/* 端点且有 refresh，则刷新并重试一次；失败跳登录。
 *         注意后端"凭证错误"也是 401，但发生在 /auth/login，已被 isAuthUrl 排除，不会误触发刷新。
 *  - 403：无权（ERR_FORBIDDEN）→ 友好提示，不跳登录。
 *  - 423：账号锁定 → 提示。其余非 0 → toast message。
 *  - hashid：ID 字段为字符串 hashid，前端原样透传，不解码。
 *  - 加密信封（CBC）：admin 不接（C 端 uni 才用），此处不预留请求加密逻辑。
 */
import axios, {
  type AxiosInstance,
  type AxiosRequestConfig,
  type InternalAxiosRequestConfig,
  type AxiosError,
} from 'axios'
import { ElMessage } from 'element-plus'
import type { ApiEnvelope } from './types'
import { getAccessToken, getRefreshToken, setTokens, clearTokens } from '@/utils/auth'

/** 带"已重试"标记的请求配置，避免刷新后无限重试。 */
interface RetriableConfig extends InternalAxiosRequestConfig {
  _retry?: boolean
}

const service: AxiosInstance = axios.create({
  // 留空走 Vite 开发代理（同源）；生产由 VITE_API_BASE_URL 指向后端
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
  timeout: 15000,
})

/** 是否认证端点（这些端点的 401 表示凭证/刷新本身失败，不应再触发刷新重试）。 */
function isAuthUrl(url?: string): boolean {
  return !!url && /\/auth\/(login|refresh|captcha)/.test(url)
}

// ---- 请求拦截：注入 JWT ----
service.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = getAccessToken()
  if (token && !isAuthUrl(config.url)) {
    config.headers.set('Authorization', `Bearer ${token}`)
  }
  return config
})

// ---- 令牌刷新（去重：并发 401 共用一次刷新）----
let refreshing: Promise<string> | null = null

async function refreshAccessToken(): Promise<string> {
  const rt = getRefreshToken()
  if (!rt) throw new Error('no refresh token')
  // 刷新走同实例：/auth/refresh 命中 isAuthUrl，不会递归触发刷新；成功后回包络解包为 data
  const data = await service.post<unknown, { access_token: string; refresh_token: string }>(
    '/auth/refresh',
    { refresh_token: rt },
  )
  setTokens(data.access_token, data.refresh_token)
  return data.access_token
}

/** 跳登录（保留来源以便登录后回跳）。延迟取 router，避免模块初始化期循环依赖。 */
async function redirectToLogin(): Promise<void> {
  clearTokens()
  const { default: router } = await import('@/router')
  const current = router.currentRoute.value
  if (current.path !== '/login') {
    router.replace({ path: '/login', query: { redirect: current.fullPath } })
  }
}

// ---- 响应拦截：成功解包 ----
service.interceptors.response.use(
  (response) => {
    // 二进制流（responseType:'blob'，如鉴权下载）：响应体是文件流非 JSON 包络，原样透传
    if (response.config.responseType === 'blob') {
      return response.data
    }
    const env = response.data as ApiEnvelope
    // 后端成功恒为 code==0；理论上不会出现 2xx+非0，稳妥起见兜底提示
    if (env && typeof env.code === 'number' && env.code !== 0) {
      ElMessage.error(env.message || '请求失败')
      return Promise.reject(env)
    }
    return env ? env.data : response.data
  },
  // ---- 响应拦截：错误分流 ----
  async (error: AxiosError<ApiEnvelope>) => {
    const config = error.config as RetriableConfig | undefined
    const status = error.response?.status
    // blob 请求失败时错误体也是 Blob（内容为 JSON 包络）→ 解析出友好 message 再分流
    const raw: unknown = error.response?.data
    let env: ApiEnvelope | undefined
    if (raw instanceof Blob) {
      try {
        env = JSON.parse(await raw.text()) as ApiEnvelope
      } catch {
        // 非 JSON 错误体，保持 undefined 走默认提示
      }
    } else {
      env = raw as ApiEnvelope | undefined
    }
    const message = env?.message

    // 401：access 失效。非认证端点 + 未重试 + 有 refresh → 刷新重试一次
    if (status === 401 && config && !config._retry && !isAuthUrl(config.url) && getRefreshToken()) {
      config._retry = true
      try {
        if (!refreshing) refreshing = refreshAccessToken().finally(() => (refreshing = null))
        const newToken = await refreshing
        config.headers.set('Authorization', `Bearer ${newToken}`)
        return service(config)
      } catch {
        await redirectToLogin()
        return Promise.reject(error)
      }
    }

    // 认证端点（login/refresh/captcha）的错误一律交调用方（登录页）处理，本层不 toast/跳转
    if (isAuthUrl(config?.url)) {
      return Promise.reject(error)
    }

    switch (status) {
      case 401:
        await redirectToLogin()
        break
      case 403:
        ElMessage.error(message || '无权访问该资源')
        break
      case 423:
        ElMessage.error(message || '账号已被锁定，请稍后再试')
        break
      default:
        if (status !== undefined) {
          ElMessage.error(message || '请求失败')
        } else {
          ElMessage.error('网络异常，请检查连接')
        }
    }
    return Promise.reject(error)
  },
)

/** 类型友好的请求方法：返回值即解包后的 data（T）。 */
export const http = {
  get<T = unknown>(url: string, config?: AxiosRequestConfig) {
    return service.get<unknown, T>(url, config)
  },
  post<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig) {
    return service.post<unknown, T>(url, data, config)
  },
  put<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig) {
    return service.put<unknown, T>(url, data, config)
  },
  delete<T = unknown>(url: string, config?: AxiosRequestConfig) {
    return service.delete<unknown, T>(url, config)
  },
}

export default service
