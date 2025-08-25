import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/api/client'
import type { CreateShareRequest, UpdateShareRequest } from '@/api/client'
import { pushToast } from '@/components/ui/toast'

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
			pushToast(`Share "${newShare.name}" created successfully`, 'success')
		},
		onError: (error: any) => {
			const message = error.data?.error?.message || error.message || 'Failed to create share'
			pushToast(message, 'error')
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
			pushToast(`Share "${updatedShare.name}" updated successfully`, 'success')
		},
		onError: (error: any) => {
			const message = error.data?.error?.message || error.message || 'Failed to update share'
			pushToast(message, 'error')
		},
	})
}

export function useDeleteShare() {
	const queryClient = useQueryClient()
	
	return useMutation({
		mutationFn: (name: string) => api.shares.delete(name),
		onSuccess: (_, name) => {
			queryClient.invalidateQueries({ queryKey: ['shares'] })
			pushToast(`Share "${name}" deleted successfully`, 'success')
		},
		onError: (error: any) => {
			const message = error.data?.error?.message || error.message || 'Failed to delete share'
			pushToast(message, 'error')
		},
	})
}

export function useTestShare() {
	return useMutation({
		mutationFn: ({ name, config }: { name: string; config: any }) => 
			api.shares.test(name, config),
		onError: (error: any) => {
			const message = error.data?.error?.message || error.message || 'Validation failed'
			pushToast(message, 'error')
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
