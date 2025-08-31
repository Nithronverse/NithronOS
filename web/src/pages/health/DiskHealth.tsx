import { useState, useEffect } from 'react'
import { motion, Variants } from 'framer-motion'
import {
  HardDrive,
  AlertTriangle,
  CheckCircle,
  XCircle,
  RefreshCw,
  Activity,
  Thermometer,
  Play,
  AlertCircle,
} from 'lucide-react'
import { Card } from '@/components/ui/card-enhanced'
import { StatusPill, HealthBadge } from '@/components/ui/status'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Progress } from '@/components/ui/progress'
import { 
  useSmartDevices,
  useSmartSummary,
  useRunSmartTest,
} from '@/hooks/use-api'
import { cn } from '@/lib/utils'
import { formatDistanceToNow } from 'date-fns'
import {
  PieChart,
  Pie,
  Cell,
  ResponsiveContainer,
  Tooltip,
} from 'recharts'

// Helper functions
function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

function getHealthStatus(smartStatus: string): 'healthy' | 'degraded' | 'critical' {
  if (smartStatus === 'PASSED' || smartStatus === 'OK') return 'healthy'
  if (smartStatus === 'WARNING') return 'degraded'
  return 'critical'
}

function getTemperatureColor(temp: number): string {
  if (temp >= 60) return 'text-red-500'
  if (temp >= 50) return 'text-yellow-500'
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

interface SmartDevice {
  device: string
  model: string
  serial: string
  capacity: number
  temperature: number
  powerOnHours: number
  smartStatus: string
  attributes: Array<{
    id: number
    name: string
    value: number
    worst: number
    threshold: number
    raw: string
    status: string
  }>
  testHistory: Array<{
    type: string
    status: string
    completedAt: string
    duration: number
  }>
}

export function DiskHealth() {
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [selectedDevice, setSelectedDevice] = useState<string | null>(null)
  const [lastScanTime, setLastScanTime] = useState<Date>(new Date())
  const [runningTests, setRunningTests] = useState<Set<string>>(new Set())
  
  // Fetch real data (with type assertions for mock data)
  const { data: devicesData, isLoading, refetch: refetchDevices } = useSmartDevices()
  const { data: summary, refetch: refetchSummary } = useSmartSummary()
  const runSmartTest = useRunSmartTest()
  
  const devices = devicesData as SmartDevice[] | undefined
  
  // Auto-refresh every 30 seconds
  useEffect(() => {
    if (!autoRefresh) return
    
    const interval = setInterval(() => {
      refetchDevices()
      refetchSummary()
      setLastScanTime(new Date())
    }, 30000)
    
    return () => clearInterval(interval)
  }, [autoRefresh, refetchDevices, refetchSummary])
  
  // Live updating time since last scan
  const [, setTick] = useState(0)
  useEffect(() => {
    const timer = setInterval(() => setTick(t => t + 1), 1000)
    return () => clearInterval(timer)
  }, [])
  
  const handleRefresh = async () => {
    setIsRefreshing(true)
    setLastScanTime(new Date())
    await Promise.all([refetchDevices(), refetchSummary()])
    setTimeout(() => setIsRefreshing(false), 500)
  }
  
  const handleRunTest = async (device: string, testType: 'short' | 'long' | 'conveyance') => {
    setRunningTests(prev => new Set(prev).add(device))
    try {
      await runSmartTest.mutateAsync({ device, type: testType })
      // Refresh after starting test
      setTimeout(() => {
        refetchDevices()
        setRunningTests(prev => {
          const next = new Set(prev)
          next.delete(device)
          return next
        })
      }, 2000)
    } catch (error) {
      console.error('Failed to start SMART test:', error)
      setRunningTests(prev => {
        const next = new Set(prev)
        next.delete(device)
        return next
      })
    }
  }
  
  // Calculate health distribution
  const healthDistribution = summary ? [
    { name: 'Healthy', value: summary.healthyDevices, fill: 'hsl(var(--chart-3))' },
    { name: 'Warning', value: summary.warningDevices, fill: 'hsl(var(--chart-2))' },
    { name: 'Critical', value: summary.criticalDevices, fill: 'hsl(var(--destructive))' },
  ].filter(item => item.value > 0) : []
  
  const selectedDeviceData = devices?.find((d: SmartDevice) => d.device === selectedDevice)
  
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
        <div className="flex items-center gap-4">
          <span className="text-sm text-muted-foreground">
            Last scan: {formatDistanceToNow(lastScanTime, { addSuffix: true })}
          </span>
          <StatusPill status={autoRefresh ? 'running' : 'stopped'} />
        </div>
      </div>
      
      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-3">
        <motion.div variants={itemVariants}>
          <Card
            title="Disk Health Summary"
            isLoading={isLoading}
          >
            {summary ? (
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-muted-foreground">Overall Status</p>
                    <div className="mt-1">
                      <HealthBadge 
                        status={
                          summary.criticalDevices > 0 ? 'critical' :
                          summary.warningDevices > 0 ? 'degraded' : 'healthy'
                        }
                      />
                    </div>
                  </div>
                  {healthDistribution.length > 0 && (
                    <div className="h-[80px] w-[80px]">
                      <ResponsiveContainer width="100%" height="100%">
                        <PieChart>
                          <Pie
                            data={healthDistribution}
                            cx="50%"
                            cy="50%"
                            innerRadius={25}
                            outerRadius={35}
                            dataKey="value"
                            strokeWidth={0}
                          >
                            {healthDistribution.map((entry, index) => (
                              <Cell key={`cell-${index}`} fill={entry.fill} />
                            ))}
                          </Pie>
                          <Tooltip />
                        </PieChart>
                      </ResponsiveContainer>
                    </div>
                  )}
                </div>
                
                <div className="grid grid-cols-3 gap-2 text-center">
                  <div>
                    <div className="text-2xl font-bold text-green-500">
                      {summary.healthyDevices}
                    </div>
                    <p className="text-xs text-muted-foreground">Healthy</p>
                  </div>
                  <div>
                    <div className="text-2xl font-bold text-yellow-500">
                      {summary.warningDevices}
                    </div>
                    <p className="text-xs text-muted-foreground">Warning</p>
                  </div>
                  <div>
                    <div className="text-2xl font-bold text-red-500">
                      {summary.criticalDevices}
                    </div>
                    <p className="text-xs text-muted-foreground">Critical</p>
                  </div>
                </div>
                
                <div className="pt-2 border-t text-xs text-muted-foreground">
                  Total: {summary.totalDevices} device{summary.totalDevices !== 1 ? 's' : ''}
                </div>
              </div>
            ) : (
              <div className="text-center py-8">
                <HardDrive className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
                <p className="text-sm text-muted-foreground">No disk data available</p>
              </div>
            )}
          </Card>
        </motion.div>
        
        <motion.div variants={itemVariants}>
          <Card
            title="Temperature Overview"
            isLoading={isLoading}
          >
            {devices && devices.length > 0 ? (
              <div className="space-y-3">
                {devices.slice(0, 4).map((device: SmartDevice) => (
                  <div key={device.device} className="space-y-1">
                    <div className="flex items-center justify-between">
                      <span className="text-sm truncate">{device.device}</span>
                      <span className={cn(
                        "text-sm font-medium",
                        getTemperatureColor(device.temperature)
                      )}>
                        {device.temperature}°C
                      </span>
                    </div>
                    <Progress 
                      value={(device.temperature / 70) * 100} 
                      className="h-1"
                    />
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center py-8">
                <Thermometer className="h-8 w-8 mx-auto text-muted-foreground mb-2" />
                <p className="text-xs text-muted-foreground">No temperature data</p>
              </div>
            )}
          </Card>
        </motion.div>
        
        <motion.div variants={itemVariants}>
          <Card
            title="Recent Tests"
            isLoading={isLoading}
          >
            {devices && devices.some((d: SmartDevice) => d.testHistory?.length > 0) ? (
              <div className="space-y-2">
                {devices
                  .flatMap((d: SmartDevice) => 
                    (d.testHistory || []).map((test: any) => ({ ...test, device: d.device }))
                  )
                  .sort((a: any, b: any) => 
                    new Date(b.completedAt).getTime() - new Date(a.completedAt).getTime()
                  )
                  .slice(0, 3)
                  .map((test: any, idx: number) => (
                    <div key={idx} className="flex items-start gap-2 text-sm">
                      <div className={cn(
                        "mt-1 h-2 w-2 rounded-full",
                        test.status === 'completed' && "bg-green-500",
                        test.status === 'running' && "bg-blue-500 animate-pulse",
                        test.status === 'failed' && "bg-red-500"
                      )} />
                      <div className="flex-1">
                        <p className="truncate">{test.device}: {test.type}</p>
                        <p className="text-xs text-muted-foreground">
                          {formatDistanceToNow(new Date(test.completedAt), { addSuffix: true })}
                        </p>
                      </div>
                    </div>
                  ))}
              </div>
            ) : (
              <div className="text-center py-8">
                <Activity className="h-8 w-8 mx-auto text-muted-foreground mb-2" />
                <p className="text-xs text-muted-foreground">No recent tests</p>
              </div>
            )}
          </Card>
        </motion.div>
      </div>
      
      {/* Device List */}
      <motion.div variants={itemVariants}>
        <Card
          title="Storage Devices"
          description="Click on a device to view detailed SMART attributes"
          isLoading={isLoading}
        >
          {devices && devices.length > 0 ? (
            <div className="space-y-2">
              {devices.map((device: SmartDevice) => (
                <div
                  key={device.device}
                  className={cn(
                    "p-4 rounded-lg border cursor-pointer transition-colors",
                    selectedDevice === device.device
                      ? "border-primary bg-primary/5"
                      : "border-border hover:bg-muted/50"
                  )}
                  onClick={() => setSelectedDevice(
                    selectedDevice === device.device ? null : device.device
                  )}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <HardDrive className="h-5 w-5 text-muted-foreground" />
                      <div>
                        <div className="flex items-center gap-2">
                          <p className="font-medium">{device.device}</p>
                          <HealthBadge 
                            status={getHealthStatus(device.smartStatus)}
                          />
                        </div>
                        <p className="text-sm text-muted-foreground">
                          {device.model} • {formatBytes(device.capacity)}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <div className="text-right">
                        <p className="text-sm">
                          <span className={getTemperatureColor(device.temperature)}>
                            {device.temperature}°C
                          </span>
                        </p>
                        <p className="text-xs text-muted-foreground">
                          {Math.floor(device.powerOnHours / 24)} days
                        </p>
                      </div>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={(e) => {
                          e.stopPropagation()
                          handleRunTest(device.device, 'short')
                        }}
                        disabled={runningTests.has(device.device)}
                      >
                        {runningTests.has(device.device) ? (
                          <>
                            <RefreshCw className="h-3 w-3 mr-1 animate-spin" />
                            Testing...
                          </>
                        ) : (
                          <>
                            <Play className="h-3 w-3 mr-1" />
                            Quick Test
                          </>
                        )}
                      </Button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-12">
              <HardDrive className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <p className="text-muted-foreground">No storage devices found</p>
            </div>
          )}
        </Card>
      </motion.div>
      
      {/* Selected Device Details */}
      {selectedDeviceData && (
        <motion.div 
          variants={itemVariants}
          initial="hidden"
          animate="visible"
        >
          <Card
            title={`SMART Attributes - ${selectedDeviceData.device}`}
            description={`${selectedDeviceData.model} (S/N: ${selectedDeviceData.serial})`}
          >
            <div className="space-y-6">
              {/* Critical Attributes */}
              <div>
                <h4 className="text-sm font-medium mb-3">Critical Attributes</h4>
                <div className="space-y-2">
                  {selectedDeviceData.attributes
                    .filter((attr: any) => 
                      ['Reallocated_Sector_Ct', 'Current_Pending_Sector', 
                       'Offline_Uncorrectable', 'UDMA_CRC_Error_Count',
                       'Reallocated_Event_Count'].includes(attr.name)
                    )
                    .map((attr: any) => (
                      <div key={attr.id} className="flex items-center justify-between p-2 rounded bg-muted/50">
                        <div className="flex items-center gap-2">
                          {attr.status === 'FAILING' ? (
                            <XCircle className="h-4 w-4 text-red-500" />
                          ) : attr.value < attr.threshold + 10 ? (
                            <AlertTriangle className="h-4 w-4 text-yellow-500" />
                          ) : (
                            <CheckCircle className="h-4 w-4 text-green-500" />
                          )}
                          <span className="text-sm">{attr.name.replace(/_/g, ' ')}</span>
                        </div>
                        <div className="text-right">
                          <p className="text-sm font-medium">{attr.value}</p>
                          <p className="text-xs text-muted-foreground">
                            Threshold: {attr.threshold}
                          </p>
                        </div>
                      </div>
                    ))}
                </div>
              </div>
              
              {/* All Attributes Table */}
              <div>
                <h4 className="text-sm font-medium mb-3">All Attributes</h4>
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b">
                        <th className="text-left py-2">ID</th>
                        <th className="text-left py-2">Attribute</th>
                        <th className="text-right py-2">Value</th>
                        <th className="text-right py-2">Worst</th>
                        <th className="text-right py-2">Threshold</th>
                        <th className="text-right py-2">Raw</th>
                        <th className="text-right py-2">Status</th>
                      </tr>
                    </thead>
                    <tbody>
                      {selectedDeviceData.attributes.map((attr: any) => (
                        <tr key={attr.id} className="border-b">
                          <td className="py-2">{attr.id}</td>
                          <td className="py-2">{attr.name.replace(/_/g, ' ')}</td>
                          <td className="text-right py-2">{attr.value}</td>
                          <td className="text-right py-2">{attr.worst}</td>
                          <td className="text-right py-2">{attr.threshold}</td>
                          <td className="text-right py-2 font-mono text-xs">{attr.raw}</td>
                          <td className="text-right py-2">
                            <StatusPill 
                              status={
                                attr.status === 'FAILING' ? 'error' :
                                attr.value < attr.threshold + 10 ? 'warning' : 'active'
                              }
                              size="sm"
                            />
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
              
              {/* Test Options */}
              <div>
                <h4 className="text-sm font-medium mb-3">Run SMART Test</h4>
                <div className="flex gap-2">
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => handleRunTest(selectedDeviceData.device, 'short')}
                    disabled={runningTests.has(selectedDeviceData.device)}
                  >
                    Short Test (~2 min)
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => handleRunTest(selectedDeviceData.device, 'conveyance')}
                    disabled={runningTests.has(selectedDeviceData.device)}
                  >
                    Conveyance Test (~5 min)
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => handleRunTest(selectedDeviceData.device, 'long')}
                    disabled={runningTests.has(selectedDeviceData.device)}
                  >
                    Extended Test (~60+ min)
                  </Button>
                </div>
              </div>
            </div>
          </Card>
        </motion.div>
      )}
      
      {/* Alerts */}
      {summary && summary.criticalDevices > 0 && (
        <motion.div variants={itemVariants}>
          <Alert className="border-destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              {summary.criticalDevices} disk{summary.criticalDevices !== 1 ? 's' : ''} 
              {summary.criticalDevices === 1 ? ' is' : ' are'} in critical condition. 
              Immediate backup and replacement recommended.
            </AlertDescription>
          </Alert>
        </motion.div>
      )}
    </motion.div>
  )
}