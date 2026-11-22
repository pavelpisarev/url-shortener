import type { CreateResult, Link, Stats } from './types'

const base = '/api'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(base + path, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })

  if (!res.ok) {
    let message = res.statusText
    try {
      const data = (await res.json()) as { error?: string }
      if (data.error) {
        message = data.error
      }
    } catch {
      message = res.statusText
    }
    throw new Error(message)
  }

  return (await res.json()) as T
}

export function createLink(url: string): Promise<CreateResult> {
  return request<CreateResult>('/links', {
    method: 'POST',
    body: JSON.stringify({ url }),
  })
}

export function listLinks(limit = 100): Promise<{ items: Link[] }> {
  return request<{ items: Link[] }>(`/links?limit=${limit}`)
}

export function getStats(code: string): Promise<Stats> {
  return request<Stats>(`/links/${encodeURIComponent(code)}/stats`)
}
