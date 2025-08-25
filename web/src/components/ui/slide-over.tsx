import * as React from 'react'
import { cn } from '@/lib/utils'
import { X } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'

interface SlideOverProps {
  isOpen: boolean
  onClose: () => void
  title: string
  description?: string
  children: React.ReactNode
  className?: string
  size?: 'sm' | 'md' | 'lg' | 'xl'
}

const sizeClasses = {
  sm: 'max-w-sm',
  md: 'max-w-md',
  lg: 'max-w-lg',
  xl: 'max-w-xl',
}

export function SlideOver({
  isOpen,
  onClose,
  title,
  description,
  children,
  className,
  size = 'md',
}: SlideOverProps) {
  // Close on escape key
  React.useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    if (isOpen) {
      document.addEventListener('keydown', handleEscape)
      return () => document.removeEventListener('keydown', handleEscape)
    }
  }, [isOpen, onClose])

  return (
    <AnimatePresence>
      {isOpen && (
        <>
          {/* Backdrop */}
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            onClick={onClose}
            className="fixed inset-0 z-40 bg-black/50 backdrop-blur-sm"
          />
          
          {/* Slide-over panel */}
          <motion.div
            initial={{ x: '100%' }}
            animate={{ x: 0 }}
            exit={{ x: '100%' }}
            transition={{ type: 'spring', damping: 30, stiffness: 300 }}
            className={cn(
              'fixed inset-y-0 right-0 z-50 w-full overflow-y-auto bg-background shadow-xl',
              sizeClasses[size],
              className
            )}
          >
            <div className="flex h-full flex-col">
              {/* Header */}
              <div className="border-b border-border px-6 py-4">
                <div className="flex items-start justify-between">
                  <div>
                    <h2 className="text-xl font-semibold">{title}</h2>
                    {description && (
                      <p className="mt-1 text-sm text-muted-foreground">
                        {description}
                      </p>
                    )}
                  </div>
                  <button
                    onClick={onClose}
                    className="rounded-lg p-1 hover:bg-muted transition-colors"
                  >
                    <X className="h-5 w-5" />
                  </button>
                </div>
              </div>
              
              {/* Content */}
              <div className="flex-1 overflow-y-auto p-6">
                {children}
              </div>
            </div>
          </motion.div>
        </>
      )}
    </AnimatePresence>
  )
}
