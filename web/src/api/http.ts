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
	}
	if (!res.ok) {
		const ct = res.headers.get('content-type') || ''
		let detail = ''
		try {
			if (ct.includes('application/json')) {
				const j = await res.json()
				detail = (j as any)?.error || JSON.stringify(j)
			} else {
				detail = await res.text()
			}
		} catch {}
		const msg = detail ? `HTTP ${res.status}: ${detail}` : `HTTP ${res.status}`
		throw new Error(msg)
	}
	if (res.status === 204) return undefined as unknown as T
	return (await res.json()) as T
}


