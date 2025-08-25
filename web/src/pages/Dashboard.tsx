import { useEffect, useState } from 'react'
import { useGlobalNotice } from '@/lib/globalNotice'
import { Activity, HardDrive, Server, AlertCircle, CheckCircle } from 'lucide-react'
import { cn } from '@/lib/utils'
import api from '@/lib/api'

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

// Format bytes to human readable
function formatBytes(bytes: number): string {
	if (bytes === 0) return '0 B'
	const k = 1024
	const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
	const i = Math.floor(Math.log(bytes) / Math.log(k))
	return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

export function Dashboard() {
	const [alerts, setAlerts] = useState<any[]>([]) 
	const [pools, setPools] = useState<any[]>([])
	const [loading, setLoading] = useState(true)
	const [error, setError] = useState<string | null>(null)
	const { notice } = useGlobalNotice()

	useEffect(() => {
		if (notice) return
		
		const fetchData = async () => {
			setLoading(true)
			setError(null)
			
			try {
				const [alertsData, poolsData] = await Promise.all([
					api.health.alerts(),
					api.pools.list()
				])
				setAlerts(alertsData.alerts || [])
				setPools(poolsData || [])
			} catch (e) {
				setError(String(e))
			} finally {
				setLoading(false)
			}
		}
		
		fetchData()
	}, [notice])

	// Calculate system status
	const systemStatus = alerts.some((a: any) => a.severity === 'crit')
		? 'critical'
		: alerts.length > 0
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
									{alerts.length} active alerts
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
									{pools.length}
								</span>
								<p className="mt-1 text-xs text-muted-foreground">
									Active pools
								</p>
							</div>
						</div>

						{/* Shares Card */}
						<div className="rounded-lg bg-card p-6">
							<div className="flex items-center justify-between">
								<h3 className="text-sm font-medium text-muted-foreground">Network Shares</h3>
								<Server className="h-4 w-4 text-muted-foreground" />
							</div>
							<div className="mt-2">
								<span className="text-2xl font-bold">
									-
								</span>
								<p className="mt-1 text-xs text-muted-foreground">
									Active shares
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
							<h2 className="mb-4 text-lg font-semibold">Health Alerts</h2>
							{!loading ? (
								<div className="space-y-3">
									{alerts.length > 0 ? (
										alerts.map((alert: any, idx: number) => (
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
									title="Loading..."
									message="Fetching health information."
								/>
							)}
						</section>

						{/* Pool Information */}
						<section className="rounded-lg bg-card p-6">
							<h2 className="mb-4 text-lg font-semibold">Storage Pools</h2>
							{!loading ? (
								<div className="space-y-3">
									{pools.length > 0 ? (
										pools.slice(0, 5).map((pool: any, idx: number) => (
											<div key={idx} className="rounded-md border border-border p-3">
												<div className="flex items-center justify-between">
													<span className="font-mono text-sm">{pool.label || pool.id}</span>
													<span className="text-xs text-muted-foreground">{pool.mountpoint}</span>
												</div>
												{(pool.size || pool.free) && (
													<div className="mt-1 flex gap-4 text-xs text-muted-foreground">
														{pool.size && <span>Size: {formatBytes(pool.size)}</span>}
														{pool.free && <span>Free: {formatBytes(pool.free)}</span>}
													</div>
												)}
											</div>
										))
									) : (
										<EmptyState
											title="No Storage Pools"
											message="No storage pools have been configured."
										/>
									)}
									{pools.length > 5 && (
										<p className="text-center text-sm text-muted-foreground">
											+{pools.length - 5} more pools
										</p>
									)}
								</div>
							) : (
								<EmptyState
									title="Loading..."
									message="Fetching pool information."
								/>
							)}
						</section>
					</>
				)}
			</div>
		</div>
	)
}


