import {
  LayoutDashboard,
  HardDrive,
  Share2,
  Package,
  Globe,
  Settings,
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
        label: 'Network',
        path: '/settings/network',
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
        label: 'Updates',
        path: '/settings/updates',
      },
    ],
  },
]
