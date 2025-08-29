import {
  LayoutDashboard,
  HardDrive,
  Share2,
  Package,
  Globe,
  Settings,
  Archive,
  Camera,
  Calendar,
  LucideIcon,
} from 'lucide-react'

export interface NavItem {
  id: string
  label: string
  icon?: LucideIcon
  path: string
  children?: NavItem[]
}

export const navItems: NavItem[] = [
  {
    id: 'dashboard',
    label: 'Dashboard',
    icon: LayoutDashboard,
    path: '/',
  },
  {
    id: 'storage',
    label: 'Storage',
    icon: HardDrive,
    path: '/storage',
  },
  {
    id: 'shares',
    label: 'Shares',
    icon: Share2,
    path: '/shares',
  },
  {
    id: 'backup',
    label: 'Backup',
    icon: Archive,
    path: '/backup/snapshots',
    children: [
      {
        id: 'snapshots',
        label: 'Snapshots',
        icon: Camera,
        path: '/backup/snapshots',
      },
      {
        id: 'schedules',
        label: 'Schedules',
        icon: Calendar,
        path: '/backup/schedules',
      },
    ],
  },
  {
    id: 'apps',
    label: 'Apps',
    icon: Package,
    path: '/apps',
  },
  {
    id: 'remote',
    label: 'Remote',
    icon: Globe,
    path: '/remote',
  },
  {
    id: 'settings',
    label: 'Settings',
    icon: Settings,
    path: '/settings',
    children: [
      {
        id: 'general',
        label: 'General',
        path: '/settings',
      },
      {
        id: 'network',
        label: 'Network & Remote',
        path: '/settings/network',
      },
      {
        id: '2fa',
        label: 'Two-Factor Auth',
        path: '/settings/2fa',
      },
      {
        id: 'users',
        label: 'Users',
        path: '/settings/users',
      },
      {
        id: 'schedules',
        label: 'Schedules',
        path: '/settings/schedules',
      },
      {
        id: 'updates',
        label: 'Updates & Releases',
        path: '/settings/updates',
      },
    ],
  },
]
