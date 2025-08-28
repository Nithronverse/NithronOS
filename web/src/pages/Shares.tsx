import { useState } from 'react'
import { motion } from 'framer-motion'
import { 
  Share2, 
  Plus,
  Edit,
  Trash2,
  Copy,
  ToggleLeft,
  ToggleRight,
  Users,
  Globe,
  Lock,
  FolderOpen,
  Terminal,
  Shield,
  Settings,
  X,
  AlertCircle,
  RefreshCw
} from 'lucide-react'
import { ColumnDef } from '@tanstack/react-table'
import { PageHeader } from '@/components/ui/page-header'
import { Card } from '@/components/ui/card-enhanced'
import { EmptyState } from '@/components/ui/empty-state'
import { StatusPill } from '@/components/ui/status'
import { Button } from '@/components/ui/button'
import { DataTable } from '@/components/ui/data-table'
import { SlideOver } from '@/components/ui/slide-over'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { 
  useShares, 
  useCreateShare, 
  useUpdateShare, 
  useDeleteShare,
  useApiStatus 
} from '@/hooks/use-api'
import { cn } from '@/lib/utils'
import { toast } from '@/components/ui/toast'
import type { Share } from '@/lib/api'

// Share form component
function ShareForm({ 
  share, 
  onSubmit, 
  onCancel 
}: { 
  share?: Share | null
  onSubmit: (data: Partial<Share>) => void
  onCancel: () => void 
}) {
  const [formData, setFormData] = useState<Partial<Share>>({
    name: share?.name || '',
    protocol: share?.protocol || 'smb',
    path: share?.path || '',
    enabled: share?.enabled ?? true,
    guestOk: share?.guestOk || false,
    readOnly: share?.readOnly || false,
    users: share?.users || [],
    groups: share?.groups || [],
    description: share?.description || '',
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    onSubmit(formData)
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* Name */}
      <div>
        <label className="block text-sm font-medium mb-2">Share Name</label>
        <input
          type="text"
          value={formData.name}
          onChange={(e) => setFormData({ ...formData, name: e.target.value })}
          className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          placeholder="e.g., Documents"
          required
        />
      </div>

      {/* Protocol */}
      <div>
        <label className="block text-sm font-medium mb-2">Protocol</label>
        <div className="grid grid-cols-3 gap-2">
          <button
            type="button"
            onClick={() => setFormData({ ...formData, protocol: 'smb' })}
            className={cn(
              "flex items-center justify-center gap-2 p-3 rounded-lg border transition-colors",
              formData.protocol === 'smb'
                ? "border-primary bg-primary/10 text-primary"
                : "border-border hover:bg-muted/50"
            )}
          >
            <Share2 className="h-4 w-4" />
            SMB/CIFS
          </button>
          <button
            type="button"
            onClick={() => setFormData({ ...formData, protocol: 'nfs' })}
            className={cn(
              "flex items-center justify-center gap-2 p-3 rounded-lg border transition-colors",
              formData.protocol === 'nfs'
                ? "border-primary bg-primary/10 text-primary"
                : "border-border hover:bg-muted/50"
            )}
          >
            <Terminal className="h-4 w-4" />
            NFS
          </button>
          <button
            type="button"
            onClick={() => setFormData({ ...formData, protocol: 'afp' })}
            className={cn(
              "flex items-center justify-center gap-2 p-3 rounded-lg border transition-colors",
              formData.protocol === 'afp'
                ? "border-primary bg-primary/10 text-primary"
                : "border-border hover:bg-muted/50"
            )}
          >
            <FolderOpen className="h-4 w-4" />
            AFP
          </button>
        </div>
      </div>

      {/* Path */}
      <div>
        <label className="block text-sm font-medium mb-2">Path</label>
        <div className="relative">
          <FolderOpen className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            value={formData.path}
            onChange={(e) => setFormData({ ...formData, path: e.target.value })}
            className="w-full rounded-md border border-input bg-background pl-10 pr-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            placeholder="/mnt/main/folder"
            required
          />
        </div>
      </div>

      {/* Description */}
      <div>
        <label className="block text-sm font-medium mb-2">Description</label>
        <textarea
          value={formData.description}
          onChange={(e) => setFormData({ ...formData, description: e.target.value })}
          className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          placeholder="Optional description"
          rows={2}
        />
      </div>

      {/* Options */}
      <div className="space-y-3">
        <label className="block text-sm font-medium">Options</label>
        
        <label className="flex items-center gap-3 cursor-pointer">
          <input
            type="checkbox"
            checked={formData.guestOk}
            onChange={(e) => setFormData({ ...formData, guestOk: e.target.checked })}
            className="rounded border-gray-300"
          />
          <div>
            <div className="flex items-center gap-2">
              <Globe className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm font-medium">Allow guest access</span>
            </div>
            <p className="text-xs text-muted-foreground">
              Allow access without authentication
            </p>
          </div>
        </label>

        <label className="flex items-center gap-3 cursor-pointer">
          <input
            type="checkbox"
            checked={formData.readOnly}
            onChange={(e) => setFormData({ ...formData, readOnly: e.target.checked })}
            className="rounded border-gray-300"
          />
          <div>
            <div className="flex items-center gap-2">
              <Lock className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm font-medium">Read-only</span>
            </div>
            <p className="text-xs text-muted-foreground">
              Prevent modifications to files
            </p>
          </div>
        </label>
      </div>

      {/* Users */}
      {!formData.guestOk && (
        <div>
          <label className="block text-sm font-medium mb-2">Allowed Users</label>
          <input
            type="text"
            value={formData.users?.join(', ')}
            onChange={(e) => setFormData({ 
              ...formData, 
              users: e.target.value.split(',').map(u => u.trim()).filter(Boolean) 
            })}
            className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            placeholder="admin, user1, user2"
          />
          <p className="text-xs text-muted-foreground mt-1">
            Comma-separated list of usernames
          </p>
        </div>
      )}

      {/* Actions */}
      <div className="flex items-center gap-3">
        <Button type="submit">
          {share ? 'Update Share' : 'Create Share'}
        </Button>
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
      </div>
    </form>
  )
}

