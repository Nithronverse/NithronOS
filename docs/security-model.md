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

### Agent allowlist & argument sanitization
- The agent executes a strict, shell-free allowlist of commands; argv is passed directly to binaries (no `/bin/sh`).
- Allowed families (subset):
  - `btrfs device add/remove … <mount>`
  - `btrfs replace start <old> <new> <mount>`, `btrfs replace status <mount>`
  - `btrfs balance start … <mount>`, `btrfs balance status <mount>`, `btrfs balance cancel <mount>`
  - `btrfs filesystem show|usage [flags] [mount]`
  - Minimal support for `mkfs.btrfs`, `mount -t btrfs`, `umount`, `blkid`, `cryptsetup` used by planners
- Sanitization rules:
  - Device paths must be absolute under `/dev/` and contain no whitespace or NUL.
  - Mount paths must be absolute and restricted to `/srv/…` or `/mnt/…`.
  - For `filesystem usage`, the last non-flag token (if present) must pass mount path checks.
  - Unknown verbs or extra tokens cause rejection.
- Execution hardening:
  - Absolute binaries (e.g., `/usr/bin/btrfs`) executed with a clean environment (`LANG=C`, `LC_ALL=C`, minimal `PATH`).
  - Context timeouts are enforced; stdout/stderr are captured separately with bounded sizes.
  - No globbing or shell metacharacters are interpreted.


## Recovery & Residual Risks
- Physical console access implies high trust; recovery can reset passwords and disable 2FA
- Ensure `nos.recovery=1` is cleared post-recovery; consider rotating secrets


