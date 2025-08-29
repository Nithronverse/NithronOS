/**
 * NithronOS TypeScript Client Library
 * Auto-generated from OpenAPI specification
 */

export interface ClientConfig {
  baseURL: string
  token?: string
  timeout?: number
  onUnauthorized?: () => void
}

export interface APIError {
  code: string
  message: string
  hint?: string
  details?: Record<string, any>
}

export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
  hasNext: boolean
  hasPrev: boolean
}

// API Scopes
export enum APIScope {
  SystemRead = 'system.read',
  SystemWrite = 'system.write',
  UsersRead = 'users.read',
  UsersWrite = 'users.write',
  StorageRead = 'storage.read',
  StorageWrite = 'storage.write',
  AppsRead = 'apps.read',
  AppsWrite = 'apps.write',
  MetricsRead = 'metrics.read',
  AlertsRead = 'alerts.read',
  AlertsWrite = 'alerts.write',
  BackupsRead = 'backups.read',
  BackupsWrite = 'backups.write',
  AdminAll = 'admin.*',
}

// Token types
export interface APIToken {
  id: string
  type: 'personal' | 'service'
  name: string
  scopes: string[]
  createdAt: string
  expiresAt?: string
  lastUsedAt?: string
  lastUsedIP?: string
  useCount: number
}

export interface CreateTokenRequest {
  type: 'personal' | 'service'
  name: string
  scopes: string[]
  expiresIn?: string
  ipAllowlist?: string[]
}

export interface CreateTokenResponse {
  token: APIToken
  value: string // Only returned on creation
}

// Webhook types
export interface Webhook {
  id: string
  url: string
  name: string
  description?: string
  events: string[]
  enabled: boolean
  headers?: Record<string, string>
  maxRetries: number
  retryDelay: number
  timeout: number
  createdAt: string
  updatedAt: string
  lastDelivery?: string
  lastStatus?: number
  lastError?: string
  deliveryCount: number
  successCount: number
  failureCount: number
}

export interface CreateWebhookRequest {
  url: string
  name: string
  description?: string
  events: string[]
  secret?: string
  headers?: Record<string, string>
  enabled: boolean
}

export interface WebhookDelivery {
  id: string
  webhookId: string
  eventId: string
  url: string
  method: string
  status: number
  error?: string
  attemptedAt: string
  duration: number
  attempt: number
  nextRetry?: string
}

// User and auth types
export interface User {
  id: string
  username: string
  email?: string
  role: 'admin' | 'operator' | 'viewer'
  enabled: boolean
  twoFactorEnabled: boolean
  createdAt: string
  updatedAt: string
  lastLoginAt?: string
  lastLoginIP?: string
  forcePasswordChange: boolean
  lockedUntil?: string
  failedLogins: number
}

export interface Session {
  id: string
  userId: string
  username: string
  role: 'admin' | 'operator' | 'viewer'
  issuedAt: string
  expiresAt: string
  lastSeenAt: string
  ip: string
  userAgent: string
  twoFactorVerified: boolean
  elevatedUntil?: string
  scopes?: string[]
}

// System types
export interface SystemInfo {
  hostname: string
  os: string
  kernel: string
  arch: string
  cpus: number
  memory: number
  nosVersion: string
  uptime: string
}

export interface SystemStatus {
  version: string
  uptime: string
  load1: number
  load5: number
  load15: number
  cpuUsage: number
  memoryUsed: number
  memoryTotal: number
  memoryPercent: number
  storageUsed: number
  storageTotal: number
  storagePercent: number
  services: ServiceStatus[]
}

export interface ServiceStatus {
  name: string
  state: string
  active: boolean
}

// Storage types
export interface StoragePool {
  uuid: string
  label: string
  devices: string[]
  used: number
  total: number
  available: number
  dataProfile: string
  metadataProfile: string
  mountpoint: string
  health: 'healthy' | 'degraded' | 'failed'
}

export interface Snapshot {
  id: string
  subvolume: string
  path: string
  createdAt: string
  size: number
  scheduleId?: string
  tags?: string[]
}

// App types
export interface App {
  id: string
  name: string
  version: string
  status: 'running' | 'stopped' | 'unhealthy' | 'installing' | 'updating'
  health: 'healthy' | 'unhealthy' | 'unknown'
  ports: number[]
  urls: string[]
  installedAt: string
  updatedAt?: string
}

