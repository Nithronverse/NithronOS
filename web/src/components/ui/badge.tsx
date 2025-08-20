import { ReactNode } from 'react'

export function Badge({ children, variant = 'default' }: { children: ReactNode; variant?: 'default' | 'outline' }) {
  const cls = variant === 'outline'
    ? 'border border-muted/40 text-foreground'
    : 'bg-muted text-foreground'
  return <span className={`inline-flex items-center rounded px-2 py-0.5 text-xs ${cls}`}>{children}</span>
}


