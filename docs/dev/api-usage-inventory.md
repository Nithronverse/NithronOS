# API Usage Inventory

## Current State Analysis

| File | Function/Component | Current Call | Uses nos-client? | Notes |
|------|-------------------|--------------|------------------|-------|
| **web/src/pages/SettingsUpdates.tsx** | checkForUpdates() | `fetch('/api/snapshots/recent')` | ❌ | Direct fetch, needs migration |
| **web/src/pages/SettingsUpdates.tsx** | pruneSnapshots() | `fetch('/api/snapshots/prune')` | ❌ | Direct fetch, needs migration |
| **web/src/pages/PoolDetails.tsx** | useEffect() | `fetch('/api/pools')` | ❌ | Direct fetch, needs migration |
| **web/src/pages/PoolDetails.tsx** | poll() | `fetch('/api/v1/pools/tx/.../log')` | ❌ | Direct fetch with /api/v1 |
| **web/src/api/client.ts** | support.bundle() | `fetch('/api/support/bundle')` | ❌ | Direct fetch |
| **web/src/api/updates.ts** | streamProgress() | `new EventSource('/api/v1/updates/progress/stream')` | ❌ | SSE, needs helper |
| **web/src/api/apps.ts** | streamLogs() | `new WebSocket(...)` | ❌ | WebSocket, needs helper |
| **web/src/api/apps.ts** | loadAppSchema() | `fetch('/api/v1/apps/schema/...')` | ❌ | Direct fetch |
| **web/src/api/http.ts** | fetchJSON() | `fetch(path, ...)` | ❌ | Legacy helper, should be removed |
| **web/src/api/http.ts** | request() | `fetch('/api/auth/refresh')` | ❌ | Auth refresh, already in nos-client |
| **web/src/components/storage/CreatePoolWizard.tsx** | startSSE() | `new EventSource('/api/v1/pools/tx/.../stream')` | ❌ | SSE for pool creation |
| **web/src/components/storage/ImportPoolModal.tsx** | fallback | `fetch('/api/disks')` | ❌ | Direct fetch |
| **web/src/lib/useApi.ts** | request() | `fetch(path, ...)` | ❌ | Legacy helper |
| **web/src/pages/PoolSnapshots.tsx** | refresh() | `fetch('/api/pools/.../snapshots')` | ❌ | Direct fetch |
| **web/src/lib/api.ts** | endpoints.* | Various | ✅ | Shim layer using nos-client |
| **web/src/lib/api-dashboard.ts** | dashboardApi.* | http.get/post | ✅ | Uses nos-client |
| **web/src/lib/api-health.ts** | healthApi.* | http.get | ✅ | Uses nos-client |
| **web/src/api/monitor.ts** | monitorApi.* | http.get/post/patch/del | ✅ | Uses nos-client |
| **web/src/api/backup.ts** | backupApi.* | http.get/post/patch/del | ✅ | Uses nos-client |
| **web/src/api/net.ts** | Various hooks | http.get/post | ✅ | Uses nos-client |
| **web/src/api/apps.ts** | appsApi.* (except streaming) | http.get/post/del | ✅ | Mostly uses nos-client |
| **web/src/api/updates.ts** | updatesApi.* (except streaming) | http.get/post/del | ✅ | Mostly uses nos-client |
| **web/src/api/schedules.ts** | getSchedules/updateSchedules | http.get/post | ✅ | Uses nos-client |

## Issues Found

1. **Direct fetch() calls**: 14 instances bypassing nos-client
2. **EventSource/WebSocket**: 3 instances need helper functions
3. **Path inconsistencies**: Mix of `/api/v1/`, `/api/`, and unversioned paths
4. **Missing endpoints in nos-client**:
   - `/api/snapshots/*`
   - `/api/pools/*/snapshots`
   - `/api/support/bundle`
   - `/api/disks`
   - SSE/WebSocket helpers

## Migration Plan

### Phase 1: Expand nos-client
- Add missing endpoint helpers
- Add SSE/WebSocket helper functions
- Relax path validation for known unversioned paths

### Phase 2: Migrate direct fetch calls
- Replace all `fetch()` with nos-client calls
- Update paths to remove `/api` prefix

### Phase 3: Remove legacy code
- Delete `api/http.ts`
- Delete `lib/useApi.ts` (if unused)
- Remove shim layers once all imports updated

### Phase 4: Testing & Validation
- Add unit tests for path validation
- Verify all endpoints work
- Ensure live data updates in Dashboard/Health
