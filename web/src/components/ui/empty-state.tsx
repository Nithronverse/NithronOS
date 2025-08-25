import { cn } from '@/lib/utils'
import { Button } from './button'
import {
  FileX,
  Search,
  ServerCrash,
  Inbox,
  LucideIcon,
} from 'lucide-react'

type EmptyStateVariant = 'no-data' | 'filtered' | 'error' | 'backend-down'

interface EmptyStateProps {
  variant?: EmptyStateVariant
  icon?: LucideIcon
  title: string
  description?: string
  action?: {
    label: string
    onClick: () => void
  }
  className?: string
}

const variantConfig: Record<
  EmptyStateVariant,
  { icon: LucideIcon; defaultTitle: string }
> = {
  'no-data': {
    icon: Inbox,
    defaultTitle: 'No data yet',
  },
  'filtered': {
    icon: Search,
    defaultTitle: 'No results found',
  },
  'error': {
    icon: FileX,
    defaultTitle: 'Something went wrong',
  },
  'backend-down': {
    icon: ServerCrash,
    defaultTitle: 'Backend unreachable',
  },
}

export function EmptyState({
  variant = 'no-data',
  icon,
  title,
  description,
  action,
  className,
}: EmptyStateProps) {
  const config = variantConfig[variant]
  const Icon = icon || config.icon

  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center py-12 text-center',
        className
      )}
    >
      <div className="mb-4 rounded-full bg-muted/30 p-3">
        <Icon className="h-12 w-12 text-muted-foreground" />
      </div>
      <h3 className="mb-2 text-lg font-semibold">{title}</h3>
      {description && (
        <p className="mb-6 max-w-sm text-sm text-muted-foreground">
          {description}
        </p>
      )}
      {action && (
        <Button onClick={action.onClick} size="sm">
          {action.label}
        </Button>
      )}
    </div>
  )
}
