# Observability (Metrics & Debug)

## Access logs
- Structured JSON logs include: request id, method, path, status, duration, uid (when present), ip.

## Metrics
- `/metrics` (Prometheus exposition) enabled by config `metrics.enabled`.
- Optional `metrics.allowlist` supports simple IP/prefix allow.

## Debug
- `/debug/pprof` gated by `metrics.pprof` and restricted to localhost.
