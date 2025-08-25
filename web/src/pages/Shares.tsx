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
  X
} from 'lucide-react'
import { ColumnDef } from '@tanstack/react-table'
import { PageHeader } from '@/components/ui/page-header'
import { Card } from '@/components/ui/card-enhanced'
import { EmptyState } from '@/components/ui/empty-state'
import { StatusPill } from '@/components/ui/status'
import { Button } from '@/components/ui/button'
import { DataTable } from '@/components/ui/data-table'
import { SlideOver } from '@/components/ui/slide-over'
import { useShares, useCreateShare, useUpdateShare, useDeleteShare } from '@/hooks/use-api'
import { cn } from '@/lib/utils'
import { pushToast } from '@/components/ui/toast'
import type { Share } from '@/lib/api-client'

// Mock data for development
const mockShares: Share[] = [
  { id: '1', name: 'Documents', protocol: 'smb', path: '/mnt/main/documents', access: 'private', status: 'active', users: ['admin', 'john'] },
  { id: '2', name: 'Media', protocol: 'smb', path: '/mnt/main/media', access: 'public', status: 'active' },
  { id: '3', name: 'Backups', protocol: 'nfs', path: '/mnt/backup/data', access: 'private', status: 'inactive', users: ['admin'] },
  { id: '4', name: 'Public', protocol: 'smb', path: '/mnt/main/public', access: 'public', status: 'active' },
]

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
    access: share?.access || 'private',
    users: share?.users || [],
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
        <div className="grid grid-cols-2 gap-2">
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
            <Share2 className="h-4 w-4" />
            NFS
          </button>
        </div>
      </div>

      {/* Path */}
      <div>
        <label className="block text-sm font-medium mb-2">Path</label>
        <div className="flex gap-2">
          <input
            type="text"
            value={formData.path}
            onChange={(e) => setFormData({ ...formData, path: e.target.value })}
            className="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring font-mono"
            placeholder="/mnt/main/folder"
            required
          />
          <Button type="button" variant="outline" size="sm">
            <FolderOpen className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Access */}
      <div>
        <label className="block text-sm font-medium mb-2">Access Control</label>
        <div className="grid grid-cols-2 gap-2">
          <button
            type="button"
            onClick={() => setFormData({ ...formData, access: 'public' })}
            className={cn(
              "flex items-center justify-center gap-2 p-3 rounded-lg border transition-colors",
              formData.access === 'public'
                ? "border-primary bg-primary/10 text-primary"
                : "border-border hover:bg-muted/50"
            )}
          >
            <Globe className="h-4 w-4" />
            Public
          </button>
          <button
            type="button"
            onClick={() => setFormData({ ...formData, access: 'private' })}
            className={cn(
              "flex items-center justify-center gap-2 p-3 rounded-lg border transition-colors",
              formData.access === 'private'
                ? "border-primary bg-primary/10 text-primary"
                : "border-border hover:bg-muted/50"
            )}
          >
            <Lock className="h-4 w-4" />
            Private
          </button>
        </div>
      </div>

      {/* Users (if private) */}
      {formData.access === 'private' && (
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
          <p className="mt-1 text-xs text-muted-foreground">
            Comma-separated list of usernames
          </p>
        </div>
      )}

      {/* Advanced settings */}
      <details className="border-t pt-4">
        <summary className="cursor-pointer text-sm font-medium flex items-center gap-2">
          <Settings className="h-4 w-4" />
          Advanced Settings
        </summary>
        <div className="mt-4 space-y-4">
          <label className="flex items-center gap-2">
            <input type="checkbox" className="rounded" />
            <span className="text-sm">Enable recycle bin</span>
          </label>
          <label className="flex items-center gap-2">
            <input type="checkbox" className="rounded" />
            <span className="text-sm">Allow guest access</span>
          </label>
          <label className="flex items-center gap-2">
            <input type="checkbox" className="rounded" />
            <span className="text-sm">Hide dot files</span>
          </label>
        </div>
      </details>

      {/* Actions */}
      <div className="flex justify-end gap-2 pt-4 border-t">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit">
          {share ? 'Update Share' : 'Create Share'}
        </Button>
      </div>
    </form>
  )
}

