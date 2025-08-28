import { useState } from 'react'
import { motion } from 'framer-motion'
import { 
  Settings as SettingsIcon,
  Network,
  Users,
  Calendar,
  Download,
  Info,
  ChevronRight,
  Wifi,
  Globe,
  Key,
  UserPlus,
  Clock,
  Package,
  Server,
  HardDrive,
  Save
} from 'lucide-react'
import { PageHeader } from '@/components/ui/page-header'
import { Card } from '@/components/ui/card-enhanced'
import { Button } from '@/components/ui/button'
import { StatusPill } from '@/components/ui/status'
import { cn } from '@/lib/utils'
import { toast } from '@/components/ui/toast'
import { useUsers } from '@/hooks/use-api'

// Settings sections
const settingsSections = [
  { id: 'general', label: 'General', icon: SettingsIcon },
  { id: 'network', label: 'Network', icon: Network },
  { id: 'users', label: 'Users', icon: Users },
  { id: 'schedules', label: 'Schedules', icon: Calendar },
  { id: 'updates', label: 'Updates', icon: Download },
  { id: 'about', label: 'About', icon: Info },
]

// General Settings
function GeneralSettings() {
  const [settings, setSettings] = useState({
    hostname: 'nithronos',
    timezone: 'America/New_York',
    language: 'en-US',
    notifications: true
  })

  return (
    <div className="space-y-6">
      <Card title="System Configuration">
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">Hostname</label>
            <input
              type="text"
              value={settings.hostname}
              onChange={(e) => setSettings({ ...settings, hostname: e.target.value })}
              className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            />
          </div>
          
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-2">Timezone</label>
              <select
                value={settings.timezone}
                onChange={(e) => setSettings({ ...settings, timezone: e.target.value })}
                className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              >
                <option value="America/New_York">America/New York</option>
                <option value="America/Los_Angeles">America/Los Angeles</option>
                <option value="Europe/London">Europe/London</option>
              </select>
            </div>
            
            <div>
              <label className="block text-sm font-medium mb-2">Language</label>
              <select
                value={settings.language}
                onChange={(e) => setSettings({ ...settings, language: e.target.value })}
                className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              >
                <option value="en-US">English (US)</option>
                <option value="es">Spanish</option>
                <option value="fr">French</option>
              </select>
            </div>
          </div>
        </div>
      </Card>

      <div className="flex justify-end">
        <Button onClick={() => toast.success('Settings saved')}>
          <Save className="h-4 w-4 mr-2" />
          Save Changes
        </Button>
      </div>
    </div>
  )
}

// Network Settings
function NetworkSettings() {
  return (
    <div className="space-y-6">
      <Card title="Network Configuration">
        <div className="space-y-4">
          <label className="flex items-center gap-2">
            <input type="checkbox" defaultChecked className="rounded" />
            <span className="text-sm font-medium">Use DHCP</span>
          </label>

          <div>
            <label className="block text-sm font-medium mb-2">IP Address</label>
            <input
              type="text"
              defaultValue="192.168.1.100"
              disabled
              className="w-full rounded-md border border-input bg-muted px-3 py-2 text-sm font-mono"
            />
          </div>
        </div>
      </Card>

      <Card title="Connection Status">
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Wifi className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm">Network Status</span>
            </div>
            <StatusPill variant="success">Connected</StatusPill>
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Globe className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm">Internet Access</span>
            </div>
            <StatusPill variant="success">Available</StatusPill>
          </div>
        </div>
      </Card>
    </div>
  )
}

// Users Settings
function UsersSettings() {
  const { data: users } = useUsers()
  const usersData = users || [
    { id: '1', username: 'admin', role: 'Administrator', email: 'admin@example.com', status: 'active' },
    { id: '2', username: 'john', role: 'User', email: 'john@example.com', status: 'active' },
  ]
  const mockUsers = Array.isArray(usersData) ? usersData : usersData?.data || []

  return (
    <div className="space-y-6">
      <Card 
        title="User Accounts"
        actions={
          <Button size="sm">
            <UserPlus className="h-4 w-4 mr-2" />
            Add User
          </Button>
        }
      >
        <div className="space-y-2">
          {mockUsers.map((user: any) => (
            <div key={user.id} className="flex items-center justify-between p-3 rounded-lg border">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-full bg-primary/10 flex items-center justify-center">
                  <span className="text-sm font-medium">{user.username[0].toUpperCase()}</span>
                </div>
                <div>
                  <div className="font-medium">{user.username}</div>
                  <div className="text-sm text-muted-foreground">{(user as any).email || user.username + '@example.com'}</div>
                </div>
              </div>
              <div className="flex items-center gap-4">
                <StatusPill variant={user.role === 'Administrator' ? 'warning' : 'info'}>
                  {user.role}
                </StatusPill>
                <Button variant="ghost" size="sm">
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </div>
            </div>
          ))}
        </div>
      </Card>

      <Card title="Security">
        <div className="flex items-center justify-between p-3 rounded-lg bg-muted/30">
          <div className="flex items-center gap-3">
            <Key className="h-5 w-5 text-muted-foreground" />
            <div>
              <div className="font-medium">Two-Factor Authentication</div>
              <div className="text-sm text-muted-foreground">Require 2FA for administrators</div>
            </div>
          </div>
          <Button variant="outline" size="sm">Configure</Button>
        </div>
      </Card>
    </div>
  )
}

