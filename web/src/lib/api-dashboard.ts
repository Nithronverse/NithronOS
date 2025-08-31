// Dashboard API types and endpoints
import { apiClient as api } from './api-client'

// Dashboard types
export interface DashboardData {
  system: SystemSummary
  storage: StorageSummary
  disks: DisksSummary
  shares: ShareInfo[]
  apps: AppInfo[]
  maintenance: MaintenanceStatus
  events: EventInfo[]
}

export interface SystemSummary {
  status: 'ok' | 'degraded' | 'critical'
  cpuPct: number
  load1: number
  mem: {
    used: number
    total: number
  }
  swap: {
    used: number
    total: number
  }
  uptimeSec: number
}

export interface StorageSummary {
  totalBytes: number
  usedBytes: number
  poolsOnline: number
  poolsTotal: number
}

export interface DisksSummary {
  total: number
  healthy: number
  warning: number
  critical: number
  lastScanISO: string
}

export interface ShareInfo {
  name: string
  proto: 'SMB' | 'NFS' | 'AFP'
  path: string
  state: 'active' | 'disabled'
}

export interface AppInfo {
  id: string
  name: string
  state: string
  version: string
}

export interface MaintenanceStatus {
  scrub: MaintenanceOp
  balance: MaintenanceOp
}

export interface MaintenanceOp {
  state: 'idle' | 'running' | 'scheduled'
  nextISO: string
}

export interface EventInfo {
  id: string
  timestamp: string
  type: string
  message: string
  severity: 'info' | 'warning' | 'error'
}

// Dashboard API endpoints
export const dashboardApi = {
  // Get all dashboard data
  getDashboard: async (): Promise<DashboardData> => {
    try {
      return await api.get<DashboardData>('/dashboard')
    } catch (error) {
      console.warn('Failed to fetch dashboard data:', error)
      // Return safe defaults
      return {
        system: {
          status: 'ok',
          cpuPct: 0,
          load1: 0,
          mem: { used: 0, total: 1 },
          swap: { used: 0, total: 1 },
          uptimeSec: 0
        },
        storage: {
          totalBytes: 0,
          usedBytes: 0,
          poolsOnline: 0,
          poolsTotal: 0
        },
        disks: {
          total: 0,
          healthy: 0,
          warning: 0,
          critical: 0,
          lastScanISO: new Date().toISOString()
        },
        shares: [],
        apps: [],
        maintenance: {
          scrub: { state: 'idle', nextISO: '' },
          balance: { state: 'idle', nextISO: '' }
        },
        events: []
      }
    }
  },

  // Individual endpoints for granular updates
  getSystemSummary: async (): Promise<SystemSummary> => {
    try {
      return await api.get<SystemSummary>('/health/system')
    } catch (error) {
      return {
        status: 'ok',
        cpuPct: 0,
        load1: 0,
        mem: { used: 0, total: 1 },
        swap: { used: 0, total: 1 },
        uptimeSec: 0
      }
    }
  },

  getStorageSummary: async (): Promise<StorageSummary> => {
    try {
      return await api.get<StorageSummary>('/storage/summary')
    } catch (error) {
      return {
        totalBytes: 0,
        usedBytes: 0,
        poolsOnline: 0,
        poolsTotal: 0
      }
    }
  },

  getDisksSummary: async (): Promise<DisksSummary> => {
    try {
      return await api.get<DisksSummary>('/disks/summary')
    } catch (error) {
      return {
        total: 0,
        healthy: 0,
        warning: 0,
        critical: 0,
        lastScanISO: new Date().toISOString()
      }
    }
  },

  getShares: async (): Promise<ShareInfo[]> => {
    try {
      const response = await api.get<ShareInfo[]>('/shares')
      return Array.isArray(response) ? response : []
    } catch (error) {
      return []
    }
  },

  getInstalledApps: async (): Promise<AppInfo[]> => {
    try {
      const response = await api.get<AppInfo[]>('/apps/installed')
      return Array.isArray(response) ? response : []
    } catch (error) {
      return []
    }
  },

  getMaintenanceStatus: async (): Promise<MaintenanceStatus> => {
    try {
      return await api.get<MaintenanceStatus>('/maintenance/status')
    } catch (error) {
      return {
        scrub: { state: 'idle', nextISO: '' },
        balance: { state: 'idle', nextISO: '' }
      }
    }
  },

  getRecentEvents: async (limit = 10): Promise<EventInfo[]> => {
    try {
      const response = await api.get<EventInfo[]>(`/events/recent?limit=${limit}`)
      return Array.isArray(response) ? response : []
    } catch (error) {
      return []
    }
  }
}
