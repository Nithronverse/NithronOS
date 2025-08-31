import { motion } from 'framer-motion'
import { 
  Activity, AlertCircle, CheckCircle, Clock, Cpu, HardDrive, 
  MemoryStick, Network, RefreshCw, Thermometer,
  TrendingDown, TrendingUp, Wifi, XCircle, Zap
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { useSystemHealth, formatBytes, formatUptime, formatPercent, getHealthStatus } from '@/hooks/use-health'
import { useQueryClient } from '@tanstack/react-query'
import { cn } from '@/lib/utils'

// Animation variants
const containerVariants: any = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.1
    }
  }
}

const itemVariants: any = {
  hidden: { opacity: 0, y: 20 },
  visible: {
    opacity: 1,
    y: 0,
    transition: {
      type: "spring",
      stiffness: 100
    }
  }
}

// Metric card component with smooth updates
function MetricCard({ 
  title, 
  value, 
  unit, 
  icon: Icon, 
  trend, 
  color = 'text-primary',
  className 
}: {
  title: string
  value: string | number
  unit?: string
  icon: React.ElementType
  trend?: 'up' | 'down' | 'stable' | string
  color?: string
  className?: string
}) {
  const TrendIcon = trend === 'up' ? TrendingUp : trend === 'down' ? TrendingDown : null

  return (
    <Card className={cn("relative overflow-hidden", className)}>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium flex items-center justify-between">
          <span className="flex items-center gap-2">
            <Icon className="h-4 w-4" />
            {title}
          </span>
          {TrendIcon && <TrendIcon className="h-4 w-4 text-muted-foreground" />}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="flex items-baseline gap-1">
          <motion.span 
            key={value}
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            className={cn("text-2xl font-bold transition-colors", color)}
          >
            {value}
          </motion.span>
          {unit && <span className="text-sm text-muted-foreground">{unit}</span>}
        </div>
      </CardContent>
    </Card>
  )
}

// Progress metric component
function ProgressMetric({ 
  label, 
  value, 
  max, 
  unit = '%'
}: {
  label: string
  value: number
  max: number
  unit?: string
}) {
  const percentage = (value / max) * 100

  return (
    <div className="space-y-2">
      <div className="flex justify-between text-sm">
        <span className="text-muted-foreground">{label}</span>
        <motion.span 
          key={value}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="font-medium"
        >
          {formatPercent(percentage)}{unit === '%' ? '' : ` (${formatBytes(value)} / ${formatBytes(max)})`}
        </motion.span>
      </div>
      <Progress 
        value={percentage} 
        className="h-2"
        indicatorClassName={cn(
          "transition-all duration-300",
          percentage > 90 ? 'bg-red-500' : percentage > 70 ? 'bg-yellow-500' : ''
        )}
      />
    </div>
  )
}

