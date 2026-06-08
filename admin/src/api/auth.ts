/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   认证 API — captcha / login / refresh / logout（对接后端 T-002）
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 14:00:00
 * +----------------------------------------------------------------------
 */
import { http } from '@/request'

/** 图形验证码（后端 auth.Captcha）。 */
export interface Captcha {
  captcha_id: string
  image_base64: string
  expires_in: number
}

/** 登录/刷新返回的令牌对（后端 auth handler）。 */
export interface TokenPair {
  access_token: string
  refresh_token: string
  access_exp: number
  refresh_exp: number
  token_type: string
}

export interface LoginPayload {
  username: string
  password: string
  captcha_id?: string
  captcha_code?: string
}

export function fetchCaptcha() {
  return http.post<Captcha>('/auth/captcha')
}

/** 登录前置检查：服务端告知该用户名当前是否需要验证码（前端按需显示）。 */
export function precheck(username: string) {
  return http.post<{ captcha_required: boolean }>('/auth/precheck', { username })
}

export function login(payload: LoginPayload) {
  return http.post<TokenPair>('/auth/login', payload)
}

export function refresh(refreshToken: string) {
  return http.post<TokenPair>('/auth/refresh', { refresh_token: refreshToken })
}

export function logout() {
  return http.post<Record<string, never>>('/auth/logout')
}
