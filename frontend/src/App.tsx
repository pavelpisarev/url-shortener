import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { createLink, listLinks } from './api'
import { useI18n } from './i18n/context'
import { MoonIcon, SunIcon } from './icons'
import LinkRow from './LinkRow'
import type { CreateResult, Link } from './types'

type Theme = 'light' | 'dark'

const STORAGE_KEY = 'theme'

function getInitialTheme(): Theme {
  try {
    return localStorage.getItem(STORAGE_KEY) === 'light' ? 'light' : 'dark'
  } catch {
    return 'dark'
  }
}

function App() {
  const { locale, setLocale, t } = useI18n()
  const [theme, setTheme] = useState<Theme>(getInitialTheme)
  const [links, setLinks] = useState<Link[]>([])
  const [url, setUrl] = useState('')
  const [created, setCreated] = useState<CreateResult | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme)
    localStorage.setItem(STORAGE_KEY, theme)
  }, [theme])

  const loadLinks = useCallback(async () => {
    try {
      const data = await listLinks()
      setLinks(data.items)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('errorGeneric'))
    }
  }, [t])

  useEffect(() => {
    void loadLinks()
  }, [loadLinks])

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError(null)
    setLoading(true)
    try {
      const result = await createLink(url)
      setCreated(result)
      setUrl('')
      await loadLinks()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('errorGeneric'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <main>
      <header className="app-header">
        <div>
          <h1>URL Shortener</h1>
          <p>{t('subtitle')}</p>
        </div>
        <div className="header-actions">
          <button
            type="button"
            className="lang-toggle"
            onClick={() => setLocale(locale === 'en' ? 'ru' : 'en')}
            aria-label={t('langToggle')}
          >
            {locale.toUpperCase()}
          </button>
          <button
            type="button"
            className="theme-toggle"
            onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
            aria-label={theme === 'dark' ? t('themeToLight') : t('themeToDark')}
          >
            {theme === 'dark' ? <SunIcon /> : <MoonIcon />}
          </button>
        </div>
      </header>

      <article>
        <form onSubmit={onSubmit}>
          <label htmlFor="url">
            {t('linkLabel')}
            <input
              id="url"
              type="url"
              name="url"
              placeholder={t('linkPlaceholder')}
              value={url}
              onChange={(event) => setUrl(event.target.value)}
              required
            />
          </label>
          <button type="submit" aria-busy={loading} disabled={loading}>
            {t('submit')}
          </button>
        </form>

        {created && (
          <p>
            {t('done')}{' '}
            <a href={`/r/${created.code}`} target="_blank" rel="noreferrer">
              {created.short_url}
            </a>
          </p>
        )}
        {error && (
          <p role="alert" className="error">
            {error}
          </p>
        )}
      </article>

      <section>
        <h2>{t('linksTitle')}</h2>
        {links.length === 0 ? (
          <p className="muted">{t('emptyLinks')}</p>
        ) : (
          <figure>
            <table>
              <thead>
                <tr>
                  <th>{t('colCode')}</th>
                  <th>{t('colDestination')}</th>
                  <th>{t('colCreated')}</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {links.map((link) => (
                  <LinkRow key={link.code} link={link} />
                ))}
              </tbody>
            </table>
          </figure>
        )}
      </section>
    </main>
  )
}

export default App
