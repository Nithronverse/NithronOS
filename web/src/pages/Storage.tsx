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
	const [scrubLoading, setScrubLoading] = useState<Record<string, boolean>>({})
	const [scrubStatus, setScrubStatus] = useState<Record<string, string>>({})

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

	async function startScrub(mount: string, key: string) {
		setScrubLoading((m) => ({ ...m, [key]: true }))
		try {
			await api.pools.scrubStart(mount)
			const st = await api.pools.scrubStatus(mount)
			setScrubStatus((m) => ({ ...m, [key]: summarizeScrub(st?.status || '') }))
			try { const { pushToast } = await import('../components/ui/toast'); pushToast('Scrub started') } catch {}
		} catch (e: any) {
			setScrubStatus((m) => ({ ...m, [key]: 'error' }))
			try { const { pushToast } = await import('../components/ui/toast'); pushToast(`Scrub failed: ${e?.message || e}`, 'error') } catch {}
		} finally {
			setScrubLoading((m) => ({ ...m, [key]: false }))
		}
	}

	async function checkScrub(mount: string, key: string) {
		setScrubLoading((m) => ({ ...m, [key]: true }))
		try {
			const st = await api.pools.scrubStatus(mount)
			setScrubStatus((m) => ({ ...m, [key]: summarizeScrub(st?.status || '') }))
		} finally {
			setScrubLoading((m) => ({ ...m, [key]: false }))
		}
	}

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
										<td>{healthBadge(d.smart)}</td>
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
								<th>Maintenance</th>
							</tr>
						</thead>
						<tbody>
							{pools.map((p: any) => {
								const key = p.uuid || p.label || p.mount || ''
								const mount: string | undefined = p.mount || undefined
								return (
									<tr key={key} className="border-t border-muted/20">
										<td className="py-2">{p.label || '-'}</td>
										<td className="font-mono text-xs">{p.uuid || '-'}</td>
										<td className="uppercase">{p.raid || '-'}</td>
										<td>{formatBytes(p.size)}</td>
										<td>{formatBytes(p.used)}</td>
										<td>{formatBytes(p.free)}</td>
										<td className="whitespace-nowrap">
											{mount ? (
												<div className="flex items-center gap-2">
													<button className="rounded border border-muted/30 px-2 py-0.5 text-xs" disabled={!!scrubLoading[key]} onClick={() => startScrub(mount, key)}>
														{scrubLoading[key] ? 'Starting…' : 'Start scrub'}
													</button>
													<button className="rounded border border-muted/30 px-2 py-0.5 text-xs" onClick={() => checkScrub(mount, key)}>Status</button>
													{scrubStatus[key] && <span className="text-xs text-muted-foreground">{scrubStatus[key]}</span>}
												</div>
											) : (
												<span className="text-xs text-muted-foreground">-</span>
											)}
										</td>
									</tr>
								)
							})}
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

function healthBadge(smart?: any) {
	if (!smart) return <span className="text-muted-foreground">-</span>
	const passed = smart.healthy === true
	const temp = typeof smart.temp_c === 'number' ? smart.temp_c : undefined
	let label = 'WARN'
	let color = 'bg-yellow-600'
	if (passed === true) { label = 'PASS'; color = 'bg-green-600' }
	if (passed === false) { label = 'FAIL'; color = 'bg-red-600' }
	return (
		<span className={`inline-flex items-center gap-2 rounded px-2 py-0.5 text-xs text-white ${color}`}>
			{label}{typeof temp === 'number' ? ` · ${temp}°C` : ''}
		</span>
	)
}

function summarizeScrub(text: string): string {
	const s = text.toLowerCase()
	if (s.includes('running')) return 'running'
	if (s.includes('no stats available')) return 'idle'
	if (s.includes('total bytes scrubbed') || s.includes('scrub status for')) return 'done'
	return text.split('\n')[0]?.slice(0, 80) || 'unknown'
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


