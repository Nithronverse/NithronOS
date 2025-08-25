import { useState } from 'react'
import { motion } from 'framer-motion'
import { 
  Cloud,
  Plus,
  Edit,
  Trash2,
  Play,
  Pause,
  Calendar,
  Clock,
  CheckCircle,
  XCircle,
  Server,
  Shield,
  Key,
  FileText,
  Globe,
  Lock
} from 'lucide-react'
import { ColumnDef } from '@tanstack/react-table'
import { PageHeader } from '@/components/ui/page-header'
import { Card } from '@/components/ui/card-enhanced'
import { EmptyState } from '@/components/ui/empty-state'
import { StatusPill, Metric } from '@/components/ui/status'
import { Button } from '@/components/ui/button'
import { DataTable } from '@/components/ui/data-table'
import { SlideOver } from '@/components/ui/slide-over'
import { cn } from '@/lib/utils'
import { pushToast } from '@/components/ui/toast'

// Mock data
const mockDestinations = [
  {
    id: '1',
    name: 'AWS S3 Backup',
    type: 's3',
    endpoint: 's3.amazonaws.com',
    bucket: 'nithronos-backup',
    lastRun: '2024-01-15T10:30:00Z',
    status: 'active',
    encrypted: true
  },
  {
    id: '2',
    name: 'Offsite NAS',
    type: 'ssh',
    endpoint: '192.168.2.100',
    path: '/backup/nithronos',
    lastRun: '2024-01-15T08:00:00Z',
    status: 'active',
    encrypted: false
  }
]

const mockJobs = [
  {
    id: 'j1',
    name: 'Daily Documents Backup',
    destination: 'AWS S3 Backup',
    schedule: '0 2 * * *',
    lastResult: 'success',
    nextRun: '2024-01-16T02:00:00Z',
    status: 'idle',
    size: 1500000000
  },
  {
    id: 'j2',
    name: 'Hourly Database Sync',
    destination: 'Offsite NAS',
    schedule: '0 * * * *',
    lastResult: 'success',
    nextRun: '2024-01-15T11:00:00Z',
    status: 'running',
    progress: 67,
    size: 250000000
  }
]

