import { useState } from 'react'
import { motion } from 'framer-motion'
import { 
  HardDrive, 
  Database,
  Activity,
  Plus,
  RefreshCw,
  Settings,
  Power,
  Lightbulb,
  Trash2,
  AlertTriangle,
  CheckCircle,
  Info,
  ChevronRight,
  Thermometer,
  Zap,
  Clock
} from 'lucide-react'
import { ColumnDef } from '@tanstack/react-table'
import { PageHeader } from '@/components/ui/page-header'
import { Card } from '@/components/ui/card-enhanced'
import { EmptyState } from '@/components/ui/empty-state'
import { StatusPill, HealthBadge } from '@/components/ui/status'
import { Button } from '@/components/ui/button'
import { DataTable } from '@/components/ui/data-table'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { useDisks, useVolumes } from '@/hooks/use-api'
import { cn } from '@/lib/utils'
import type { Disk, Volume } from '@/lib/api-client'

// Mock SMART data
const mockSmartData = {
  sda: {
    temperature: 35,
    powerOnHours: 8760,
    attributes: [
      { id: 5, name: 'Reallocated Sectors', value: 0, threshold: 36, status: 'ok' },
      { id: 9, name: 'Power On Hours', value: 8760, threshold: 0, status: 'ok' },
      { id: 194, name: 'Temperature', value: 35, threshold: 0, status: 'ok' },
      { id: 197, name: 'Current Pending Sectors', value: 0, threshold: 0, status: 'ok' },
    ]
  }
}

// Helper functions
function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

// Disk columns
const diskColumns: ColumnDef<Disk>[] = [
  {
    accessorKey: 'name',
    header: 'Device',
    cell: ({ row }) => (
      <div className="flex items-center gap-2">
        <HardDrive className="h-4 w-4 text-muted-foreground" />
        <span className="font-mono">{row.original.name}</span>
      </div>
    ),
  },
  {
    accessorKey: 'model',
    header: 'Model',
    cell: ({ row }) => row.original.model || 'Unknown',
  },
  {
    accessorKey: 'size',
    header: 'Size',
    cell: ({ row }) => formatBytes(row.original.size),
  },
  {
    accessorKey: 'used',
    header: 'Usage',
    cell: ({ row }) => {
      const used = row.original.used || 0
      const size = row.original.size
      const percentage = size > 0 ? (used / size) * 100 : 0
      return (
        <div className="flex items-center gap-2">
          <div className="w-24">
            <div className="h-2 bg-muted rounded-full overflow-hidden">
              <div 
                className={cn(
                  "h-full transition-all",
                  percentage < 70 ? "bg-green-500" : 
                  percentage < 90 ? "bg-yellow-500" : "bg-red-500"
                )}
                style={{ width: `${percentage}%` }}
              />
            </div>
          </div>
          <span className="text-sm text-muted-foreground">
            {percentage.toFixed(1)}%
          </span>
        </div>
      )
    },
  },
  {
    accessorKey: 'health',
    header: 'Health',
    cell: ({ row }) => {
      const health = row.original.health || 'healthy'
      return <HealthBadge status={health === 'healthy' ? 'healthy' : health === 'warning' ? 'degraded' : 'critical'} />
    },
  },
  {
    accessorKey: 'temperature',
    header: 'Temp',
    cell: ({ row }) => {
      const temp = row.original.temperature
      if (!temp) return '-'
      return (
        <div className="flex items-center gap-1">
          <Thermometer className="h-4 w-4 text-muted-foreground" />
          <span className={cn(
            "text-sm",
            temp < 40 ? "text-green-600" :
            temp < 50 ? "text-yellow-600" : "text-red-600"
          )}>
            {temp}°C
          </span>
        </div>
      )
    },
  },
  {
    id: 'actions',
    header: 'Actions',
    cell: () => (
      <div className="flex items-center gap-1">
        <Button variant="ghost" size="sm">
          <Info className="h-4 w-4" />
        </Button>
        <Button variant="ghost" size="sm">
          <Lightbulb className="h-4 w-4" />
        </Button>
        <Button variant="ghost" size="sm">
          <Power className="h-4 w-4" />
        </Button>
      </div>
    ),
  },
]

