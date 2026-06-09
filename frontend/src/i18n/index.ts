import { createI18n } from 'vue-i18n'
import en from './en'
import zh from './zh'

export type Locale = 'en' | 'zh'

function readLang(): Locale {
  try {
    const saved = localStorage.getItem('lang')
    if (saved === 'zh' || saved === 'en') return saved
  } catch {
    /* localStorage unavailable (tests / SSR) */
  }
  return 'en' // English default
}
const initial: Locale = readLang()

export const i18n = createI18n({
  legacy: false,
  globalInjection: true, // $t available in every template
  locale: initial,
  fallbackLocale: 'en',
  messages: { en, zh },
})

if (typeof document !== 'undefined') document.documentElement.lang = initial

export function setLocale(l: Locale) {
  i18n.global.locale.value = l
  try {
    localStorage.setItem('lang', l)
  } catch {
    /* localStorage unavailable */
  }
  if (typeof document !== 'undefined') document.documentElement.lang = l
}

export function currentLocale(): Locale {
  return i18n.global.locale.value as Locale
}
