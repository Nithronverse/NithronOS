import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api-client'

// Health hook
export function useHealth() {
  return useQuery({
    queryKey: ['health'],
    queryFn: () => apiClient.getHealth(),
    refetchInterval: 30000, // Poll every 30 seconds
  })
}

// Disks hook
export function useDisks() {
  return useQuery({
    queryKey: ['disks'],
    queryFn: () => apiClient.getDisks(),
    staleTime: 60000, // 1 minute
  })
}

// Volumes hook
export function useVolumes() {
  return useQuery({
    queryKey: ['volumes'],
    queryFn: () => apiClient.getVolumes(),
    staleTime: 60000,
  })
}

// Shares hook
export function useShares() {
  return useQuery({
    queryKey: ['shares'],
    queryFn: () => apiClient.getShares(),
    staleTime: 30000,
  })
}

// Share mutations
export function useCreateShare() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: apiClient.createShare,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['shares'] })
    },
  })
}

export function useUpdateShare() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: any }) =>
      apiClient.updateShare(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['shares'] })
    },
  })
}

export function useDeleteShare() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: apiClient.deleteShare,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['shares'] })
    },
  })
}

// Apps hook
export function useApps() {
  return useQuery({
    queryKey: ['apps'],
    queryFn: () => apiClient.getApps(),
    staleTime: 60000,
  })
}

// Marketplace hook
export function useMarketplace() {
  return useQuery({
    queryKey: ['marketplace'],
    queryFn: () => apiClient.getMarketplace(),
    staleTime: 300000, // 5 minutes
  })
}

// Users hook
export function useUsers() {
  return useQuery({
    queryKey: ['users'],
    queryFn: () => apiClient.getUsers(),
    staleTime: 60000,
  })
}

// Schedules hook
export function useSchedules() {
  return useQuery({
    queryKey: ['schedules'],
    queryFn: () => apiClient.getSchedules(),
    staleTime: 30000,
  })
}

// Schedule mutations
export function useUpdateSchedule() {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: any }) =>
      apiClient.updateSchedule(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['schedules'] })
    },
  })
}