// Destination columns
const destinationColumns: ColumnDef<any>[] = [
  {
    accessorKey: 'name',
    header: 'Destination',
    cell: ({ row }) => (
      <div className="flex items-center gap-3">
        <div className="p-2 rounded-lg bg-muted">
          {row.original.type === 's3' ? <Cloud className="h-4 w-4 text-muted-foreground" /> :
           row.original.type === 'ssh' ? <Server className="h-4 w-4 text-muted-foreground" /> :
           <Globe className="h-4 w-4 text-muted-foreground" />}
        </div>
        <div>
          <div className="font-medium">{row.original.name}</div>
          <div className="text-xs text-muted-foreground">{row.original.endpoint}</div>
        </div>
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
    accessorKey: 'encrypted',
    header: 'Encryption',
    cell: ({ row }) => (
      <div className="flex items-center gap-2">
        {row.original.encrypted ? (
          <>
            <Lock className="h-4 w-4 text-green-500" />
            <span className="text-sm">Enabled</span>
          </>
        ) : (
          <>
            <Lock className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm text-muted-foreground">Disabled</span>
          </>
        )}
      </div>
    ),
  },
  {
    accessorKey: 'status',
    header: 'Status',
    cell: ({ row }) => {
      const status = row.original.status
      return (
        <StatusPill variant={
          status === 'active' ? 'success' :
          status === 'error' ? 'error' : 'muted'
        }>
          {status}
        </StatusPill>
      )
    },
  },
  {
    id: 'actions',
    header: 'Actions',
    cell: () => (
      <div className="flex items-center gap-1">
        <Button variant="ghost" size="sm">
          <Edit className="h-4 w-4" />
        </Button>
        <Button variant="ghost" size="sm" className="text-destructive">
          <Trash2 className="h-4 w-4" />
        </Button>
      </div>
    ),
  },
]

// Job columns
const jobColumns: ColumnDef<any>[] = [
  {
    accessorKey: 'name',
    header: 'Job',
    cell: ({ row }) => (
      <div>
        <div className="font-medium">{row.original.name}</div>
        <div className="text-xs text-muted-foreground">{row.original.destination}</div>
      </div>
    ),
  },
  {
    accessorKey: 'schedule',
    header: 'Schedule',
    cell: ({ row }) => (
      <div className="flex items-center gap-2">
        <Calendar className="h-4 w-4 text-muted-foreground" />
        <span className="text-sm font-mono">{row.original.schedule}</span>
      </div>
    ),
  },
  {
    accessorKey: 'status',
    header: 'Status',
    cell: ({ row }) => {
      const { status, progress } = row.original
      if (status === 'running' && progress) {
        return (
          <div className="space-y-1">
            <StatusPill variant="info">Running</StatusPill>
            <div className="w-24 h-1.5 bg-muted rounded-full overflow-hidden">
              <div 
                className="h-full bg-primary transition-all"
                style={{ width: `${progress}%` }}
              />
            </div>
            <span className="text-xs text-muted-foreground">{progress}%</span>
          </div>
        )
      }
      return (
        <StatusPill variant={status === 'idle' ? 'muted' : 'info'}>
          {status}
        </StatusPill>
      )
    },
  },
  {
    accessorKey: 'lastResult',
    header: 'Last Result',
    cell: ({ row }) => {
      const result = row.original.lastResult
      return result ? (
        <div className="flex items-center gap-2">
          {result === 'success' ? (
            <CheckCircle className="h-4 w-4 text-green-500" />
          ) : (
            <XCircle className="h-4 w-4 text-red-500" />
          )}
          <span className="text-sm capitalize">{result}</span>
        </div>
      ) : (
        <span className="text-sm text-muted-foreground">-</span>
      )
    },
  },
  {
    accessorKey: 'nextRun',
    header: 'Next Run',
    cell: ({ row }) => {
      const date = row.original.nextRun ? new Date(row.original.nextRun) : null
      return date ? (
        <div className="flex items-center gap-2 text-sm">
          <Clock className="h-4 w-4 text-muted-foreground" />
          <span>{date.toLocaleString()}</span>
        </div>
      ) : (
        <span className="text-sm text-muted-foreground">-</span>
      )
    },
  },
  {
    id: 'actions',
    header: 'Actions',
    cell: ({ row }) => {
      const isRunning = row.original.status === 'running'
      return (
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => pushToast(isRunning ? 'Stopping job...' : 'Starting job...', 'success')}
          >
            {isRunning ? <Pause className="h-4 w-4" /> : <Play className="h-4 w-4" />}
          </Button>
          <Button variant="ghost" size="sm">
            <FileText className="h-4 w-4" />
          </Button>
        </div>
      )
    },
  },
]

