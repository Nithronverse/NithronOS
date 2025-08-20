import { useEffect, useMemo, useRef, useState } from 'react'

type Share = { id: string; type: 'smb' | 'nfs'; path: string; name: string; ro?: boolean; users?: string[] }

export function Shares() {
	const [shares, setShares] = useState<Share[]>([])
	const [filter, setFilter] = useState<'all' | 'smb' | 'nfs'>('all')
	const [error, setError] = useState<string | null>(null)
	const [show, setShow] = useState(false)

	useEffect(() => { refresh() }, [])

	async function refresh() {
		try {
			const r = await fetch('/api/shares')
			setShares(await r.json())
		} catch (e: any) { setError(e.message) }
	}

	async function del(id: string) {
		await fetch(`/api/shares/${encodeURIComponent(id)}`, { method: 'DELETE', headers: { 'X-CSRF-Token': getCSRF() } })
		refresh()
	}

	const filtered = useMemo(() => shares.filter(s => filter === 'all' ? true : s.type === filter), [shares, filter])

	return (
		<div className="space-y-6">
			<div className="flex items-center justify-between">
				<h1 className="text-2xl font-semibold">Shares</h1>
				<button className="btn bg-primary text-primary-foreground" onClick={() => setShow(true)}>New Share</button>
			</div>
			{error && <div className="text-sm text-red-400">{error}</div>}
			<div className="flex items-center gap-2">
				<label className="text-sm">Filter</label>
				<select className="rounded bg-card p-1" value={filter} onChange={e => setFilter(e.target.value as any)}>
					<option value="all">All</option>
					<option value="smb">SMB</option>
					<option value="nfs">NFS</option>
				</select>
			</div>
			<table className="w-full text-sm">
				<thead className="text-left text-muted-foreground">
					<tr><th>Name</th><th>Type</th><th>Path</th><th>RO</th><th>Users</th><th></th></tr>
				</thead>
				<tbody>
					{filtered.map(s => (
						<tr key={s.id} className="border-t border-muted/20">
							<td className="py-2">{s.name}</td>
							<td className="uppercase">{s.type}</td>
							<td className="font-mono text-xs">{s.path}</td>
							<td>{s.ro ? 'yes' : 'no'}</td>
							<td className="text-xs">{(s.users||[]).join(', ')}</td>
							<td className="text-right"><button className="text-red-400 text-xs" onClick={() => del(s.id)}>Delete</button></td>
						</tr>
					))}
				</tbody>
			</table>

			{show && (
				<ShareModal onClose={() => setShow(false)} onSaved={() => { setShow(false); refresh() }} />
			)}
		</div>
	)
}

function getCSRF(): string {
	const m = document.cookie.match(/(?:^|; )nos_csrf=([^;]*)/)
	return m ? decodeURIComponent(m[1]) : ''
}

