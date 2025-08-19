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


