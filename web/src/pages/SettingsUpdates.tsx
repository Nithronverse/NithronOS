import { useEffect, useState } from 'react'
import http from '@/lib/nos-client'
import { toast } from '@/components/ui/toast'

type CheckResp = { plan:any; snapshot_roots:string[] }
type ApplyResp = { ok:boolean; tx_id:string; snapshots_count:number; updates_count:number }

export function SettingsUpdates(){
  const [loading,setLoading]=useState(false)
  const [applying,setApplying]=useState(false)
  const [pruning,setPruning]=useState(false)
  const [snapshot,setSnapshot]=useState(true)
  const [plan,setPlan]=useState<CheckResp|null>(null)
  const [recent,setRecent]=useState<any[]>([])
  const [pruneResult,setPruneResult]=useState<any|null>(null)
  const [error,setError]=useState<string>('')

  const load = async()=>{
    setLoading(true)
    setError('')
    try{
      const p = await http.updates.check() as unknown as CheckResp
      setPlan(p)
      const rec = await http.snapshots.recent() as any[]
      setRecent(rec||[])
    }catch(e:any){ setError(e?.message||'Failed to load updates') }
    finally{ setLoading(false) }
  }
  useEffect(()=>{ load() },[])

  const apply = async()=>{
    setApplying(true)
    setError('')
    try{
      const resp = await http.updates.apply({ snapshot, confirm:'yes' }) as unknown as ApplyResp
      toast.success(`Updates applied (tx ${resp.tx_id})`)
      await load()
    }catch(e:any){
      const msg = e?.message||'Failed to apply updates'
      setError(msg)
      toast.error(msg)
    }
    finally{ setApplying(false) }
  }

  const rollback = async(tx_id:string)=>{
    if (!confirm(`Rollback updates for ${tx_id}?`)) return
    setApplying(true)
    setError('')
    try{
      await http.updates.rollback({ tx_id, confirm:'yes' })
      toast.success('Rollback requested')
      await load()
    }catch(e:any){
      const msg = e?.message||'Failed to rollback'
      setError(msg)
      toast.error(msg)
    }
    finally{ setApplying(false) }
  }

  const updates = (plan?.plan?.updates as any[])||[]
  const reboot = !!(plan?.plan?.reboot_required)

  const prune = async()=>{
    if (!confirm('Prune old snapshots now?')) return
    setPruning(true)
    setError('')
    try{
      const resp = await http.snapshots.prune({ keep_per_target: 5 }) as any
      if (!resp?.ok) throw new Error(resp?.message || 'Prune failed')
      const data = resp
      setPruneResult(data)
      toast.success('Prune completed')
    }catch(e:any){
      const msg = e?.message||'Failed to prune snapshots'
      setError(msg)
      toast.error(msg)
    }
    finally{ setPruning(false) }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Updates</h1>
        <button className="btn-outline" onClick={load} disabled={loading||applying}>Refresh</button>
      </div>
      {error && <div className="text-red-500 text-sm">{error}</div>}

      <section className="card p-4 space-y-3">
        <div className="flex items-center gap-3">
          <h2 className="text-lg font-medium">Available updates</h2>
          {reboot && <span className="px-2 py-1 text-xs rounded bg-yellow-200 text-yellow-900">Reboot required</span>}
        </div>
        {loading ? <div className="text-sm text-muted-foreground">Checking…</div> : (
          updates.length ? (
            <ul className="text-sm list-disc pl-5">
              {updates.map((u:any)=> (
                <li key={u.name}>{u.name} <span className="text-muted-foreground">{u.current} → {u.candidate}</span></li>
              ))}
            </ul>
          ) : <div className="text-sm text-muted-foreground">Your system is up to date.</div>
        )}
        <label className="flex items-center gap-2 text-sm"><input type="checkbox" checked={snapshot} onChange={e=>setSnapshot(e.target.checked)}/> Snapshot before update</label>
        <button className="btn" onClick={apply} disabled={applying||loading}>{applying?'Applying…':'Apply Updates'}</button>
      </section>

      <section className="card p-4 space-y-3">
        <h2 className="text-lg font-medium">Previous updates</h2>
        {!recent?.length ? (
          <div className="text-sm text-muted-foreground">No update history yet.</div>
        ) : (
          <table className="w-full text-sm">
            <thead><tr><th>Time</th><th>Tx</th><th>Packages</th><th>Targets</th><th>Status</th><th/></tr></thead>
            <tbody>
              {recent.map((tx:any)=> (
                <tr key={tx.tx_id} className="border-b border-border/50">
                  <td>{new Date(tx.time).toLocaleString()}</td>
                  <td className="font-mono">{tx.tx_id}</td>
                  <td>{(tx.packages||[]).join(', ')}</td>
                  <td>{(tx.targets||[]).map((t:any)=>t.type).join(', ')}</td>
                  <td>{tx.success? 'OK':'Failed'}</td>
                  <td className="text-right"><button className="btn-outline" onClick={()=>rollback(tx.tx_id)} disabled={applying}>Rollback</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      <section className="card p-4 space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-medium">Snapshot retention</h2>
          <button className="btn-outline" onClick={prune} disabled={pruning}>{pruning?'Pruning…':'Prune snapshots now'}</button>
        </div>
        {pruneResult && (
          <div className="text-sm">
            <div className="font-medium mb-1">Last prune summary</div>
            <ul className="list-disc pl-5">
              {Object.entries(pruneResult.pruned||{}).map(([k,v]:any)=> (
                <li key={k}>{k}: removed {v as number}</li>
              ))}
            </ul>
          </div>
        )}
      </section>
    </div>
  )
}


