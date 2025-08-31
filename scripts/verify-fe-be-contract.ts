import fs from 'fs'
import path from 'path'

type Route = { method: string; path: string }

const repoRoot = path.resolve(__dirname, '..')
const fePath = path.resolve(repoRoot, 'nos_client_contract.json')
const bePath = path.resolve(repoRoot, 'route_dump.json')

if (!fs.existsSync(fePath)) {
  console.error(`Missing ${fePath}. Run: node web/scripts/extract-nos-contract.ts`)
  process.exit(1)
}
if (!fs.existsSync(bePath)) {
  console.error(`Missing ${bePath}. Run: go run backend/cmd/route-dump/main.go > route_dump.json`)
  process.exit(1)
}

const fe: Route[] = JSON.parse(fs.readFileSync(fePath, 'utf-8'))
const be: Route[] = JSON.parse(fs.readFileSync(bePath, 'utf-8'))

function toRegex(p: string): RegExp {
  return new RegExp('^' + p
    .replace(/\//g, '\\/')
    .replace(/\{[^}]+\}/g, '[^/]+') + '$')
}

const beByMethod = new Map<string, RegExp[]>()
for (const r of be) {
  const arr = beByMethod.get(r.method) || []
  arr.push(toRegex(r.path))
  beByMethod.set(r.method, arr)
}

const missing: Route[] = []
for (const f of fe) {
  const regs = beByMethod.get(f.method) || []
  const ok = regs.some(rx => rx.test(f.path))
  if (!ok) missing.push(f)
}

if (missing.length) {
  console.error('MISSING_ON_BACKEND:')
  for (const m of missing) console.error(`${m.method} ${m.path}`)
  process.exit(1)
}

console.log('Contract OK')