// Share columns
const shareColumns: ColumnDef<Share>[] = [
  {
    accessorKey: 'name',
    header: 'Name',
    cell: ({ row }) => (
      <div className="flex items-center gap-2">
        <Share2 className="h-4 w-4 text-muted-foreground" />
        <div>
          <div className="font-medium">{row.original.name}</div>
          <div className="text-xs text-muted-foreground">{row.original.path}</div>
        </div>
      </div>
    ),
  },
  {
    accessorKey: 'protocol',
    header: 'Protocol',
    cell: ({ row }) => (
      <span className="uppercase text-xs font-medium">
        {row.original.protocol}
      </span>
    ),
  },
  {
    accessorKey: 'access',
    header: 'Access',
    cell: ({ row }) => (
      <div className="flex items-center gap-2">
        {row.original.guestOk ? (
          <>
            <Globe className="h-3 w-3" />
            <span className="text-sm">Public</span>
          </>
        ) : (
          <>
            <Lock className="h-3 w-3" />
            <span className="text-sm">Private</span>
          </>
        )}
      </div>
    ),
  },
  {
    accessorKey: 'users',
    header: 'Users',
    cell: ({ row }) => {
      const users = row.original.users || []
      const groups = row.original.groups || []
      const total = users.length + groups.length
      
      if (row.original.guestOk) {
        return <span className="text-sm text-muted-foreground">Everyone</span>
      }
      
      if (total === 0) {
        return <span className="text-sm text-muted-foreground">None</span>
      }
      
      return (
        <div className="flex items-center gap-2">
          <Users className="h-3 w-3" />
          <span className="text-sm">
            {users.length} user{users.length !== 1 ? 's' : ''}
            {groups.length > 0 && `, ${groups.length} group${groups.length !== 1 ? 's' : ''}`}
          </span>
        </div>
      )
    },
  },
  {
    accessorKey: 'status',
    header: 'Status',
    cell: ({ row }) => (
      <StatusPill 
        status={row.original.enabled ? 'active' : 'inactive'} 
        size="sm" 
      />
    ),
  },
  {
    accessorKey: 'actions',
    header: '',
    cell: ({ row }) => (
      <div className="flex items-center gap-1">
        <Button 
          variant="ghost" 
          size="icon"
          onClick={() => window.dispatchEvent(new CustomEvent('editShare', { detail: row.original }))}
        >
          <Edit className="h-3 w-3" />
        </Button>
        <Button 
          variant="ghost" 
          size="icon"
          onClick={() => window.dispatchEvent(new CustomEvent('deleteShare', { detail: row.original }))}
        >
          <Trash2 className="h-3 w-3" />
        </Button>
      </div>
    ),
  },
]

