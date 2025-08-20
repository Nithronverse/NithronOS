import { apiGet, apiPost } from './http'
import type { paths } from './schema'

type GetResponse<Path extends keyof paths> = paths[Path] extends { get: any }
	? paths[Path]['get'] extends {
			responses: { 200: { content: { 'application/json': infer T } } }
	  }
		? T
		: never
	: never

export type Health = GetResponse<'/api/health'>
export type Disks = GetResponse<'/api/disks'>
export type Pools = GetResponse<'/api/pools'>
export type PoolCandidates = GetResponse<'/api/pools/candidates'>
export type Shares = GetResponse<'/api/shares'>
export type Apps = GetResponse<'/api/apps'>
export type RemoteStatus = GetResponse<'/api/remote/status'>
type PostResponse<Path extends keyof paths> = paths[Path] extends { post: any }
	? paths[Path]['post'] extends {
			responses: { 200: { content: { 'application/json': infer T } } }
		}
		? T
		: never
	: never

export const api = {
	health: {
		get: () => apiGet<Health>('/api/health'),
	},
	disks: {
		get: () => apiGet<Disks>('/api/disks'),
	},
	pools: {
		get: () => apiGet<Pools>('/api/pools'),
		planCreate: (body: any) => apiPost<PostResponse<'/api/pools/plan-create'>>('/api/pools/plan-create', body),
		create: (body: any) => apiPost<PostResponse<'/api/pools/create'>>('/api/pools/create', body),
		candidates: () => apiGet<PoolCandidates>('/api/pools/candidates'),
		import: (deviceOrUuid: string) => apiPost<PostResponse<'/api/pools/import'>>('/api/pools/import', { device_or_uuid: deviceOrUuid }),
	},
	shares: {
		get: () => apiGet<Shares>('/api/shares'),
	},
	apps: {
		get: () => apiGet<Apps>('/api/apps'),
	},
	remote: {
		status: () => apiGet<RemoteStatus>('/api/remote/status'),
	},
	auth: {
		me: () => apiGet<any>('/api/auth/me'),
	},
	firewall: {
		status: () => apiGet<GetResponse<'/api/firewall/status'>>('/api/firewall/status'),
		plan: (mode: string) => apiPost<PostResponse<'/api/firewall/plan'>>('/api/firewall/plan', { mode }),
		apply: (mode: string, twoFactorToken?: string) =>
			apiPost<PostResponse<'/api/firewall/apply'>>('/api/firewall/apply', { mode, twoFactorToken }),
		rollback: () => apiPost<PostResponse<'/api/firewall/rollback'>>('/api/firewall/rollback'),
	},
	support: {
		bundle: async (): Promise<Blob> => {
			const res = await fetch('/api/support/bundle', { headers: { Accept: 'application/gzip' } })
			if (!res.ok) throw new Error(`HTTP ${res.status}`)
			return await res.blob()
		},
	},
}


