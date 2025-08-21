# API (v1)

## Base and versioning
- All routes under `/api/v1`.
- Legacy routes may 301/410 temporarily.

## Error envelope
Responses use a standard shape:
```json
{ "error": { "code": "string", "message": "string", "retryAfterSec": 0 } }
```

## OpenAPI and client types
- Spec: `docs/api/openapi.yaml`
- Generate TS types for the web client:
```bash
bash scripts/gen-api-types.sh
# or from /web
npm run gen:api:types
```
- Output: `web/src/types/api.d.ts`
