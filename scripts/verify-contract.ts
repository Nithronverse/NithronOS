import fs from 'fs'
import path from 'path'
import { NOS_ENDPOINTS } from '../web/src/lib/nos-client'

type Route = { method: string; path: string }

const dumpPath = path.resolve(__dirname, '..', 'route_dump.json')
if (!fs.existsSync(dumpPath)) {
  console.error(`route_dump.json not found at ${dumpPath}. Run route-dump first.`)
  process.exit(1)
}

const be: Route[] = JSON.parse(fs.readFileSync(dumpPath, 'utf-8'))

// Normalize backend path with {param} to regex
function toRegex(p: string): RegExp {
  // Support chi patterns like /{id} and wildcards
  const esc = p
    .replace(/\//g, '\\/')
    .replace(/\{[^}]+\}/g, '[^/]+')
  return new RegExp('^' + esc + '$')
}

const beByMethod = new Map<string, RegExp[]>()
for (const r of be) {
  const arr = beByMethod.get(r.method) || []
  arr.push(toRegex(r.path))
  beByMethod.set(r.method, arr)
}

const missing: Route[] = []
for (const fe of NOS_ENDPOINTS) {
  const regs = beByMethod.get(fe.method) || []
  const ok = regs.some(rx => rx.test(fe.path))
  if (!ok) missing.push(fe)
}

if (missing.length) {
  console.error('FE endpoints missing on BE:')
  for (const m of missing) console.error(`  ${m.method} ${m.path}`)
  process.exit(1)
}

// Optional: warn for BE unused by FE
const feSet = new Set(NOS_ENDPOINTS.map(e => e.method + ' ' + e.path))
const unused = be.filter(r => !Array.from(feSet).some(f => {
  const [m, p] = f.split(' ')
  return m === r.method && toRegex(r.path).test(p)
}))
if (unused.length) {
  console.warn('BE routes not referenced by FE (warn):')
  for (const u of unused) console.warn(`  ${u.method} ${u.path}`)
}

console.log('Contract OK')