function ShareModal({ onClose, onSaved }: { onClose: () => void; onSaved: () => void }) {
  const [type, setType] = useState<'smb'|'nfs'>('smb')
  const [path, setPath] = useState('')
  const [name, setName] = useState('')
  const [ro, setRO] = useState(false)
  const [users, setUsers] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [poolPaths, setPoolPaths] = useState<string[]>([])
  const [dirs, setDirs] = useState<string[]>([])
  const [loadingDirs, setLoadingDirs] = useState(false)
  const debounceRef = useRef<number | undefined>(undefined)
  const [currentDir, setCurrentDir] = useState('')

  useEffect(() => {
    fetch('/api/pools')
      .then(r => r.json())
      .then((pools: any[]) => {
        const mounts = pools.map(p => p.mount || p.id).filter(Boolean)
        setPoolPaths(mounts)
      })
      .catch(() => {})
  }, [])

  async function browse(p?: string) {
    const base = p || path || poolPaths[0]
    if (!base) return
    setLoadingDirs(true)
    try {
      const r = await fetch(`/api/fs/list?path=${encodeURIComponent(base)}`)
      const j = await r.json()
      setDirs(j.dirs || [])
      setCurrentDir(base)
    } finally {
      setLoadingDirs(false)
    }
  }

  useEffect(() => {
    // Debounce browsing when path changes manually
    if (!path) return
    if (debounceRef.current) window.clearTimeout(debounceRef.current)
    debounceRef.current = window.setTimeout(() => {
      browse(path)
    }, 300)
    return () => {
      if (debounceRef.current) window.clearTimeout(debounceRef.current)
    }
  }, [path])

  async function save() {
    setErr(null)
    if (!name.match(/^[a-zA-Z0-9_-]{1,32}$/)) { setErr('Invalid name'); return }
    if (!path.startsWith('/mnt') && !path.startsWith('/srv')) { setErr('Path must be under /mnt or /srv'); return }
    const body: any = { type, path, name, ro }
    if (type === 'smb') body.users = users.split(',').map(s => s.trim()).filter(Boolean)
    setSaving(true)
    const res = await fetch('/api/shares', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCSRF() }, body: JSON.stringify(body) })
    setSaving(false)
    if (!res.ok) {
      try {
        const j = await res.json()
        setErr(j.error || `Failed to create share (HTTP ${res.status})`)
      } catch {
        setErr(`Failed to create share (HTTP ${res.status})`)
      }
      return
    }
    onSaved()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="w-full max-w-lg rounded-lg bg-card p-4 shadow-lg">
        <div className="mb-3 flex items-center justify-between">
          <h3 className="text-lg font-medium">New Share</h3>
          <button onClick={onClose} className="text-sm text-muted-foreground">Close</button>
        </div>
        {err && <div className="mb-2 text-sm text-red-400">{err}</div>}
        <div className="space-y-3">
          <div>
            <label className="mb-1 block text-sm">Type</label>
            <select className="w-full rounded bg-background p-2" value={type} onChange={e => setType(e.target.value as any)}>
              <option value="smb">SMB</option>
              <option value="nfs">NFS</option>
            </select>
          </div>
          <div>
            <label className="mb-1 block text-sm">Path</label>
            <input list="paths-list" className="w-full rounded bg-background p-2" placeholder="/mnt/pool/data" value={path} onChange={e => setPath(e.target.value)} />
            <datalist id="paths-list">
              {poolPaths.map((m) => (
                <option key={m} value={m.endsWith('/') ? m : m + '/'} />
              ))}
            </datalist>
            {poolPaths.length > 0 && (
              <div className="mt-2 flex flex-wrap gap-2 text-xs">
                {poolPaths.map((m) => (
                  <button key={m} type="button" className="rounded bg-card px-2 py-1" onClick={() => { const v = m.endsWith('/') ? m : m + '/'; setPath(v); browse(v) }}>{m}</button>
                ))}
              </div>
            )}
            {(loadingDirs || dirs.length > 0) && (
              <div className="mt-2">
                <div className="mb-1 flex items-center justify-between text-xs">
                  <div className="flex flex-wrap gap-1">
                    {breadcrumb(currentDir).map((seg, idx) => (
                      <span key={idx} className="flex items-center gap-1">
                        {idx > 0 && <span className="text-muted-foreground">/</span>}
                        <button className="text-primary" type="button" onClick={() => browse(seg.path)}>{seg.name || '/'}</button>
                      </span>
                    ))}
                  </div>
                  <button className="text-primary" type="button" onClick={() => goUp(currentDir, browse)}>Up</button>
                </div>
                {loadingDirs ? (
                  <div className="space-y-2 p-2">
                    <div className="h-3 w-1/2 animate-pulse rounded bg-muted" />
                    <div className="h-3 w-2/3 animate-pulse rounded bg-muted" />
                    <div className="h-3 w-1/3 animate-pulse rounded bg-muted" />
                  </div>
                ) : (
                  <div className="max-h-40 overflow-auto rounded border border-muted/20">
                    {dirs.map((d) => (
                      <div key={d} className="flex items-center justify-between border-b border-muted/10 px-2 py-1 text-xs">
                        <span className="font-mono">{d}</span>
                        <div className="flex gap-2">
                          <button className="text-primary" type="button" onClick={() => browse(d)}>Open</button>
                          <button className="text-primary" type="button" onClick={() => setPath(d.endsWith('/') ? d : d + '/')}>Use</button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
          <div>
            <label className="mb-1 block text-sm">Name</label>
            <input className="w-full rounded bg-background p-2" placeholder="share name" value={name} onChange={e => setName(e.target.value)} />
          </div>
          <div className="flex items-center gap-2">
            <input type="checkbox" checked={ro} onChange={e => setRO(e.target.checked)} />
            <span className="text-sm">Read-only</span>
          </div>
          {type === 'smb' && (
            <div>
              <label className="mb-1 block text-sm">SMB Users (comma-separated)</label>
              <input className="w-full rounded bg-background p-2" placeholder="alice,bob" value={users} onChange={e => setUsers(e.target.value)} />
            </div>
          )}
          <div className="flex justify-end gap-2">
            <button className="btn bg-card" onClick={onClose}>Cancel</button>
            <button className="btn bg-primary text-primary-foreground disabled:opacity-60" disabled={saving} onClick={save}>{saving ? 'Creating...' : 'Create'}</button>
          </div>
        </div>
      </div>
    </div>
  )
}

function breadcrumb(path: string): { name: string; path: string }[] {
  if (!path) return []
  const norm = path.replace(/\\+/g, '/').replace(/\/+$/, '')
  const parts = norm.split('/').filter(Boolean)
  const out: { name: string; path: string }[] = []
  let acc = ''
  for (const p of parts) {
    acc += '/' + p
    out.push({ name: p, path: acc })
  }
  return [{ name: '', path: '/' }, ...out]
}

function goUp(path: string, cb: (p: string) => void) {
  if (!path || path === '/') return
  const norm = path.replace(/\\+/g, '/').replace(/\/+$/, '')
  const idx = norm.lastIndexOf('/')
  const up = idx > 0 ? norm.slice(0, idx) : '/'
  cb(up)
}


