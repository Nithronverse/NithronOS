import type { paths, components } from '@/api-types'
export type Share = components['schemas']['Share'];
export type Pool = { id:string; label?:string; mountpoint:string; size?:number; free?:number };

const csrf = () => localStorage.getItem('csrf') || '';
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

export const api = {
  shares: {
    list: () => req<Share[]>('/api/shares'),
    create: (body: Partial<Share>) => req<Share>('/api/shares', { method:'POST', body: JSON.stringify(body) }),
    del: (id:string) => req<void>(`/api/shares/${id}`, { method:'DELETE' }),
  },
  pools: {
    list: () => req<Pool[]>('/api/pools'),
    roots: () => req<PoolsRootsResp>('/api/pools/roots').then(r => (r as any).roots as string[]),
  },
  smb: {
    usersList: () => req<SmbUsersGet>('/api/smb/users').catch(()=>[] as SmbUsersGet),
    userCreate: (u: SmbUserCreateReq) => req<SmbUserCreateResp>('/api/smb/users', { method:'POST', body: JSON.stringify(u) }),
  },
};
export default api;


