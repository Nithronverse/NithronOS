import type { paths, components } from '@/api-types'
export type Share = components['schemas']['Share'];
export type Pool = { id:string; label?:string; mountpoint:string; size?:number; free?:number };

const csrf = () => {
  const m = document.cookie.match(/(?:^|; )nos_csrf=([^;]*)/)
  return m ? decodeURIComponent(m[1]) : ''
};
async function req<T>(url:string, init:RequestInit={}) {
  const r = await fetch(url, {
    credentials: 'include',
    headers: { 'Content-Type':'application/json', 'X-CSRF-Token': csrf(), ...(init.headers||{}) },
    ...init,
  });
  if (r.status === 204) return undefined as unknown as T;
  if (!r.ok) {
    const ct = r.headers.get('content-type') || ''
    try {
      if (ct.includes('application/json')) {
        const j = await r.json() as { error?: string };
        if (j && typeof j.error === 'string') throw new Error(j.error);
      }
    } catch { /* fallthrough to text */ }
    throw new Error((await r.text()) || r.statusText);
  }
  const ct = r.headers.get('content-type') || ''
  if (ct.includes('application/json')) return (await r.json()) as T;
  return undefined as unknown as T;
}

import type {} from '@/api-types'

type SmbUsersGet = paths['/api/smb/users']['get']['responses'][200]['content']['application/json']
type SmbUserCreateReq = components['schemas']['SmbUserCreate']
type SmbUserCreateResp = paths['/api/smb/users']['post']['responses'][201]
type PoolsRootsResp = paths['/api/pools/roots']['get']['responses'][200]['content']['application/json']
type UpdatesApplyResp = paths['/api/updates/apply']['post']['responses'][200]['content']['application/json']
type UpdatesRollbackResp = paths['/api/updates/rollback']['post']['responses'][200]['content']['application/json']
type UpdatesCheckResp = paths['/api/updates/check']['get']['responses'][200]['content']['application/json']

export const api = {
  shares: {
    list: () => req<Share[]>('/api/shares'),
    create: (body: Partial<Share>) => req<Share>('/api/shares', { method:'POST', body: JSON.stringify(body) }),
    del: (id:string) => req<void>(`/api/shares/${id}`, { method:'DELETE' }),
  },
  pools: {
    list: () => req<Pool[]>('/api/pools'),
    roots: () => req<PoolsRootsResp>('/api/pools/roots').then(r => (r as any).roots as string[]),
    getMountOptions: (id: string) => req<{ mountOptions: string }>(`/api/v1/pools/${encodeURIComponent(id)}/options`),
    setMountOptions: (id: string, mountOptions: string) =>
      req<{ ok: boolean; mountOptions: string; rebootRequired?: boolean }>(
        `/api/v1/pools/${encodeURIComponent(id)}/options`,
        { method: 'POST', body: JSON.stringify({ mountOptions }) },
      ),
    planDevice: (id: string, body: any) => req<{ planId: string; steps: any[]; warnings: string[]; requiresBalance?: boolean }>(`/api/v1/pools/${encodeURIComponent(id)}/plan-device`, { method:'POST', body: JSON.stringify(body) }),
    applyDevice: (id: string, steps: { id:string; description:string; command:string }[], confirm?: 'ADD'|'REMOVE'|'REPLACE') =>
      req<{ ok:boolean; tx_id:string }>(`/api/v1/pools/${encodeURIComponent(id)}/apply-device`, { method:'POST', body: JSON.stringify({ steps, confirm }) }),
  },
  health: {
    alerts: () => req<{ alerts: Array<{ id:string; severity:'warn'|'crit'; kind:string; device:string; messages:string[]; createdAt:string }> }>(`/api/v1/alerts`),
    scan: () => req<{ ok:boolean }>(`/api/v1/health/scan`, { method:'POST', body: JSON.stringify({}) }),
  },
  smb: {
    usersList: () => req<SmbUsersGet>('/api/smb/users').catch(()=>[] as SmbUsersGet),
    userCreate: (u: SmbUserCreateReq) => req<SmbUserCreateResp>('/api/smb/users', { method:'POST', body: JSON.stringify(u) }),
  },
  updates: {
    check: () => req<UpdatesCheckResp>('/api/updates/check'),
    apply: (body: { packages?: string[]; snapshot?: boolean; confirm: 'yes' }) =>
      req<UpdatesApplyResp>('/api/updates/apply', { method:'POST', body: JSON.stringify(body) }),
    rollback: (body: { tx_id: string; confirm: 'yes' }) =>
      req<UpdatesRollbackResp>('/api/updates/rollback', { method:'POST', body: JSON.stringify(body) }),
  },
};
export default api;


