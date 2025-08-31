import fs from 'fs'
import path from 'path'
import { fileURLToPath } from 'url'
import ts from 'typescript'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const projectRoot = path.resolve(__dirname, '..')
const nosClientPath = path.resolve(projectRoot, 'src', 'lib', 'nos-client.ts')

const srcText = fs.readFileSync(nosClientPath, 'utf-8')
const source = ts.createSourceFile('nos-client.ts', srcText, ts.ScriptTarget.ES2022, true, ts.ScriptKind.TS)

const endpoints = []

function canon(p) {
  if (!p.startsWith('/')) return null
  if (!p.startsWith('/api/v1') && !p.startsWith('/v1/')) return null
  const out = p.startsWith('/api/') ? p : `/api${p}`
  return out !== '/' && out.endsWith('/') ? out.slice(0, -1) : out
}

function add(method, p) {
  const out = canon(p)
  if (!out) return
  endpoints.push({ method, path: out })
}

function getStringLiteral(node) {
  return (ts.isStringLiteralLike(node) && node.text) ? node.text : null
}

let foundManifest = false
function extractNosEndpoints(node) {
  if (!ts.isVariableStatement(node)) return
  const decl = node.declarationList.declarations.find(d => d.name && d.name.getText() === 'NOS_ENDPOINTS')
  if (!decl || !decl.initializer) return
  const init = decl.initializer
  if (!ts.isArrayLiteralExpression(init)) return
  // Reset any scanned endpoints; prefer explicit manifest
  endpoints.length = 0
  for (const el of init.elements) {
    if (!ts.isObjectLiteralExpression(el)) continue
    let method = null, pathVal = null
    for (const prop of el.properties) {
      if (!ts.isPropertyAssignment(prop)) continue
      const key = prop.name.getText()
      if (key === 'method') method = getStringLiteral(prop.initializer)
      if (key === 'path') pathVal = getStringLiteral(prop.initializer)
    }
    if (method && pathVal) add(method, pathVal)
  }
  foundManifest = true
}

function tryAddHttpCoreCall(node) {
  if (!ts.isCallExpression(node)) return
  const expr = node.expression
  if (!ts.isPropertyAccessExpression(expr)) return
  const name = expr.name.getText()
  if (!['get','post','put','patch','del'].includes(name)) return
  const arg0 = node.arguments[0]
  const p = getStringLiteral(arg0)
  if (!p) return
  const method = name === 'del' ? 'DELETE' : name.toUpperCase()
  add(method, p)
}

function walk(node) {
  extractNosEndpoints(node)
  if (!foundManifest) tryAddHttpCoreCall(node)
  node.forEachChild(walk)
}

walk(source)

const uniq = new Map()
for (const e of endpoints) uniq.set(`${e.method} ${e.path}`, e)
const out = Array.from(uniq.values())

const outPath = path.resolve(projectRoot, '..', 'nos_client_contract.json')
fs.writeFileSync(outPath, JSON.stringify(out, null, 2))
console.log(outPath)
