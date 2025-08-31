import fs from 'fs'
import path from 'path'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const repoRoot = path.resolve(__dirname, '..')

const fePath = path.resolve(repoRoot, 'nos_client_contract.json')
const bePath = path.resolve(repoRoot, 'route_dump.json')

if (!fs.existsSync(fePath)) {
  console.error(`Missing ${fePath}. Run: node web/scripts/extract-nos-contract.mjs`)
  process.exit(1)
}
if (!fs.existsSync(bePath)) {
  console.error(`Missing ${bePath}. Run: go run backend/nosd/cmd/route-dump > route_dump.json`) 
  process.exit(1)
}

const fe = JSON.parse(fs.readFileSync(fePath, 'utf-8'))
const be = JSON.parse(fs.readFileSync(bePath, 'utf-8'))

function normalizeMethod(m) { return String(m || '').toUpperCase() }
function normalizePath(p) {
  if (!p) return ''
  // Ensure starts with /api
  if (p.startsWith('/v1/')) p = '/api' + p
  // Drop trailing slash for comparison convenience
  if (p.length > 1 && p.endsWith('/')) p = p.slice(0, -1)
  return p
}

function feToRegex(p) {
  const base = normalizePath(p)
  const pattern = base
    .replace(/\//g, '\\/')
    .replace(/\{[^}]+\}/g, '[^/]+')
  // Allow optional trailing slash in backend route dump
  return new RegExp('^' + pattern + '\\/?$')
}

const beByMethod = new Map()
for (const r of be) {
  const m = normalizeMethod(r.method)
  const list = beByMethod.get(m) || []
  list.push(normalizePath(r.path))
  beByMethod.set(m, list)
}

const missing = []
for (const f of fe) {
  const m = normalizeMethod(f.method)
  const candidates = beByMethod.get(m) || []
  const rx = feToRegex(f.path)
  const ok = candidates.some(p => rx.test(p))
  if (!ok) missing.push({ method: m, path: normalizePath(f.path) })
}

if (missing.length) {
  console.error('MISSING_ON_BACKEND:')
  for (const m of missing) console.error(`${m.method} ${m.path}`)
  process.exit(1)
}

console.log('Contract OK')
