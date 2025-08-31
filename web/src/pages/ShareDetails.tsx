import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, Loader2 } from 'lucide-react'
import { PageHeader } from '@/components/layouts/PageHeader'
import { Button } from '@/components/ui/button'
import { ShareForm } from '@/components/shares/ShareForm'
import { useShare, useUpdateShare, useCreateShare } from '@/hooks/use-shares'
import type { CreateShareRequest, UpdateShareRequest } from '@/api/client'

export function ShareDetails() {
	const { name } = useParams<{ name: string }>()
	const navigate = useNavigate()
	const isNew = !name || name === 'new'
	
	const { data: share, isLoading } = useShare(isNew ? undefined : name)
	const updateShare = useUpdateShare()
	const createShare = useCreateShare()

	const handleSubmit = async (data: CreateShareRequest | UpdateShareRequest) => {
		try {
			if (isNew) {
				await createShare.mutateAsync(data as CreateShareRequest)
				navigate('/shares')
			} else if (name) {
				await updateShare.mutateAsync({ name, data: data as UpdateShareRequest })
				navigate('/shares')
			}
		} catch (err) {
			console.error('Failed to save share:', err)
		}
	}

	const handleCancel = () => {
		navigate('/shares')
	}

	if (isLoading && !isNew) {
		return (
			<div className="flex items-center justify-center h-full">
				<Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
			</div>
		)
	}

	if (!isNew && !share && !isLoading) {
		return (
			<div className="flex flex-col h-full">
				<PageHeader
					title="Share Not Found"
					description="The requested share could not be found"
				/>
				<div className="flex-1 flex items-center justify-center">
					<div className="text-center space-y-4">
						<p className="text-muted-foreground">
							The share "{name}" does not exist or has been deleted.
						</p>
						<Button onClick={() => navigate('/shares')}>
							<ArrowLeft className="h-4 w-4 mr-2" />
							Back to Shares
						</Button>
					</div>
				</div>
			</div>
		)
	}

	return (
		<div className="flex flex-col h-full">
			<PageHeader
				title={isNew ? 'Create Share' : `Edit Share: ${name}`}
				description={isNew ? 'Configure a new network file share' : 'Modify share settings and permissions'}
				breadcrumbs={[
					{ label: 'Shares', href: '/shares' },
					{ label: isNew ? 'New' : name || '' }
				]}
			/>
			
			<div className="flex-1 p-6 max-w-4xl mx-auto w-full">
				<ShareForm
					share={share || undefined}
					onSubmit={handleSubmit}
					onCancel={handleCancel}
					mode={isNew ? 'create' : 'edit'}
				/>
			</div>
		</div>
	)
}
