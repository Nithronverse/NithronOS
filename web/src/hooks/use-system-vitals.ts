// Shared hook for real-time system vitals used by Dashboard and Monitoring
import { useEffect, useRef, useState } from 'react'
import { openSSE } from '@/lib/nos-client'
import { useQuery } from '@tanstack/react-query'
import http from '@/lib/nos-client'

export interface SystemVitals {
  cpuPct: number
  memUsed: number
  memTotal: number
  swapUsed: number
  swapTotal: number
  uptime: number
  load1: number
  load5: number
  load15: number
  timestamp: number
}

// Check if SSE endpoint exists
async function checkSSEAvailable(): Promise<boolean> {
  try {
    // Try to connect briefly to see if SSE endpoint exists
    const testSource = openSSE('/v1/metrics/stream')
    return new Promise((resolve) => {
      const timeout = setTimeout(() => {
        testSource.close()
        resolve(false)
      }, 1000)
      
      testSource.onopen = () => {
        clearTimeout(timeout)
        testSource.close()
        resolve(true)
      }
      
      testSource.onerror = () => {
        clearTimeout(timeout)
        testSource.close()
        resolve(false)
      }
    })
  } catch {
    return false
  }
}

// Shared hook for system vitals with SSE fallback to polling
export function useSystemVitals() {
  const [vitals, setVitals] = useState<SystemVitals | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<Error | null>(null)
  const sseRef = useRef<EventSource | null>(null)
  const intervalRef = useRef<number | null>(null)
  const [useSSE, setUseSSE] = useState<boolean | null>(null)
  
  // Check if SSE is available on mount
  useEffect(() => {
    checkSSEAvailable().then(setUseSSE)
  }, [])
  
  // Polling fallback
  const { data: pollingData, error: pollingError } = useQuery({
    queryKey: ['system', 'vitals'],
    queryFn: () => http.get<SystemVitals>('/v1/health/system'),
    enabled: useSSE === false, // Only poll if SSE is not available
    refetchInterval: 1000, // 1Hz
    refetchIntervalInBackground: true,
    staleTime: 500,
  })
  
  // SSE connection
  useEffect(() => {
    if (useSSE === null) return // Still checking
    if (!useSSE) return // Use polling instead
    
    try {
      const eventSource = openSSE('/v1/metrics/stream')
      sseRef.current = eventSource
      
      eventSource.onopen = () => {
        setIsLoading(false)
        setError(null)
      }
      
      eventSource.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          setVitals({
            cpuPct: data.cpu_pct || 0,
            memUsed: data.mem_used || 0,
            memTotal: data.mem_total || 1,
            swapUsed: data.swap_used || 0,
            swapTotal: data.swap_total || 1,
            uptime: data.uptime || 0,
            load1: data.load_1 || 0,
            load5: data.load_5 || 0,
            load15: data.load_15 || 0,
            timestamp: Date.now(),
          })
          setError(null)
        } catch (err) {
          console.error('Failed to parse SSE data:', err)
        }
      }
      
      eventSource.onerror = (err) => {
        console.error('SSE connection error:', err)
        // Fall back to polling
        setUseSSE(false)
        if (sseRef.current) {
          sseRef.current.close()
          sseRef.current = null
        }
      }
    } catch (err) {
      console.error('Failed to open SSE:', err)
      setUseSSE(false)
    }
    
    return () => {
      if (sseRef.current) {
        sseRef.current.close()
        sseRef.current = null
      }
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
        intervalRef.current = null
      }
    }
  }, [useSSE])
  
  // Use polling data if SSE is not available
  useEffect(() => {
    if (useSSE === false && pollingData) {
      setVitals({
        cpuPct: pollingData.cpuPct || 0,
        memUsed: pollingData.memUsed || 0,
        memTotal: pollingData.memTotal || 1,
        swapUsed: pollingData.swapUsed || 0,
        swapTotal: pollingData.swapTotal || 1,
        uptime: pollingData.uptime || 0,
        load1: pollingData.load1 || 0,
        load5: pollingData.load5 || 0,
        load15: pollingData.load15 || 0,
        timestamp: Date.now(),
      })
      setIsLoading(false)
    }
  }, [useSSE, pollingData])
  
  // Handle errors
  useEffect(() => {
    if (useSSE === false && pollingError) {
      setError(pollingError as Error)
      setIsLoading(false)
    }
  }, [useSSE, pollingError])
  
  return {
    vitals,
    isLoading: useSSE === null || isLoading,
    error,
    isSSE: useSSE === true,
  }
}

// Format helpers
export function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  
  if (days > 0) {
    return `${days}d ${hours}h ${minutes}m`
  } else if (hours > 0) {
    return `${hours}h ${minutes}m`
  } else {
    return `${minutes}m`
  }
}

export function formatBytes(bytes: number): string {
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = bytes
  let unitIndex = 0
  
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex++
  }
  
  return `${value.toFixed(1)} ${units[unitIndex]}`
}
