## Config and Safe Hot Reload

File: `/etc/nos/config.yaml` (env can override).

### Keys
- `http.bind`: address to listen on (e.g. `127.0.0.1:9000`)
- `cors.origin`: allowed UI origin (credentials allowed)
- `rate.*`: `otpPerMin`, `loginPer15m`, `otpWindowSec`, `loginWindowSec`
- `trustProxy`: if true, client IP is taken from `X-Forwarded-For`
- `logging.level`: `trace|debug|info|warn|error`
- `sessions.accessTTL`, `sessions.refreshTTL`: Go durations (e.g. `15m`, `168h`)
- `metrics.enabled`: enable `/metrics` endpoint
- `metrics.pprof`: enable `/debug/pprof` (localhost only)
- `metrics.allowlist`: optional list of IPs or dot-suffix prefixes allowed to read `/metrics`

### Env overrides
Prefer environment in containers or dev:

```
NOS_HTTP_BIND=0.0.0.0:9000
NOS_CORS_ORIGIN=https://ui.example
NOS_TRUST_PROXY=true
NOS_LOG=debug
NOS_RATE_OTP_PER_MIN=5
NOS_RATE_LOGIN_PER_15M=5
NOS_RATE_OTP_WINDOW_SEC=60
NOS_RATE_LOGIN_WINDOW_SEC=900
NOS_SESSION_ACCESS_TTL=15m
NOS_SESSION_REFRESH_TTL=168h
NOS_METRICS=1
NOS_PPROF=0
NOS_METRICS_ALLOWLIST=127.0.0.1,10.0.0.
```

### Safe hot reload
Send `SIGHUP` (Unix) to apply safe fields without restart:

```
sudo kill -HUP $(pidof nosd)
```

Applied live: `cors.origin`, `trustProxy`, `logging.level`. Changes are logged with a diff.


