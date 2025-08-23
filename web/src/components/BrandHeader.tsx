export default function BrandHeader({ className = "" }: { className?: string }) {
  return (
    <div className={`absolute left-1/2 -translate-x-1/2 -top-28 flex flex-col items-center ${className}`}>
      <img
        src="/brand/nithronos-logo-mark.svg"
        alt="NithronOS"
        width={144}
        height={144}
        className="h-36 w-36"
      />
      <span className="mt-2 text-2xl font-bold text-foreground">NithronOS</span>
    </div>
  )
}


