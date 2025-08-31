// Health API endpoints with proper types
import { api } from './api'

// System health types
export interface SystemHealth {
  cpu: number
  load1: number
  load5: number
  load15: number
  memory: {
    total: number
    used: number
    free: number
    available: number
    usagePct: number
    cached: number
    buffers: number
  }
  swap: {
    total: number
    used: number
    free: number
    usagePct: number
  }
  uptimeSec: number
  tempCpu?: number
  timestamp: number
  network: {
    bytesRecv: number
    bytesSent: number
    packetsRecv: number
    packetsSent: number
    rxSpeed: number
    txSpeed: number
  }
  diskIO: {
    readBytes: number
    writeBytes: number
    readOps: number
    writeOps: number
    readSpeed: number
    writeSpeed: number
  }
}

// Disk health types
export interface DiskHealth {
  id: string
  name: string
  model: string
  serial: string
  sizeBytes: number
  state: 'healthy' | 'warning' | 'critical'
  tempC?: number
  usagePct: number
  smart: {
    passed: boolean
    attrs?: Record<string, any>
    testStatus: string
  }
  filesystem: string
  mountPoint: string
}

// Health API endpoints
export const healthApi = {
  // Get system health metrics
  getSystemHealth: async (): Promise<SystemHealth> => {
    // Let errors propagate so UI can show an unreachable banner
    return await api.get<SystemHealth>('/v1/health/system')
  },

  // Get disk health information
  getDiskHealth: async (): Promise<DiskHealth[]> => {
    const response = await api.get<DiskHealth[]>('/v1/health/disks')
    return Array.isArray(response) ? response : []
  },

  // Get monitoring data (reuses system health)
  getMonitoringData: async (): Promise<SystemHealth> => {
    return await api.get<SystemHealth>('/v1/monitor/system')
  }
}
