import { motion } from 'framer-motion'
import { 
  Activity, AlertCircle, CheckCircle, Database,
  HardDrive, Info, RefreshCw, Thermometer, Play,
  AlertTriangle
} from 'lucide-react'
import { useMemo, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useDiskHealth, formatBytes, getDiskHealthColor } from '@/hooks/use-health'
import { useQueryClient } from '@tanstack/react-query'
import { cn } from '@/lib/utils'
import type { DiskHealth } from '@/lib/api-health'

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

// Health status component
function HealthStatus({ state }: { state: string }) {
  const Icon = state === 'critical' ? AlertTriangle :
               state === 'warning' ? AlertCircle : CheckCircle
  
  const color = getDiskHealthColor(state)
  
  return (
    <div className={cn("flex items-center gap-1", color)}>
      <Icon className="h-4 w-4" />
      <span className="capitalize">{state}</span>
    </div>
  )
}

// Disk card component
function DiskCard({ 
  disk, 
  isSelected, 
  onClick 
}: { 
  disk: DiskHealth
  isSelected: boolean
  onClick: () => void 
}) {
  const usageColor = disk.usagePct > 90 ? 'bg-red-500' :
                    disk.usagePct > 80 ? 'bg-yellow-500' : ''

  return (
    <motion.div variants={itemVariants}>
      <Card 
        className={cn(
          "cursor-pointer transition-all hover:shadow-md",
          isSelected && "ring-2 ring-primary"
        )}
        onClick={onClick}
      >
        <CardHeader className="pb-3">
          <div className="flex justify-between items-start">
            <div className="space-y-1">
              <CardTitle className="text-base flex items-center gap-2">
                <HardDrive className="h-4 w-4" />
                {disk.name}
              </CardTitle>
              <CardDescription className="text-xs">
                {disk.model || 'Unknown Model'} • {formatBytes(disk.sizeBytes)}
              </CardDescription>
            </div>
            <HealthStatus state={disk.state} />
          </div>
        </CardHeader>
        <CardContent className="space-y-3">
          <div className="space-y-2">
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">Usage</span>
              <span className="font-medium">{Math.round(disk.usagePct)}%</span>
            </div>
            <Progress 
              value={disk.usagePct} 
              className="h-2"
              indicatorClassName={cn("transition-all", usageColor)}
            />
          </div>
          
          {disk.tempC !== undefined && (
            <div className="flex justify-between items-center text-sm">
              <span className="text-muted-foreground flex items-center gap-1">
                <Thermometer className="h-3 w-3" />
                Temperature
              </span>
              <span className={cn(
                "font-medium",
                disk.tempC > 50 ? 'text-red-500' : disk.tempC > 40 ? 'text-yellow-500' : ''
              )}>
                {disk.tempC.toFixed(0)}°C
              </span>
            </div>
          )}

          <div className="flex justify-between items-center text-sm">
            <span className="text-muted-foreground">SMART</span>
            <Badge variant={disk.smart.passed ? "default" : "destructive"} className="text-xs">
              {disk.smart.passed ? 'Passed' : 'Failed'}
            </Badge>
          </div>

          <div className="pt-2 border-t">
            <div className="grid grid-cols-2 gap-2 text-xs">
              <div>
                <p className="text-muted-foreground">Filesystem</p>
                <p className="font-medium">{disk.filesystem || 'Unknown'}</p>
              </div>
              <div>
                <p className="text-muted-foreground">Mount</p>
                <p className="font-medium truncate" title={disk.mountPoint}>
                  {disk.mountPoint || 'Not mounted'}
                </p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </motion.div>
  )
}

