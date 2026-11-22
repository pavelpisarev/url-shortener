import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { I18nContext, type I18nValue } from './context'
import { dictionaries, persistLocale, resolveLocale, type Locale } from './locale'

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocale] = useState<Locale>(resolveLocale)

  useEffect(() => {
    document.documentElement.lang = locale
    persistLocale(locale)
  }, [locale])

  const value = useMemo<I18nValue>(
    () => ({
      locale,
      setLocale,
      t: (key) => dictionaries[locale][key],
    }),
    [locale],
  )

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>
}
