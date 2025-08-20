import { useEffect, useState } from 'react'

type Toast = { id: number; message: string; kind: 'success' | 'error' }

let pushImpl: ((t: Omit<Toast, 'id'>) => void) | null = null

export function pushToast(message: string, kind: 'success' | 'error' = 'success') {
  if (pushImpl) pushImpl({ message, kind } as any)
}

export function Toasts() {
  const [toasts, setToasts] = useState<Toast[]>([])
  useEffect(() => {
    pushImpl = (t) => {
      const id = Date.now()
      setToasts((prev) => [...prev, { id, message: t.message, kind: t.kind }])
      setTimeout(() => setToasts((prev) => prev.filter((x) => x.id !== id)), 3500)
    }
    return () => { pushImpl = null }
  }, [])
  return (
    <div className="fixed right-4 top-16 z-[100] space-y-2">
      {toasts.map((t) => (
        <div
          key={t.id}
          className={`rounded border px-3 py-2 text-sm shadow ${
            t.kind === 'success'
              ? 'border-green-500/30 bg-green-500/10 text-green-300'
              : 'border-red-500/30 bg-red-500/10 text-red-300'
          }`}
        >
          {t.message}
        </div>
      ))}
    </div>
  )
}