export interface AppManifest {
  meta: {
    name: string
    id: string
    version: string
    icon?: string
    description: string
    upstream?: string
    author?: string
    license?: string
    homepage?: string
    categories?: string[]
  }
  runtime: {
    dockerComposePath: string
    envSchema?: Record<string, any>
    mounts?: Array<{
      name: string
      path: string
      description?: string
      required?: boolean
      persistent?: boolean
    }>
    ports?: Array<{
      port: number
      protocol?: string
      description?: string
      webui?: boolean
    }>
    healthChecks?: Array<{
      type: 'container' | 'http' | 'tcp'
      container?: string
      url?: string
      port?: number
      interval?: number
      timeout?: number
      retries?: number
    }>
  }
  permissions?: {
    apiScopes?: string[]
    agentOps?: string[]
    privileged?: boolean
    hostNetwork?: boolean
    hostPID?: boolean
    capabilities?: string[]
  }
  backup?: {
    includePaths?: string[]
    excludePaths?: string[]
    backupSize?: string
  }
  webui?: {
    path: string
    port: number
    authMode?: 'none' | 'inherit' | 'custom'
    stripPath?: boolean
    headers?: Record<string, string>
  }
}

// Alert types
export interface AlertRule {
  id: string
  name: string
  metric: string
  operator: '>' | '<' | '>=' | '<=' | '==' | '!='
  threshold: number
  duration: number
  severity: 'info' | 'warning' | 'critical'
  enabled: boolean
  currentState: {
    firing: boolean
    since?: string
    value?: number
  }
  createdAt: string
  updatedAt: string
}

export interface AlertChannel {
  id: string
  type: 'email' | 'webhook' | 'ntfy'
  name: string
  enabled: boolean
  config: Record<string, any>
  createdAt: string
  updatedAt: string
}

// Backup types
export interface BackupSchedule {
  id: string
  name: string
  subvols: string[]
  frequency: string // cron expression
  retention: {
    minKeep: number
    days?: number
    weeks?: number
    months?: number
    years?: number
  }
  enabled: boolean
  lastRun?: string
  nextRun?: string
  createdAt: string
  updatedAt: string
}

export interface BackupJob {
  id: string
  type: 'backup' | 'restore' | 'replicate'
  state: 'pending' | 'running' | 'succeeded' | 'failed' | 'cancelled'
  progress: number
  startedAt: string
  finishedAt?: string
  error?: string
  metadata?: Record<string, any>
}

/**
 * NithronOS API Client
 */
export class NOSClient {
  private config: ClientConfig
  private abortControllers: Map<string, AbortController> = new Map()

  constructor(config: ClientConfig) {
    this.config = {
      timeout: 30000,
      ...config,
    }
  }

  // Helper methods

