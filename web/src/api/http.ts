export async function apiGet<T>(path: string, init?: RequestInit): Promise<T> {
	const res = await fetch(path, {
		...init,
		headers: {
			'Accept': 'application/json',
			...(init?.headers || {}),
		},
	})
	if (!res.ok) {
		throw new Error(`HTTP ${res.status}`)
	}
	return (await res.json()) as T
}

export async function apiPost<T>(path: string, body?: any): Promise<T> {
	const csrf = getCSRFCookie()
	const res = await fetch(path, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
			'X-CSRF-Token': csrf ?? '',
		},
		body: body ? JSON.stringify(body) : undefined,
	})
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
	return (await res.json()) as T
}

function getCSRFCookie(): string | null {
	const m = document.cookie.match(/(?:^|; )nos_csrf=([^;]*)/)
	return m ? decodeURIComponent(m[1]) : null
}


