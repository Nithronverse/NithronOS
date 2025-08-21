# Security Model (Short Threat Model)

## Assets
- Admin credentials (password hashes, TOTP secrets, recovery codes)
- Session cookies and server-side sessions
- System configuration and state (`/etc/nos`, `/var/lib/nos`)
- Agent ↔ daemon trust (agent tokens)

## Trust Boundaries
- External network vs. local network (default bind loopback)
- Reverse proxy (when `trustProxy=true`) vs. direct connections
- System user `nos` vs. root-only agent via Unix socket

## Attacker Goals
- Brute-force admin login
- Session hijacking / cookie theft
- Persistence via tampered state files
- Privilege escalation via agent API

## Mitigations
- Rate limiting with persistence (IP + username), standardized 429 with Retry-After
- Temporary account lockout after failures; generic auth errors
- Server-side sessions with UA/IP binding and refresh rotation; reuse detection revokes
- Atomic JSON writes with fsync + rename; advisory file locks; crash recovery of `.tmp`
- Systemd hardening and least-privilege `nos` user; read/write paths restricted
- Agent trust bootstrap: per-agent rotated token after bootstrap token; option to disable registration
- Recovery mode localhost-only; explicit warnings; OTP regeneration guarded

## Recovery & Residual Risks
- Physical console access implies high trust; recovery can reset passwords and disable 2FA
- Ensure `nos.recovery=1` is cleared post-recovery; consider rotating secrets


