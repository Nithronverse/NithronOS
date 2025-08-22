# Observability (Metrics & Debug)

## Access logs
- Structured JSON logs include: request id, method, path, status, duration, uid (when present), ip.

## Metrics
- `/metrics` (Prometheus exposition) enabled by config `metrics.enabled`.
- Optional `metrics.allowlist` supports simple IP/prefix allow.

### Scraping metrics
- Recommended single target: `GET /metrics/all` on `nosd` to retrieve a combined exposition that includes both `nosd` and agent metrics.
- Fallback targets:
  - `GET /metrics` on `nosd` (nosd-only).
  - `GET /metrics` on the agent (served on the agent’s Unix socket). Note: Prometheus cannot scrape Unix sockets directly; prefer the combined `nosd` endpoint above.

Notes:
- The combined endpoint appends the agent exposition after the `nosd` exposition with a separating comment line.
- If the agent scrape fails (timeout ~500ms), `nosd` still returns its own metrics and adds a comment `# agent metrics unavailable: <err>`.

## Debug
- `/debug/pprof` gated by `metrics.pprof` and restricted to localhost.
