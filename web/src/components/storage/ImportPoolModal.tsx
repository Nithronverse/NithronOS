import { useEffect, useState } from 'react'
import { Dialog, DialogHeader, DialogTitle } from '../ui/dialog'
import { Badge } from '../ui/badge'
import { api } from '../../api/client'

export function ImportPoolModal({ open, onClose, onImported }: { open: boolean; onClose: () => void; onImported?: () => void }) {
  const [candidates, setCandidates] = useState<any[]>([])
  const [sel, setSel] = useState<string>('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!open) return
    api.pools.candidates().then(setCandidates).catch(async () => {
      // Fallback: derive candidates from disks (minimal)
      try {
        const disks = await fetch('/api/disks').then(r => r.json())
        const list = (disks?.disks || []).filter((d: any) => d.fstype === 'btrfs' && d.mountpoint).map((d: any) => ({ id: d.mountpoint, label: d.mountpoint, uuid: '', devices: [d.path], size: d.size }))
        setCandidates(list)
      } catch { setCandidates([]) }
    })
  }, [open])

  async function doImport() {
    setLoading(true); setError(null)
    try {
      await api.pools.import(sel)
      try { const { pushToast } = await import('../ui/toast'); pushToast('Pool imported') } catch {}
      onImported?.(); onClose()
    } catch (e: any) {
      setError(e?.message || String(e))
      try { const { pushToast } = await import('../ui/toast'); pushToast(`Import failed: ${e?.message || e}`, 'error') } catch {}
    } finally { setLoading(false) }
  }

  return (
    <Dialog open={open} onOpenChange={(v) => !v && onClose()}>
      <DialogHeader>
        <DialogTitle>Import Pool</DialogTitle>
        <button className="text-sm underline" onClick={onClose}>Close</button>
      </DialogHeader>
      <div className="space-y-2 text-sm">
        <label className="block">Select a detected pool</label>
        <select className="w-full rounded border border-muted/30 bg-background px-2 py-1" value={sel} onChange={(e) => setSel(e.target.value)}>
          <option value="">-- choose --</option>
          {candidates.map((c) => (
            <option key={c.uuid || c.id} value={c.uuid || c.id}>
              {c.label || c.id} — {(c.devices||[]).length} devices
            </option>
          ))}
        </select>
        {sel && (
          <div className="text-xs text-muted-foreground">
            {(() => { const c = candidates.find((x) => (x.uuid || x.id) === sel) || {}; return (
              <div className="mt-2 flex flex-wrap items-center gap-2">
                <Badge>{c.raid || 'unknown'}</Badge>
                <Badge variant="outline">{(c.devices||[]).length} devices</Badge>
                {c.size ? <Badge variant="outline">{formatBytes(c.size)}</Badge> : null}
              </div>
            )})()}
          </div>
        )}
      </div>
      {error && <div className="mt-2 rounded border border-red-500/30 bg-red-500/10 p-2 text-red-400 text-xs">{error}</div>}
      <div className="mt-4 flex justify-end gap-2">
        <button className="rounded border border-muted/30 px-3 py-1 text-sm" onClick={onClose}>Cancel</button>
        <button className="rounded bg-primary px-3 py-1 text-sm text-background disabled:opacity-50" onClick={doImport} disabled={!sel || loading}>Import</button>
      </div>
    </Dialog>
  )
}

// no-op helper removed; using api client
function formatBytes(n?: number): string { if (!n || Number.isNaN(n)) return '-'; const u = ['B','KB','MB','GB','TB','PB']; let i=0,v=n; while(v>=1024 && i<u.length-1){v/=1024;i++} return `${v.toFixed(1)} ${u[i]}` }