// Volume columns
const volumeColumns: ColumnDef<Volume>[] = [
  {
    accessorKey: 'name',
    header: 'Name',
    cell: ({ row }) => (
      <div className="flex items-center gap-2">
        <Database className="h-4 w-4 text-muted-foreground" />
        <span className="font-medium">{row.original.name}</span>
      </div>
    ),
  },
  {
    accessorKey: 'type',
    header: 'Type',
    cell: ({ row }) => (
      <StatusPill variant="info">
        {row.original.type.toUpperCase()}
      </StatusPill>
    ),
  },
  {
    accessorKey: 'size',
    header: 'Size',
    cell: ({ row }) => {
      const { size, used } = row.original
      return (
        <div>
          <div className="text-sm">{formatBytes(used)} / {formatBytes(size)}</div>
          <div className="mt-1 h-1.5 bg-muted rounded-full overflow-hidden w-20">
            <div 
              className="h-full bg-primary transition-all"
              style={{ width: `${(used / size) * 100}%` }}
            />
          </div>
        </div>
      )
    },
  },
  {
    accessorKey: 'pool',
    header: 'Pool',
    cell: ({ row }) => row.original.pool || '-',
  },
  {
    accessorKey: 'status',
    header: 'Status',
    cell: ({ row }) => {
      const status = row.original.status
      return (
        <StatusPill variant={
          status === 'online' ? 'success' :
          status === 'degraded' ? 'warning' : 'error'
        }>
          {status}
        </StatusPill>
      )
    },
  },
  {
    accessorKey: 'mountpoint',
    header: 'Mount',
    cell: ({ row }) => (
      <code className="text-xs bg-muted px-1 py-0.5 rounded">
        {row.original.mountpoint}
      </code>
    ),
  },
  {
    id: 'actions',
    header: 'Actions',
    cell: () => (
      <div className="flex items-center gap-1">
        <Button variant="ghost" size="sm">
          <Settings className="h-4 w-4" />
        </Button>
        <Button variant="ghost" size="sm" className="text-destructive">
          <Trash2 className="h-4 w-4" />
        </Button>
      </div>
    ),
  },
]

