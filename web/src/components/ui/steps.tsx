export function Steps({ steps, current }: { steps: string[]; current: number }) {
  return (
    <ol className="mb-2 flex items-center gap-3 text-sm">
      {steps.map((s, idx) => {
        const active = current === idx + 1
        const done = current > idx + 1
        return (
          <li key={s} className="flex items-center gap-2">
            <span
              className={`flex h-6 w-6 items-center justify-center rounded-full text-xs font-medium ${
                done ? 'bg-green-600 text-white' : active ? 'bg-primary text-background' : 'bg-muted text-foreground'
              }`}
            >
              {idx + 1}
            </span>
            <span className={`${active ? 'font-medium' : 'text-muted-foreground'}`}>{s}</span>
            {idx < steps.length - 1 && <span className="mx-1 h-px w-8 bg-muted" />}
          </li>
        )
      })}
    </ol>
  )
}


