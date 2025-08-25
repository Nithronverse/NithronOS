import { useState } from 'react'
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

} from 'lucide-react'
import { PageHeader } from '@/components/ui/page-header'
import { Card } from '@/components/ui/card-enhanced'
import { EmptyState } from '@/components/ui/empty-state'
import { StatusPill, HealthBadge, Metric } from '@/components/ui/status'
import { Button } from '@/components/ui/button'
import { useHealth, useDisks, useVolumes, useShares, useApps } from '@/hooks/use-api'
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

// Mock data for development - remove when API is ready
const mockHealth = {
  status: 'healthy' as const,
  cpu: 45,
  memory: 62,
  uptime: 1234567,
  alerts: []
}

const mockDisks = [
  { name: 'sda', model: 'Samsung SSD', size: 500000000000, used: 250000000000, health: 'healthy' as const, temperature: 35 },
  { name: 'sdb', model: 'WD Red', size: 2000000000000, used: 1500000000000, health: 'healthy' as const, temperature: 38 },
]

const mockVolumes = [
  { id: '1', name: 'main-pool', type: 'zfs' as const, size: 2000000000000, used: 1200000000000, status: 'online' as const, mountpoint: '/mnt/main' },
]

const mockShares = [
  { id: '1', name: 'Documents', protocol: 'smb' as const, path: '/mnt/main/docs', access: 'private' as const, status: 'active' as const },
  { id: '2', name: 'Media', protocol: 'smb' as const, path: '/mnt/main/media', access: 'public' as const, status: 'active' as const },
]

const mockApps = [
  { id: '1', name: 'Plex', version: '1.32.5', status: 'running' as const, autoUpdate: true },
  { id: '2', name: 'Nextcloud', version: '27.1.0', status: 'running' as const, autoUpdate: false },
]

