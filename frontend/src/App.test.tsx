import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App'
import { createLink, listLinks } from './api'
import { I18nProvider } from './i18n/I18nProvider'
import type { CreateResult, Link } from './types'

vi.mock('./api', () => ({
  createLink: vi.fn(),
  listLinks: vi.fn(),
  getStats: vi.fn(),
}))

const link: Link = {
  code: 'abc1234',
  url: 'https://example.com',
  created_at: '2026-01-02T10:00:00Z',
}

const created: CreateResult = {
  ...link,
  short_url: 'http://short/r/abc1234',
}

function renderApp() {
  return render(
    <I18nProvider>
      <App />
    </I18nProvider>,
  )
}

function fillAndSubmit(url: string) {
  const input = screen.getByLabelText('Link')
  fireEvent.change(input, { target: { value: url } })
  fireEvent.submit(input.closest('form') as HTMLFormElement)
}

beforeEach(() => {
  localStorage.clear()
  localStorage.setItem('lang', 'en')
  vi.mocked(createLink).mockReset()
  vi.mocked(listLinks).mockReset()
})

describe('App', () => {
  it('renders links from the API', async () => {
    vi.mocked(listLinks).mockResolvedValue({ items: [link] })

    renderApp()

    expect(await screen.findByText('abc1234')).toBeInTheDocument()
    expect(screen.getByText('https://example.com')).toBeInTheDocument()
  })

  it('shows the empty state', async () => {
    vi.mocked(listLinks).mockResolvedValue({ items: [] })

    renderApp()

    expect(await screen.findByText('No links yet.')).toBeInTheDocument()
  })

  it('creates a link and shows the short url', async () => {
    vi.mocked(listLinks).mockResolvedValueOnce({ items: [] }).mockResolvedValue({ items: [link] })
    vi.mocked(createLink).mockResolvedValue(created)

    renderApp()
    await screen.findByText('No links yet.')

    fillAndSubmit('https://example.com')

    await waitFor(() => expect(createLink).toHaveBeenCalledWith('https://example.com'))
    expect(await screen.findByRole('link', { name: 'http://short/r/abc1234' })).toBeInTheDocument()
    expect(screen.getByLabelText('Link')).toHaveValue('')
    expect(vi.mocked(listLinks)).toHaveBeenCalledTimes(2)
  })

  it('shows an error when creation fails', async () => {
    vi.mocked(listLinks).mockResolvedValue({ items: [] })
    vi.mocked(createLink).mockRejectedValue(new Error('invalid url'))

    renderApp()
    await screen.findByText('No links yet.')

    fillAndSubmit('nope')

    expect(await screen.findByRole('alert')).toHaveTextContent('invalid url')
  })
})
