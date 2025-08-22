import { pushToast } from '@/components/ui/toast'

export async function apiGet<T>(path: string, init?: RequestInit): Promise<T> {
	return request<T>(path, { method: 'GET', ...(init || {}) })
}

export async function apiPost<T>(path: string, body?: any): Promise<T> {
	return request<T>(path, { method: 'POST', body: body ? JSON.stringify(body) : undefined })
}

function getCSRFCookie(): string | null {
	const m = document.cookie.match(/(?:^|; )nos_csrf=([^;]*)/)
	return m ? decodeURIComponent(m[1]) : null
}

export async function apiPostAuth<T>(path: string, token: string, body?: any): Promise<T> {
	return request<T>(path, { method: 'POST', body: body ? JSON.stringify(body) : undefined, headers: { Authorization: `Bearer ${token}` } })
}

async function request<T>(path: string, init: RequestInit, retried = false): Promise<T> {
	const isSetup = path.startsWith('/api/setup/')
	const csrf = getCSRFCookie()
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
	// Handle setup 410 gracefully: let callers treat as non-firstBoot
	if (isSetup && res.status === 410) {
		pushToast('Setup already completed. You can sign in.', 'error')
		throw new Error('HTTP 410')
	}
	if (res.status === 401 && !retried) {
		// Try refresh once
		const r = await fetch('/api/auth/refresh', {
			method: 'POST',
			credentials: 'include',
			headers: {
				...(csrf ? { 'X-CSRF-Token': csrf } : {}),
			},
		})
		if (r.ok) {
			return request<T>(path, init, true)
		}
		// refresh failed — log out
		window.location.href = '/login'
	}
	if (!res.ok) {
		const ct = res.headers.get('content-type') || ''
		let message = ''
		let retryAfterSec = 0
		let body: any = undefined
		try {
			if (ct.includes('application/json')) {
				body = await res.json()
				const err = (body as any)?.error
				if (err) {
					message = String(err.message || '')
					retryAfterSec = Number(err.retryAfterSec || 0)
				}
			} else {
				message = await res.text()
			}
		} catch {}
		// Global toasts for common statuses
		if (res.status === 429) {
			const ra = retryAfterSec || parseInt(res.headers.get('Retry-After') || '0', 10) || 0
			pushToast(ra > 0 ? `Rate limited. Try again in ${ra}s` : 'Rate limited. Please try again shortly.', 'error')
		} else if (res.status === 423) {
			pushToast('Account temporarily locked. Please try again later.', 'error')
		} else if (res.status >= 500) {
			pushToast(message || `Request failed (${res.status})`, 'error')
		}
		const msg = message ? `HTTP ${res.status}: ${message}` : `HTTP ${res.status}`
		const error: any = new Error(msg)
		error.status = res.status
		if (body !== undefined) error.data = body
		throw error
	}
	if (res.status === 204) return undefined as unknown as T
	return (await res.json()) as T
}