// Share columns
const createShareColumns = (
  onEdit: (share: Share) => void,
  onDelete: (share: Share) => void,
  onToggle: (share: Share) => void,
  onCopyPath: (path: string) => void
): ColumnDef<Share>[] => [
  {
    accessorKey: 'name',
    header: 'Name',
    cell: ({ row }) => (
      <div className="flex items-center gap-2">
        <Share2 className="h-4 w-4 text-muted-foreground" />
        <span className="font-medium">{row.original.name}</span>
      </div>
    ),
  },
  {
    accessorKey: 'protocol',
    header: 'Protocol',
    cell: ({ row }) => (
      <StatusPill variant="info">
        {row.original.protocol.toUpperCase()}
      </StatusPill>
    ),
  },
  {
    accessorKey: 'path',
    header: 'Path',
    cell: ({ row }) => (
      <div className="flex items-center gap-1">
        <code className="text-xs bg-muted px-2 py-1 rounded font-mono">
          {row.original.path}
        </code>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onCopyPath(row.original.path)}
        >
          <Copy className="h-3 w-3" />
        </Button>
      </div>
    ),
  },
  {
    accessorKey: 'access',
    header: 'Access',
    cell: ({ row }) => (
      <div className="flex items-center gap-1">
        {row.original.access === 'public' ? (
          <>
            <Globe className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm">Public</span>
          </>
        ) : (
          <>
            <Lock className="h-4 w-4 text-muted-foreground" />
            <span className="text-sm">Private</span>
            {row.original.users && (
              <span className="ml-1 text-xs text-muted-foreground">
                ({row.original.users.length} users)
              </span>
            )}
          </>
        )}
      </div>
    ),
  },
  {
    accessorKey: 'status',
    header: 'Status',
    cell: ({ row }) => {
      const isActive = row.original.status === 'active'
      return (
        <button
          onClick={() => onToggle(row.original)}
          className="flex items-center gap-2 hover:opacity-80 transition-opacity"
        >
          {isActive ? (
            <ToggleRight className="h-5 w-5 text-green-500" />
          ) : (
            <ToggleLeft className="h-5 w-5 text-muted-foreground" />
          )}
          <StatusPill variant={isActive ? 'success' : 'muted'}>
            {row.original.status}
          </StatusPill>
        </button>
      )
    },
  },
  {
    id: 'actions',
    header: 'Actions',
    cell: ({ row }) => (
      <div className="flex items-center gap-1">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onEdit(row.original)}
        >
          <Edit className="h-4 w-4" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          onClick={() => onDelete(row.original)}
          className="text-destructive hover:text-destructive"
        >
          <Trash2 className="h-4 w-4" />
        </Button>
      </div>
    ),
  },
]

