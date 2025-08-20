import { useEffect, useState } from 'react'
import { api } from '../api/client'
import { apiPost } from '../api/http'

export function PoolsCreate() {
  const [label, setLabel] = useState('pool1')
  const [devices, setDevices] = useState<string[]>([])
  const [raid, setRaid] = useState<'single'|'raid1'|'raid10'>('raid1')
  const [diskList, setDiskList] = useState<any[]>([])
  const [plan, setPlan] = useState<string[]>([])
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api.disks.get().then((d: any) => setDiskList(d.disks || []))
  }, [])

  async function doPlan() {
    setError(null)
    try {
      const res = await apiPost<any>('/api/pools/plan-create', { label, devices, raid })
      setPlan(res.plan || res.Plan || [])
    } catch (e: any) {
      setError(e.message)
    }
  }

  async function doCreate() {
    setError(null)
    try {
      await fetch('/api/pools/create', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCSRF(), 'Confirm': 'yes' }, body: JSON.stringify({ label, devices, raid }) })
    } catch (e: any) {
      setError(e.message)
    }
  }

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">Create Pool</h1>
      {error && <div className="text-sm text-red-400">{error}</div>}
      <div className="rounded-lg bg-card p-4 space-y-3">
        <div>
          <label className="block text-sm mb-1">Label</label>
          <input className="w-full rounded bg-background p-2" value={label} onChange={(e) => setLabel(e.target.value)} />
        </div>
        <div>
          <label className="block text-sm mb-1">RAID</label>
          <select className="w-full rounded bg-background p-2" value={raid} onChange={(e) => setRaid(e.target.value as any)}>
            <option value="single">single</option>
            <option value="raid1">raid1</option>
            <option value="raid10">raid10</option>
          </select>
        </div>
        <div>
          <label className="block text-sm mb-1">Devices</label>
          <div className="space-y-1">
            {diskList.map((d) => (
              <label key={d.path || d.name} className="flex items-center gap-2">
                <input type="checkbox" checked={devices.includes(d.path)} onChange={(e) => {
                  const p = d.path || d.name
                  setDevices((prev) => e.target.checked ? [...prev, p] : prev.filter((x) => x !== p))
                }} />
                <span>{d.name} <span className="text-muted-foreground">({formatBytes(Number(d.size))})</span></span>
              </label>
            ))}
          </div>
        </div>
        <div className="flex gap-2">
          <button className="btn bg-primary text-primary-foreground" onClick={doPlan}>Plan</button>
          <button className="btn bg-primary text-primary-foreground" onClick={doCreate}>Create</button>
        </div>
      </div>
      {plan.length > 0 && (
        <div className="rounded-lg bg-card p-4">
          <h2 className="mb-2 text-lg font-medium">Plan</h2>
          <pre className="text-xs text-muted-foreground whitespace-pre-wrap">{plan.join('\n')}</pre>
        </div>
      )}
    </div>
  )
}

function formatBytes(n: number): string {
  if (!n || Number.isNaN(n)) return '-'
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) { v/=1024; i++ }
  return `${v.toFixed(1)} ${units[i]}`
}

function getCSRF(): string {
  const m = document.cookie.match(/(?:^|; )nos_csrf=([^;]*)/)
  return m ? decodeURIComponent(m[1]) : ''
}


