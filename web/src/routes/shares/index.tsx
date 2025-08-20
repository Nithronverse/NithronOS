import { useEffect, useState } from 'react';
import api from '@/lib/api';
import type { components } from '@/api-types';
type Share = components['schemas']['Share'];
import ShareTable from '@/components/shares/ShareTable';
import NewShareWizard from '@/components/shares/NewShareWizard';

export default function SharesPage() {
  const [items,setItems]=useState<Share[]>([]);
  const [open,setOpen]=useState(false);
  const load = async()=> setItems(await api.shares.list());
  useEffect(()=>{ load(); },[]);
  const del = async (id:string) => {
    const s = items.find(x=>x.id===id)
    if (!confirm(`Delete ${s?.name ?? id}?`)) return
    try {
      await api.shares.del(id)
      await load()
    } catch (err:any) {
      alert(err?.message || 'Failed to delete share')
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Shares</h1>
        <div className="flex gap-2">
          <button className="btn" onClick={()=>setOpen(true)}>New Share</button>
          <button className="btn-outline" onClick={load}>Refresh</button>
        </div>
      </div>
      <ShareTable items={items} onDelete={del}/>
      <NewShareWizard open={open} onClose={async(ok)=>{ setOpen(false); if (ok) await load(); }}/>
    </div>
  );
}