// Schedules Settings
function SchedulesSettings() {
  const schedules = [
    { id: '1', name: 'Daily Backup', schedule: '0 2 * * *', enabled: true },
    { id: '2', name: 'Weekly Scrub', schedule: '0 3 * * 0', enabled: true },
  ]

  return (
    <Card title="Scheduled Tasks">
      <div className="space-y-2">
        {schedules.map(schedule => (
          <div key={schedule.id} className="flex items-center justify-between p-3 rounded-lg border">
            <div className="flex items-center gap-3">
              <Clock className="h-5 w-5 text-muted-foreground" />
              <div>
                <div className="font-medium">{schedule.name}</div>
                <div className="text-sm text-muted-foreground font-mono">{schedule.schedule}</div>
              </div>
            </div>
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                defaultChecked={schedule.enabled}
                className="rounded"
              />
              <span className="text-sm">Enabled</span>
            </label>
          </div>
        ))}
      </div>
    </Card>
  )
}

// Updates Settings
function UpdatesSettings() {
  return (
    <div className="space-y-6">
      <Card title="Update Settings">
        <label className="flex items-center justify-between p-3 rounded-lg hover:bg-muted/50">
          <div className="flex items-center gap-3">
            <Download className="h-5 w-5 text-muted-foreground" />
            <div>
              <div className="font-medium">Automatic Updates</div>
              <div className="text-sm text-muted-foreground">Install security updates automatically</div>
            </div>
          </div>
          <input type="checkbox" defaultChecked className="rounded" />
        </label>
      </Card>

      <Card title="Available Updates">
        <div className="flex items-center justify-between p-3 rounded-lg border">
          <div className="flex items-center gap-3">
            <Package className="h-5 w-5 text-blue-500" />
            <div>
              <div className="font-medium">NithronOS Core</div>
              <div className="text-sm text-muted-foreground">v2.1.0 → v2.2.0</div>
            </div>
          </div>
          <Button size="sm">Install</Button>
        </div>
      </Card>
    </div>
  )
}

// About Section
function AboutSection() {
  return (
    <div className="space-y-6">
      <Card title="System Information">
        <div className="grid grid-cols-2 gap-4">
          <div>
            <p className="text-sm text-muted-foreground">Version</p>
            <p className="font-medium">NithronOS v2.1.0</p>
          </div>
          <div>
            <p className="text-sm text-muted-foreground">Kernel</p>
            <p className="font-medium">Linux 6.1.0</p>
          </div>
        </div>
      </Card>

      <Card title="Hardware">
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Server className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm">CPU</span>
            </div>
            <span className="text-sm font-medium">Intel Xeon E-2288G</span>
          </div>
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <HardDrive className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm">Memory</span>
            </div>
            <span className="text-sm font-medium">32 GB DDR4</span>
          </div>
        </div>
      </Card>
    </div>
  )
}

export function Settings() {
  const [activeSection, setActiveSection] = useState('general')

  const renderSection = () => {
    switch (activeSection) {
      case 'general': return <GeneralSettings />
      case 'network': return <NetworkSettings />
      case 'users': return <UsersSettings />
      case 'schedules': return <SchedulesSettings />
      case 'updates': return <UpdatesSettings />
      case 'about': return <AboutSection />
      default: return <GeneralSettings />
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Settings"
        description="Configure system settings and preferences"
      />

      <div className="flex gap-6">
        {/* Sidebar */}
        <div className="w-64 shrink-0">
          <Card>
            <nav className="space-y-1">
              {settingsSections.map(section => (
                <button
                  key={section.id}
                  onClick={() => setActiveSection(section.id)}
                  className={cn(
                    "w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm transition-colors text-left",
                    activeSection === section.id
                      ? "bg-primary text-primary-foreground"
                      : "hover:bg-muted/50"
                  )}
                >
                  <section.icon className="h-4 w-4" />
                  <span className="font-medium">{section.label}</span>
                </button>
              ))}
            </nav>
          </Card>
        </div>

        {/* Content */}
        <div className="flex-1">
          <motion.div
            key={activeSection}
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.2 }}
          >
            {renderSection()}
          </motion.div>
        </div>
      </div>
    </div>
  )
}