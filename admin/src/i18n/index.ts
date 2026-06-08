/**
 * +----------------------------------------------------------------------
 * | @project   本心通用管理后台 / BenxinAdminPro
 * | @mission   i18n 框架 — 中文默认 + 预留 en；语言持久化
 * | @author    仗键天涯(daxing)
 * | @email     3442535897@qq.com
 * | @date      2026-06-08 14:00:00
 * +----------------------------------------------------------------------
 */
import { createI18n } from 'vue-i18n'
import zhCN from './locales/zh-CN'
import en from './locales/en'

export type LocaleKey = 'zh-CN' | 'en'
const LOCALE_KEY = 'bxap_locale'

export function getStoredLocale(): LocaleKey {
  const v = localStorage.getItem(LOCALE_KEY)
  return v === 'en' ? 'en' : 'zh-CN'
}

const i18n = createI18n({
  legacy: false,
  locale: getStoredLocale(),
  fallbackLocale: 'zh-CN',
  messages: { 'zh-CN': zhCN, en },
})

export function setLocale(locale: LocaleKey): void {
  i18n.global.locale.value = locale
  localStorage.setItem(LOCALE_KEY, locale)
  document.documentElement.lang = locale
}

export default i18n
