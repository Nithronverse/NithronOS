import { ReactNode } from 'react'
import { cn } from '@/lib/utils'

interface BadgeProps {
  children: ReactNode
  variant?: 'default' | 'outline' | 'secondary'
  className?: string
}

export function Badge({ children, variant = 'default', className }: BadgeProps) {
  const variants = {
    default: 'bg-primary text-primary-foreground',
    outline: 'border border-muted-foreground/40 text-foreground',
    secondary: 'bg-secondary text-secondary-foreground',
  }
  
  return (
    <span 
      className={cn(
        'inline-flex items-center rounded-md px-2.5 py-0.5 text-xs font-semibold transition-colors',
        variants[variant],
        className
      )}
    >
      {children}
    </span>
  )
}


