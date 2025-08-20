import { useEffect, useMemo, useState } from 'react'
import { api, type Disks } from '../api/client'

export function Storage() {
	const [disks, setDisks] = useState<Disks | null>(null)
	const [loading, setLoading] = useState(true)
	const [sortKey, setSortKey] = useState<'name' | 'size'>('name')
	const [sortAsc, setSortAsc] = useState(true)

	useEffect(() => {
		api.disks.get().then(setDisks).finally(() => setLoading(false))
	}, [])

	const sorted = useMemo(() => {
		const rows = (disks?.disks ?? []).slice()
		rows.sort((a: any, b: any) => {
			const va = sortKey === 'size' ? Number(a.size) : String(a.name)
			const vb = sortKey === 'size' ? Number(b.size) : String(b.name)
			if (va < vb) return sortAsc ? -1 : 1
			if (va > vb) return sortAsc ? 1 : -1
			return 0
		})
		return rows
	}, [disks, sortKey, sortAsc])

	return (
		<div className="space-y-6">
			<h1 className="text-2xl font-semibold">Storage</h1>
			<div className="rounded-lg bg-card p-4">
				<div className="mb-4 flex items-center justify-between">
					<h2 className="text-lg font-medium">Disks</h2>
				</div>
				{loading ? (
					<div className="space-y-2">
						<div className="h-4 w-1/3 animate-pulse rounded bg-muted" />
						<div className="h-4 w-1/2 animate-pulse rounded bg-muted" />
						<div className="h-4 w-2/3 animate-pulse rounded bg-muted" />
					</div>
				) : (
					<div className="overflow-x-auto">
						<table className="w-full text-sm">
							<thead className="text-left text-muted-foreground">
								<tr>
									<th className="cursor-pointer" onClick={() => (setSortKey('name'), setSortAsc(sortKey === 'name' ? !sortAsc : true))}>Name</th>
									<th>Model</th>
									<th>Serial</th>
									<th className="cursor-pointer" onClick={() => (setSortKey('size'), setSortAsc(sortKey === 'size' ? !sortAsc : true))}>Size</th>
									<th>Type</th>
									<th>Transport</th>
									<th>Mount</th>
									<th>FS</th>
									<th>Health</th>
								</tr>
							</thead>
							<tbody>
								{sorted.map((d: any) => (
									<tr key={d.path || d.name} className="border-t border-muted/20">
										<td className="py-2">{d.name}</td>
										<td>{d.model || '-'}</td>
										<td>{d.serial || '-'}</td>
										<td>{formatBytes(Number(d.size))}</td>
										<td className="uppercase">{d.type}</td>
										<td className="uppercase">{d.tran || '-'}</td>
										<td>{d.mountpoint || '-'}</td>
										<td>{d.fstype || '-'}</td>
										<td>{healthDot(d.smart?.healthy)}</td>
									</tr>
								))}
							</tbody>
						</table>
					</div>
				)}
			</div>

			<div className="rounded-lg bg-card p-4">
				<div className="mb-4 flex items-center justify-between">
					<h2 className="text-lg font-medium">Pools</h2>
					<button className="btn bg-primary text-primary-foreground opacity-60 cursor-not-allowed">
						Create Pool
					</button>
				</div>
				<div className="text-sm text-muted-foreground">No pools yet.</div>
			</div>
		</div>
	)
}

function formatBytes(n: number): string {
	if (!n || Number.isNaN(n)) return '-'
	const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
	let i = 0
	let v = n
	while (v >= 1024 && i < units.length - 1) {
		v /= 1024
		i++
	}
	return `${v.toFixed(1)} ${units[i]}`
}

function healthDot(healthy?: boolean) {
	const color = healthy === true ? 'bg-green-500' : 'bg-muted'
	return <span className={`inline-block h-2.5 w-2.5 rounded-full ${color}`} />
}