export function Storage() {
  const [activeTab, setActiveTab] = useState('disks')
  const [selectedDisk, setSelectedDisk] = useState<string | null>(null)
  const [isRefreshing, setIsRefreshing] = useState(false)
  
  const { data: disks, isLoading: disksLoading, refetch: refetchDisks } = useDisks()
  const { data: volumes, isLoading: volumesLoading, refetch: refetchVolumes } = useVolumes()

  const handleRefresh = async () => {
    setIsRefreshing(true)
    await Promise.all([refetchDisks(), refetchVolumes()])
    setTimeout(() => setIsRefreshing(false), 500)
  }

  const handleCreateVolume = () => {
    // TODO: Open create volume wizard
    console.log('Create volume')
  }

  // Use mock data if API fails
  const diskData = disks || [
    { name: 'sda', model: 'Samsung SSD 970', size: 500000000000, used: 250000000000, health: 'healthy' as const, temperature: 35 },
    { name: 'sdb', model: 'WD Red Plus', size: 4000000000000, used: 3200000000000, health: 'healthy' as const, temperature: 38 },
    { name: 'sdc', model: 'Seagate IronWolf', size: 8000000000000, used: 1000000000000, health: 'warning' as const, temperature: 42 },
  ]

  const volumeData = volumes || [
    { id: '1', name: 'main-pool', type: 'zfs' as const, size: 12000000000000, used: 4450000000000, status: 'online' as const, mountpoint: '/mnt/main', pool: 'tank' },
    { id: '2', name: 'backup-vol', type: 'btrfs' as const, size: 4000000000000, used: 2800000000000, status: 'online' as const, mountpoint: '/mnt/backup' },
  ]

  const smartData = selectedDisk ? mockSmartData[selectedDisk as keyof typeof mockSmartData] : null

  return (
    <div className="space-y-6">
      <PageHeader
        title="Storage"
        description="Manage disks, volumes, and storage pools"
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
            <Button size="sm" onClick={handleCreateVolume}>
              <Plus className="h-4 w-4 mr-2" />
              Create Volume
            </Button>
          </>
        }
      />

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="disks">
            <HardDrive className="h-4 w-4 mr-2" />
            Disks
          </TabsTrigger>
          <TabsTrigger value="volumes">
            <Database className="h-4 w-4 mr-2" />
            Volumes
          </TabsTrigger>
          <TabsTrigger value="smart">
            <Activity className="h-4 w-4 mr-2" />
            SMART
          </TabsTrigger>
        </TabsList>

        <TabsContent value="disks">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
          >
            <Card
              title="Physical Disks"
              description="All connected storage devices"
              isLoading={disksLoading}
            >
              {diskData.length > 0 ? (
                <DataTable
                  columns={diskColumns}
                  data={diskData}
                  searchKey="disks"
                />
              ) : (
                <EmptyState
                  variant="no-data"
                  icon={HardDrive}
                  title="No disks detected"
                  description="No storage devices are connected to the system"
                />
              )}
            </Card>

            {/* Disk warnings */}
            {diskData.some(d => d.health === 'warning' || d.health === 'critical') && (
              <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                className="mt-4"
              >
                <Card className="border-yellow-600/50 bg-yellow-600/10">
                  <div className="flex items-start gap-3">
                    <AlertTriangle className="h-5 w-5 text-yellow-600 mt-0.5" />
                    <div className="flex-1">
                      <h4 className="font-medium text-yellow-600">Disk Health Warning</h4>
                      <p className="text-sm text-muted-foreground mt-1">
                        One or more disks are reporting health issues. Check SMART data for details.
                      </p>
                    </div>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setActiveTab('smart')}
                    >
                      View SMART
                      <ChevronRight className="h-4 w-4 ml-1" />
                    </Button>
                  </div>
                </Card>
              </motion.div>
            )}
          </motion.div>
        </TabsContent>

        <TabsContent value="volumes">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
          >
            <Card
              title="Storage Volumes"
              description="Logical volumes and pools"
              isLoading={volumesLoading}
            >
              {volumeData.length > 0 ? (
                <DataTable
                  columns={volumeColumns}
                  data={volumeData}
                  searchKey="volumes"
                />
              ) : (
                <EmptyState
                  variant="no-data"
                  icon={Database}
                  title="No volumes configured"
                  description="Create your first storage volume to get started"
                  action={{
                    label: "Create Volume",
                    onClick: handleCreateVolume
                  }}
                />
              )}
            </Card>
          </motion.div>
        </TabsContent>

        <TabsContent value="smart">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
          >
            <div className="grid gap-4 lg:grid-cols-3">
              {/* Disk selector */}
              <Card title="Select Disk" className="lg:col-span-1">
                <div className="space-y-2">
                  {diskData.map(disk => (
                    <button
                      key={disk.name}
                      onClick={() => setSelectedDisk(disk.name)}
                      className={cn(
                        "w-full flex items-center justify-between p-3 rounded-lg transition-colors",
                        selectedDisk === disk.name
                          ? "bg-primary/10 border border-primary"
                          : "bg-muted/30 hover:bg-muted/50 border border-transparent"
                      )}
                    >
                      <div className="flex items-center gap-2">
                        <HardDrive className="h-4 w-4 text-muted-foreground" />
                        <span className="font-mono text-sm">{disk.name}</span>
                      </div>
                      <HealthBadge status={
                        disk.health === 'healthy' ? 'healthy' :
                        disk.health === 'warning' ? 'degraded' : 'critical'
                      } />
                    </button>
                  ))}
                </div>
              </Card>

              {/* SMART details */}
              <Card 
                title={selectedDisk ? `SMART Data - ${selectedDisk}` : 'SMART Data'}
                className="lg:col-span-2"
              >
                {selectedDisk && smartData ? (
                  <div className="space-y-4">
                    {/* Overview */}
                    <div className="grid grid-cols-3 gap-4">
                      <div className="p-3 bg-muted/30 rounded-lg">
                        <div className="flex items-center gap-2 text-sm text-muted-foreground mb-1">
                          <Thermometer className="h-4 w-4" />
                          Temperature
                        </div>
                        <div className="text-xl font-bold">{smartData.temperature}°C</div>
                      </div>
                      <div className="p-3 bg-muted/30 rounded-lg">
                        <div className="flex items-center gap-2 text-sm text-muted-foreground mb-1">
                          <Clock className="h-4 w-4" />
                          Power On Hours
                        </div>
                        <div className="text-xl font-bold">{smartData.powerOnHours.toLocaleString()}</div>
                      </div>
                      <div className="p-3 bg-muted/30 rounded-lg">
                        <div className="flex items-center gap-2 text-sm text-muted-foreground mb-1">
                          <Zap className="h-4 w-4" />
                          Overall Status
                        </div>
                        <div className="mt-1">
                          <StatusPill variant="success">Healthy</StatusPill>
                        </div>
                      </div>
                    </div>

                    {/* Attributes table */}
                    <div className="border rounded-lg overflow-hidden">
                      <table className="w-full">
                        <thead className="bg-muted/30">
                          <tr>
                            <th className="px-4 py-2 text-left text-sm font-medium">ID</th>
                            <th className="px-4 py-2 text-left text-sm font-medium">Attribute</th>
                            <th className="px-4 py-2 text-left text-sm font-medium">Value</th>
                            <th className="px-4 py-2 text-left text-sm font-medium">Threshold</th>
                            <th className="px-4 py-2 text-left text-sm font-medium">Status</th>
                          </tr>
                        </thead>
                        <tbody>
                          {smartData.attributes.map((attr, idx) => (
                            <tr key={attr.id} className={idx % 2 === 0 ? 'bg-muted/10' : ''}>
                              <td className="px-4 py-2 text-sm font-mono">{attr.id}</td>
                              <td className="px-4 py-2 text-sm">{attr.name}</td>
                              <td className="px-4 py-2 text-sm font-mono">{attr.value}</td>
                              <td className="px-4 py-2 text-sm font-mono">{attr.threshold}</td>
                              <td className="px-4 py-2">
                                {attr.status === 'ok' ? (
                                  <CheckCircle className="h-4 w-4 text-green-500" />
                                ) : (
                                  <AlertTriangle className="h-4 w-4 text-yellow-500" />
                                )}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </div>
                ) : (
                  <EmptyState
                    variant="no-data"
                    title="Select a disk"
                    description="Choose a disk from the list to view its SMART data"
                  />
                )}
              </Card>
            </div>
          </motion.div>
        </TabsContent>
      </Tabs>
    </div>
  )
}