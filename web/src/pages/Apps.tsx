import { useEffect, useState } from 'react'
import { api, type Apps as AppList } from '../api/client'

export function Apps() {
	const [apps, setApps] = useState<AppList>([])
	const [error, setError] = useState<string | null>(null)

	useEffect(() => {
		api.apps.get().then(setApps).catch((e) => setError(String(e)))
	}, [])

	return (
		<div className="space-y-6">
			<h1 className="text-2xl font-semibold">Apps</h1>
			{error && <div className="text-red-400 text-sm">{error}</div>}
			<div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
				{apps.map((a) => (
					<div key={a.id} className="rounded-lg bg-card p-4">
						<div className="mb-2 text-lg font-medium capitalize">{a.id}</div>
						<div className="mb-3 text-sm text-muted-foreground">Status: {a.status}</div>
						<button className="btn bg-primary text-primary-foreground opacity-60 cursor-not-allowed">
							Install
						</button>
					</div>
				))}
			</div>
		</div>
	)
}


