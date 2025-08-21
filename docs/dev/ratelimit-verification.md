## Rate limit verification report

### What changed

- Unified OTP and login throttling to use a single persisted store at `/var/lib/nos/ratelimit.json`.
- Removed legacy in-memory limiters; added proxy-aware client IP extraction with `TrustProxy` knob.
- Added config knobs: `NOS_RATE_OTP_PER_MIN` (default 5), `NOS_RATE_LOGIN_PER_15M` (default 5), and `NOS_TRUST_PROXY`.
- Added tests for persistence, proxy parsing, and handler integration.

### Files touched (high-level)

- `backend/nosd/internal/ratelimit/store.go` (Allow semantics, throttled persist, RFC3339Nano timestamps)
- `backend/nosd/internal/config/config.go` (TrustProxy, RateOTPPerMin, RateLoginPer15m)
- `backend/nosd/internal/server/router.go` (use persisted store for OTP/login; clientIP helper)
- Tests:
  - `backend/nosd/internal/ratelimit/store_persistence_test.go`
  - `backend/nosd/internal/server/ip_extractor_test.go`
  - `backend/nosd/internal/server/rate_integration_test.go`

### Proof of removal

Searches (should have no functional usages):

- `(limiter|throttle|map\[string\]int|sync\.Map).*OTP|login` → Only references in test files and new log text; no in-memory limiters remain in code paths.
- `(TODO|FIXME).*(rate|limit|OTP|login)` → No matches.
- `(os\.WriteFile|ioutil\.WriteFile|os\.Create).*ratelimit.*\.json` → No matches (fsatomic used).

### 429 behavior

- Typed error: `{"error":{"code":"rate.limited","message":"try later","retryAfterSec":N}}`
- `Retry-After` header set with remaining seconds.
- Structured warn logs include: `event=rate.limited`, route, key, remaining, and `resetAt`.

### CI checks

- `go vet ./...` → OK
- `go test ./...` → OK (environment without CGO; race can be run in CI with CGO enabled)
- `web: npm run typecheck` → OK


