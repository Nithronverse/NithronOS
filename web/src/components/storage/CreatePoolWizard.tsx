import { useEffect, useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import { api, type Disks } from '../../api/client'
import { Steps } from '../ui/steps'

const schema = z.object({
  label: z.string().regex(/^[A-Za-z0-9_-]{1,32}$/),
  raid: z.enum(['single', 'raid1', 'raid10']),
  devices: z.array(z.string()).min(1),
})
type FormValues = z.infer<typeof schema>

export function CreatePoolWizard({ onCreated }: { onCreated?: () => void }) {
  const [step, setStep] = useState(1)
  const [disks, setDisks] = useState<Disks | null>(null)
  const [plan, setPlan] = useState<string>('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const { register, handleSubmit, setValue, watch, formState: { errors } } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { label: '', raid: 'single', devices: [] },
  })
  const values = watch()
  const reqError = useMemo(() => validateDevicesForRaid(values.raid, values.devices.length), [values.raid, values.devices])

  useEffect(() => { api.disks.get().then(setDisks).catch(() => {}) }, [])

  const rows = useMemo(() => (disks?.disks ?? []).map((d: any) => ({
    id: d.path || d.name,
    name: d.name,
    model: d.model,
    serial: d.serial,
    size: Number(d.size),
    inUse: !!d.mountpoint || (d.fstype && d.fstype !== '') || d.type !== 'disk',
  })), [disks])

  function toggleDevice(id: string) {
    const list = new Set(values.devices)
    if (list.has(id)) list.delete(id); else list.add(id)
    setValue('devices', Array.from(list))
  }

  useEffect(() => {
    // enforce device count based on raid
    const n = values.devices.length
    if (values.raid === 'raid1' && n < 2) setPlan('')
    if (values.raid === 'raid10' && (n < 4 || n % 2 !== 0)) setPlan('')
  }, [values.devices, values.raid])

  async function doPlan() {
    setLoading(true)
    setError(null)
    try {
      const res: any = await api.pools.planCreate({ label: values.label, devices: values.devices, raid: values.raid })
      setPlan(JSON.stringify(res, null, 2))
      setStep(4)
    } catch (e: any) {
      setError(e?.message || String(e))
    } finally {
      setLoading(false)
    }
  }

  async function doCreate() {
    setLoading(true)
    setError(null)
    try {
      await fetch('/api/pools/create', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCSRF(), 'Confirm': 'yes' }, body: JSON.stringify({ label: values.label, devices: values.devices, raid: values.raid }) })
      try { const { pushToast } = await import('../ui/toast'); pushToast('Pool created') } catch {}
      onCreated?.()
    } catch (e: any) {
      setError(e?.message || String(e))
      try { const { pushToast } = await import('../ui/toast'); pushToast(`Create failed: ${e?.message || e}`, 'error') } catch {}
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-4">
      <Steps steps={["Devices", "Profile", "Plan", "Create"]} current={step} />
      {step === 1 && (
        <div className="space-y-2">
          <h3 className="font-medium">Select devices</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="text-left text-muted-foreground"><tr><th></th><th>Name</th><th>Model</th><th>Serial</th><th>Size</th></tr></thead>
              <tbody>
                {rows.map((r) => (
                  <tr key={r.id} className="border-t border-muted/20 opacity-100">
                    <td className="py-2">
                      <input type="checkbox" disabled={r.inUse} checked={values.devices.includes(r.id)} onChange={() => toggleDevice(r.id)} />
                    </td>
                    <td>{r.name}</td><td>{r.model || '-'}</td><td>{r.serial || '-'}</td><td>{formatBytes(r.size)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {(errors.devices || reqError) && (
            <div className="text-xs text-red-400">{reqError || 'Select at least one device'}</div>
          )}
          <div className="flex gap-2"><button className="rounded bg-primary px-3 py-1 text-sm text-background" onClick={() => setStep(2)} disabled={!!reqError}>Next</button></div>
        </div>
      )}
      {step === 2 && (
        <form className="space-y-2" onSubmit={handleSubmit(() => setStep(3))}>
          <h3 className="font-medium">Profile</h3>
          <div>
            <label className="block text-sm">Label</label>
            <input className="mt-1 w-full rounded border border-muted/30 bg-background px-2 py-1 focus:outline-none focus:ring-2 focus:ring-primary/50" placeholder="mydata" {...register('label')} />
            {errors.label && <div className="text-xs text-red-400">Use 1-32 characters [A-Za-z0-9_-]</div>}
          </div>
          <div className="text-xs text-muted-foreground">
            Estimated capacity: {estimateCapacity(values.raid, values.devices.length, rows)}
          </div>
          <div>
            <label className="block text-sm">RAID</label>
            <select className="mt-1 w-full rounded border border-muted/30 bg-background px-2 py-1 focus:outline-none focus:ring-2 focus:ring-primary/50" {...register('raid')}>
              <option value="single">single</option>
              <option value="raid1">raid1</option>
              <option value="raid10">raid10</option>
            </select>
          </div>
          <div className="flex gap-2"><button className="rounded border border-muted/30 px-3 py-1 text-sm" type="button" onClick={() => setStep(1)}>Back</button><button className="rounded bg-primary px-3 py-1 text-sm text-background" type="submit">Next</button></div>
        </form>
      )}
      {step === 3 && (
        <div className="space-y-2">
          <h3 className="font-medium">Plan</h3>
          <button className="inline-flex items-center gap-2 rounded bg-primary px-3 py-1 text-sm text-background disabled:opacity-50" onClick={doPlan} disabled={loading || !!reqError}>
            {loading && <span className="h-3 w-3 animate-spin rounded-full border-2 border-background border-t-transparent" />}
            Fetch Plan
          </button>
          {plan && <pre className="text-xs text-muted-foreground overflow-auto max-h-60 whitespace-pre-wrap">{plan}</pre>}
          <div className="flex gap-2"><button className="rounded border border-muted/30 px-3 py-1 text-sm" onClick={() => setStep(2)}>Back</button><button className="rounded bg-primary px-3 py-1 text-sm text-background disabled:opacity-50" onClick={() => setStep(4)} disabled={!plan}>Next</button></div>
        </div>
      )}
      {step === 4 && (
        <div className="space-y-2">
          <h3 className="font-medium">Confirm & Create</h3>
          <button className="inline-flex items-center gap-2 rounded bg-primary px-3 py-1 text-sm text-background disabled:opacity-50" onClick={doCreate} disabled={loading || !plan}>
            {loading && <span className="h-3 w-3 animate-spin rounded-full border-2 border-background border-t-transparent" />}
            Create
          </button>
        </div>
      )}
      {error && <div className="rounded border border-red-500/30 bg-red-500/10 p-2 text-red-400 text-xs">{error}</div>}
    </div>
  )
}

function getCSRF(): string { const m = document.cookie.match(/(?:^|; )nos_csrf=([^;]*)/); return m ? decodeURIComponent(m[1]) : '' }
function formatBytes(n: number): string { if (!n || Number.isNaN(n)) return '-'; const u = ['B','KB','MB','GB','TB','PB']; let i=0,v=n; while(v>=1024 && i<u.length-1){v/=1024;i++} return `${v.toFixed(1)} ${u[i]}` }
function validateDevicesForRaid(raid: string, count: number): string | null {
  if (raid === 'single' && count < 1) return 'Select at least one device'
  if (raid === 'raid1' && count < 2) return 'RAID1 requires at least 2 devices'
  if (raid === 'raid10') {
    if (count < 4) return 'RAID10 requires at least 4 devices'
    if (count % 2 !== 0) return 'RAID10 requires an even number of devices'
  }
  return null
}
function estimateCapacity(raid: string, _count: number, rows: any[]): string {
  const selectedSizes = rows.filter(r => r && r.id && r.size).map(r => r.size).sort((a,b)=>a-b)
  if (selectedSizes.length === 0) return '-'
  if (raid === 'single') return formatBytes(selectedSizes.reduce((a,b)=>a+b,0))
  if (raid === 'raid1') return formatBytes(selectedSizes[0])
  if (raid === 'raid10') {
    const pairs = Math.floor(selectedSizes.length/2)
    const usablePerPair = Math.min(...selectedSizes.slice(0, pairs*2))
    return formatBytes(usablePerPair * pairs)
  }
  return '-'
}


