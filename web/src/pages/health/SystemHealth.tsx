import { useState, useEffect } from 'react'
import { motion, Variants } from 'framer-motion'
import {
  Thermometer,
  RefreshCw,
  AlertCircle,
  TrendingUp,
  TrendingDown,
} from 'lucide-react'
import { Card } from '@/components/ui/card-enhanced'
import { StatusPill, HealthBadge } from '@/components/ui/status'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { useSystemInfo, useSystemMetrics } from '@/hooks/use-api'
import { cn } from '@/lib/utils'
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
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
  
  if (days > 0) return `${days}d ${hours}h ${minutes}m`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

function getHealthColor(value: number, thresholds: { warning: number; critical: number }): string {
  if (value >= thresholds.critical) return 'text-red-500'
  if (value >= thresholds.warning) return 'text-yellow-500'
  return 'text-green-500'
}

// Animation variants
const containerVariants: Variants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: { staggerChildren: 0.05 }
  }
}

const itemVariants: Variants = {
  hidden: { opacity: 0, y: 20 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { type: "spring", stiffness: 100 }
  }
}

export function SystemHealth() {
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [cpuHistory, setCpuHistory] = useState<any[]>([])
  const [memoryHistory, setMemoryHistory] = useState<any[]>([])
  const [networkHistory, setNetworkHistory] = useState<any[]>([])
  
  // Fetch real data (with proper typing)
  const { data: systemInfo, isLoading, refetch } = useSystemInfo()
  const { data: metricsData, refetch: refetchMetrics } = useSystemMetrics()
  const metrics = metricsData as any // Type assertion for mock data
  
  // Auto-refresh every 5 seconds
  useEffect(() => {
    if (!autoRefresh) return
    
    const interval = setInterval(() => {
      refetch()
      refetchMetrics()
    }, 5000)
    
    return () => clearInterval(interval)
  }, [autoRefresh, refetch, refetchMetrics])
  
  // Update history data
  useEffect(() => {
    if (!metrics) return
    
    const timestamp = new Date().toLocaleTimeString('en-US', { 
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
    
    // CPU history
    setCpuHistory(prev => {
      const newData = [...prev, { time: timestamp, value: metrics.cpuUsage || 0 }]
      return newData.slice(-20) // Keep last 20 data points
    })
    
    // Memory history
    setMemoryHistory(prev => {
      const memoryPercent = systemInfo?.memoryTotal 
        ? ((systemInfo.memoryUsed || 0) / systemInfo.memoryTotal) * 100
        : 0
      const newData = [...prev, { time: timestamp, value: memoryPercent }]
      return newData.slice(-20)
    })
    
    // Network history
    setNetworkHistory(prev => {
      const newData = [...prev, {
        time: timestamp,
        rx: metrics.networkRx || 0,
        tx: metrics.networkTx || 0,
      }]
      return newData.slice(-20)
    })
  }, [metrics, systemInfo])
  
  const handleRefresh = async () => {
    setIsRefreshing(true)
    await Promise.all([refetch(), refetchMetrics()])
    setTimeout(() => setIsRefreshing(false), 500)
  }
  
  // Calculate overall health
  const overallHealth = (() => {
    if (!systemInfo) return 'unknown'
    const cpuUsage = metrics?.cpuUsage || 0
    const memoryPercent = systemInfo.memoryTotal 
      ? ((systemInfo.memoryUsed || 0) / systemInfo.memoryTotal) * 100
      : 0
    
    if (cpuUsage > 90 || memoryPercent > 90) return 'critical'
    if (cpuUsage > 75 || memoryPercent > 75) return 'degraded'
    return 'healthy'
  })()
  
  // Memory chart data
  const memoryData = systemInfo?.memoryTotal ? [
    { name: 'Used', value: systemInfo.memoryUsed || 0, fill: 'hsl(var(--primary))' },
    { name: 'Free', value: systemInfo.memoryTotal - (systemInfo.memoryUsed || 0), fill: 'hsl(var(--muted))' },
  ] : []
  
  // Extended system info interface
  const sysInfoExt = systemInfo as any
  
  return (
    <motion.div
      variants={containerVariants}
      initial="hidden"
      animate="visible"
      className="space-y-6"
    >
      {/* Controls */}
      <div className="flex justify-between items-center">
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={handleRefresh}
            disabled={isRefreshing}
          >
            <RefreshCw className={cn("h-4 w-4 mr-2", isRefreshing && "animate-spin")} />
            Refresh
          </Button>
          <Button
            variant={autoRefresh ? "default" : "outline"}
            size="sm"
            onClick={() => setAutoRefresh(!autoRefresh)}
          >
            Auto-refresh: {autoRefresh ? 'On' : 'Off'}
          </Button>
        </div>
        <StatusPill status={autoRefresh ? 'running' : 'stopped'} />
      </div>
      
      {/* Overall Status Card */}
      <motion.div variants={itemVariants}>
        <Card
          title="Overall System Health"
          isLoading={isLoading}
        >
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">Status</p>
              <HealthBadge status={overallHealth} />
            </div>
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">Hostname</p>
              <p className="font-medium">{systemInfo?.hostname || 'N/A'}</p>
            </div>
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">Uptime</p>
              <p className="font-medium">{formatUptime(systemInfo?.uptime || 0)}</p>
            </div>
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">Load Average</p>
              <p className="font-medium">
                {sysInfoExt?.loadAverage?.map((l: number) => l.toFixed(2)).join(', ') || 'N/A'}
              </p>
            </div>
          </div>
        </Card>
      </motion.div>
      
      <div className="grid gap-6 md:grid-cols-2">
        {/* CPU Usage */}
        <motion.div variants={itemVariants}>
          <Card
            title="CPU Usage"
            isLoading={isLoading}
          >
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <div className="text-3xl font-bold">
                    <span className={getHealthColor(metrics?.cpuUsage || 0, { warning: 75, critical: 90 })}>
                      {(metrics?.cpuUsage || 0).toFixed(1)}%
                    </span>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    {systemInfo?.cpuCount || 0} cores • {sysInfoExt?.cpuModel || 'Unknown'}
                  </p>
                </div>
                <div className="text-right">
                  <div className="flex items-center gap-1">
                    <Thermometer className="h-3 w-3" />
                    <span className="text-sm">{metrics?.cpuTemp || 0}°C</span>
                  </div>
                </div>
              </div>
              
              {cpuHistory.length > 0 && (
                <div className="h-[150px]">
                  <ResponsiveContainer width="100%" height="100%">
                    <AreaChart data={cpuHistory}>
                      <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
                      <XAxis 
                        dataKey="time" 
                        stroke="hsl(var(--muted-foreground))"
                        fontSize={10}
                        tickFormatter={(value) => value.split(':').slice(1).join(':')}
                      />
                      <YAxis 
                        stroke="hsl(var(--muted-foreground))"
                        fontSize={10}
                        domain={[0, 100]}
                      />
                      <Tooltip
                        contentStyle={{
                          backgroundColor: 'hsl(var(--card))',
                          border: '1px solid hsl(var(--border))',
                          borderRadius: '6px',
                        }}
                      />
                      <Area
                        type="monotone"
                        dataKey="value"
                        stroke="hsl(var(--primary))"
                        fill="hsl(var(--primary))"
                        fillOpacity={0.3}
                      />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>
              )}
              
              {/* Per-core usage */}
              {metrics?.cpuCores && (
                <div className="grid grid-cols-4 gap-2">
                  {metrics.cpuCores.map((core: number, idx: number) => (
                    <div key={idx} className="text-center">
                      <div className="text-xs text-muted-foreground mb-1">Core {idx}</div>
                      <div className="h-1 bg-secondary rounded-full overflow-hidden">
                        <div
                          className="h-full bg-primary transition-all"
                          style={{ width: `${core}%` }}
                        />
                      </div>
                      <div className="text-xs mt-1">{core.toFixed(0)}%</div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </Card>
        </motion.div>
        
        {/* Memory Usage */}
        <motion.div variants={itemVariants}>
          <Card
            title="Memory Usage"
            isLoading={isLoading}
          >
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <div className="text-3xl font-bold">
                    <span className={getHealthColor(
                      systemInfo?.memoryTotal ? ((systemInfo.memoryUsed || 0) / systemInfo.memoryTotal) * 100 : 0,
                      { warning: 75, critical: 90 }
                    )}>
                      {systemInfo?.memoryTotal 
                        ? `${(((systemInfo.memoryUsed || 0) / systemInfo.memoryTotal) * 100).toFixed(1)}%`
                        : 'N/A'}
                    </span>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    {formatBytes(systemInfo?.memoryUsed || 0)} / {formatBytes(systemInfo?.memoryTotal || 0)}
                  </p>
                </div>
                <div className="h-[80px] w-[80px]">
                  <ResponsiveContainer width="100%" height="100%">
                    <PieChart>
                      <Pie
                        data={memoryData}
                        cx="50%"
                        cy="50%"
                        innerRadius={25}
                        outerRadius={35}
                        dataKey="value"
                        strokeWidth={0}
                      >
                        {memoryData.map((entry, index) => (
                          <Cell key={`cell-${index}`} fill={entry.fill} />
                        ))}
                      </Pie>
                    </PieChart>
                  </ResponsiveContainer>
                </div>
              </div>
              
              {memoryHistory.length > 0 && (
                <div className="h-[150px]">
                  <ResponsiveContainer width="100%" height="100%">
                    <AreaChart data={memoryHistory}>
                      <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
                      <XAxis 
                        dataKey="time" 
                        stroke="hsl(var(--muted-foreground))"
                        fontSize={10}
                        tickFormatter={(value) => value.split(':').slice(1).join(':')}
                      />
                      <YAxis 
                        stroke="hsl(var(--muted-foreground))"
                        fontSize={10}
                        domain={[0, 100]}
                      />
                      <Tooltip
                        contentStyle={{
                          backgroundColor: 'hsl(var(--card))',
                          border: '1px solid hsl(var(--border))',
                          borderRadius: '6px',
                        }}
                      />
                      <Area
                        type="monotone"
                        dataKey="value"
                        stroke="hsl(var(--chart-2))"
                        fill="hsl(var(--chart-2))"
                        fillOpacity={0.3}
                      />
                    </AreaChart>
                  </ResponsiveContainer>
                </div>
              )}
              
              {/* Memory breakdown */}
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="text-muted-foreground">Cached:</span>
                  <span className="ml-2">{formatBytes(sysInfoExt?.memoryCached || 0)}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">Buffers:</span>
                  <span className="ml-2">{formatBytes(sysInfoExt?.memoryBuffers || 0)}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">Swap Used:</span>
                  <span className="ml-2">{formatBytes(sysInfoExt?.swapUsed || 0)}</span>
                </div>
                <div>
                  <span className="text-muted-foreground">Swap Total:</span>
                  <span className="ml-2">{formatBytes(sysInfoExt?.swapTotal || 0)}</span>
                </div>
              </div>
            </div>
          </Card>
        </motion.div>
        
        {/* Network Activity */}
        <motion.div variants={itemVariants}>
          <Card
            title="Network Activity"
            isLoading={isLoading}
          >
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <div className="flex items-center gap-2">
                    <TrendingUp className="h-4 w-4 text-green-500" />
                    <span className="text-sm text-muted-foreground">Upload</span>
                  </div>
                  <div className="text-2xl font-bold">
                    {formatBytes(metrics?.networkTx || 0)}/s
                  </div>
                </div>
                <div className="space-y-1">
                  <div className="flex items-center gap-2">
                    <TrendingDown className="h-4 w-4 text-blue-500" />
                    <span className="text-sm text-muted-foreground">Download</span>
                  </div>
                  <div className="text-2xl font-bold">
                    {formatBytes(metrics?.networkRx || 0)}/s
                  </div>
                </div>
              </div>
              
              {networkHistory.length > 0 && (
                <div className="h-[150px]">
                  <ResponsiveContainer width="100%" height="100%">
                    <LineChart data={networkHistory}>
                      <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" />
                      <XAxis 
                        dataKey="time" 
                        stroke="hsl(var(--muted-foreground))"
                        fontSize={10}
                        tickFormatter={(value) => value.split(':').slice(1).join(':')}
                      />
                      <YAxis 
                        stroke="hsl(var(--muted-foreground))"
                        fontSize={10}
                        tickFormatter={(value) => formatBytes(value)}
                      />
                      <Tooltip
                        contentStyle={{
                          backgroundColor: 'hsl(var(--card))',
                          border: '1px solid hsl(var(--border))',
                          borderRadius: '6px',
                        }}
                        formatter={(value: number) => formatBytes(value)}
                      />
                      <Line
                        type="monotone"
                        dataKey="tx"
                        stroke="hsl(var(--chart-3))"
                        strokeWidth={2}
                        dot={false}
                        name="Upload"
                      />
                      <Line
                        type="monotone"
                        dataKey="rx"
                        stroke="hsl(var(--chart-4))"
                        strokeWidth={2}
                        dot={false}
                        name="Download"
                      />
                    </LineChart>
                  </ResponsiveContainer>
                </div>
              )}
              
              {/* Network interfaces */}
              {metrics?.networkInterfaces && (
                <div className="space-y-2">
                  <p className="text-sm font-medium">Active Interfaces</p>
                  {metrics.networkInterfaces.map((iface: any) => (
                    <div key={iface.name} className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground">{iface.name}</span>
                      <span>{iface.address}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </Card>
        </motion.div>
        
        {/* Disk I/O */}
        <motion.div variants={itemVariants}>
          <Card
            title="Disk I/O"
            isLoading={isLoading}
          >
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <span className="text-sm text-muted-foreground">Read</span>
                  <div className="text-2xl font-bold">
                    {formatBytes(metrics?.diskRead || 0)}/s
                  </div>
                </div>
                <div className="space-y-1">
                  <span className="text-sm text-muted-foreground">Write</span>
                  <div className="text-2xl font-bold">
                    {formatBytes(metrics?.diskWrite || 0)}/s
                  </div>
                </div>
              </div>
              
              <div className="space-y-2">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">IOPS Read</span>
                  <span>{metrics?.iopsRead || 0}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">IOPS Write</span>
                  <span>{metrics?.iopsWrite || 0}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Queue Depth</span>
                  <span>{metrics?.diskQueueDepth || 0}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Avg Latency</span>
                  <span>{metrics?.diskLatency || 0} ms</span>
                </div>
              </div>
            </div>
          </Card>
        </motion.div>
      </div>
      
      {/* System Information */}
      <motion.div variants={itemVariants}>
        <Card
          title="System Information"
          isLoading={isLoading}
        >
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">Operating System</p>
              <p className="font-medium">{sysInfoExt?.os || 'N/A'}</p>
            </div>
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">Kernel Version</p>
              <p className="font-medium">{systemInfo?.kernel || 'N/A'}</p>
            </div>
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">Architecture</p>
              <p className="font-medium">{systemInfo?.arch || 'N/A'}</p>
            </div>
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">CPU Model</p>
              <p className="font-medium">{sysInfoExt?.cpuModel || 'N/A'}</p>
            </div>
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">Total Memory</p>
              <p className="font-medium">{formatBytes(systemInfo?.memoryTotal || 0)}</p>
            </div>
            <div className="space-y-2">
              <p className="text-sm text-muted-foreground">Boot Time</p>
              <p className="font-medium">
                {sysInfoExt?.bootTime 
                  ? new Date(sysInfoExt.bootTime).toLocaleString()
                  : 'N/A'}
              </p>
            </div>
          </div>
        </Card>
      </motion.div>
      
      {/* Alerts */}
      {overallHealth === 'critical' && (
        <motion.div variants={itemVariants}>
          <Alert className="border-destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              System resources are critically high. Consider investigating running processes or scaling resources.
            </AlertDescription>
          </Alert>
        </motion.div>
      )}
    </motion.div>
  )
}