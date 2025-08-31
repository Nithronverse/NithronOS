import fs from 'fs'
import path from 'path'
import ts from 'typescript'
import { fileURLToPath } from 'url'

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)

const projectRoot = path.resolve(__dirname, '..')
const nosClientPath = path.resolve(projectRoot, 'src', 'lib', 'nos-client.ts')

const source = ts.createSourceFile(
  'nos-client.ts',
  fs.readFileSync(nosClientPath, 'utf-8'),
  ts.ScriptTarget.ES2022,
  true,
  ts.ScriptKind.TS
)

type Endpoint = { method: 'GET'|'POST'|'PUT'|'PATCH'|'DELETE'; path: string }
const endpoints: Endpoint[] = []

function add(method: Endpoint['method'], p: string) {
  if (!p.startsWith('/')) return
  const pathOut = p.startsWith('/api/') ? p : `/api${p}`
  endpoints.push({ method, path: pathOut })
}

function isHttpCoreCall(expr: ts.CallExpression): { m: Endpoint['method'], p: string } | null {
  const prop = expr.expression
  if (ts.isPropertyAccessExpression(prop)) {
    const name = prop.name.getText()
    if (['get','post','put','patch','del'].includes(name)) {
      const args = expr.arguments
      if (args.length >= 1 && ts.isStringLiteralLike(args[0])) {
        const pathLit = (args[0] as ts.StringLiteral).text
        const method = name === 'del' ? 'DELETE' : name.toUpperCase() as Endpoint['method']
        return { m: method, p: pathLit }
      }
    }
  }
  return null
}

function walk(node: ts.Node) {
  if (ts.isCallExpression(node)) {
    const match = isHttpCoreCall(node)
    if (match) add(match.m, match.p)
  }
  node.forEachChild(walk)
}

walk(source)

try {
  const { NOS_ENDPOINTS } = await import('../src/lib/nos-client.ts') as unknown as { NOS_ENDPOINTS: Endpoint[] }
  for (const e of NOS_ENDPOINTS) add(e.method, e.path.replace('/api',''))
} catch {}

const uniq = new Map<string, Endpoint>()
for (const e of endpoints) uniq.set(`${e.method} ${e.path}`, e)
const out = Array.from(uniq.values())

const outPath = path.resolve(projectRoot, '..', 'nos_client_contract.json')
fs.writeFileSync(outPath, JSON.stringify(out, null, 2))
console.log(outPath)
