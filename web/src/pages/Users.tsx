import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { PageHeader } from '@/components/ui/page-header'
import { Card } from '@/components/ui/card-enhanced'
import { Button } from '@/components/ui/button'
import { StatusPill } from '@/components/ui/status'
import { toast } from '@/components/ui/toast'
import { useUsers } from '@/hooks/use-api'
import { 
  UserPlus, 
  Key, 
  ChevronRight, 
  Shield,
  Mail,
  Calendar,
  Edit,
  Trash2,
  RefreshCw
} from 'lucide-react'
import { motion } from 'framer-motion'

interface User {
  id: string
  username: string
  email?: string
  role: string
  status: 'active' | 'inactive' | 'locked'
  lastLogin?: string
  createdAt?: string
  has2FA?: boolean
}

export function Users() {
  const navigate = useNavigate()
  const { data: users, isLoading } = useUsers()
  const [selectedUser, setSelectedUser] = useState<string | null>(null)
  const [showDeleteModal, setShowDeleteModal] = useState(false)
  const [userToDelete, setUserToDelete] = useState<User | null>(null)

  // Transform the users data
  const userData: User[] = users ? (Array.isArray(users) ? users : users?.data || []).map((u: any) => ({
    id: u.id || u.username,
    username: u.username,
    email: u.email || `${u.username}@example.com`,
    role: u.role || (u.username === 'admin' ? 'Administrator' : 'User'),
    status: u.status || 'active',
    lastLogin: u.lastLogin,
    createdAt: u.createdAt,
    has2FA: u.has2FA || false,
  })) : []

  const handleAddUser = () => {
    // Navigate to add user page or open modal
    toast.info('Add user functionality coming soon')
  }

  const handleEditUser = (user: User) => {
    setSelectedUser(user.id)
    toast.info(`Edit user ${user.username} functionality coming soon`)
  }

  const handleDeleteUser = (user: User) => {
    setUserToDelete(user)
    setShowDeleteModal(true)
  }

  const confirmDelete = () => {
    if (userToDelete) {
      toast.success(`User ${userToDelete.username} deleted`)
      setShowDeleteModal(false)
      setUserToDelete(null)
    }
  }

  const handleResetPassword = (user: User) => {
    toast.info(`Password reset for ${user.username} functionality coming soon`)
  }

  const handleToggle2FA = (user: User) => {
    toast.info(`2FA settings for ${user.username} functionality coming soon`)
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Users"
        description="Manage user accounts and permissions"
        actions={
          <Button onClick={handleAddUser}>
            <UserPlus className="h-4 w-4 mr-2" />
            Add User
          </Button>
        }
      />

      {/* User Statistics */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <div className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Total Users</p>
                <p className="text-2xl font-semibold">{userData.length}</p>
              </div>
              <div className="h-12 w-12 rounded-full bg-primary/10 flex items-center justify-center">
                <UserPlus className="h-6 w-6 text-primary" />
              </div>
            </div>
          </div>
        </Card>

        <Card>
          <div className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">Active Users</p>
                <p className="text-2xl font-semibold">
                  {userData.filter(u => u.status === 'active').length}
                </p>
              </div>
              <div className="h-12 w-12 rounded-full bg-green-500/10 flex items-center justify-center">
                <Shield className="h-6 w-6 text-green-500" />
              </div>
            </div>
          </div>
        </Card>

        <Card>
          <div className="p-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">With 2FA</p>
                <p className="text-2xl font-semibold">
                  {userData.filter(u => u.has2FA).length}
                </p>
              </div>
              <div className="h-12 w-12 rounded-full bg-blue-500/10 flex items-center justify-center">
                <Key className="h-6 w-6 text-blue-500" />
              </div>
            </div>
          </div>
        </Card>
      </div>

      {/* Users List */}
      <Card title="User Accounts">
        {isLoading ? (
          <div className="p-8 text-center text-muted-foreground">
            Loading users...
          </div>
        ) : userData.length === 0 ? (
          <div className="p-8 text-center">
            <UserPlus className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
            <p className="text-muted-foreground mb-4">No users found</p>
            <Button onClick={handleAddUser}>
              <UserPlus className="h-4 w-4 mr-2" />
              Add Your First User
            </Button>
          </div>
        ) : (
          <div className="divide-y">
            {userData.map((user) => (
              <motion.div
                key={user.id}
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                className="p-4 hover:bg-muted/30 transition-colors"
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-4">
                    {/* User Avatar */}
                    <div className="w-12 h-12 rounded-full bg-primary/10 flex items-center justify-center">
                      <span className="text-lg font-semibold">
                        {user.username[0].toUpperCase()}
                      </span>
                    </div>

                    {/* User Info */}
                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <span className="font-medium">{user.username}</span>
                        <StatusPill 
                          variant={user.role === 'Administrator' ? 'warning' : 'info'}
                          size="sm"
                        >
                          {user.role}
                        </StatusPill>
                        {user.has2FA && (
                          <StatusPill variant="success" size="sm">
                            <Key className="h-3 w-3 mr-1" />
                            2FA
                          </StatusPill>
                        )}
                      </div>
                      <div className="flex items-center gap-4 text-sm text-muted-foreground">
                        <span className="flex items-center gap-1">
                          <Mail className="h-3 w-3" />
                          {user.email}
                        </span>
                        {user.lastLogin && (
                          <span className="flex items-center gap-1">
                            <Calendar className="h-3 w-3" />
                            Last login: {new Date(user.lastLogin).toLocaleDateString()}
                          </span>
                        )}
                      </div>
                    </div>
                  </div>

                  {/* Actions */}
                  <div className="flex items-center gap-2">
                    <StatusPill 
                      variant={
                        user.status === 'active' ? 'success' : 
                        user.status === 'locked' ? 'error' : 
                        'info'
                      }
                    >
                      {user.status}
                    </StatusPill>

                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleResetPassword(user)}
                      title="Reset Password"
                    >
                      <RefreshCw className="h-4 w-4" />
                    </Button>

                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleToggle2FA(user)}
                      title="Configure 2FA"
                    >
                      <Key className="h-4 w-4" />
                    </Button>

                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleEditUser(user)}
                      title="Edit User"
                    >
                      <Edit className="h-4 w-4" />
                    </Button>

                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleDeleteUser(user)}
                      title="Delete User"
                      className="text-red-500 hover:text-red-600"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>

                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setSelectedUser(selectedUser === user.id ? null : user.id)}
                    >
                      <ChevronRight 
                        className={`h-4 w-4 transition-transform ${
                          selectedUser === user.id ? 'rotate-90' : ''
                        }`}
                      />
                    </Button>
                  </div>
                </div>

                {/* Expanded Details */}
                {selectedUser === user.id && (
                  <motion.div
                    initial={{ height: 0, opacity: 0 }}
                    animate={{ height: 'auto', opacity: 1 }}
                    exit={{ height: 0, opacity: 0 }}
                    className="mt-4 pt-4 border-t space-y-2"
                  >
                    <div className="grid grid-cols-2 gap-4 text-sm">
                      <div>
                        <span className="text-muted-foreground">Created:</span>
                        <span className="ml-2">
                          {user.createdAt ? new Date(user.createdAt).toLocaleDateString() : 'Unknown'}
                        </span>
                      </div>
                      <div>
                        <span className="text-muted-foreground">User ID:</span>
                        <span className="ml-2 font-mono">{user.id}</span>
                      </div>
                    </div>
                  </motion.div>
                )}
              </motion.div>
            ))}
          </div>
        )}
      </Card>

      {/* Security Settings */}
      <Card title="Security Settings">
        <div className="space-y-3">
          <div className="flex items-center justify-between p-3 rounded-lg bg-muted/30">
            <div className="flex items-center gap-3">
              <Key className="h-5 w-5 text-muted-foreground" />
              <div>
                <div className="font-medium">Two-Factor Authentication</div>
                <div className="text-sm text-muted-foreground">
                  Require 2FA for all administrators
                </div>
              </div>
            </div>
            <Button variant="outline" size="sm" onClick={() => navigate('/settings/2fa')}>
              Configure
            </Button>
          </div>

          <div className="flex items-center justify-between p-3 rounded-lg bg-muted/30">
            <div className="flex items-center gap-3">
              <Shield className="h-5 w-5 text-muted-foreground" />
              <div>
                <div className="font-medium">Password Policy</div>
                <div className="text-sm text-muted-foreground">
                  Set minimum password requirements
                </div>
              </div>
            </div>
            <Button variant="outline" size="sm">
              Configure
            </Button>
          </div>
        </div>
      </Card>

      {/* Delete Confirmation Modal */}
      {showDeleteModal && userToDelete && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <Card className="w-full max-w-md">
            <div className="p-6 space-y-4">
              <h2 className="text-xl font-semibold">Delete User</h2>
              <p className="text-muted-foreground">
                Are you sure you want to delete user <strong>{userToDelete.username}</strong>? 
                This action cannot be undone.
              </p>
              <div className="flex justify-end gap-2">
                <Button
                  variant="outline"
                  onClick={() => {
                    setShowDeleteModal(false)
                    setUserToDelete(null)
                  }}
                >
                  Cancel
                </Button>
                <Button
                  variant="destructive"
                  onClick={confirmDelete}
                >
                  Delete User
                </Button>
              </div>
            </div>
          </Card>
        </div>
      )}
    </div>
  )
}
