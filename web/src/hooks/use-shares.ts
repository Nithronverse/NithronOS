import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/api/client'
import type { CreateShareRequest, UpdateShareRequest } from '@/api/client'
import { toast } from '@/components/ui/toast'

export function useShares() {
	return useQuery({
		queryKey: ['shares'],
		queryFn: api.shares.get,
		staleTime: 30000,
	})
}

export function useShare(name?: string) {
	return useQuery({
		queryKey: ['shares', name],
		queryFn: async () => {
			if (!name) return null
			const shares = await api.shares.get()
			return shares.find(s => s.name === name) || null
		},
		enabled: !!name,
	})
}

export function useCreateShare() {
	const queryClient = useQueryClient()
	
	return useMutation({
		mutationFn: (data: CreateShareRequest) => api.shares.create(data),
		onSuccess: (newShare) => {
			queryClient.invalidateQueries({ queryKey: ['shares'] })
			toast.success(`Share "${newShare.name}" created successfully`)
		},
		onError: (error: any) => {
			const message = error.data?.error?.message || error.message || 'Failed to create share'
			toast.error(message)
		},
	})
}

export function useUpdateShare() {
	const queryClient = useQueryClient()
	
	return useMutation({
		mutationFn: ({ name, data }: { name: string; data: UpdateShareRequest }) => 
			api.shares.update(name, data),
		onSuccess: (updatedShare) => {
			queryClient.invalidateQueries({ queryKey: ['shares'] })
			queryClient.invalidateQueries({ queryKey: ['shares', updatedShare.name] })
			toast.success(`Share "${updatedShare.name}" updated successfully`)
		},
		onError: (error: any) => {
			const message = error.data?.error?.message || error.message || 'Failed to update share'
			toast.error(message)
		},
	})
}

export function useDeleteShare() {
	const queryClient = useQueryClient()
	
	return useMutation({
		mutationFn: (name: string) => api.shares.delete(name),
		onSuccess: (_, name) => {
			queryClient.invalidateQueries({ queryKey: ['shares'] })
			toast.success(`Share "${name}" deleted successfully`)
		},
		onError: (error: any) => {
			const message = error.data?.error?.message || error.message || 'Failed to delete share'
			toast.error(message)
		},
	})
}

export function useTestShare() {
	return useMutation({
		mutationFn: ({ name, config }: { name: string; config: any }) => 
			api.shares.test(name, config),
		onError: (error: any) => {
			const message = error.data?.error?.message || error.message || 'Validation failed'
			toast.error(message)
		},
	})
}

export function useUsers() {
	return useQuery({
		queryKey: ['users'],
		queryFn: api.users.get,
		staleTime: 60000,
	})
}

export function useGroups() {
	return useQuery({
		queryKey: ['groups'],
		queryFn: api.groups.get,
		staleTime: 60000,
	})
}

export function usePolicy() {
	return useQuery({
		queryKey: ['policy'],
		queryFn: api.policy.get,
		staleTime: 300000, // 5 minutes
	})
}
