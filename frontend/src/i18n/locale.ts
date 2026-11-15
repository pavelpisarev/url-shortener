import { en, type TranslationKey } from './en'
import { ru } from './ru'

export type Locale = 'en' | 'ru'

export const dictionaries: Record<Locale, Record<TranslationKey, string>> = { en, ru }

const STORAGE_KEY = 'lang'

function readSavedLocale(): Locale | null {
  try {
    const value = localStorage.getItem(STORAGE_KEY)
    return value === 'en' || value === 'ru' ? value : null
  } catch {
    return null
  }
}

export function persistLocale(locale: Locale) {
  try {
    localStorage.setItem(STORAGE_KEY, locale)
  } catch {
    return
  }
}

export function resolveLocale(): Locale {
  const saved = readSavedLocale()
  if (saved) {
    return saved
  }

  const languages = navigator.languages?.length ? navigator.languages : [navigator.language]
  for (const language of languages) {
    if (language.toLowerCase().startsWith('ru')) {
      return 'ru'
    }
  }
  return 'en'
}

export function intlLocale(locale: Locale): string {
  return locale === 'ru' ? 'ru-RU' : 'en-US'
}
