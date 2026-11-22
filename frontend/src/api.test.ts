import { afterEach, describe, expect, it, vi } from 'vitest'
import { createLink, getStats, listLinks } from './api'

function jsonResponse(data: unknown, options: { ok?: boolean; statusText?: string } = {}): Response {
  return {
    ok: options.ok ?? true,
    statusText: options.statusText ?? '',
    json: async () => data,
  } as unknown as Response
}

function stubFetch(result: Response) {
  const fetchMock = vi.fn().mockResolvedValue(result)
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

afterEach(() => {
  vi.unstubAllGlobals()
  vi.clearAllMocks()
})

describe('api', () => {
  it('creates a link', async () => {
    const payload = {
      code: 'abc1234',
      url: 'https://example.com',
      short_url: 'http://short/r/abc1234',
      created_at: '2026-01-02T10:00:00Z',
    }
    const fetchMock = stubFetch(jsonResponse(payload))

    await expect(createLink('https://example.com')).resolves.toEqual(payload)

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [path, init] = fetchMock.mock.calls[0]
    expect(path).toBe('/api/links')
    expect(init).toMatchObject({
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ url: 'https://example.com' }),
    })
  })

  it('surfaces the error message from a JSON body', async () => {
    stubFetch(jsonResponse({ error: 'invalid url' }, { ok: false }))

    await expect(createLink('nope')).rejects.toThrow('invalid url')
  })

  it('falls back to statusText for non-JSON errors', async () => {
    stubFetch({
      ok: false,
      statusText: 'Bad Gateway',
      json: async () => {
        throw new Error('not json')
      },
    } as unknown as Response)

    await expect(listLinks()).rejects.toThrow('Bad Gateway')
  })

  it('encodes the code in the stats path', async () => {
    const fetchMock = stubFetch(jsonResponse({ code: 'a/b', total: 0, days: [] }))

    await getStats('a/b')

    expect(fetchMock.mock.calls[0][0]).toBe('/api/links/a%2Fb/stats')
  })
})
