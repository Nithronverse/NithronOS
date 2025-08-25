import { useEffect, useState } from 'react'
import { api, type Health, type Disks as Lsblk } from '../api/client'
import { useGlobalNotice } from '@/lib/globalNotice'
import { Activity, HardDrive, Server, AlertCircle, CheckCircle } from 'lucide-react'
import { cn } from '@/lib/utils'

// Loading skeleton component
function CardSkeleton() {
	return (
		<div className="animate-pulse rounded-lg bg-card p-6">
			<div className="mb-4 h-6 w-32 rounded bg-muted/20" />
			<div className="space-y-2">
				<div className="h-4 w-full rounded bg-muted/20" />
				<div className="h-4 w-3/4 rounded bg-muted/20" />
				<div className="h-4 w-1/2 rounded bg-muted/20" />
			</div>
		</div>
	)
}

// Empty state component
function EmptyState({ title, message }: { title: string; message: string }) {
	return (
		<div className="flex flex-col items-center justify-center rounded-lg bg-card p-8 text-center">
			<Server className="mb-4 h-12 w-12 text-muted-foreground" />
			<h3 className="mb-2 text-lg font-medium">{title}</h3>
			<p className="text-sm text-muted-foreground">{message}</p>
		</div>
	)
}

export function Dashboard() {
	const [health, setHealth] = useState<Health | null>(null)
	const [disks, setDisks] = useState<Lsblk | null>(null)
	const [loading, setLoading] = useState(true)
	const [error, setError] = useState<string | null>(null)
	const { notice } = useGlobalNotice()

	useEffect(() => {
		if (notice) return
		
		const fetchData = async () => {
			setLoading(true)
			setError(null)
			
			try {
				const [healthData, disksData] = await Promise.all([
					api.health.get(),
					api.disks.get()
				])
				setHealth(healthData)
				setDisks(disksData)
			} catch (e) {
				setError(String(e))
			} finally {
				setLoading(false)
			}
		}
		
		fetchData()
	}, [notice])

	// Calculate system status
	const systemStatus = health?.alerts?.some((a: any) => a.severity === 'crit')
		? 'critical'
		: health?.alerts?.length
		? 'warning'
		: 'healthy'

	return (
		<div className="space-y-6">
			<div>
				<h1 className="text-3xl font-bold">Dashboard</h1>
				<p className="mt-2 text-muted-foreground">System overview and health status</p>
			</div>

			{error && (
				<div className="rounded-lg border border-destructive bg-destructive/10 p-4">
					<div className="flex items-center gap-2">
						<AlertCircle className="h-5 w-5 text-destructive" />
						<span className="text-sm">{error}</span>
					</div>
				</div>
			)}

			{/* Status Cards */}
			<div className="grid gap-4 md:grid-cols-3">
				{loading ? (
					<>
						<CardSkeleton />
						<CardSkeleton />
						<CardSkeleton />
					</>
				) : (
					<>
						{/* System Status Card */}
						<div className="rounded-lg bg-card p-6">
							<div className="flex items-center justify-between">
								<h3 className="text-sm font-medium text-muted-foreground">System Status</h3>
								<Activity className="h-4 w-4 text-muted-foreground" />
							</div>
							<div className="mt-2">
								<div className={cn(
									"flex items-center gap-2",
									systemStatus === 'critical' && "text-destructive",
									systemStatus === 'warning' && "text-yellow-500",
									systemStatus === 'healthy' && "text-green-500"
								)}>
									{systemStatus === 'healthy' ? (
										<CheckCircle className="h-5 w-5" />
									) : (
										<AlertCircle className="h-5 w-5" />
									)}
									<span className="text-2xl font-bold capitalize">{systemStatus}</span>
								</div>
								<p className="mt-1 text-xs text-muted-foreground">
									{health?.alerts?.length || 0} active alerts
								</p>
							</div>
						</div>

						{/* Storage Card */}
						<div className="rounded-lg bg-card p-6">
							<div className="flex items-center justify-between">
								<h3 className="text-sm font-medium text-muted-foreground">Storage Pools</h3>
								<HardDrive className="h-4 w-4 text-muted-foreground" />
							</div>
							<div className="mt-2">
								<span className="text-2xl font-bold">
									{health?.pools?.length || 0}
								</span>
								<p className="mt-1 text-xs text-muted-foreground">
									Active pools
								</p>
							</div>
						</div>

						{/* Disks Card */}
						<div className="rounded-lg bg-card p-6">
							<div className="flex items-center justify-between">
								<h3 className="text-sm font-medium text-muted-foreground">Physical Disks</h3>
								<Server className="h-4 w-4 text-muted-foreground" />
							</div>
							<div className="mt-2">
								<span className="text-2xl font-bold">
									{disks?.blockdevices?.length || 0}
								</span>
								<p className="mt-1 text-xs text-muted-foreground">
									Connected devices
								</p>
							</div>
						</div>
					</>
				)}
			</div>

			{/* Detailed Sections */}
			<div className="grid gap-4 lg:grid-cols-2">
				{loading ? (
					<>
						<CardSkeleton />
						<CardSkeleton />
					</>
				) : (
					<>
						{/* Health Details */}
						<section className="rounded-lg bg-card p-6">
							<h2 className="mb-4 text-lg font-semibold">Health Details</h2>
							{health ? (
								<div className="space-y-3">
									{health.alerts && health.alerts.length > 0 ? (
										health.alerts.map((alert: any, idx: number) => (
											<div
												key={idx}
												className={cn(
													"rounded-md border p-3",
													alert.severity === 'crit'
														? "border-destructive bg-destructive/10"
														: "border-yellow-600 bg-yellow-600/10"
												)}
											>
												<div className="flex items-center justify-between">
													<span className="font-mono text-sm">{alert.device}</span>
													<span className={cn(
														"rounded px-2 py-0.5 text-xs font-medium",
														alert.severity === 'crit'
															? "bg-destructive text-destructive-foreground"
															: "bg-yellow-600 text-yellow-50"
													)}>
														{alert.severity}
													</span>
												</div>
												<p className="mt-1 text-sm text-muted-foreground">
													{alert.messages?.join(', ')}
												</p>
											</div>
										))
									) : (
										<EmptyState
											title="No Alerts"
											message="Your system is running smoothly with no active alerts."
										/>
									)}
								</div>
							) : (
								<EmptyState
									title="No Health Data"
									message="Health information is currently unavailable."
								/>
							)}
						</section>

						{/* Disk Information */}
						<section className="rounded-lg bg-card p-6">
							<h2 className="mb-4 text-lg font-semibold">Disk Information</h2>
							{disks && disks.blockdevices && disks.blockdevices.length > 0 ? (
								<div className="space-y-3">
									{disks.blockdevices.slice(0, 5).map((disk: any, idx: number) => (
										<div key={idx} className="rounded-md border border-border p-3">
											<div className="flex items-center justify-between">
												<span className="font-mono text-sm">{disk.name}</span>
												<span className="text-xs text-muted-foreground">{disk.size}</span>
											</div>
											<div className="mt-1 flex gap-4 text-xs text-muted-foreground">
												<span>Type: {disk.type || 'Unknown'}</span>
												<span>Model: {disk.model || 'N/A'}</span>
											</div>
										</div>
									))}
									{disks.blockdevices.length > 5 && (
										<p className="text-center text-sm text-muted-foreground">
											+{disks.blockdevices.length - 5} more devices
										</p>
									)}
								</div>
							) : (
								<EmptyState
									title="No Disks Detected"
									message="No storage devices are currently connected to the system."
								/>
							)}
						</section>
					</>
				)}
			</div>
		</div>
	)
}