// Simple destination form
function DestinationForm({ onSubmit, onCancel }: { onSubmit: (data: any) => void; onCancel: () => void }) {
  const [formData, setFormData] = useState({
    name: '',
    type: 's3',
    endpoint: '',
    bucket: '',
    encrypted: true,
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSubmit(formData)
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label className="block text-sm font-medium mb-2">Name</label>
        <input
          type="text"
          value={formData.name}
          onChange={(e) => setFormData({ ...formData, name: e.target.value })}
          className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
          placeholder="e.g., AWS Backup"
          required
        />
      </div>

      <div>
        <label className="block text-sm font-medium mb-2">Type</label>
        <div className="grid grid-cols-3 gap-2">
          {['s3', 'ssh', 'webdav'].map(type => (
            <button
              key={type}
              type="button"
              onClick={() => setFormData({ ...formData, type })}
              className={cn(
                "p-2 rounded-lg border transition-colors",
                formData.type === type
                  ? "border-primary bg-primary/10"
                  : "border-border hover:bg-muted/50"
              )}
            >
              {type.toUpperCase()}
            </button>
          ))}
        </div>
      </div>

      <div>
        <label className="block text-sm font-medium mb-2">Endpoint</label>
        <input
          type="text"
          value={formData.endpoint}
          onChange={(e) => setFormData({ ...formData, endpoint: e.target.value })}
          className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
          placeholder="s3.amazonaws.com"
          required
        />
      </div>

      {formData.type === 's3' && (
        <div>
          <label className="block text-sm font-medium mb-2">Bucket</label>
          <input
            type="text"
            value={formData.bucket}
            onChange={(e) => setFormData({ ...formData, bucket: e.target.value })}
            className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            placeholder="my-backup-bucket"
            required
          />
        </div>
      )}

      <label className="flex items-center gap-2">
        <input
          type="checkbox"
          checked={formData.encrypted}
          onChange={(e) => setFormData({ ...formData, encrypted: e.target.checked })}
          className="rounded"
        />
        <span className="text-sm">Enable encryption</span>
      </label>

      <div className="flex justify-end gap-2 pt-4 border-t">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit">
          Save Destination
        </Button>
      </div>
    </form>
  )
}

// Helper function
function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

export function Remote() {
  const [isDestinationOpen, setIsDestinationOpen] = useState(false)

  const handleCreateDestination = async (_data: any) => {
    pushToast('Destination created successfully', 'success')
    setIsDestinationOpen(false)
  }

  // Calculate stats
  const totalBackupSize = mockJobs.reduce((acc, job) => acc + (job.size || 0), 0)
  const successfulJobs = mockJobs.filter(j => j.lastResult === 'success').length

  return (
    <div className="space-y-6">
      <PageHeader
        title="Remote"
        description="Backup destinations and replication jobs"
        actions={
          <Button size="sm" onClick={() => setIsDestinationOpen(true)}>
            <Plus className="h-4 w-4 mr-2" />
            Add Destination
          </Button>
        }
      />

      {/* Stats */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="grid gap-4 md:grid-cols-4"
      >
        <Card>
          <Metric
            label="Total Destinations"
            value={mockDestinations.length}
            sublabel={`${mockDestinations.filter(d => d.status === 'active').length} active`}
          />
        </Card>
        <Card>
          <Metric
            label="Backup Jobs"
            value={mockJobs.length}
            sublabel={`${mockJobs.filter(j => j.status === 'running').length} running`}
          />
        </Card>
        <Card>
          <Metric
            label="Success Rate"
            value={`${Math.round((successfulJobs / mockJobs.length) * 100)}%`}
            sublabel="Last 24 hours"
          />
        </Card>
        <Card>
          <Metric
            label="Total Size"
            value={formatBytes(totalBackupSize)}
            sublabel="Backed up data"
          />
        </Card>
      </motion.div>

      {/* Destinations */}
      <Card
        title="Backup Destinations"
        description="Remote storage locations for your backups"
      >
        {mockDestinations.length > 0 ? (
          <DataTable
            columns={destinationColumns}
            data={mockDestinations}
            searchKey="destinations"
          />
        ) : (
          <EmptyState
            variant="no-data"
            icon={Cloud}
            title="No destinations configured"
            description="Add a backup destination to get started"
            action={{
              label: "Add Destination",
              onClick: () => setIsDestinationOpen(true)
            }}
          />
        )}
      </Card>

      {/* Jobs */}
      <Card
        title="Backup Jobs"
        description="Scheduled backup and replication tasks"
      >
        {mockJobs.length > 0 ? (
          <DataTable
            columns={jobColumns}
            data={mockJobs}
            searchKey="jobs"
          />
        ) : (
          <EmptyState
            variant="no-data"
            title="No backup jobs"
            description="Create a backup job to start protecting your data"
          />
        )}
      </Card>

      {/* Encryption settings */}
      <Card
        title="Encryption Settings"
        description="Manage backup encryption"
      >
        <div className="flex items-start gap-3 p-4 rounded-lg bg-muted/30">
          <Shield className="h-5 w-5 text-primary mt-0.5" />
          <div className="flex-1">
            <h4 className="font-medium">End-to-End Encryption</h4>
            <p className="text-sm text-muted-foreground mt-1">
              All backups are encrypted before leaving your system using AES-256 encryption.
            </p>
          </div>
          <Button variant="outline" size="sm">
            <Key className="h-4 w-4 mr-2" />
            Manage Keys
          </Button>
        </div>
      </Card>

      {/* Add Destination SlideOver */}
      <SlideOver
        isOpen={isDestinationOpen}
        onClose={() => setIsDestinationOpen(false)}
        title="Add Backup Destination"
        description="Configure a remote storage location"
        size="md"
      >
        <DestinationForm
          onSubmit={handleCreateDestination}
          onCancel={() => setIsDestinationOpen(false)}
        />
      </SlideOver>
    </div>
  )
}