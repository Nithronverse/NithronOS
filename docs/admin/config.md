# Config & Hot Reload

File: `/etc/nos/config.yaml` (env overrides allowed). Safe fields can be hot-reloaded via `SIGHUP`.

## Keys
- `http.bind`: e.g. `127.0.0.1:9000`
- `cors.origin`: allowed UI origin
- `rate`: `otpPerMin`, `loginPer15m`, `otpWindowSec`, `loginWindowSec`
- `trustProxy`: use last untrusted hop from `X-Forwarded-For`
- `logging.level`: `trace|debug|info|warn|error`
- `sessions`: `accessTTL`, `refreshTTL` (Go durations)
- `metrics`: `enabled`, `pprof`, `allowlist`
- `agents`: `allowRegistration`

## Env overrides
Examples:
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

## Hot reload
- Send `SIGHUP` to `nosd` to apply updated `cors.origin`, `trustProxy`, and `logging.level`.
- Changes are logged with field diffs.
