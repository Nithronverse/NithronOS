import { z } from 'zod'

// ============================================================================
// Core API Client with strict JSON parsing and error handling
// ============================================================================

class APIError extends Error {
  constructor(
    message: string,
    public status: number,
    public code?: string,
    public details?: any
  ) {
    super(message)
    this.name = 'APIError'
  }
}

class ApiClient {
  private baseURL = '/api'
  private tokenPromise: Promise<void> | null = null

  private getCsrfToken(): string {
    const match = document.cookie.match(/(?:^|; )nos_csrf=([^;]*)/)
    return match ? decodeURIComponent(match[1]) : ''
  }

  private async refreshToken(): Promise<void> {
    if (this.tokenPromise) return this.tokenPromise
    
    this.tokenPromise = fetch('/api/auth/refresh', {
      method: 'POST',
      credentials: 'include',
    }).then(async (res) => {
      if (!res.ok) throw new APIError('Session expired', 401)
      this.tokenPromise = null
    })
    
    return this.tokenPromise
  }

  async request<T>(
    path: string,
    options: RequestInit = {}
  ): Promise<T> {
    const url = `${this.baseURL}${path}`
    
    const response = await fetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': this.getCsrfToken(),
        ...options.headers,
      },
      credentials: 'include',
    })

    // Check for HTML response (proxy misconfiguration)
    const contentType = response.headers.get('content-type') || ''
    if (contentType.includes('text/html')) {
      throw new APIError('Backend unreachable or proxy misconfigured', 502)
    }

    // Handle 401 with token refresh
    if (response.status === 401) {
      await this.refreshToken()
      // Retry once
      return this.request<T>(path, options)
    }

    // Handle 204 No Content
    if (response.status === 204) {
      return undefined as unknown as T
    }

    // Parse error response
    if (!response.ok) {
      let message = response.statusText
      let code: string | undefined
      let details: any
      
      try {
        if (contentType.includes('application/json')) {
          const error = await response.json()
          message = error.error || error.message || message
          code = error.code
          details = error.details
        } else {
          message = await response.text() || message
        }
      } catch {
        // Use default message
      }
      
      throw new APIError(message, response.status, code, details)
    }

    // Parse success response
    if (contentType.includes('application/json')) {
      return response.json()
    }
    
    return undefined as unknown as T
  }

  // Helper methods
  get<T>(path: string, params?: Record<string, any>) {
    const query = params ? '?' + new URLSearchParams(params).toString() : ''
    return this.request<T>(`${path}${query}`)
  }

  post<T>(path: string, body?: any) {
    return this.request<T>(path, {
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
    })
  }

  put<T>(path: string, body?: any) {
    return this.request<T>(path, {
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined,
    })
  }

  del<T>(path: string) {
    return this.request<T>(path, { method: 'DELETE' })
  }
}

export const api = new ApiClient()

// ============================================================================
// Type Schemas for M1-M3 Features
// ============================================================================

// System Info
export const SystemInfoSchema = z.object({
  hostname: z.string(),
  uptime: z.number(),
  kernel: z.string(),
  version: z.string(),
  arch: z.string().optional(),
  cpuCount: z.number().optional(),
  memoryTotal: z.number().optional(),
  memoryUsed: z.number().optional(),
})

// Storage Pools (M1)
export const PoolSchema = z.object({
  id: z.string(),
  uuid: z.string(),
  label: z.string().optional(),
  mountpoint: z.string(),
  size: z.number(),
  used: z.number(),
  free: z.number(),
  raid: z.string(),
  status: z.enum(['online', 'degraded', 'offline']),
  devices: z.array(z.object({
    path: z.string(),
    size: z.number(),
    used: z.number().optional(),
    status: z.string().optional(),
  })),
  subvolumes: z.array(z.object({
    id: z.string(),
    path: z.string(),
    size: z.number().optional(),
  })).optional(),
})

export const PoolSummarySchema = z.object({
  totalPools: z.number(),
  totalSize: z.number(),
  totalUsed: z.number(),
  poolsOnline: z.number(),
  poolsDegraded: z.number(),
})

// Storage Devices
export const DeviceSchema = z.object({
  path: z.string(),
  model: z.string().optional(),
  serial: z.string().optional(),
  size: z.number(),
  type: z.enum(['ssd', 'hdd']).optional(),
  inUse: z.boolean(),
  pool: z.string().optional(),
})

// SMART Health (M1)
export const SmartDataSchema = z.object({
  device: z.string(),
  status: z.enum(['healthy', 'warning', 'critical']),
  temperature: z.number().optional(),
  powerOnHours: z.number().optional(),
  attributes: z.array(z.object({
    id: z.number(),
    name: z.string(),
    value: z.number(),
    worst: z.number(),
    threshold: z.number(),
    rawValue: z.string(),
    status: z.enum(['ok', 'warning', 'critical']),
  })).optional(),
})

