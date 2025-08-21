## API versioning, typed errors, and OpenAPI types

### Base path and shims
- All application routes are mounted under `/api/v1`.
- Temporary shims can return 301 (redirect) or 410 (gone) for old paths if needed during migrations.

### Standard error envelope
Back-end errors follow a single shape:

```json
{ "error": { "code": "string", "message": "string", "retryAfterSec": 0 } }
```

- Use helpers in `backend/nosd/pkg/httpx`:
  - `WriteError(w, status, message)`
  - `WriteTypedError(w, status, code, message, retryAfterSec)`
- 429 responses also set `Retry-After` header (seconds).

### OpenAPI spec
- Seed OpenAPI document lives at `docs/api/openapi.yaml`.
- Keep it updated when endpoints are added/changed (auth/setup endpoints are documented initially).

### Type generation (frontend)
- Script (root): `scripts/gen-api-types.sh`
- Web script: from `web/` run:

```bash
npm run gen:api:types
```

This generates `web/src/types/api.d.ts` using `openapi-typescript`.


