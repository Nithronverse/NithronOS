import { useEffect, useState } from 'react'
import { api, type Apps as AppList } from '../api/client'

export function Apps() {
	const [apps, setApps] = useState<AppList>([])
	const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState<string | null>(null)
  const [showCfg, setShowCfg] = useState<string | null>(null)

	useEffect(() => {
		api.apps.get().then(setApps).catch((e) => setError(String(e)))
	}, [])

  async function install(id: string, cfg?: any) {
    setBusy(id)
    try {
      const res = await fetch('/api/apps/install', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id, config: cfg||{} }) })
      if (!res.ok) throw new Error('Install failed')
      const list = await api.apps.get()
      setApps(list)
    } catch (e: any) {
      setError(String(e.message || e))
    } finally { setBusy(null); setShowCfg(null) }
  }

  async function uninstall(id: string) {
    if (!confirm(`Uninstall ${id}?`)) return
    setBusy(id)
    try {
      const res = await fetch('/api/apps/uninstall', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ id }) })
      if (!res.ok) throw new Error('Uninstall failed')
      const list = await api.apps.get()
      setApps(list)
    } catch (e: any) {
      setError(String(e.message || e))
    } finally { setBusy(null) }
  }

	return (
		<div className="space-y-6">
			<h1 className="text-2xl font-semibold">Apps</h1>
			{error && <div className="text-red-400 text-sm">{error}</div>}
			<div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
				{apps.map((a) => (
					<div key={a.id} className="rounded-lg bg-card p-4">
						<div className="mb-2 text-lg font-medium capitalize">{a.id}</div>
						<div className="mb-3 text-sm text-muted-foreground">Status: {a.status}</div>
						<div className="flex gap-2">
							{a.status === 'installed' ? (
								<button className="btn bg-card" disabled={busy===a.id} onClick={() => uninstall(a.id)}>{busy===a.id? 'Uninstalling...' : 'Uninstall'}</button>
							) : (
								<>
									<button className="btn bg-primary text-primary-foreground" disabled={busy===a.id} onClick={() => install(a.id)}>{busy===a.id? 'Installing...' : 'Install'}</button>
									<button className="btn bg-card" onClick={() => setShowCfg(a.id)}>Configure</button>
								</>
							)}
						</div>
					</div>
				))}
			</div>
      {showCfg && (
        <ConfigModal id={showCfg} onClose={() => setShowCfg(null)} onSave={(cfg) => install(showCfg, cfg)} />
      )}
		</div>
	)
}

function ConfigModal({ id, onClose, onSave }: { id: string; onClose: () => void; onSave: (cfg: any) => void }) {
  const [ports, setPorts] = useState('')
  const [dataDir, setDataDir] = useState('')
  const [mediaDir, setMediaDir] = useState('')
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="w-full max-w-lg rounded-lg bg-card p-4 shadow-lg">
        <div className="mb-3 flex items-center justify-between">
          <h3 className="text-lg font-medium">Configure {id}</h3>
          <button onClick={onClose} className="text-sm text-muted-foreground">Close</button>
        </div>
        <div className="space-y-3">
          <div>
            <label className="mb-1 block text-sm">Ports (e.g., 8080:8080, 2283:2283)</label>
            <input className="w-full rounded bg-background p-2" value={ports} onChange={e=>setPorts(e.target.value)} />
          </div>
          <div>
            <label className="mb-1 block text-sm">Data Dir (e.g., /srv/data/app)</label>
            <input className="w-full rounded bg-background p-2" value={dataDir} onChange={e=>setDataDir(e.target.value)} />
          </div>
          <div>
            <label className="mb-1 block text-sm">Media Dir (optional, e.g., /srv/media)</label>
            <input className="w-full rounded bg-background p-2" value={mediaDir} onChange={e=>setMediaDir(e.target.value)} />
          </div>
          <div className="flex justify-end gap-2">
            <button className="btn bg-card" onClick={onClose}>Cancel</button>
            <button className="btn bg-primary text-primary-foreground" onClick={() => onSave({ ports, dataDir, mediaDir })}>Save & Install</button>
          </div>
        </div>
      </div>
    </div>
  )
}