export const SmartSummarySchema = z.object({
  totalDevices: z.number(),
  healthyDevices: z.number(),
  warningDevices: z.number(),
  criticalDevices: z.number(),
  lastScan: z.string().optional(),
})

// Scrub Status (M1)
export const ScrubStatusSchema = z.object({
  poolId: z.string(),
  status: z.enum(['idle', 'running', 'paused', 'finished', 'cancelled']),
  progress: z.number().optional(),
  bytesScanned: z.number().optional(),
  bytesTotal: z.number().optional(),
  errorsFixed: z.number().optional(),
  startedAt: z.string().optional(),
  finishedAt: z.string().optional(),
  nextRun: z.string().optional(),
})

// Balance Status (M1)
export const BalanceStatusSchema = z.object({
  poolId: z.string(),
  status: z.enum(['idle', 'running', 'paused', 'finished', 'cancelled']),
  progress: z.number().optional(),
  bytesBalanced: z.number().optional(),
  bytesTotal: z.number().optional(),
  startedAt: z.string().optional(),
  finishedAt: z.string().optional(),
})

// Schedules (M1)
export const ScheduleSchema = z.object({
  id: z.string(),
  type: z.enum(['smart_scan', 'btrfs_scrub', 'snapshot', 'backup']),
  cron: z.string(),
  enabled: z.boolean(),
  target: z.string().optional(),
  lastRun: z.string().optional(),
  nextRun: z.string().optional(),
})

// Services Status
export const ServiceSchema = z.object({
  name: z.string(),
  status: z.enum(['running', 'stopped', 'failed', 'unknown']),
  enabled: z.boolean(),
  uptime: z.number().optional(),
})

// Jobs/Events
export const JobSchema = z.object({
  id: z.string(),
  type: z.string(),
  status: z.enum(['pending', 'running', 'completed', 'failed']),
  progress: z.number().optional(),
  message: z.string().optional(),
  startedAt: z.string(),
  completedAt: z.string().optional(),
  error: z.string().optional(),
})

// Shares (M2)
export const ShareSchema = z.object({
  name: z.string(),
  path: z.string(),
  protocol: z.enum(['smb', 'nfs', 'afp']),
  enabled: z.boolean(),
  guestOk: z.boolean().optional(),
  readOnly: z.boolean().optional(),
  users: z.array(z.string()).optional(),
  groups: z.array(z.string()).optional(),
  description: z.string().optional(),
  createdAt: z.string().optional(),
  modifiedAt: z.string().optional(),
})

// Apps (M3)
export const AppCatalogEntrySchema = z.object({
  id: z.string(),
  name: z.string(),
  version: z.string(),
  description: z.string(),
  categories: z.array(z.string()),
  icon: z.string().optional(),
  compose: z.string(),
  schema: z.string(),
  defaults: z.record(z.any()).optional(),
  health: z.object({
    type: z.enum(['container', 'http']),
    url: z.string().optional(),
    interval_s: z.number(),
    timeout_s: z.number(),
  }).optional(),
  needs_privileged: z.boolean().optional(),
  notes: z.string().optional(),
})

export const InstalledAppSchema = z.object({
  id: z.string(),
  name: z.string(),
  version: z.string(),
  status: z.enum(['running', 'stopped', 'unhealthy', 'installing', 'upgrading']),
  health: z.object({
    status: z.enum(['healthy', 'unhealthy', 'unknown']),
    checks: z.array(z.any()).optional(),
    lastCheck: z.string().optional(),
  }).optional(),
  ports: z.array(z.object({
    host: z.number(),
    container: z.number(),
    protocol: z.string(),
  })).optional(),
  urls: z.array(z.string()).optional(),
  installedAt: z.string(),
  updatedAt: z.string().optional(),
})

// Type exports
export type SystemInfo = z.infer<typeof SystemInfoSchema>
export type Pool = z.infer<typeof PoolSchema>
export type PoolSummary = z.infer<typeof PoolSummarySchema>
export type Device = z.infer<typeof DeviceSchema>
export type SmartData = z.infer<typeof SmartDataSchema>
export type SmartSummary = z.infer<typeof SmartSummarySchema>
export type ScrubStatus = z.infer<typeof ScrubStatusSchema>
export type BalanceStatus = z.infer<typeof BalanceStatusSchema>
export type Schedule = z.infer<typeof ScheduleSchema>
export type Service = z.infer<typeof ServiceSchema>
export type Job = z.infer<typeof JobSchema>
export type Share = z.infer<typeof ShareSchema>
export type AppCatalogEntry = z.infer<typeof AppCatalogEntrySchema>
export type InstalledApp = z.infer<typeof InstalledAppSchema>

