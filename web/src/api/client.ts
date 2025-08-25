import { apiGet, apiPost, apiDelete, apiPatch } from './http'
import type { paths, components } from './schema'

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
export type Share = components['schemas']['Share']
export type Shares = GetResponse<'/api/shares'>
export type Apps = GetResponse<'/api/apps'>
export type RemoteStatus = GetResponse<'/api/remote/status'>

// Additional types for shares
export interface SMBConfig {
	enabled: boolean
	guest?: boolean
	time_machine?: boolean
	recycle?: {
		enabled: boolean
		directory?: string
	}
}

export interface NFSConfig {
	enabled: boolean
	networks?: string[]
	read_only?: boolean
}

export interface CreateShareRequest {
	name: string
	smb?: SMBConfig
	nfs?: NFSConfig
	owners?: string[]
	readers?: string[]
	description?: string
}

export interface UpdateShareRequest {
	smb?: SMBConfig
	nfs?: NFSConfig
	owners?: string[]
	readers?: string[]
	description?: string
}

export interface TestShareResponse {
	valid: boolean
	errors?: Array<{
		code: string
		message: string
		field?: string
	}>
}

export interface User {
	username: string
	uid?: number
	groups?: string[]
}

export interface Group {
	name: string
	gid?: number
	members?: string[]
}

export interface Policy {
	shares?: {
		guest_access_forbidden?: boolean
		max_shares?: number
	}
}

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
		planCreateV1: (body: any) => apiPost<any>('/api/v1/pools/plan-create', body),
		applyCreateV1: (body: any) => apiPost<any>('/api/v1/pools/apply-create', body),
		txStatus: (id: string) => apiGet<any>(`/api/v1/pools/tx/${id}/status`),
		txLog: (id: string, cursor = 0, max = 1000) => apiGet<any>(`/api/v1/pools/tx/${id}/log?cursor=${cursor}&max=${max}`),
		planCreate: (body: any) => apiPost<PostResponse<'/api/pools/plan-create'>>('/api/pools/plan-create', body),
		create: (body: any) => apiPost<PostResponse<'/api/pools/create'>>('/api/pools/create', body),
		candidates: () => apiGet<PoolCandidates>('/api/pools/candidates'),
		import: (deviceOrUuid: string) => apiPost<PostResponse<'/api/pools/import'>>('/api/pools/import', { device_or_uuid: deviceOrUuid }),
		discoverV1: () => apiGet<any>('/api/v1/pools/discover'),
		importV1: (body: { uuid: string; label?: string; mountpoint: string }) => apiPost<any>('/api/v1/pools/import', body),
		scrubStart: (mount: string) => apiPost<any>('/api/v1/pools/scrub/start', { mount }),
		scrubStatus: (mount: string) => apiGet<any>(`/api/v1/pools/scrub/status?mount=${encodeURIComponent(mount)}`),
	},
	shares: {
		get: () => apiGet<Share[]>('/api/shares'),
		create: (data: CreateShareRequest) => apiPost<Share>('/api/shares', data),
		update: (name: string, data: UpdateShareRequest) => apiPatch<Share>(`/api/shares/${name}`, data),
		delete: (name: string) => apiDelete(`/api/shares/${name}`),
		test: (name: string, config: any) => apiPost<TestShareResponse>(`/api/shares/${name}/test`, { config }),
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
	users: {
		get: () => apiGet<User[]>('/api/users'),
	},
	groups: {
		get: () => apiGet<Group[]>('/api/groups'),
	},
	policy: {
		get: () => apiGet<Policy>('/api/policy'),
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


