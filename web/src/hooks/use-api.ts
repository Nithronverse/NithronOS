import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { endpoints, type PoolSummary, type Device, 
         type SmartData, type SmartSummary, type ScrubStatus, type BalanceStatus,
         type Schedule, type Service, type Job, type Share, type AppCatalogEntry, 
         type InstalledApp } from '@/lib/api'

// ============================================================================
// System Queries
// ============================================================================

export function useSystemInfo() {
  return useQuery({
    queryKey: ['system', 'info'],
    queryFn: endpoints.system.info,
    staleTime: 10_000,
    retry: 1,
  })
}

export function useServices() {
  return useQuery({
    queryKey: ['system', 'services'],
    queryFn: endpoints.system.services,
    staleTime: 5_000,
    retry: 1,
  })
}

// ============================================================================
// Storage Pool Queries (M1)
// ============================================================================

export function usePools() {
  return useQuery({
    queryKey: ['pools'],
    queryFn: endpoints.pools.list,
    staleTime: 10_000,
    retry: 1,
  })
}

export function usePoolsSummary() {
  return useQuery({
    queryKey: ['pools', 'summary'],
    queryFn: endpoints.pools.summary,
    staleTime: 10_000,
    retry: 1,
  })
}

export function usePool(uuid: string) {
  return useQuery({
    queryKey: ['pools', uuid],
    queryFn: () => endpoints.pools.get(uuid),
    enabled: !!uuid,
    staleTime: 10_000,
    retry: 1,
  })
}

export function usePoolSubvolumes(uuid: string) {
  return useQuery({
    queryKey: ['pools', uuid, 'subvolumes'],
    queryFn: () => endpoints.pools.subvolumes(uuid),
    enabled: !!uuid,
    staleTime: 10_000,
    retry: 1,
  })
}

export function usePoolMountOptions(uuid: string) {
  return useQuery({
    queryKey: ['pools', uuid, 'mount-options'],
    queryFn: () => endpoints.pools.getMountOptions(uuid),
    enabled: !!uuid,
    staleTime: 30_000,
    retry: 1,
  })
}

export function useUpdatePoolMountOptions() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ uuid, options }: { uuid: string; options: string }) =>
      endpoints.pools.setMountOptions(uuid, options),
    onSuccess: (_, { uuid }) => {
      queryClient.invalidateQueries({ queryKey: ['pools', uuid, 'mount-options'] })
    },
  })
}

// ============================================================================
// Storage Device Queries
// ============================================================================

export function useDevices() {
  return useQuery({
    queryKey: ['devices'],
    queryFn: endpoints.devices.list,
    staleTime: 10_000,
    retry: 1,
  })
}

// ============================================================================
// SMART Health Queries (M1)
// ============================================================================

export function useSmartSummary() {
  return useQuery({
    queryKey: ['smart', 'summary'],
    queryFn: endpoints.smart.summary,
    staleTime: 10_000,
    retry: 1,
  })
}

export function useSmartDevice(device: string) {
  return useQuery({
    queryKey: ['smart', device],
    queryFn: () => endpoints.smart.device(device),
    enabled: !!device,
    staleTime: 10_000,
    retry: 1,
  })
}

export function useSmartScan() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: endpoints.smart.scan,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['smart'] })
    },
  })
}

// ============================================================================
// Scrub Queries (M1)
// ============================================================================

export function useScrubStatus() {
  return useQuery({
    queryKey: ['scrub', 'status'],
    queryFn: endpoints.scrub.status,
    staleTime: 5_000,
    retry: 1,
    refetchInterval: (data) => {
      // Poll more frequently if scrub is running
      const hasRunning = data?.some(s => s.status === 'running')
      return hasRunning ? 5_000 : false
    },
  })
}

export function useStartScrub() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: endpoints.scrub.start,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['scrub'] })
    },
  })
}

export function useCancelScrub() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: endpoints.scrub.cancel,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['scrub'] })
    },
  })
}

// ============================================================================
// Balance Queries (M1)
// ============================================================================

export function useBalanceStatus() {
  return useQuery({
    queryKey: ['balance', 'status'],
    queryFn: endpoints.balance.status,
    staleTime: 5_000,
    retry: 1,
    refetchInterval: (data) => {
      // Poll more frequently if balance is running
      const hasRunning = data?.some(b => b.status === 'running')
      return hasRunning ? 5_000 : false
    },
  })
}

export function useStartBalance() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: endpoints.balance.start,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['balance'] })
    },
  })
}

export function useCancelBalance() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: endpoints.balance.cancel,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['balance'] })
    },
  })
}

// ============================================================================
// Schedule Queries (M1)
// ============================================================================

export function useSchedules() {
  return useQuery({
    queryKey: ['schedules'],
    queryFn: endpoints.schedules.list,
    staleTime: 30_000,
    retry: 1,
  })
}