  private async request<T>(
    method: string,
    path: string,
    body?: any,
    options?: RequestInit
  ): Promise<T> {
    const url = `${this.config.baseURL}${path}`
    const abortController = new AbortController()
    const requestId = Math.random().toString(36).substr(2, 9)
    
    this.abortControllers.set(requestId, abortController)
    
    const timeout = setTimeout(() => {
      abortController.abort()
    }, this.config.timeout!)
    
    try {
      const response = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
          ...(this.config.token && { Authorization: `Bearer ${this.config.token}` }),
          ...options?.headers,
        },
        body: body ? JSON.stringify(body) : undefined,
        signal: abortController.signal,
        ...options,
      })
      
      clearTimeout(timeout)
      this.abortControllers.delete(requestId)
      
      if (response.status === 401 && this.config.onUnauthorized) {
        this.config.onUnauthorized()
      }
      
      if (!response.ok) {
        const error = await response.json().catch(() => ({}))
        throw new APIClientError(
          error.message || `Request failed with status ${response.status}`,
          error.code || 'UNKNOWN',
          error.hint,
          error.details
        )
      }
      
      if (response.status === 204) {
        return {} as T
      }
      
      return response.json()
    } catch (error) {
      clearTimeout(timeout)
      this.abortControllers.delete(requestId)
      
      if (error instanceof APIClientError) {
        throw error
      }
      
      if ((error as Error).name === 'AbortError') {
        throw new APIClientError('Request timeout', 'TIMEOUT')
      }
      
      throw new APIClientError(
        (error as Error).message || 'Network error',
        'NETWORK_ERROR'
      )
    }
  }

  private get<T>(path: string, options?: RequestInit): Promise<T> {
    return this.request<T>('GET', path, undefined, options)
  }

  private post<T>(path: string, body?: any, options?: RequestInit): Promise<T> {
    return this.request<T>('POST', path, body, options)
  }

  // Commented out as it's currently unused
  // private put<T>(path: string, body?: any, options?: RequestInit): Promise<T> {
  //   return this.request<T>('PUT', path, body, options)
  // }

  private patch<T>(path: string, body?: any, options?: RequestInit): Promise<T> {
    return this.request<T>('PATCH', path, body, options)
  }

  private delete<T>(path: string, body?: any, options?: RequestInit): Promise<T> {
    return this.request<T>('DELETE', path, body, options)
  }

  // Cancel all pending requests
  cancelAll(): void {
    this.abortControllers.forEach(controller => controller.abort())
    this.abortControllers.clear()
  }

  // API Methods

  // System
  async getSystemInfo(): Promise<SystemInfo> {
    return this.get<SystemInfo>('/api/v1/system/info')
  }

  async getSystemStatus(): Promise<SystemStatus> {
    return this.get<SystemStatus>('/api/v1/system/status')
  }

  // Tokens
  async listTokens(): Promise<APIToken[]> {
    const response = await this.get<{ tokens: APIToken[] }>('/api/v1/tokens')
    return response.tokens
  }

  async createToken(request: CreateTokenRequest): Promise<CreateTokenResponse> {
    return this.post<CreateTokenResponse>('/api/v1/tokens', request)
  }

  async deleteToken(id: string): Promise<void> {
    await this.delete<void>(`/api/v1/tokens/${id}`)
  }

  // Webhooks
  async listWebhooks(): Promise<Webhook[]> {
    const response = await this.get<{ webhooks: Webhook[] }>('/api/v1/webhooks')
    return response.webhooks
  }

  async createWebhook(request: CreateWebhookRequest): Promise<Webhook> {
    return this.post<Webhook>('/api/v1/webhooks', request)
  }

  async updateWebhook(id: string, request: Partial<CreateWebhookRequest>): Promise<Webhook> {
    return this.patch<Webhook>(`/api/v1/webhooks/${id}`, request)
  }

  async deleteWebhook(id: string): Promise<void> {
    await this.delete<void>(`/api/v1/webhooks/${id}`)
  }

  async testWebhook(id: string): Promise<void> {
    await this.post<void>(`/api/v1/webhooks/${id}/test`)
  }

  async getWebhookDeliveries(id: string, limit?: number): Promise<WebhookDelivery[]> {
    const query = limit ? `?limit=${limit}` : ''
    const response = await this.get<{ deliveries: WebhookDelivery[] }>(
      `/api/v1/webhooks/${id}/deliveries${query}`
    )
    return response.deliveries
  }

  // Users
  async listUsers(): Promise<User[]> {
    const response = await this.get<{ users: User[] }>('/api/v1/auth/users')
    return response.users
  }

  async createUser(request: {
    username: string
    email?: string
    password: string
    role: 'admin' | 'operator' | 'viewer'
  }): Promise<User> {
    return this.post<User>('/api/v1/auth/users', request)
  }

  async updateUser(id: string, request: {
    email?: string
    role?: 'admin' | 'operator' | 'viewer'
    enabled?: boolean
  }): Promise<User> {
    return this.patch<User>(`/api/v1/auth/users/${id}`, request)
  }

  async deleteUser(id: string): Promise<void> {
    await this.delete<void>(`/api/v1/auth/users/${id}`)
  }

  // Sessions
  async listSessions(): Promise<Session[]> {
    const response = await this.get<{ sessions: Session[] }>('/api/v1/auth/sessions')
    return response.sessions
  }

  async revokeSession(id?: string): Promise<void> {
    await this.post<void>('/api/v1/auth/sessions/revoke', { sessionId: id })
  }

  // Storage
  async listStoragePools(): Promise<StoragePool[]> {
    const response = await this.get<{ pools: StoragePool[] }>('/api/v1/storage/pools')
    return response.pools
  }

  async getStoragePool(uuid: string): Promise<StoragePool> {
    return this.get<StoragePool>(`/api/v1/storage/pools/${uuid}`)
  }

  async listSnapshots(): Promise<Snapshot[]> {
    const response = await this.get<{ items: Snapshot[] }>('/api/v1/backup/snapshots')
    return response.items
  }

  async createSnapshot(subvols: string[], tag?: string): Promise<BackupJob> {
    return this.post<BackupJob>('/api/v1/backup/snapshots/create', { subvols, tag })
  }

  async deleteSnapshot(id: string): Promise<void> {
    await this.delete<void>(`/api/v1/backup/snapshots/${id}`)
  }

  // Apps
  async listApps(): Promise<App[]> {
    const response = await this.get<{ items: App[] }>('/api/v1/apps/installed')
    return response.items
  }

  async getApp(id: string): Promise<App> {
    return this.get<App>(`/api/v1/apps/${id}`)
  }

  async installApp(id: string, params?: Record<string, any>): Promise<void> {
    await this.post<void>('/api/v1/apps/install', { id, params })
  }

  async uninstallApp(id: string, keepData?: boolean): Promise<void> {
    await this.delete<void>(`/api/v1/apps/${id}`, { keep_data: keepData })
  }

  async startApp(id: string): Promise<void> {
    await this.post<void>(`/api/v1/apps/${id}/start`)
  }

  async stopApp(id: string): Promise<void> {
    await this.post<void>(`/api/v1/apps/${id}/stop`)
  }

  async restartApp(id: string): Promise<void> {
    await this.post<void>(`/api/v1/apps/${id}/restart`)
  }

  // Alerts
  async listAlertRules(): Promise<AlertRule[]> {
    const response = await this.get<{ rules: AlertRule[] }>('/api/v1/monitor/alerts/rules')
    return response.rules
  }

  async createAlertRule(request: {
    name: string
    metric: string
    operator: '>' | '<' | '>=' | '<=' | '==' | '!='
    threshold: number
    duration: number
    severity: 'info' | 'warning' | 'critical'
    enabled: boolean
  }): Promise<AlertRule> {
    return this.post<AlertRule>('/api/v1/monitor/alerts/rules', request)
  }

  async updateAlertRule(id: string, request: Partial<{
    name: string
    enabled: boolean
    threshold: number
    duration: number
    severity: 'info' | 'warning' | 'critical'
  }>): Promise<AlertRule> {
    return this.patch<AlertRule>(`/api/v1/monitor/alerts/rules/${id}`, request)
  }

  async deleteAlertRule(id: string): Promise<void> {
    await this.delete<void>(`/api/v1/monitor/alerts/rules/${id}`)
  }

  async listAlertChannels(): Promise<AlertChannel[]> {
    const response = await this.get<{ channels: AlertChannel[] }>('/api/v1/monitor/alerts/channels')
    return response.channels
  }

  async createAlertChannel(request: {
    type: 'email' | 'webhook' | 'ntfy'
    name: string
    enabled: boolean
    config: Record<string, any>
  }): Promise<AlertChannel> {
    return this.post<AlertChannel>('/api/v1/monitor/alerts/channels', request)
  }

  async testAlertChannel(id: string): Promise<void> {
    await this.post<void>(`/api/v1/monitor/alerts/channels/${id}/test`)
  }

  // Backups
  async listBackupSchedules(): Promise<BackupSchedule[]> {
    const response = await this.get<{ schedules: BackupSchedule[] }>('/api/v1/backup/schedules')
    return response.schedules
  }

  async createBackupSchedule(request: {
    name: string
    subvols: string[]
    frequency: string
    retention: {
      minKeep: number
      days?: number
      weeks?: number
      months?: number
      years?: number
    }
    enabled: boolean
  }): Promise<BackupSchedule> {
    return this.post<BackupSchedule>('/api/v1/backup/schedules', request)
  }

  async updateBackupSchedule(id: string, request: Partial<{
    name: string
    enabled: boolean
    frequency: string
    retention: {
      minKeep: number
      days?: number
      weeks?: number
      months?: number
      years?: number
    }
  }>): Promise<BackupSchedule> {
    return this.patch<BackupSchedule>(`/api/v1/backup/schedules/${id}`, request)
  }

  async deleteBackupSchedule(id: string): Promise<void> {
    await this.delete<void>(`/api/v1/backup/schedules/${id}`)
  }

  async runBackup(scheduleId: string): Promise<BackupJob> {
    return this.post<BackupJob>('/api/v1/backup/run', { schedule_id: scheduleId })
  }

  async listBackupJobs(): Promise<BackupJob[]> {
    const response = await this.get<{ jobs: BackupJob[] }>('/api/v1/backup/jobs')
    return response.jobs
  }

  async getBackupJob(id: string): Promise<BackupJob> {
    return this.get<BackupJob>(`/api/v1/backup/jobs/${id}`)
  }

  async cancelBackupJob(id: string): Promise<void> {
    await this.post<void>(`/api/v1/backup/jobs/${id}/cancel`)
  }

  // OpenAPI
  async getOpenAPISpec(): Promise<any> {
    return this.get<any>('/api/v1/openapi.json')
  }
}

/**
 * API Client Error
 */
export class APIClientError extends Error {
  constructor(
    message: string,
    public code: string,
    public hint?: string,
    public details?: Record<string, any>
  ) {
    super(message)
    this.name = 'APIClientError'
  }
}

/**
 * Create a NithronOS API client
 */
export function createNOSClient(config: ClientConfig): NOSClient {
  return new NOSClient(config)
}

// Export all types
export * from './nos-client'