const mockActivity = [
  { id: '1', type: 'share', action: 'created', target: 'Documents', time: '2 hours ago', icon: Share2 },
  { id: '2', type: 'app', action: 'updated', target: 'Plex', time: '5 hours ago', icon: Package },
  { id: '3', type: 'backup', action: 'completed', target: 'Daily Backup', time: '1 day ago', icon: Download },
]

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
  
  // Use real hooks with fallback to mock data
  const { data: healthData, isLoading: healthLoading, refetch: refetchHealth } = useHealth()
  const { data: disksData, isLoading: disksLoading, refetch: refetchDisks } = useDisks()
  const { data: volumesData, isLoading: volumesLoading, refetch: refetchVolumes } = useVolumes()
  const { data: sharesData, isLoading: sharesLoading, refetch: refetchShares } = useShares()
  const { data: appsData, isLoading: appsLoading, refetch: refetchApps } = useApps()

  // Use mock data if API fails
  const health = healthData || mockHealth
  const disks = disksData || mockDisks
  const volumes = volumesData || mockVolumes
  const shares = sharesData || mockShares
  const apps = appsData || mockApps
  const activity = mockActivity // TODO: Replace with real activity API

  const handleRefresh = async () => {
    setIsRefreshing(true)
    await Promise.all([
      refetchHealth(),
      refetchDisks(),
      refetchVolumes(),
      refetchShares(),
      refetchApps(),
    ])
    setTimeout(() => setIsRefreshing(false), 500)
  }

  // Calculate storage usage
  const totalStorage = disks.reduce((acc, disk) => acc + disk.size, 0)
  const usedStorage = disks.reduce((acc, disk) => acc + (disk.used || 0), 0)
  const storagePercentage = totalStorage > 0 ? (usedStorage / totalStorage) * 100 : 0

  const storageData = [
    { name: 'Used', value: usedStorage, fill: 'hsl(var(--primary))' },
    { name: 'Free', value: totalStorage - usedStorage, fill: 'hsl(var(--muted))' },
  ]

  const diskUsageData = disks.slice(0, 5).map(disk => ({
    name: disk.name,
    usage: disk.used ? (disk.used / disk.size) * 100 : 0,
  }))

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
              Rescan
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
            isLoading={healthLoading}
            className="h-full"
          >
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Overall Status</p>
                  <div className="mt-1">
                    <HealthBadge status={health.status} />
                  </div>
                  <p className="mt-1 text-xs text-muted-foreground">
                    Uptime: {formatUptime(health.uptime)}
                  </p>
                </div>
                <Activity className="h-8 w-8 text-muted-foreground" />
              </div>
              
              <div className="space-y-2">
                <div className="flex items-center justify-between text-sm">
                  <span className="flex items-center gap-2">
                    <Cpu className="h-4 w-4 text-muted-foreground" />
                    CPU Load
                  </span>
                  <span className="font-medium">{health.cpu}%</span>
                </div>
                <div className="h-2 bg-muted rounded-full overflow-hidden">
                  <div 
                    className={cn(
                      "h-full transition-all",
                      health.cpu < 70 ? "bg-green-500" : 
                      health.cpu < 90 ? "bg-yellow-500" : "bg-red-500"
                    )}
                    style={{ width: `${health.cpu}%` }}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <div className="flex items-center justify-between text-sm">
                  <span className="flex items-center gap-2">
                    <MemoryStick className="h-4 w-4 text-muted-foreground" />
                    Memory
                  </span>
                  <span className="font-medium">{health.memory}%</span>
                </div>
                <div className="h-2 bg-muted rounded-full overflow-hidden">
                  <div 
                    className={cn(
                      "h-full transition-all",
                      health.memory < 70 ? "bg-green-500" : 
                      health.memory < 90 ? "bg-yellow-500" : "bg-red-500"
                    )}
                    style={{ width: `${health.memory}%` }}
                  />
                </div>
              </div>

              {health.alerts.length > 0 && (
                <div className="pt-2 border-t">
                  <StatusPill variant="warning">
                    {health.alerts.length} Active Alert{health.alerts.length !== 1 ? 's' : ''}
                  </StatusPill>
                </div>
              )}
            </div>
          </Card>
        </motion.div>

        {/* Storage Card */}
        <motion.div variants={itemVariants}>
          <Card
            title="Storage"
            isLoading={disksLoading || volumesLoading}
            className="h-full"
            actions={
              <Button variant="ghost" size="sm" onClick={() => window.location.href = '/storage'}>
                View All
              </Button>
            }
          >
            <div className="space-y-4">
              <div className="h-32">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={storageData}
                      cx="50%"
                      cy="50%"
                      innerRadius={40}
                      outerRadius={60}
                      paddingAngle={2}
                      dataKey="value"
                    >
                      {storageData.map((entry, index) => (
                        <Cell key={`cell-${index}`} fill={entry.fill} />
                      ))}
                    </Pie>
                    <Tooltip formatter={(value: number) => formatBytes(value)} />
                  </PieChart>
                </ResponsiveContainer>
              </div>
              
              <div className="text-center">
                <p className="text-2xl font-bold">{formatBytes(usedStorage)}</p>
                <p className="text-sm text-muted-foreground">
                  of {formatBytes(totalStorage)} used ({storagePercentage.toFixed(1)}%)
                </p>
              </div>

              <div className="space-y-1">
                <div className="flex items-center justify-between text-xs">
                  <span className="flex items-center gap-1">
                    <HardDrive className="h-3 w-3" />
                    {disks.length} Disk{disks.length !== 1 ? 's' : ''}
                  </span>
                  <span>{volumes.length} Volume{volumes.length !== 1 ? 's' : ''}</span>
                </div>
              </div>
            </div>
          </Card>
        </motion.div>

        {/* Shares Card */}
        <motion.div variants={itemVariants}>
          <Card
            title="Shares"
            isLoading={sharesLoading}
            className="h-full"
            actions={
              <Button variant="ghost" size="sm" onClick={() => window.location.href = '/shares'}>
                Manage
              </Button>
            }
          >
            <div className="space-y-4">
              <Metric
                label="Active Shares"
                value={shares.filter(s => s.status === 'active').length}
                sublabel={`${shares.length} total configured`}
              />
              
              {shares.length > 0 ? (
                <div className="space-y-2">
                  {shares.slice(0, 3).map(share => (
                    <div key={share.id} className="flex items-center justify-between p-2 rounded-lg bg-muted/30">
                      <div className="flex items-center gap-2">
                        <Share2 className="h-4 w-4 text-muted-foreground" />
                        <span className="text-sm font-medium">{share.name}</span>
                      </div>
                      <StatusPill variant={share.status === 'active' ? 'success' : 'muted'}>
                        {share.protocol.toUpperCase()}
                      </StatusPill>
                    </div>
                  ))}
                </div>
              ) : (
                <EmptyState
                  variant="no-data"
                  title="No shares"
                  description="Create your first share"
                  action={{
                    label: "Create Share",
                    onClick: () => window.location.href = '/shares'
                  }}
                  className="py-4"
                />
              )}
            </div>
          </Card>
        </motion.div>

        {/* Apps Card */}
        <motion.div variants={itemVariants}>
          <Card
            title="Applications"
            isLoading={appsLoading}
            className="h-full"
            actions={
              <Button variant="ghost" size="sm" onClick={() => window.location.href = '/apps'}>
                View All
              </Button>
            }
          >
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <Metric
                  label="Installed"
                  value={apps.length}
                />
                <Metric
                  label="Running"
                  value={apps.filter(a => a.status === 'running').length}
                />
              </div>
              
              {apps.length > 0 ? (
                <div className="space-y-2">
                  {apps.slice(0, 3).map(app => (
                    <div key={app.id} className="flex items-center justify-between p-2 rounded-lg bg-muted/30">
                      <div className="flex items-center gap-2">
                        <Package className="h-4 w-4 text-muted-foreground" />
                        <span className="text-sm font-medium">{app.name}</span>
                      </div>
                      <StatusPill 
                        variant={
                          app.status === 'running' ? 'success' : 
                          app.status === 'stopped' ? 'muted' : 'error'
                        }
                      >
                        {app.status}
                      </StatusPill>
                    </div>
                  ))}
                </div>
              ) : (
                <EmptyState
                  variant="no-data"
                  title="No apps"
                  description="Install your first app"
                  action={{
                    label: "Browse Marketplace",
                    onClick: () => window.location.href = '/apps'
                  }}
                  className="py-4"
                />
              )}
            </div>
          </Card>
        </motion.div>
      </motion.div>

      {/* Disk Usage Chart */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.2 }}
        className="grid gap-4 lg:grid-cols-2"
      >
        <Card
          title="Disk Usage"
          description="Storage utilization by disk"
          isLoading={disksLoading}
        >
          {diskUsageData.length > 0 ? (
            <div className="h-64">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={diskUsageData}>
                  <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
                  <XAxis dataKey="name" className="text-xs" />
                  <YAxis className="text-xs" />
                  <Tooltip formatter={(value: number) => `${value.toFixed(1)}%`} />
                  <Bar dataKey="usage" fill="hsl(var(--primary))" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          ) : (
            <EmptyState
              variant="no-data"
              title="No disk data"
              description="Disk information will appear here"
            />
          )}
        </Card>

        {/* Recent Activity */}
        <Card
          title="Recent Activity"
          description="Latest system events"
        >
          {activity.length > 0 ? (
            <div className="space-y-2">
              {activity.map((item) => {
                const Icon = item.icon
                return (
                  <div key={item.id} className="flex items-center gap-3 p-2 rounded-lg hover:bg-muted/30 transition-colors">
                    <div className="p-2 rounded-full bg-muted">
                      <Icon className="h-4 w-4 text-muted-foreground" />
                    </div>
                    <div className="flex-1">
                      <p className="text-sm">
                        <span className="font-medium">{item.target}</span>
                        <span className="text-muted-foreground"> was {item.action}</span>
                      </p>
                      <p className="text-xs text-muted-foreground flex items-center gap-1">
                        <Clock className="h-3 w-3" />
                        {item.time}
                      </p>
                    </div>
                  </div>
                )
              })}
            </div>
          ) : (
            <EmptyState
              variant="no-data"
              title="No recent activity"
              description="System events will appear here"
            />
          )}
        </Card>
      </motion.div>
    </div>
  )
}