// ============================================================================
// API Endpoints for M1-M3 Features
// ============================================================================

export const endpoints = {
  // System
  system: {
    info: () => api.get<SystemInfo>('/v1/system/info'),
    services: () => api.get<Service[]>('/v1/system/services'),
  },

  // Storage Pools (M1)
  pools: {
    list: () => api.get<Pool[]>('/v1/storage/pools'),
    summary: () => api.get<PoolSummary>('/v1/storage/pools', { summary: '1' }),
    get: (uuid: string) => api.get<Pool>(`/v1/storage/pools/${uuid}`),
    subvolumes: (uuid: string) => api.get<any[]>(`/v1/storage/pools/${uuid}/subvols`),
    getMountOptions: (uuid: string) => 
      api.get<{ mountOptions: string }>(`/v1/storage/pools/${uuid}/options`),
    setMountOptions: (uuid: string, options: string) =>
      api.put<{ ok: boolean; mountOptions: string; rebootRequired?: boolean }>(
        `/v1/storage/pools/${uuid}/options`,
        { mountOptions: options }
      ),
  },

  // Storage Devices
  devices: {
    list: () => api.get<Device[]>('/v1/storage/devices'),
  },

  // SMART Health (M1)
  smart: {
    summary: () => api.get<SmartSummary>('/v1/health/smart/summary'),
    device: (device: string) => api.get<SmartData>(`/v1/health/smart/${device}`),
    scan: () => api.post('/v1/health/smart/scan'),
  },

  // Scrub (M1)
  scrub: {
    status: () => api.get<ScrubStatus[]>('/v1/btrfs/scrub/status'),
    start: (poolId: string) => api.post(`/v1/btrfs/scrub/start`, { poolId }),
    cancel: (poolId: string) => api.post(`/v1/btrfs/scrub/cancel`, { poolId }),
  },

  // Balance (M1)
  balance: {
    status: () => api.get<BalanceStatus[]>('/v1/btrfs/balance/status'),
    start: (poolId: string) => api.post(`/v1/btrfs/balance/start`, { poolId }),
    cancel: (poolId: string) => api.post(`/v1/btrfs/balance/cancel`, { poolId }),
  },

  // Schedules (M1)
  schedules: {
    list: () => api.get<Schedule[]>('/v1/schedules'),
    create: (schedule: Partial<Schedule>) => api.post<Schedule>('/v1/schedules', schedule),
    update: (id: string, schedule: Partial<Schedule>) => 
      api.put<Schedule>(`/v1/schedules/${id}`, schedule),
    delete: (id: string) => api.del(`/v1/schedules/${id}`),
  },

  // Jobs
  jobs: {
    recent: (limit = 10) => api.get<Job[]>('/v1/jobs/recent', { limit: limit.toString() }),
  },

  // Shares (M2)
  shares: {
    list: () => api.get<Share[]>('/v1/shares'),
    get: (name: string) => api.get<Share>(`/v1/shares/${name}`),
    create: (share: Partial<Share>) => api.post<Share>('/v1/shares', share),
    update: (name: string, share: Partial<Share>) => 
      api.put<Share>(`/v1/shares/${name}`, share),
    delete: (name: string) => api.del(`/v1/shares/${name}`),
    test: (name: string, config: any) => 
      api.post(`/v1/shares/${name}/test`, config),
  },

  // Apps (M3)
  apps: {
    catalog: () => api.get<{ entries: AppCatalogEntry[] }>('/v1/apps/catalog'),
    installed: () => api.get<{ items: InstalledApp[] }>('/v1/apps/installed'),
    get: (id: string) => api.get<InstalledApp>(`/v1/apps/${id}`),
    install: (params: { id: string; params?: Record<string, any> }) =>
      api.post<InstalledApp>('/v1/apps/install', params),
    start: (id: string) => api.post(`/v1/apps/${id}/start`),
    stop: (id: string) => api.post(`/v1/apps/${id}/stop`),
    restart: (id: string) => api.post(`/v1/apps/${id}/restart`),
    upgrade: (id: string, params?: Record<string, any>) =>
      api.post(`/v1/apps/${id}/upgrade`, { params }),
    delete: (id: string, keepData = false) =>
      api.del(`/v1/apps/${id}?keep_data=${keepData}`),
    logs: (id: string, options?: { follow?: boolean; limit?: number }) =>
      api.get(`/v1/apps/${id}/logs`, options),
  },

  // Auth
  auth: {
    refresh: () => api.post('/auth/refresh'),
    logout: () => api.post('/auth/logout'),
  },
}

export default api