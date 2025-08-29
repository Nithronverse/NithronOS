import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import React from 'react'
import { 
  useSystemInfo, 
  usePools, 
  useSmartSummary, 
  useShares,
  useInstalledApps,
} from '../hooks/use-api'

// Mock the fetch function
global.fetch = vi.fn()

// Create a wrapper with QueryClient
const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { 
        retry: false,
        gcTime: 0,
        staleTime: 0,
      },
    },
  })
  
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  )
}

describe('API Wiring Tests', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('System API', () => {
    it('should fetch system info from /api/v1/system/info', async () => {
      const mockData = {
        hostname: 'nithronos',
        uptime: 123456,
        kernel: '5.15.0',
        version: '0.3.0',
      }

      ;(global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockData,
        headers: new Headers({ 'content-type': 'application/json' }),
      })

      const { result } = renderHook(() => useSystemInfo(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => expect(result.current.isSuccess).toBe(true))

      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/system/info'),
        expect.any(Object)
      )
      expect(result.current.data).toEqual(mockData)
    })
  })

  describe('Storage API', () => {
    it('should fetch pools from /api/v1/storage/pools', async () => {
      const mockPools = [
        {
          id: 'pool1',
          uuid: 'uuid1',
          label: 'Main Pool',
          mountpoint: '/mnt/main',
          size: 1000000,
          used: 500000,
          free: 500000,
          raid: 'raid1',
          status: 'online',
          devices: [],
        },
      ]

      ;(global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockPools,
        headers: new Headers({ 'content-type': 'application/json' }),
      })

      const { result } = renderHook(() => usePools(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => expect(result.current.isSuccess).toBe(true))

      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/storage/pools'),
        expect.any(Object)
      )
      expect(result.current.data).toEqual(mockPools)
    })
  })

  describe('Health API', () => {
    it('should fetch SMART summary from /api/v1/health/smart/summary', async () => {
      const mockSummary = {
        totalDevices: 4,
        healthyDevices: 3,
        warningDevices: 1,
        criticalDevices: 0,
      }

      ;(global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockSummary,
        headers: new Headers({ 'content-type': 'application/json' }),
      })

      const { result } = renderHook(() => useSmartSummary(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => expect(result.current.isSuccess).toBe(true))

      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/health/smart/summary'),
        expect.any(Object)
      )
      expect(result.current.data).toEqual(mockSummary)
    })
  })

  describe('Shares API', () => {
    it('should fetch shares from /api/v1/shares', async () => {
      const mockShares = [
        {
          name: 'Documents',
          path: '/mnt/main/docs',
          protocol: 'smb',
          enabled: true,
        },
      ]

      ;(global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockShares,
        headers: new Headers({ 'content-type': 'application/json' }),
      })

      const { result } = renderHook(() => useShares(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => expect(result.current.isSuccess).toBe(true))

      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/shares'),
        expect.any(Object)
      )
      expect(result.current.data).toEqual(mockShares)
    })
  })

  describe('Apps API', () => {
    it('should fetch installed apps from /api/v1/apps/installed', async () => {
      const mockApps = {
        items: [
          {
            id: 'nextcloud',
            name: 'Nextcloud',
            version: '28.0',
            status: 'running',
          },
        ],
      }

      ;(global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockApps,
        headers: new Headers({ 'content-type': 'application/json' }),
      })

      const { result } = renderHook(() => useInstalledApps(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => expect(result.current.isSuccess).toBe(true))

      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/apps/installed'),
        expect.any(Object)
      )
      expect(result.current.data).toEqual(mockApps.items)
    })
  })

  describe('Error Handling', () => {
    it('should handle backend unreachable error', async () => {
      ;(global.fetch as any).mockResolvedValueOnce({
        ok: false,
        status: 502,
        headers: new Headers({ 'content-type': 'text/html' }),
        text: async () => '<html>502 Bad Gateway</html>',
        json: async () => { throw new Error('Not JSON') },
      })

      const { result } = renderHook(() => useSystemInfo(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => expect(result.current.isError).toBe(true), { timeout: 5000 })
      
      expect(result.current.error).toBeDefined()
    })

    it('should handle JSON parse errors gracefully', async () => {
      ;(global.fetch as any).mockResolvedValueOnce({
        ok: true,
        headers: new Headers({ 'content-type': 'application/json' }),
        json: async () => {
          throw new Error('Invalid JSON')
        },
        text: async () => 'not json',
      })

      const { result } = renderHook(() => useSystemInfo(), {
        wrapper: createWrapper(),
      })

      await waitFor(() => expect(result.current.isError).toBe(true), { timeout: 5000 })
      
      expect(result.current.error).toBeDefined()
    })
  })
})
