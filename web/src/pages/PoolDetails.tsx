import { useEffect, useMemo, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { PoolSnapshots } from './PoolSnapshots'

export function PoolDetails() {
  const { id } = useParams()
  const [pools, setPools] = useState<any[]>([])
  const pool = useMemo(() => pools.find((p) => (p.mount || p.id) === id || p.uuid === id), [pools, id])

  useEffect(() => {
    fetch('/api/pools').then((r) => r.json()).then(setPools)
  }, [])

  if (!id) return <div className="p-4">Missing id</div>

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Pool Details</h1>
        <Link to="/storage" className="text-sm text-primary">Back to Storage</Link>
      </div>
      {pool ? (
        <div className="rounded-lg bg-card p-4 space-y-2">
          <div className="grid gap-2 md:grid-cols-3">
            <Field label="Label" value={pool.label || '-'} />
            <Field label="UUID" value={pool.uuid || '-'} mono />
            <Field label="RAID" value={(pool.raid || '-').toUpperCase()} />
            <Field label="Mount" value={pool.mount || pool.id} mono />
            <Field label="Size" value={formatBytes(pool.size)} />
            <Field label="Used / Free" value={`${formatBytes(pool.used)} / ${formatBytes(pool.free)}`} />
          </div>
        </div>
      ) : (
        <div className="text-sm text-muted-foreground">Loading...</div>
      )}

      <section className="space-y-3">
        <h2 className="text-lg font-medium">Snapshots</h2>
        <PoolSnapshots id={id} />
      </section>
    </div>
  )
}

function Field({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className={mono ? 'font-mono text-xs' : ''}>{value}</div>
    </div>
  )
}

function formatBytes(n: number): string {
  if (!n || Number.isNaN(n as any)) return '-'
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) { v/=1024; i++ }
  return `${v.toFixed(1)} ${units[i]}`
}