export function useCreateSchedule() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: endpoints.schedules.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['schedules'] })
    },
  })
}

export function useUpdateSchedule() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ id, schedule }: { id: string; schedule: Partial<Schedule> }) =>
      endpoints.schedules.update(id, schedule),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['schedules'] })
    },
  })
}

export function useDeleteSchedule() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: endpoints.schedules.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['schedules'] })
    },
  })
}

// ============================================================================
// Job Queries
// ============================================================================

export function useRecentJobs(limit = 10) {
  return useQuery({
    queryKey: ['jobs', 'recent', limit],
    queryFn: () => endpoints.jobs.recent(limit),
    staleTime: 5_000,
    retry: 1,
  })
}

// ============================================================================
// Share Queries (M2)
// ============================================================================

export function useShares() {
  return useQuery({
    queryKey: ['shares'],
    queryFn: endpoints.shares.list,
    staleTime: 10_000,
    retry: 1,
  })
}

export function useShare(name: string) {
  return useQuery({
    queryKey: ['shares', name],
    queryFn: () => endpoints.shares.get(name),
    enabled: !!name,
    staleTime: 10_000,
    retry: 1,
  })
}

export function useCreateShare() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: endpoints.shares.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['shares'] })
    },
  })
}

export function useUpdateShare() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ name, share }: { name: string; share: Partial<Share> }) =>
      endpoints.shares.update(name, share),
    onSuccess: (_, { name }) => {
      queryClient.invalidateQueries({ queryKey: ['shares'] })
      queryClient.invalidateQueries({ queryKey: ['shares', name] })
    },
  })
}

export function useDeleteShare() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: endpoints.shares.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['shares'] })
    },
  })
}

export function useTestShare() {
  return useMutation({
    mutationFn: ({ name, config }: { name: string; config: any }) =>
      endpoints.shares.test(name, config),
  })
}

// ============================================================================
// App Queries (M3)
// ============================================================================

export function useAppCatalog() {
  return useQuery({
    queryKey: ['apps', 'catalog'],
    queryFn: async () => {
      const data = await endpoints.apps.catalog()
      return data.entries
    },
    staleTime: 60_000,
    retry: 1,
  })
}

export function useInstalledApps() {
  return useQuery({
    queryKey: ['apps', 'installed'],
    queryFn: async () => {
      const data = await endpoints.apps.installed()
      return data.items
    },
    staleTime: 10_000,
    retry: 1,
  })
}

export function useApp(id: string) {
  return useQuery({
    queryKey: ['apps', id],
    queryFn: () => endpoints.apps.get(id),
    enabled: !!id,
    staleTime: 5_000,
    retry: 1,
  })
}

export function useInstallApp() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: endpoints.apps.install,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['apps'] })
    },
  })
}

export function useAppActions() {
  const queryClient = useQueryClient()
  
  return {
    start: useMutation({
      mutationFn: endpoints.apps.start,
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['apps'] })
      },
    }),
    stop: useMutation({
      mutationFn: endpoints.apps.stop,
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['apps'] })
      },
    }),
    restart: useMutation({
      mutationFn: endpoints.apps.restart,
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['apps'] })
      },
    }),
    upgrade: useMutation({
      mutationFn: ({ id, params }: { id: string; params?: Record<string, any> }) =>
        endpoints.apps.upgrade(id, params),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['apps'] })
      },
    }),
    delete: useMutation({
      mutationFn: ({ id, keepData }: { id: string; keepData: boolean }) =>
        endpoints.apps.delete(id, keepData),
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['apps'] })
      },
    }),
  }
}

// ============================================================================
// Error Handling Helper
// ============================================================================

export function useApiErrorHandler() {
  const queryClient = useQueryClient()
  
  return {
    handleError: (error: any) => {
      // Check for proxy error
      if (error?.status === 502 || error?.message?.includes('Backend unreachable')) {
        // Set global error state that UI can check
        queryClient.setQueryData(['api', 'status'], { 
          isReachable: false,
          error: 'Backend unreachable or proxy misconfigured' 
        })
        return
      }
      
      // Check for auth error
      if (error?.status === 401) {
        // Redirect to login
        window.location.href = '/login'
        return
      }
      
      // Log other errors
      console.error('API Error:', error)
    },
    
    clearError: () => {
      queryClient.setQueryData(['api', 'status'], { isReachable: true })
    }
  }
}

export function useApiStatus() {
  return useQuery({
    queryKey: ['api', 'status'],
    queryFn: async () => {
      // Check if API is reachable
      try {
        await endpoints.system.info()
        return { isReachable: true }
      } catch (error: any) {
        if (error?.status === 502 || error?.message?.includes('Backend unreachable')) {
          return { 
            isReachable: false, 
            error: 'Backend unreachable or proxy misconfigured' 
          }
        }
        throw error
      }
    },
    staleTime: 30_000,
    retry: false,
  })
}