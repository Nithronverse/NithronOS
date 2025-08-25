import { useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { 
  Activity, 
  HardDrive, 
  Share2, 
  Package,
  RefreshCw,
  Download,
  Clock,
  Cpu,
  MemoryStick,
  AlertCircle,
  CheckCircle,
  XCircle,
  Server,
  Database,
  FolderOpen,
} from 'lucide-react'
import { PageHeader } from '@/components/ui/page-header'
import { Card } from '@/components/ui/card-enhanced'
import { EmptyState } from '@/components/ui/empty-state'
import { StatusPill, HealthBadge, Metric } from '@/components/ui/status'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Skeleton } from '@/components/ui/skeleton'
import { 
  useSystemInfo, 
  usePoolsSummary, 
  useSmartSummary,
  useScrubStatus,
  useBalanceStatus,
  useRecentJobs,
  useShares,
  useInstalledApps,
  useApiStatus,
} from '@/hooks/use-api'
import { cn } from '@/lib/utils'
import { 
  PieChart, 
  Pie, 
  Cell, 
  ResponsiveContainer, 
  Tooltip,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid
} from 'recharts'

// Helper functions
function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

// Animation variants
const containerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.05
    }
  }
}

const itemVariants = {
  hidden: { opacity: 0, y: 20 },
  visible: {
    opacity: 1,
    y: 0,
    transition: {
      type: "spring" as const,
      stiffness: 100
    }
  }
}

