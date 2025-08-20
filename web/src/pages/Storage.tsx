import { useEffect, useMemo, useState } from 'react'
import { api, type Disks } from '../api/client'
import { CreatePoolWizard } from '../components/storage/CreatePoolWizard'
import { ImportPoolModal } from '../components/storage/ImportPoolModal'
import { Dialog, DialogHeader, DialogTitle } from '../components/ui/dialog'

export function Storage() {
	const [disks, setDisks] = useState<Disks | null>(null)
	const { pools, refresh: refreshPools, loadingPools } = usePools()
	const [loading, setLoading] = useState(true)
	const [sortKey, setSortKey] = useState<'name' | 'size'>('name')
	const [sortAsc, setSortAsc] = useState(true)
	const [showCreate, setShowCreate] = useState(false)
	const [showImport, setShowImport] = useState(false)

	useEffect(() => {
		Promise.all([
			api.disks.get().then(setDisks),
			refreshPools(),
		]).finally(() => setLoading(false))
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
					<div className="flex gap-2">
						<button className="rounded bg-primary px-3 py-1 text-sm text-background" onClick={() => setShowCreate(true)}>Create Pool</button>
						<button className="rounded border border-muted/30 px-3 py-1 text-sm" onClick={() => setShowImport(true)}>Import Pool</button>
					</div>
				</div>
				{loadingPools ? (
					<div className="space-y-2">
						<div className="h-4 w-1/3 animate-pulse rounded bg-muted" />
						<div className="h-4 w-1/2 animate-pulse rounded bg-muted" />
					</div>
				) : pools.length === 0 ? (
					<div className="text-sm text-muted-foreground">No pools yet.</div>
				) : (
					<table className="w-full text-sm">
						<thead className="text-left text-muted-foreground">
							<tr>
								<th>Label</th>
								<th>UUID</th>
								<th>RAID</th>
								<th>Size</th>
								<th>Used</th>
								<th>Free</th>
							</tr>
						</thead>
						<tbody>
							{pools.map((p: any) => (
								<tr key={p.uuid || p.label} className="border-t border-muted/20">
									<td className="py-2">{p.label || '-'}</td>
									<td className="font-mono text-xs">{p.uuid || '-'}</td>
									<td className="uppercase">{p.raid || '-'}</td>
									<td>{formatBytes(p.size)}</td>
									<td>{formatBytes(p.used)}</td>
									<td>{formatBytes(p.free)}</td>
								</tr>
							))}
						</tbody>
					</table>
				)}
			</div>

			<Dialog open={showCreate} onOpenChange={setShowCreate}>
				<DialogHeader><DialogTitle>Create Pool</DialogTitle><button className="text-sm underline" onClick={() => setShowCreate(false)}>Close</button></DialogHeader>
				<CreatePoolWizard onCreated={() => { setShowCreate(false); refreshPools() }} />
			</Dialog>
			<ImportPoolModal open={showImport} onClose={() => setShowImport(false)} onImported={() => { setShowImport(false); refreshPools() }} />
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

function usePools() {
  const [pools, setPools] = useState<any[]>([])
  const [loadingPools, setLoading] = useState(false)
  const refresh = async () => {
    setLoading(true)
    try { const r = await fetch('/api/pools'); setPools(await r.json()) } finally { setLoading(false) }
  }
  return { pools, refresh, loadingPools }
}


