import { useEffect, useState } from 'react'
import { api, type Health, type Disks as Lsblk } from '../api/client'
import { useGlobalNotice } from '@/lib/globalNotice'

export function Dashboard() {
	const [health, setHealth] = useState<Health | null>(null)
	const [disks, setDisks] = useState<Lsblk | null>(null)
	const [error, setError] = useState<string | null>(null)
    const { notice } = useGlobalNotice()

	useEffect(() => {
		if (notice) return
		api.health.get().then(setHealth).catch((e) => setError(String(e)))
		api.disks.get().then(setDisks).catch((e) => setError(String(e)))
	}, [notice])

	return (
		<div className="space-y-6">
			<h1 className="text-2xl font-semibold">Dashboard</h1>
			{error && <div className="text-red-400 text-sm">{error}</div>}
			<div className="grid gap-4 md:grid-cols-2">
				<section className="rounded-lg bg-card p-4">
					<h2 className="mb-2 text-lg font-medium">Health</h2>
					<pre className="text-sm text-muted-foreground">{JSON.stringify(health, null, 2)}</pre>
				</section>
				<section className="rounded-lg bg-card p-4">
					<h2 className="mb-2 text-lg font-medium">Disks</h2>
					<pre className="text-sm text-muted-foreground">{JSON.stringify(disks, null, 2)}</pre>
				</section>
			</div>
		</div>
	)
}


