// Health API endpoints with proper types
import { apiClient as api } from './api-client'

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
    try {
      const response = await api.get<SystemHealth>('/health/system')
      return response
    } catch (error) {
      console.warn('Failed to fetch system health:', error)
      // Return default values on error
      return {
        cpu: 0,
        load1: 0,
        load5: 0,
        load15: 0,
        memory: {
          total: 0,
          used: 0,
          free: 0,
          available: 0,
          usagePct: 0,
          cached: 0,
          buffers: 0
        },
        swap: {
          total: 0,
          used: 0,
          free: 0,
          usagePct: 0
        },
        uptimeSec: 0,
        timestamp: Date.now() / 1000,
        network: {
          bytesRecv: 0,
          bytesSent: 0,
          packetsRecv: 0,
          packetsSent: 0,
          rxSpeed: 0,
          txSpeed: 0
        },
        diskIO: {
          readBytes: 0,
          writeBytes: 0,
          readOps: 0,
          writeOps: 0,
          readSpeed: 0,
          writeSpeed: 0
        }
      }
    }
  },

  // Get disk health information
  getDiskHealth: async (): Promise<DiskHealth[]> => {
    try {
      const response = await api.get<DiskHealth[]>('/health/disks')
      // Ensure we always return an array
      return Array.isArray(response) ? response : []
    } catch (error) {
      console.warn('Failed to fetch disk health:', error)
      // Return empty array on error
      return []
    }
  },

  // Get monitoring data (reuses system health)
  getMonitoringData: async (): Promise<SystemHealth> => {
    try {
      const response = await api.get<SystemHealth>('/monitor/system')
      return response
    } catch (error) {
      console.warn('Failed to fetch monitoring data:', error)
      // Return default values on error
      return {
        cpu: 0,
        load1: 0,
        load5: 0,
        load15: 0,
        memory: {
          total: 0,
          used: 0,
          free: 0,
          available: 0,
          usagePct: 0,
          cached: 0,
          buffers: 0
        },
        swap: {
          total: 0,
          used: 0,
          free: 0,
          usagePct: 0
        },
        uptimeSec: 0,
        timestamp: Date.now() / 1000,
        network: {
          bytesRecv: 0,
          bytesSent: 0,
          packetsRecv: 0,
          packetsSent: 0,
          rxSpeed: 0,
          txSpeed: 0
        },
        diskIO: {
          readBytes: 0,
          writeBytes: 0,
          readOps: 0,
          writeOps: 0,
          readSpeed: 0,
          writeSpeed: 0
        }
      }
    }
  }
}