export default function SystemHealth() {
  const queryClient = useQueryClient()
  const { data: health, isLoading, error, refetch } = useSystemHealth()
  const [isManualRefreshing, setIsManualRefreshing] = useState(false)

  // Calculate derived metrics
  const metrics = useMemo(() => {
    if (!health) return null

    const cpuTrend = health.cpu > 50 ? 'up' : health.cpu < 20 ? 'down' : 'stable'
    const memoryTrend = health.memory.usagePct > 70 ? 'up' : 'stable'
    const overallStatus = getHealthStatus(
      health.cpu,
      health.memory.usagePct,
      0 // We'll get disk usage from disk health
    )

    return {
      cpu: health.cpu,
      cpuTrend,
      memory: health.memory,
      memoryTrend,
      swap: health.swap,
      uptime: health.uptimeSec,
      load: {
        load1: health.load1,
        load5: health.load5,
        load15: health.load15
      },
      network: health.network,
      diskIO: health.diskIO,
      tempCpu: health.tempCpu,
      overallStatus
    }
  }, [health])

  const handleManualRefresh = async () => {
    setIsManualRefreshing(true)
    await queryClient.invalidateQueries({ queryKey: ['health', 'system'] })
    await refetch()
    setTimeout(() => setIsManualRefreshing(false), 1000)
  }

  if (isLoading && !health) {
    return (
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <h2 className="text-2xl font-bold">System Health</h2>
        </div>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {[...Array(8)].map((_, i) => (
            <Card key={i} className="animate-pulse">
              <CardHeader className="pb-2">
                <div className="h-4 bg-muted rounded w-24"></div>
              </CardHeader>
              <CardContent>
                <div className="h-8 bg-muted rounded w-16"></div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-6">
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            Failed to load system health data. 
            <Button 
              variant="link" 
              className="px-2"
              onClick={() => refetch()}
            >
              Retry
            </Button>
          </AlertDescription>
        </Alert>
      </div>
    )
  }

  const StatusIcon = metrics?.overallStatus === 'critical' ? XCircle :
                     metrics?.overallStatus === 'warning' ? AlertCircle : CheckCircle
  
  const statusColor = metrics?.overallStatus === 'critical' ? 'text-red-500' :
                      metrics?.overallStatus === 'warning' ? 'text-yellow-500' : 'text-green-500'

  return (
    <motion.div
      variants={containerVariants}
      initial="hidden"
      animate="visible"
      className="space-y-6"
    >
      {/* Header with status */}
      <div className="flex justify-between items-center">
        <div className="flex items-center gap-3">
          <h2 className="text-2xl font-bold">System Health</h2>
          <Badge variant="outline" className="gap-1">
            <StatusIcon className={cn("h-3 w-3", statusColor)} />
            <span className="capitalize">{metrics?.overallStatus || 'Unknown'}</span>
          </Badge>
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={handleManualRefresh}
          disabled={isManualRefreshing}
        >
          <RefreshCw className={cn("h-4 w-4 mr-2", isManualRefreshing && "animate-spin")} />
          Refresh
        </Button>
      </div>

      {/* Quick metrics grid */}
      <motion.div 
        variants={containerVariants}
        className="grid gap-4 md:grid-cols-2 lg:grid-cols-4"
      >
        <motion.div variants={itemVariants}>
          <MetricCard
            title="CPU Usage"
            value={formatPercent(metrics?.cpu || 0)}
            icon={Cpu}
            trend={metrics?.cpuTrend}
            color={metrics?.cpu && metrics.cpu > 80 ? 'text-red-500' : 'text-primary'}
          />
        </motion.div>
        
        <motion.div variants={itemVariants}>
          <MetricCard
            title="Memory"
            value={formatPercent(metrics?.memory?.usagePct || 0)}
            icon={MemoryStick}
            trend={metrics?.memoryTrend}
            color={metrics?.memory?.usagePct && metrics.memory.usagePct > 80 ? 'text-red-500' : 'text-primary'}
          />
        </motion.div>

        <motion.div variants={itemVariants}>
          <MetricCard
            title="System Load"
            value={metrics?.load?.load1?.toFixed(2) || '0.00'}
            icon={Activity}
            trend={metrics?.load?.load1 && metrics.load.load1 > metrics.load.load5 ? 'up' : 'down'}
          />
        </motion.div>

        <motion.div variants={itemVariants}>
          <MetricCard
            title="Uptime"
            value={formatUptime(metrics?.uptime || 0)}
            icon={Clock}
          />
        </motion.div>

        <motion.div variants={itemVariants}>
          <MetricCard
            title="Network RX"
            value={formatBytes(metrics?.network?.rxSpeed || 0)}
            unit="/s"
            icon={Network}
          />
        </motion.div>

        <motion.div variants={itemVariants}>
          <MetricCard
            title="Network TX"
            value={formatBytes(metrics?.network?.txSpeed || 0)}
            unit="/s"
            icon={Wifi}
          />
        </motion.div>

        <motion.div variants={itemVariants}>
          <MetricCard
            title="Disk Read"
            value={formatBytes(metrics?.diskIO?.readSpeed || 0)}
            unit="/s"
            icon={HardDrive}
          />
        </motion.div>

        <motion.div variants={itemVariants}>
          <MetricCard
            title="Disk Write"
            value={formatBytes(metrics?.diskIO?.writeSpeed || 0)}
            unit="/s"
            icon={Zap}
          />
        </motion.div>
      </motion.div>

      {/* Detailed metrics tabs */}
      <motion.div variants={itemVariants}>
        <Tabs defaultValue="resources" className="space-y-4">
          <TabsList>
            <TabsTrigger value="resources">Resources</TabsTrigger>
            <TabsTrigger value="network">Network</TabsTrigger>
            <TabsTrigger value="system">System</TabsTrigger>
          </TabsList>

          <TabsContent value="resources" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Resource Usage</CardTitle>
                <CardDescription>CPU, Memory, and Swap utilization</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <ProgressMetric
                  label="CPU"
                  value={metrics?.cpu || 0}
                  max={100}
                />
                <ProgressMetric
                  label="Memory"
                  value={metrics?.memory?.used || 0}
                  max={metrics?.memory?.total || 1}
                  unit="bytes"
                />
                {metrics?.swap?.total ? (
                  <ProgressMetric
                    label="Swap"
                    value={metrics?.swap?.used || 0}
                    max={metrics?.swap?.total || 1}
                    unit="bytes"
                  />
                ) : null}
                <div className="grid grid-cols-3 gap-4 pt-4 border-t">
                  <div>
                    <p className="text-sm text-muted-foreground">Load (1m)</p>
                    <p className="text-lg font-semibold">{metrics?.load?.load1?.toFixed(2) || '0.00'}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Load (5m)</p>
                    <p className="text-lg font-semibold">{metrics?.load?.load5?.toFixed(2) || '0.00'}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Load (15m)</p>
                    <p className="text-lg font-semibold">{metrics?.load?.load15?.toFixed(2) || '0.00'}</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="network" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Network Statistics</CardTitle>
                <CardDescription>Network interface traffic and speeds</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <p className="text-sm text-muted-foreground">Download Speed</p>
                    <p className="text-2xl font-bold">{formatBytes(metrics?.network?.rxSpeed || 0)}/s</p>
                    <p className="text-xs text-muted-foreground">
                      Total: {formatBytes(metrics?.network?.bytesRecv || 0)}
                    </p>
                  </div>
                  <div className="space-y-2">
                    <p className="text-sm text-muted-foreground">Upload Speed</p>
                    <p className="text-2xl font-bold">{formatBytes(metrics?.network?.txSpeed || 0)}/s</p>
                    <p className="text-xs text-muted-foreground">
                      Total: {formatBytes(metrics?.network?.bytesSent || 0)}
                    </p>
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-4 pt-4 border-t">
                  <div>
                    <p className="text-sm text-muted-foreground">Packets Received</p>
                    <p className="text-lg font-semibold">{metrics?.network?.packetsRecv?.toLocaleString() || '0'}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Packets Sent</p>
                    <p className="text-lg font-semibold">{metrics?.network?.packetsSent?.toLocaleString() || '0'}</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="system" className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">System Information</CardTitle>
                <CardDescription>Temperature, uptime, and system details</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm text-muted-foreground">Uptime</p>
                    <p className="text-lg font-semibold">{formatUptime(metrics?.uptime || 0)}</p>
                  </div>
                  {metrics?.tempCpu !== undefined && (
                    <div>
                      <p className="text-sm text-muted-foreground">CPU Temperature</p>
                      <p className="text-lg font-semibold flex items-center gap-1">
                        <Thermometer className="h-4 w-4" />
                        {metrics.tempCpu.toFixed(1)}°C
                      </p>
                    </div>
                  )}
                </div>
                <div className="space-y-2 pt-4 border-t">
                  <h4 className="text-sm font-medium">Disk I/O</h4>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-1">
                      <p className="text-sm text-muted-foreground">Read Speed</p>
                      <p className="text-lg font-semibold">{formatBytes(metrics?.diskIO?.readSpeed || 0)}/s</p>
                      <p className="text-xs text-muted-foreground">
                        Total: {formatBytes(metrics?.diskIO?.readBytes || 0)}
                      </p>
                    </div>
                    <div className="space-y-1">
                      <p className="text-sm text-muted-foreground">Write Speed</p>
                      <p className="text-lg font-semibold">{formatBytes(metrics?.diskIO?.writeSpeed || 0)}/s</p>
                      <p className="text-xs text-muted-foreground">
                        Total: {formatBytes(metrics?.diskIO?.writeBytes || 0)}
                      </p>
                    </div>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </motion.div>
    </motion.div>
  )
}