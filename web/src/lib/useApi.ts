import { useCallback, useState } from 'react'

function getCSRFCookie(): string | null {
  const m = document.cookie.match(/(?:^|; )nos_csrf=([^;]*)/)
  return m ? decodeURIComponent(m[1]) : null
}

async function request<T>(path: string, init: RequestInit = {}, retried = false): Promise<T> {
  const csrf = getCSRFCookie() || ''
  const res = await fetch(path, {
    ...init,
    credentials: 'include',
    headers: {
      'Accept': 'application/json',
      ...(init.body ? { 'Content-Type': 'application/json' } : {}),
      ...(csrf ? { 'X-CSRF-Token': csrf } : {}),
      ...(init.headers || {}),
    },
  })
  if (res.status === 401 && !retried) {
    const r = await fetch('/api/auth/refresh', {
      method: 'POST',
      credentials: 'include',
      headers: { ...(csrf ? { 'X-CSRF-Token': csrf } : {}) },
    })
    if (r.ok) return request<T>(path, init, true)
  }
  if (!res.ok) throw new Error(await safeDetail(res))
  if (res.status === 204) return undefined as unknown as T
  return (await res.json()) as T
}

export function useApi() {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const get = useCallback(async function <T>(path: string, init?: RequestInit): Promise<T> {
    setError(null); setLoading(true)
    try {
      return await request<T>(path, { method: 'GET', ...(init || {}) })
    } finally {
      setLoading(false)
    }
  }, [])

  const post = useCallback(async function <T>(path: string, body?: any): Promise<T> {
    setError(null); setLoading(true)
    try {
      return await request<T>(path, { method: 'POST', body: body ? JSON.stringify(body) : undefined })
    } finally {
      setLoading(false)
    }
  }, [])

  const postAuth = useCallback(async function <T>(path: string, token: string, body?: any): Promise<T> {
    setError(null); setLoading(true)
    try {
      return await request<T>(path, { method: 'POST', body: body ? JSON.stringify(body) : undefined, headers: { Authorization: `Bearer ${token}` } })
    } finally {
      setLoading(false)
    }
  }, [])

  return { loading, error, setError, get, post, postAuth }
}

async function safeDetail(res: Response): Promise<string> {
  const status = res.status
  try {
    const ct = res.headers.get('content-type') || ''
    if (ct.includes('application/json')) {
      const j = await res.json()
      const det = (j as any)?.error || JSON.stringify(j)
      return det ? `HTTP ${status}: ${det}` : `HTTP ${status}`
    }
    const t = await res.text()
    return t ? `HTTP ${status}: ${t}` : `HTTP ${status}`
  } catch {
    return `HTTP ${status}`
  }
}


