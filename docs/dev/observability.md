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

### Live Metrics Summary (JSON)

For lightweight UI polling at 1 Hz, the backend exposes a compact JSON summary:

- `GET /api/metrics/summary`

Response shape (example):

```
{
  "timestamp": 1730000000,
  "cpu": 12.3,
  "load1": 0.42,
  "load5": 0.31,
  "load15": 0.25,
  "memory": { "total": 0, "used": 0, "free": 0, "available": 0, "usagePct": 0, "cached": 0, "buffers": 0 },
  "swap": { "total": 0, "used": 0, "free": 0, "usagePct": 0 },
  "uptimeSec": 12345,
  "tempCpu": 45.1,
  "network": { "bytesRecv": 0, "bytesSent": 0, "packetsRecv": 0, "packetsSent": 0, "rxSpeed": 0, "txSpeed": 0 },
  "diskIO": { "readBytes": 0, "writeBytes": 0, "readOps": 0, "writeOps": 0, "readSpeed": 0, "writeSpeed": 0 }
}
```

Notes:
- All keys are stable. If a field is unavailable on a platform, the key remains with a zero value or null.
- This endpoint is optimized to be fast (<10ms typical) and suitable for UI polling.

### Optional SSE Stream

An optional Server-Sent Events stream emits the same payload every second:

- `GET /api/metrics/stream`

Headers:
- `Content-Type: text/event-stream`
- `Cache-Control: no-cache`
- `Connection: keep-alive`

This is disabled by default in the UI which uses polling; switch to SSE if needed.

## Debug
- `/debug/pprof` gated by `metrics.pprof` and restricted to localhost.