export function Dashboard() {
  const [isRefreshing, setIsRefreshing] = useState(false)
  
  // Check API status
  const { data: apiStatus } = useApiStatus()
  
  // Fetch real data from API
  const { data: systemInfo, isLoading: systemLoading, refetch: refetchSystem } = useSystemInfo()
  const { data: poolsSummary, isLoading: poolsLoading, refetch: refetchPools } = usePoolsSummary()
  const { data: smartSummary, isLoading: smartLoading, refetch: refetchSmart } = useSmartSummary()
  const { data: scrubStatus, isLoading: scrubLoading, refetch: refetchScrub } = useScrubStatus()
  const { data: balanceStatus, isLoading: balanceLoading, refetch: refetchBalance } = useBalanceStatus()
  const { data: recentJobs, isLoading: jobsLoading, refetch: refetchJobs } = useRecentJobs(10)
  const { data: shares, isLoading: sharesLoading, refetch: refetchShares } = useShares()
  const { data: apps, isLoading: appsLoading, refetch: refetchApps } = useInstalledApps()

  const handleRefresh = async () => {
    setIsRefreshing(true)
    await Promise.all([
      refetchSystem(),
      refetchPools(),
      refetchSmart(),
      refetchScrub(),
      refetchBalance(),
      refetchJobs(),
      refetchShares(),
      refetchApps(),
    ])
    setTimeout(() => setIsRefreshing(false), 500)
  }

  // Show backend error banner if API is unreachable
  if (apiStatus && !apiStatus.isReachable) {
    return (
      <div className="space-y-6">
        <PageHeader
          title="Dashboard"
          description="System overview and health status"
        />
        <Alert className="border-destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            Backend unreachable or proxy misconfigured. Please check that the backend service is running.
          </AlertDescription>
        </Alert>
      </div>
    )
  }

  // Calculate overall health status based on SMART and pool status
  const overallHealth = (() => {
    if (!smartSummary && !poolsSummary) return 'unknown'
    if (smartSummary?.criticalDevices || poolsSummary?.poolsDegraded) return 'critical'
    if (smartSummary?.warningDevices) return 'degraded'
    return 'healthy'
  })()

  // Calculate storage usage
  const storageData = poolsSummary ? [
    { name: 'Used', value: poolsSummary.totalUsed, fill: 'hsl(var(--primary))' },
    { name: 'Free', value: poolsSummary.totalSize - poolsSummary.totalUsed, fill: 'hsl(var(--muted))' },
  ] : []

  const storagePercentage = poolsSummary && poolsSummary.totalSize > 0 
    ? (poolsSummary.totalUsed / poolsSummary.totalSize) * 100 
    : 0

  // Get running scrub/balance operations
  const runningScrubs = scrubStatus?.filter(s => s.status === 'running') || []
  const runningBalances = balanceStatus?.filter(b => b.status === 'running') || []

  return (
    <div className="space-y-6">
      <PageHeader
        title="Dashboard"
        description="System overview and health status"
        actions={
          <>
            <Button
              variant="outline"
              size="sm"
              onClick={handleRefresh}
              disabled={isRefreshing}
            >
              <RefreshCw className={cn("h-4 w-4 mr-2", isRefreshing && "animate-spin")} />
              Refresh
            </Button>
            <Button size="sm">
              <Download className="h-4 w-4 mr-2" />
              Check Updates
            </Button>
          </>
        }
      />

      <motion.div
        variants={containerVariants}
        initial="hidden"
        animate="visible"
        className="grid gap-4 md:grid-cols-2 lg:grid-cols-4"
      >
        {/* System Health Card */}
        <motion.div variants={itemVariants}>
          <Card
            title="System Health"
            isLoading={systemLoading || smartLoading}
            className="h-full"
          >
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Overall Status</p>
                  <div className="mt-1">
                    <HealthBadge status={overallHealth} />
                  </div>
                </div>
              </div>
              
              {systemInfo && (
                <>
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground flex items-center gap-1">
                        <Cpu className="h-3 w-3" />
                        CPU
                      </span>
                      <span>{systemInfo.cpuCount || 'N/A'} cores</span>
                    </div>
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground flex items-center gap-1">
                        <MemoryStick className="h-3 w-3" />
                        Memory
                      </span>
                      <span>
                        {systemInfo.memoryUsed && systemInfo.memoryTotal 
                          ? `${formatBytes(systemInfo.memoryUsed)} / ${formatBytes(systemInfo.memoryTotal)}`
                          : 'N/A'}
                      </span>
                    </div>
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground flex items-center gap-1">
                        <Clock className="h-3 w-3" />
                        Uptime
                      </span>
                      <span>{formatUptime(systemInfo.uptime)}</span>
                    </div>
                  </div>
                  <div className="pt-2 border-t text-xs text-muted-foreground">
                    {systemInfo.hostname} • {systemInfo.kernel}
                  </div>
                </>
              )}
            </div>
          </Card>
        </motion.div>

        {/* Storage Overview Card */}
        <motion.div variants={itemVariants}>
          <Card
            title="Storage"
            isLoading={poolsLoading}
            className="h-full"
          >
            {poolsSummary ? (
              <div className="space-y-4">
                <div className="h-[100px]">
                  <ResponsiveContainer width="100%" height="100%">
                    <PieChart>
                      <Pie
                        data={storageData}
                        cx="50%"
                        cy="50%"
                        innerRadius={30}
                        outerRadius={40}
                        dataKey="value"
                        strokeWidth={0}
                      >
                        {storageData.map((entry, index) => (
                          <Cell key={`cell-${index}`} fill={entry.fill} />
                        ))}
                      </Pie>
                      <Tooltip
                        formatter={(value: number) => formatBytes(value)}
                        contentStyle={{
                          backgroundColor: 'hsl(var(--card))',
                          border: '1px solid hsl(var(--border))',
                          borderRadius: '6px',
                        }}
                      />
                    </PieChart>
                  </ResponsiveContainer>
                </div>
                
                <div className="space-y-2">
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Used</span>
                    <span>{formatBytes(poolsSummary.totalUsed)}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Total</span>
                    <span>{formatBytes(poolsSummary.totalSize)}</span>
                  </div>
                  <div className="w-full bg-secondary rounded-full h-2">
                    <div
                      className="bg-primary h-2 rounded-full transition-all"
                      style={{ width: `${Math.min(storagePercentage, 100)}%` }}
                    />
                  </div>
                </div>

                <div className="pt-2 border-t text-xs text-muted-foreground">
                  {poolsSummary.totalPools} pool{poolsSummary.totalPools !== 1 ? 's' : ''} • 
                  {poolsSummary.poolsOnline} online
                  {poolsSummary.poolsDegraded > 0 && ` • ${poolsSummary.poolsDegraded} degraded`}
                </div>
              </div>
            ) : (
              <EmptyState
                icon={Database}
                title="No pools"
                description="Create a storage pool to get started"
                size="sm"
              />
            )}
          </Card>
        </motion.div>

        {/* SMART Health Card */}
        <motion.div variants={itemVariants}>
          <Card
            title="Disk Health"
            isLoading={smartLoading}
            className="h-full"
          >
            {smartSummary ? (
              <div className="space-y-4">
                <div className="grid grid-cols-3 gap-2">
                  <div className="text-center">
                    <div className="text-2xl font-bold text-green-500">
                      {smartSummary.healthyDevices}
                    </div>
                    <p className="text-xs text-muted-foreground">Healthy</p>
                  </div>
                  <div className="text-center">
                    <div className="text-2xl font-bold text-yellow-500">
                      {smartSummary.warningDevices}
                    </div>
                    <p className="text-xs text-muted-foreground">Warning</p>
                  </div>
                  <div className="text-center">
                    <div className="text-2xl font-bold text-red-500">
                      {smartSummary.criticalDevices}
                    </div>
                    <p className="text-xs text-muted-foreground">Critical</p>
                  </div>
                </div>
                
                <div className="space-y-2">
                  <div className="flex justify-between text-sm">
                    <span className="text-muted-foreground">Total Devices</span>
                    <span>{smartSummary.totalDevices}</span>
                  </div>
                  {smartSummary.lastScan && (
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Last Scan</span>
                      <span>{new Date(smartSummary.lastScan).toLocaleTimeString()}</span>
                    </div>
                  )}
                </div>

                {smartSummary.criticalDevices > 0 && (
                  <Alert className="py-2">
                    <AlertCircle className="h-3 w-3" />
                    <AlertDescription className="text-xs">
                      {smartSummary.criticalDevices} disk{smartSummary.criticalDevices !== 1 ? 's' : ''} need attention
                    </AlertDescription>
                  </Alert>
                )}
              </div>
            ) : (
              <EmptyState
                icon={HardDrive}
                title="No data"
                description="Run SMART scan to check disk health"
                size="sm"
              />
            )}
          </Card>
        </motion.div>

        {/* Activity Card */}
        <motion.div variants={itemVariants}>
          <Card
            title="Recent Activity"
            isLoading={jobsLoading}
            className="h-full"
          >
            {recentJobs && recentJobs.length > 0 ? (
              <div className="space-y-2">
                {recentJobs.slice(0, 5).map((job) => (
                  <div key={job.id} className="flex items-start gap-2 text-sm">
                    <div className={cn(
                      "mt-1 h-2 w-2 rounded-full",
                      job.status === 'completed' && "bg-green-500",
                      job.status === 'running' && "bg-blue-500 animate-pulse",
                      job.status === 'failed' && "bg-red-500",
                      job.status === 'pending' && "bg-yellow-500"
                    )} />
                    <div className="flex-1 min-w-0">
                      <p className="truncate">{job.type}</p>
                      <p className="text-xs text-muted-foreground">
                        {new Date(job.startedAt).toLocaleTimeString()}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <EmptyState
                icon={Activity}
                title="No activity"
                description="Recent jobs will appear here"
                size="sm"
              />
            )}
          </Card>
        </motion.div>
      </motion.div>

      <motion.div
        variants={containerVariants}
        initial="hidden"
        animate="visible"
        className="grid gap-6 md:grid-cols-2 lg:grid-cols-3"
      >
        {/* Shares Card */}
        <motion.div variants={itemVariants}>
          <Card
            title="Network Shares"
            subtitle={`${shares?.length || 0} configured`}
            isLoading={sharesLoading}
            actions={
              <Button variant="outline" size="sm" onClick={() => window.location.href = '/shares'}>
                View All
              </Button>
            }
          >
            {shares && shares.length > 0 ? (
              <div className="space-y-3">
                {shares.slice(0, 4).map((share) => (
                  <div key={share.name} className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <FolderOpen className="h-4 w-4 text-muted-foreground" />
                      <div>
                        <p className="font-medium">{share.name}</p>
                        <p className="text-xs text-muted-foreground">
                          {share.protocol.toUpperCase()} • {share.path}
                        </p>
                      </div>
                    </div>
                    <StatusPill 
                      status={share.enabled ? 'active' : 'inactive'} 
                      size="sm" 
                    />
                  </div>
                ))}
              </div>
            ) : (
              <EmptyState
                icon={Share2}
                title="No shares"
                description="Create a network share to get started"
                size="sm"
                action={
                  <Button size="sm" onClick={() => window.location.href = '/shares'}>
                    Create Share
                  </Button>
                }
              />
            )}
          </Card>
        </motion.div>

        {/* Apps Card */}
        <motion.div variants={itemVariants}>
          <Card
            title="Installed Apps"
            subtitle={`${apps?.length || 0} installed`}
            isLoading={appsLoading}
            actions={
              <Button variant="outline" size="sm" onClick={() => window.location.href = '/apps'}>
                View All
              </Button>
            }
          >
            {apps && apps.length > 0 ? (
              <div className="space-y-3">
                {apps.slice(0, 4).map((app) => (
                  <div key={app.id} className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <Package className="h-4 w-4 text-muted-foreground" />
                      <div>
                        <p className="font-medium">{app.name}</p>
                        <p className="text-xs text-muted-foreground">v{app.version}</p>
                      </div>
                    </div>
                    <StatusPill 
                      status={app.status === 'running' ? 'running' : app.status === 'stopped' ? 'stopped' : 'error'} 
                      size="sm" 
                    />
                  </div>
                ))}
              </div>
            ) : (
              <EmptyState
                icon={Package}
                title="No apps"
                description="Install apps from the catalog"
                size="sm"
                action={
                  <Button size="sm" onClick={() => window.location.href = '/apps'}>
                    Browse Catalog
                  </Button>
                }
              />
            )}
          </Card>
        </motion.div>

        {/* Operations Card */}
        <motion.div variants={itemVariants}>
          <Card
            title="Maintenance Operations"
            isLoading={scrubLoading || balanceLoading}
          >
            <div className="space-y-3">
              {/* Scrub Status */}
              {runningScrubs.length > 0 ? (
                runningScrubs.map((scrub) => (
                  <div key={scrub.poolId} className="space-y-2">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium">Scrubbing {scrub.poolId}</span>
                      <StatusPill status="running" size="sm" />
                    </div>
                    {scrub.progress !== undefined && (
                      <div className="w-full bg-secondary rounded-full h-2">
                        <div
                          className="bg-primary h-2 rounded-full transition-all"
                          style={{ width: `${Math.min(scrub.progress, 100)}%` }}
                        />
                      </div>
                    )}
                  </div>
                ))
              ) : (
                <div className="flex items-center justify-between text-sm">
                  <span className="text-muted-foreground">Scrub</span>
                  <span>Idle</span>
                </div>
              )}

              {/* Balance Status */}
              {runningBalances.length > 0 ? (
                runningBalances.map((balance) => (
                  <div key={balance.poolId} className="space-y-2">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium">Balancing {balance.poolId}</span>
                      <StatusPill status="running" size="sm" />
                    </div>
                    {balance.progress !== undefined && (
                      <div className="w-full bg-secondary rounded-full h-2">
                        <div
                          className="bg-primary h-2 rounded-full transition-all"
                          style={{ width: `${Math.min(balance.progress, 100)}%` }}
                        />
                      </div>
                    )}
                  </div>
                ))
              ) : (
                <div className="flex items-center justify-between text-sm">
                  <span className="text-muted-foreground">Balance</span>
                  <span>Idle</span>
                </div>
              )}

              {/* Next scheduled operations */}
              {scrubStatus && scrubStatus[0]?.nextRun && (
                <div className="pt-2 border-t text-xs text-muted-foreground">
                  Next scrub: {new Date(scrubStatus[0].nextRun).toLocaleString()}
                </div>
              )}
            </div>
          </Card>
        </motion.div>
      </motion.div>
    </div>
  )
}