export function Shares() {
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [editingShare, setEditingShare] = useState<Share | null>(null)
  const [selectedShare, setSelectedShare] = useState<Share | null>(null)
  
  const { data: shares, isLoading } = useShares()
  const createMutation = useCreateShare()
  const updateMutation = useUpdateShare()
  const deleteMutation = useDeleteShare()

  // Use mock data if API fails
  const shareData = shares || mockShares

  const handleCreate = async (data: Partial<Share>) => {
    try {
      await createMutation.mutateAsync(data)
      pushToast('Share created successfully', 'success')
      setIsCreateOpen(false)
    } catch (error) {
      pushToast('Failed to create share', 'error')
    }
  }

  const handleUpdate = async (data: Partial<Share>) => {
    if (!editingShare) return
    try {
      await updateMutation.mutateAsync({ id: editingShare.id, data })
      pushToast('Share updated successfully', 'success')
      setEditingShare(null)
    } catch (error) {
      pushToast('Failed to update share', 'error')
    }
  }

  const handleDelete = async (share: Share) => {
    if (!confirm(`Delete share "${share.name}"?`)) return
    try {
      await deleteMutation.mutateAsync(share.id)
      pushToast('Share deleted successfully', 'success')
    } catch (error) {
      pushToast('Failed to delete share', 'error')
    }
  }

  const handleToggle = async (share: Share) => {
    const newStatus = share.status === 'active' ? 'inactive' : 'active'
    try {
      await updateMutation.mutateAsync({ 
        id: share.id, 
        data: { status: newStatus } 
      })
      pushToast(`Share ${newStatus === 'active' ? 'enabled' : 'disabled'}`, 'success')
    } catch (error) {
      pushToast('Failed to toggle share', 'error')
    }
  }

  const handleCopyPath = (path: string) => {
    navigator.clipboard.writeText(path)
    pushToast('Path copied to clipboard', 'success')
  }

  const columns = createShareColumns(
    setEditingShare,
    handleDelete,
    handleToggle,
    handleCopyPath
  )

  return (
    <div className="space-y-6">
      <PageHeader
        title="Shares"
        description="Network shares and file access configuration"
        actions={
          <Button size="sm" onClick={() => setIsCreateOpen(true)}>
            <Plus className="h-4 w-4 mr-2" />
            New Share
          </Button>
        }
      />

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Main table */}
        <div className="lg:col-span-2">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
          >
            <Card
              title="Network Shares"
              description="Configure SMB and NFS shares"
              isLoading={isLoading}
            >
              {shareData.length > 0 ? (
                <DataTable
                  columns={columns}
                  data={shareData}
                  searchKey="shares"
                  onRowClick={setSelectedShare}
                />
              ) : (
                <EmptyState
                  variant="no-data"
                  icon={Share2}
                  title="No shares configured"
                  description="Create your first network share to start sharing files"
                  action={{
                    label: "Create Share",
                    onClick: () => setIsCreateOpen(true)
                  }}
                />
              )}
            </Card>
          </motion.div>
        </div>

        {/* Sidebar - Share details / connections */}
        <div className="lg:col-span-1">
          <motion.div
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.3, delay: 0.1 }}
          >
            {selectedShare ? (
              <Card
                title="Share Details"
                description={selectedShare.name}
                actions={
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setSelectedShare(null)}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                }
              >
                <div className="space-y-4">
                  {/* Connection info */}
                  <div>
                    <h4 className="text-sm font-medium mb-2">Connection Info</h4>
                    <div className="space-y-2">
                      <div className="p-3 bg-muted/30 rounded-lg">
                        <p className="text-xs text-muted-foreground mb-1">
                          {selectedShare.protocol === 'smb' ? 'Windows (SMB)' : 'Linux/Mac (NFS)'}
                        </p>
                        <code className="text-xs block bg-background p-2 rounded border">
                          {selectedShare.protocol === 'smb' 
                            ? `\\\\nithron.os\\${selectedShare.name}`
                            : `nithron.os:${selectedShare.path}`
                          }
                        </code>
                      </div>
                    </div>
                  </div>

                  {/* Active connections */}
                  <div>
                    <h4 className="text-sm font-medium mb-2">Active Connections</h4>
                    <div className="space-y-2">
                      <div className="flex items-center justify-between p-2 rounded-lg bg-muted/30">
                        <div className="flex items-center gap-2">
                          <Users className="h-4 w-4 text-muted-foreground" />
                          <span className="text-sm">john@192.168.1.100</span>
                        </div>
                        <span className="text-xs text-muted-foreground">2h ago</span>
                      </div>
                      <div className="flex items-center justify-between p-2 rounded-lg bg-muted/30">
                        <div className="flex items-center gap-2">
                          <Users className="h-4 w-4 text-muted-foreground" />
                          <span className="text-sm">admin@192.168.1.105</span>
                        </div>
                        <span className="text-xs text-muted-foreground">5m ago</span>
                      </div>
                    </div>
                  </div>

                  {/* Mount commands */}
                  <div>
                    <h4 className="text-sm font-medium mb-2 flex items-center gap-2">
                      <Terminal className="h-4 w-4" />
                      Mount Commands
                    </h4>
                    <div className="space-y-2">
                      <div className="p-2 bg-muted/30 rounded-lg">
                        <p className="text-xs text-muted-foreground mb-1">Linux</p>
                        <code className="text-xs block">
                          {selectedShare.protocol === 'smb'
                            ? `sudo mount -t cifs //nithron.os/${selectedShare.name} /mnt/share`
                            : `sudo mount -t nfs nithron.os:${selectedShare.path} /mnt/share`
                          }
                        </code>
                      </div>
                    </div>
                  </div>
                </div>
              </Card>
            ) : (
              <Card
                title="Quick Tips"
              >
                <div className="space-y-3">
                  <div className="flex gap-3">
                    <Shield className="h-5 w-5 text-muted-foreground shrink-0 mt-0.5" />
                    <div>
                      <p className="text-sm font-medium">Security Best Practice</p>
                      <p className="text-xs text-muted-foreground mt-1">
                        Use private shares with specific user access for sensitive data.
                      </p>
                    </div>
                  </div>
                  <div className="flex gap-3">
                    <Globe className="h-5 w-5 text-muted-foreground shrink-0 mt-0.5" />
                    <div>
                      <p className="text-sm font-medium">Public Shares</p>
                      <p className="text-xs text-muted-foreground mt-1">
                        Public shares allow guest access without authentication.
                      </p>
                    </div>
                  </div>
                  <div className="flex gap-3">
                    <Share2 className="h-5 w-5 text-muted-foreground shrink-0 mt-0.5" />
                    <div>
                      <p className="text-sm font-medium">Protocol Choice</p>
                      <p className="text-xs text-muted-foreground mt-1">
                        Use SMB for Windows clients, NFS for Linux/Unix systems.
                      </p>
                    </div>
                  </div>
                </div>
              </Card>
            )}
          </motion.div>
        </div>
      </div>

      {/* Create/Edit Share SlideOver */}
      <SlideOver
        isOpen={isCreateOpen || !!editingShare}
        onClose={() => {
          setIsCreateOpen(false)
          setEditingShare(null)
        }}
        title={editingShare ? 'Edit Share' : 'Create New Share'}
        description={editingShare ? `Editing ${editingShare.name}` : 'Configure a new network share'}
        size="md"
      >
        <ShareForm
          share={editingShare}
          onSubmit={editingShare ? handleUpdate : handleCreate}
          onCancel={() => {
            setIsCreateOpen(false)
            setEditingShare(null)
          }}
        />
      </SlideOver>
    </div>
  )
}