export default function DiskHealth() {
  const queryClient = useQueryClient()
  const { data: rawDisks, isLoading, error, refetch } = useDiskHealth()
  const [selectedDisk, setSelectedDisk] = useState<string | null>(null)
  const [isManualRefreshing, setIsManualRefreshing] = useState(false)
  const [runningTests, setRunningTests] = useState<Set<string>>(new Set())

  // Ensure disks is always an array
  const disks = useMemo(() => {
    if (!rawDisks) return []
    return Array.isArray(rawDisks) ? rawDisks : []
  }, [rawDisks])

  // Calculate summary statistics
  const summary = useMemo(() => {
    if (!disks || disks.length === 0) return null

    const totalCapacity = disks.reduce((sum, disk) => sum + disk.sizeBytes, 0)
    const totalUsed = disks.reduce((sum, disk) => sum + (disk.sizeBytes * disk.usagePct / 100), 0)
    const healthyDisks = disks.filter(d => d.state === 'healthy').length
    const warningDisks = disks.filter(d => d.state === 'warning').length
    const criticalDisks = disks.filter(d => d.state === 'critical').length
    const avgTemp = disks.filter(d => d.tempC !== undefined).reduce((sum, d) => sum + (d.tempC || 0), 0) / 
                   (disks.filter(d => d.tempC !== undefined).length || 1)

    return {
      totalDisks: disks.length,
      totalCapacity,
      totalUsed,
      usagePercent: (totalUsed / totalCapacity) * 100,
      healthyDisks,
      warningDisks,
      criticalDisks,
      avgTemp: disks.some(d => d.tempC !== undefined) ? avgTemp : undefined
    }
  }, [disks])

  const selectedDiskData = useMemo(() => {
    if (!selectedDisk || !disks) return null
    return disks.find((d: DiskHealth) => d.id === selectedDisk) || null
  }, [selectedDisk, disks])

  const handleManualRefresh = async () => {
    setIsManualRefreshing(true)
    await queryClient.invalidateQueries({ queryKey: ['health', 'disks'] })
    await refetch()
    setTimeout(() => setIsManualRefreshing(false), 1000)
  }

  const handleRunTest = async (diskId: string, testType: 'short' | 'long' | 'conveyance') => {
    setRunningTests(prev => new Set(prev).add(diskId))
    
    // Simulate test execution
    setTimeout(() => {
      setRunningTests(prev => {
        const next = new Set(prev)
        next.delete(diskId)
        return next
      })
    }, testType === 'short' ? 5000 : testType === 'conveyance' ? 10000 : 30000)
  }

  if (isLoading && !disks) {
    return (
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <h2 className="text-2xl font-bold">Disk Health</h2>
        </div>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[...Array(3)].map((_, i) => (
            <Card key={i} className="animate-pulse">
              <CardHeader className="pb-3">
                <div className="h-4 bg-muted rounded w-32"></div>
                <div className="h-3 bg-muted rounded w-48 mt-2"></div>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  <div className="h-2 bg-muted rounded"></div>
                  <div className="h-4 bg-muted rounded w-20"></div>
                </div>
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
            Failed to load disk health data. 
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

  if (!disks || disks.length === 0) {
    return (
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <h2 className="text-2xl font-bold">Disk Health</h2>
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
        <Alert>
          <Info className="h-4 w-4" />
          <AlertDescription>
            No disks detected. Make sure disks are properly connected and the system has necessary permissions.
          </AlertDescription>
        </Alert>
      </div>
    )
  }

  return (
    <motion.div
      variants={containerVariants}
      initial="hidden"
      animate="visible"
      className="space-y-6"
    >
      {/* Header */}
      <div className="flex justify-between items-center">
        <div className="flex items-center gap-3">
          <h2 className="text-2xl font-bold">Disk Health</h2>
          {summary && (
            <Badge variant="outline" className="gap-1">
              <Database className="h-3 w-3" />
              {summary.totalDisks} {summary.totalDisks === 1 ? 'Disk' : 'Disks'}
            </Badge>
          )}
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

      {/* Summary stats */}
      {summary && (
        <motion.div variants={itemVariants}>
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Storage Overview</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div>
                  <p className="text-sm text-muted-foreground">Total Capacity</p>
                  <p className="text-xl font-bold">{formatBytes(summary.totalCapacity)}</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Used Space</p>
                  <p className="text-xl font-bold">{formatBytes(summary.totalUsed)}</p>
                  <p className="text-xs text-muted-foreground">{Math.round(summary.usagePercent)}% used</p>
                </div>
                <div>
                  <p className="text-sm text-muted-foreground">Disk Status</p>
                  <div className="flex gap-2 mt-1">
                    {summary.healthyDisks > 0 && (
                      <Badge variant="default" className="text-xs">
                        {summary.healthyDisks} Healthy
                      </Badge>
                    )}
                    {summary.warningDisks > 0 && (
                      <Badge variant="secondary" className="text-xs bg-yellow-100">
                        {summary.warningDisks} Warning
                      </Badge>
                    )}
                    {summary.criticalDisks > 0 && (
                      <Badge variant="destructive" className="text-xs">
                        {summary.criticalDisks} Critical
                      </Badge>
                    )}
                  </div>
                </div>
                {summary.avgTemp !== undefined && (
                  <div>
                    <p className="text-sm text-muted-foreground">Avg Temperature</p>
                    <p className="text-xl font-bold flex items-center gap-1">
                      <Thermometer className="h-4 w-4" />
                      {summary.avgTemp.toFixed(0)}°C
                    </p>
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </motion.div>
      )}

      {/* Disk cards grid */}
      <motion.div 
        variants={containerVariants}
        className="grid gap-4 md:grid-cols-2 lg:grid-cols-3"
      >
        {disks.map((disk) => (
          <DiskCard
            key={disk.id}
            disk={disk}
            isSelected={selectedDisk === disk.id}
            onClick={() => setSelectedDisk(disk.id === selectedDisk ? null : disk.id)}
          />
        ))}
      </motion.div>

      {/* Selected disk details */}
      {selectedDiskData && (
        <motion.div variants={itemVariants}>
          <Card>
            <CardHeader>
              <CardTitle className="text-base flex items-center gap-2">
                <HardDrive className="h-4 w-4" />
                {selectedDiskData.name} Details
              </CardTitle>
              <CardDescription>
                Detailed information and SMART attributes
              </CardDescription>
            </CardHeader>
            <CardContent>
              <Tabs defaultValue="info" className="space-y-4">
                <TabsList>
                  <TabsTrigger value="info">Information</TabsTrigger>
                  <TabsTrigger value="smart">SMART Data</TabsTrigger>
                  <TabsTrigger value="tests">Tests</TabsTrigger>
                </TabsList>

                <TabsContent value="info" className="space-y-4">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <p className="text-sm text-muted-foreground">Model</p>
                      <p className="font-medium">{selectedDiskData.model || 'Unknown'}</p>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Serial Number</p>
                      <p className="font-medium">{selectedDiskData.serial || 'Unknown'}</p>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Capacity</p>
                      <p className="font-medium">{formatBytes(selectedDiskData.sizeBytes)}</p>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Filesystem</p>
                      <p className="font-medium">{selectedDiskData.filesystem || 'Unknown'}</p>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Mount Point</p>
                      <p className="font-medium">{selectedDiskData.mountPoint || 'Not mounted'}</p>
                    </div>
                    <div>
                      <p className="text-sm text-muted-foreground">Usage</p>
                      <p className="font-medium">{Math.round(selectedDiskData.usagePct)}%</p>
                    </div>
                  </div>
                </TabsContent>

                <TabsContent value="smart" className="space-y-4">
                  <div className="space-y-3">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium">SMART Status</span>
                      <Badge variant={selectedDiskData.smart.passed ? "default" : "destructive"}>
                        {selectedDiskData.smart.passed ? 'PASSED' : 'FAILED'}
                      </Badge>
                    </div>
                    
                    {selectedDiskData.smart.attrs && Object.keys(selectedDiskData.smart.attrs).length > 0 ? (
                      <Table>
                        <TableHeader>
                          <TableRow>
                            <TableHead>Attribute</TableHead>
                            <TableHead>Value</TableHead>
                            <TableHead>Threshold</TableHead>
                            <TableHead>Status</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {Object.entries(selectedDiskData.smart.attrs).map(([key, value]: [string, any]) => (
                            <TableRow key={key}>
                              <TableCell className="font-medium">{key}</TableCell>
                              <TableCell>{value.value || '-'}</TableCell>
                              <TableCell>{value.threshold || '-'}</TableCell>
                              <TableCell>
                                <Badge variant={value.status === 'ok' ? 'default' : 'destructive'} className="text-xs">
                                  {value.status || 'Unknown'}
                                </Badge>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    ) : (
                      <Alert>
                        <Info className="h-4 w-4" />
                        <AlertDescription>
                          No SMART attributes available for this disk.
                        </AlertDescription>
                      </Alert>
                    )}
                  </div>
                </TabsContent>

                <TabsContent value="tests" className="space-y-4">
                  <div className="space-y-3">
                    <p className="text-sm text-muted-foreground">
                      Run SMART self-tests to check disk health
                    </p>
                    <div className="flex gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleRunTest(selectedDiskData.id, 'short')}
                        disabled={runningTests.has(selectedDiskData.id)}
                      >
                        <Play className="h-3 w-3 mr-1" />
                        Short Test (~2 min)
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleRunTest(selectedDiskData.id, 'conveyance')}
                        disabled={runningTests.has(selectedDiskData.id)}
                      >
                        <Play className="h-3 w-3 mr-1" />
                        Conveyance Test (~5 min)
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => handleRunTest(selectedDiskData.id, 'long')}
                        disabled={runningTests.has(selectedDiskData.id)}
                      >
                        <Play className="h-3 w-3 mr-1" />
                        Extended Test (~30 min)
                      </Button>
                    </div>
                    {runningTests.has(selectedDiskData.id) && (
                      <Alert>
                        <Activity className="h-4 w-4 animate-spin" />
                        <AlertDescription>
                          Test is running... This may take several minutes.
                        </AlertDescription>
                      </Alert>
                    )}
                  </div>
                </TabsContent>
              </Tabs>
            </CardContent>
          </Card>
        </motion.div>
      )}
    </motion.div>
  )
}