// Null-safe number formatting helpers used across dashboard/health widgets
import { formatBytes as humanBytes } from '@/lib/utils'

export function toFixedSafe(n: number | null | undefined, d = 0, fallback = '0') {
  if (n === null || n === undefined || Number.isNaN(n)) return fallback
  const num = Number(n)
  if (!Number.isFinite(num)) return fallback
  return num.toFixed(d)
}

export function pctSafe(n: number | null | undefined, d = 0, fallback = 'N/A') {
  if (n === null || n === undefined || Number.isNaN(n)) return fallback
  const num = Number(n)
  if (!Number.isFinite(num)) return fallback
  return `${num.toFixed(d)}%`
}

export function bytesSafe(b: number | null | undefined, fallback = 'N/A') {
  if (b === null || b === undefined || Number.isNaN(b)) return fallback
  const num = Number(b)
  if (!Number.isFinite(num)) return fallback
  return humanBytes(num)
}


