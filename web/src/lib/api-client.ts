import { z } from 'zod'

// API Schemas
export const DiskSchema = z.object({
  name: z.string(),
  model: z.string().optional(),
  size: z.number(),
  used: z.number().optional(),
  health: z.enum(['healthy', 'warning', 'critical']).optional(),
  temperature: z.number().optional(),
  mountpoints: z.array(z.string()).optional(),
})

export const VolumeSchema = z.object({
  id: z.string(),
  name: z.string(),
  type: z.enum(['zfs', 'ext4', 'btrfs', 'xfs']),
  size: z.number(),
  used: z.number(),
  pool: z.string().optional(),
  status: z.enum(['online', 'degraded', 'offline']),
  mountpoint: z.string(),
})

export const ShareSchema = z.object({
  id: z.string(),
  name: z.string(),
  protocol: z.enum(['smb', 'nfs']),
  path: z.string(),
  access: z.enum(['public', 'private']),
  status: z.enum(['active', 'inactive']),
  users: z.array(z.string()).optional(),
})

export const AppSchema = z.object({
  id: z.string(),
  name: z.string(),
  version: z.string(),
  status: z.enum(['running', 'stopped', 'error']),
  autoUpdate: z.boolean(),
  category: z.string().optional(),
  description: z.string().optional(),
})

export const HealthSchema = z.object({
  status: z.enum(['healthy', 'degraded', 'critical']),
  cpu: z.number(),
  memory: z.number(),
  uptime: z.number(),
  alerts: z.array(z.object({
    id: z.string(),
    severity: z.enum(['info', 'warn', 'crit']),
    message: z.string(),
    timestamp: z.string(),
  })),
})

export const UserSchema = z.object({
  id: z.string(),
  username: z.string(),
  role: z.enum(['admin', 'user', 'readonly']),
  twoFactor: z.boolean(),
  lastLogin: z.string().optional(),
})

export const ScheduleSchema = z.object({
  id: z.string(),
  name: z.string(),
  cron: z.string(),
  enabled: z.boolean(),
  lastRun: z.string().optional(),
  nextRun: z.string().optional(),
  timezone: z.string(),
})

// Types
export type Disk = z.infer<typeof DiskSchema>
export type Volume = z.infer<typeof VolumeSchema>
export type Share = z.infer<typeof ShareSchema>
export type App = z.infer<typeof AppSchema>
export type Health = z.infer<typeof HealthSchema>
export type User = z.infer<typeof UserSchema>
export type Schedule = z.infer<typeof ScheduleSchema>

// API Client
class ApiClient {
  private baseURL = '/api'

  private async request<T>(
    path: string,
    options: RequestInit = {}
  ): Promise<T> {
    const response = await fetch(`${this.baseURL}${path}`, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      credentials: 'include',
    })

    // Check if response is HTML (proxy error)
    const contentType = response.headers.get('content-type')
    if (contentType && contentType.includes('text/html')) {
      throw new Error('PROXY_ERROR')
    }

    if (!response.ok) {
      const error = await response.text()
      throw new Error(error || response.statusText)
    }

    const data = await response.json()
    return data
  }

  // Health endpoints
  async getHealth() {
    const data = await this.request<any>('/health')
    return HealthSchema.parse(data)
  }

  // Disk endpoints
  async getDisks() {
    const data = await this.request<any>('/disks')
    return z.array(DiskSchema).parse(data.disks || [])
  }

  // Volume endpoints
  async getVolumes() {
    const data = await this.request<any>('/pools')
    return z.array(VolumeSchema).parse(data)
  }

  async createVolume(data: any) {
    return this.request('/pools', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  // Share endpoints
  async getShares() {
    const data = await this.request<any>('/shares')
    return z.array(ShareSchema).parse(data)
  }

  async createShare(data: Partial<Share>) {
    return this.request('/shares', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async updateShare(id: string, data: Partial<Share>) {
    return this.request(`/shares/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async deleteShare(id: string) {
    return this.request(`/shares/${id}`, {
      method: 'DELETE',
    })
  }

  // App endpoints
  async getApps() {
    const data = await this.request<any>('/apps')
    return z.array(AppSchema).parse(data.installed || [])
  }

  async getMarketplace() {
    const data = await this.request<any>('/apps/marketplace')
    return z.array(AppSchema).parse(data.apps || [])
  }

  // User endpoints
  async getUsers() {
    const data = await this.request<any>('/users')
    return z.array(UserSchema).parse(data.users || [])
  }

  // Schedule endpoints
  async getSchedules() {
    const data = await this.request<any>('/schedules')
    return z.array(ScheduleSchema).parse(data.schedules || [])
  }

  async updateSchedule(id: string, data: Partial<Schedule>) {
    return this.request(`/schedules/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }
}

export const apiClient = new ApiClient()
