# Auth & Sessions (Admin Guide)

This guide summarizes authentication, sessions, and recovery for NithronOS.

## Login
- Username + password (Argon2id, PHC format)
- Optional TOTP (6 digits, 30s period, ±1 step window)
- Account lockout after repeated failures; generic error responses

## Cookies
- `nos_session`: short-lived session (default 15m); httpOnly; SameSite=Lax; Secure
- `nos_refresh`: optional refresh (default 7d); httpOnly; SameSite=Lax; Secure
- `nos_csrf`: CSRF token used with `X-CSRF-Token` header for state-changing requests

## Server-side sessions
- Each login also creates a server-side record bound to:
  - `sid` (ULID)
  - UA fingerprint hash and IP (/24 for IPv4, /64 for IPv6)
  - Creation/expiry timestamps and last-seen time
- Bind enforcement is lightweight; misuse rotates/invalidates tokens on refresh

## Session management APIs
- `GET /api/v1/auth/sessions`: lists own sessions
  - Response: `[{ sid, createdAt, lastSeenAt, ipPrefix, uaFingerprint, current }]`
- `POST /api/v1/auth/sessions/revoke`
  - Body: `{ "scope": "current" | "all" | "sid", "sid"?: "<sid>" }`
  - Current scope clears auth cookies; actions are audit-logged

## Refresh hardening
- Refresh rotates both the `sid` and the refresh token
- Reuse detection (presenting an already-rotated refresh) revokes all sessions for that user

## Rate limits
- OTP: default 5/min per IP; configurable window and limit
- Login: default 5/15m per IP and per username; persisted across restarts
- Standard 429 error with `Retry-After` header and structured body `{ error: { code: "rate.limited", retryAfterSec } }`

## Recovery
- See [Recovery](recovery.md) for console-only procedures to reset password, disable 2FA, or regenerate a first-boot OTP.