export function Shares() {
  const [showCreateForm, setShowCreateForm] = useState(false)
  const [editingShare, setEditingShare] = useState<Share | null>(null)
  const [isRefreshing, setIsRefreshing] = useState(false)
  
  // Check API status
  const { data: apiStatus } = useApiStatus()
  
  // Fetch shares from API
  const { data: shares, isLoading, refetch } = useShares()
  const createShare = useCreateShare()
  const updateShare = useUpdateShare()
  const deleteShare = useDeleteShare()

  const handleRefresh = async () => {
    setIsRefreshing(true)
    await refetch()
    setTimeout(() => setIsRefreshing(false), 500)
  }

  const handleCreateShare = async (data: Partial<Share>) => {
    try {
      await createShare.mutateAsync(data)
      toast.success('Share created successfully')
      setShowCreateForm(false)
    } catch (error: any) {
      toast.error(error.message || 'Failed to create share')
    }
  }

  const handleUpdateShare = async (data: Partial<Share>) => {
    if (!editingShare) return
    
    try {
      await updateShare.mutateAsync({ name: editingShare.name, share: data })
      toast.success('Share updated successfully')
      setEditingShare(null)
    } catch (error: any) {
      toast.error(error.message || 'Failed to update share')
    }
  }

  const handleDeleteShare = async (share: Share) => {
    if (!confirm(`Delete share "${share.name}"? This action cannot be undone.`)) {
      return
    }
    
    try {
      await deleteShare.mutateAsync(share.name)
      toast.success('Share deleted successfully')
    } catch (error: any) {
      toast.error(error.message || 'Failed to delete share')
    }
  }

  // Listen for edit/delete events from table actions
  useState(() => {
    const handleEdit = (e: any) => setEditingShare(e.detail)
    const handleDelete = (e: any) => handleDeleteShare(e.detail)
    
    window.addEventListener('editShare', handleEdit)
    window.addEventListener('deleteShare', handleDelete)
    
    return () => {
      window.removeEventListener('editShare', handleEdit)
      window.removeEventListener('deleteShare', handleDelete)
    }
  })

  // Show backend error if API is unreachable
  if (apiStatus && !apiStatus.isReachable) {
    return (
      <div className="space-y-6">
        <PageHeader
          title="Network Shares"
          description="Manage SMB, NFS, and AFP network shares"
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

  return (
    <div className="space-y-6">
      <PageHeader
        title="Network Shares"
        description="Manage SMB, NFS, and AFP network shares"
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
            <Button size="sm" onClick={() => setShowCreateForm(true)}>
              <Plus className="h-4 w-4 mr-2" />
              Create Share
            </Button>
          </>
        }
      />

      {/* Stats Cards */}
      {shares && shares.length > 0 && (
        <div className="grid gap-4 md:grid-cols-4">
          <Card className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Total Shares</p>
                <p className="text-2xl font-bold">{shares.length}</p>
              </div>
              <Share2 className="h-8 w-8 text-muted-foreground" />
            </div>
          </Card>
          
          <Card className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Active</p>
                <p className="text-2xl font-bold">
                  {shares.filter(s => s.enabled).length}
                </p>
              </div>
              <ToggleRight className="h-8 w-8 text-green-500" />
            </div>
          </Card>
          
          <Card className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Public</p>
                <p className="text-2xl font-bold">
                  {shares.filter(s => s.guestOk).length}
                </p>
              </div>
              <Globe className="h-8 w-8 text-muted-foreground" />
            </div>
          </Card>
          
          <Card className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Protocols</p>
                <div className="flex gap-2 mt-1">
                  {['smb', 'nfs', 'afp'].map(protocol => {
                    const count = shares.filter(s => s.protocol === protocol).length
                    if (count === 0) return null
                    return (
                      <span key={protocol} className="text-xs uppercase font-medium">
                        {protocol}: {count}
                      </span>
                    )
                  })}
                </div>
              </div>
              <Settings className="h-8 w-8 text-muted-foreground" />
            </div>
          </Card>
        </div>
      )}

      {/* Shares Table */}
      <Card
        title="Network Shares"
        subtitle={`${shares?.length || 0} configured`}
        isLoading={isLoading}
      >
        {shares && shares.length > 0 ? (
          <DataTable 
            columns={shareColumns} 
            data={shares}
            searchKey="name"
            searchPlaceholder="Search shares..."
          />
        ) : (
          <EmptyState
            icon={Share2}
            title="No shares configured"
            description="Create your first network share to start sharing files"
            action={
              <Button onClick={() => setShowCreateForm(true)}>
                <Plus className="h-4 w-4 mr-2" />
                Create Share
              </Button>
            }
          />
        )}
      </Card>

      {/* Create Share Slide-over */}
      <SlideOver
        open={showCreateForm}
        onOpenChange={setShowCreateForm}
        title="Create Network Share"
        description="Configure a new SMB, NFS, or AFP share"
      >
        <ShareForm
          onSubmit={handleCreateShare}
          onCancel={() => setShowCreateForm(false)}
        />
      </SlideOver>

      {/* Edit Share Slide-over */}
      <SlideOver
        open={!!editingShare}
        onOpenChange={(open: boolean) => !open && setEditingShare(null)}
        title="Edit Network Share"
        description={`Modify settings for ${editingShare?.name}`}
      >
        {editingShare && (
          <ShareForm
            share={editingShare}
            onSubmit={handleUpdateShare}
            onCancel={() => setEditingShare(null)}
          />
        )}
      </SlideOver>
    </div>
  )
}