import { useState } from 'react'
import { getStats } from './api'
import { ChevronsDownIcon, ChevronsUpIcon } from './icons'
import { useI18n } from './i18n/context'
import { intlLocale } from './i18n/locale'
import type { Link, Stats } from './types'

function formatDate(value: string, locale: string): string {
  return new Date(value).toLocaleDateString(locale)
}

function formatTime(value: string, locale: string): string {
  return new Date(value).toLocaleTimeString(locale)
}

interface Props {
  link: Link
}

function LinkRow({ link }: Props) {
  const { locale, t } = useI18n()
  const localeTag = intlLocale(locale)
  const [open, setOpen] = useState(false)
  const [stats, setStats] = useState<Stats | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function toggle() {
    const next = !open
    setOpen(next)
    if (!next || stats || loading) {
      return
    }

    setLoading(true)
    setError(null)
    try {
      setStats(await getStats(link.code))
    } catch (err) {
      setError(err instanceof Error ? err.message : t('errorGeneric'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <>
      <tr>
        <td>
          <a href={`/r/${link.code}`} target="_blank" rel="noreferrer">
            {link.code}
          </a>
        </td>
        <td className="url-cell">{link.url}</td>
        <td className="created-cell">
          <span>{formatDate(link.created_at, localeTag)}</span>
          <span className="muted">{formatTime(link.created_at, localeTag)}</span>
        </td>
        <td>
          <button
            type="button"
            className="row-toggle"
            onClick={toggle}
            aria-expanded={open}
            aria-controls={`stats-${link.code}`}
            aria-label={open ? t('hideStats') : t('showStats')}
          >
            {open ? <ChevronsUpIcon /> : <ChevronsDownIcon />}
          </button>
        </td>
      </tr>
      {open && (
        <tr>
          <td colSpan={4}>
            <div id={`stats-${link.code}`} className="stats-panel">
              {loading && <p className="muted">{t('loading')}</p>}
              {error && (
                <p role="alert" className="error">
                  {error}
                </p>
              )}
              {stats && !loading && (
                <>
                  <p>
                    {t('totalClicks')} <strong>{stats.total}</strong>
                  </p>
                  {stats.days.length === 0 ? (
                    <p className="muted">{t('noClicks')}</p>
                  ) : (
                    <div className="stats-scroll">
                      <table>
                        <thead>
                          <tr>
                            <th>{t('colDay')}</th>
                            <th>{t('colClicks')}</th>
                          </tr>
                        </thead>
                        <tbody>
                          {stats.days.map((day) => (
                            <tr key={day.day}>
                              <td>{day.day}</td>
                              <td>{day.count}</td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </>
              )}
            </div>
          </td>
        </tr>
      )}
    </>
  )
}

export default LinkRow
