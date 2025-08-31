# No Placeholders Audit

## Audit Results

| File | Component/Hook | Symptom | Current Source | Required Fix |
|------|---------------|---------|----------------|--------------|
| `web/src/lib/api-dashboard.ts` | dashboardApi.getDashboard | Returns zeros on error | `cpuPct: 0, mem: {used: 0, total: 1}` | Throw error instead of returning fake defaults |
| `web/src/lib/api-dashboard.ts` | dashboardApi.getSystemSummary | Returns zeros on error | `cpuPct: 0, mem: {used: 0, total: 1}` | Throw error instead of returning fake defaults |
| `web/src/lib/api-dashboard.ts` | dashboardApi.getStorageSummary | Returns zeros on error | `totalBytes: 0, poolsOnline: 0` | Throw error instead of returning fake defaults |
| `web/src/lib/api-dashboard.ts` | dashboardApi.getDisksSummary | Returns zeros on error | `total: 0, healthy: 0` | Throw error instead of returning fake defaults |
| `web/src/pages/Dashboard.tsx` | Dashboard widgets | Shows zeros when API fails | Uses api-dashboard defaults | Handle errors properly, show "unavailable" |
| `web/src/pages/MonitoringDashboard.tsx` | Monitoring | No SSE/WebSocket | Polling via useMonitoringData | Consider SSE for real-time updates |
| `web/src/hooks/use-dashboard.ts` | 1Hz refresh | Multiple intervals | Each widget has own interval | Consolidate to single interval |
| Test files | Mocks | Mock implementations | vi.mock() patterns | Expected for tests, no fix needed |

## Key Issues Found

### 1. **Fake Defaults on Error** (CRITICAL)
The dashboard API returns zero values when the backend fails, making it impossible to distinguish between:
- Real zero values (e.g., 0% CPU usage)
- Backend errors/unavailability

### 2. **No SSE/WebSocket for Real-time Data**
Dashboard and Monitoring use polling (1Hz) instead of SSE/WebSocket for real-time metrics.

### 3. **Direct Placeholders**
No hardcoded placeholders like "MOCK", "FAKE", "Lorem ipsum" were found in production code.

### 4. **Network Calls**
All network calls correctly use `nos-client.ts`. No direct `fetch()` or `axios` calls found.

### 5. **Path Validation**  
No `/api/` prefixes found at call sites. All paths correctly use `/v1/` or `/setup/`.

## Recommendations

1. **Remove fake defaults** - Let errors bubble up so UI can show proper error states
2. **Implement SSE** - Use SSE for real-time metrics in Dashboard and Monitoring
3. **Consolidate intervals** - Single 1Hz interval per page to reduce overhead
4. **Add explicit "unavailable" states** - When backend endpoints don't exist yet

## Status
- ✅ No mock/placeholder strings in production code
- ✅ All using nos-client for API calls
- ✅ No `/api/` path prefixes at call sites
- ✅ Dashboard API no longer returns fake defaults on error
- ✅ SystemHealthWidget uses shared useSystemVitals hook
- ✅ SSE support added for real-time metrics (with polling fallback)
- ✅ Dashboard shows "unavailable" on error instead of zeros

## Changes Made

### 1. Fixed api-dashboard.ts
- Removed all try/catch blocks that returned fake zeros
- Errors now bubble up to UI for proper handling

### 2. Created useSystemVitals hook
- Shared by Dashboard and Monitoring for consistency
- Attempts SSE connection first (`/v1/metrics/stream`)
- Falls back to 1Hz polling if SSE unavailable
- Shows "Live" badge when using SSE

### 3. Updated Dashboard
- SystemHealthWidget now uses real-time data
- No fake zeros shown on error
- Proper loading and error states
