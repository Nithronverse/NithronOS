import { apiGet } from './http'
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
export type Shares = GetResponse<'/api/shares'>
export type Apps = GetResponse<'/api/apps'>
export type RemoteStatus = GetResponse<'/api/remote/status'>

export const api = {
	health: {
		get: () => apiGet<Health>('/api/health'),
	},
	disks: {
		get: () => apiGet<Disks>('/api/disks'),
	},
	pools: {
		get: () => apiGet<Pools>('/api/pools'),
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
}


