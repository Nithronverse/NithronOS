# NithronOS (nOS)
![NithronOS](./assets/brand/nithronos-readme-banner.svg)

[![CI](https://github.com/NotTekk/NithronOS/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/NotTekk/NithronOS/actions/workflows/ci.yml)
[![Snapshots On Update](https://img.shields.io/badge/Snapshots%20On%20Update-Enabled%20by%20default-2D7FF9)](docs/updates.md)
[![Release](https://img.shields.io/badge/NithronOS-v0.1.0--pre--alpha-yellow)](https://github.com/NotTekk/NithronOS/releases/tag/v0.1.0-pre-alpha)

**Open-source Linux-based OS for NAS & homelabs.**  
Local-first storage management (Btrfs/ZFS*), snapshots, shares, backups, and a modern web dashboard with an optional app catalog — all without cloud lock-in.

> **Status:** v0 public preview (pre-alpha).

* ZFS availability depends on platform licensing constraints.

---

## Table of Contents
- [Why NithronOS?](#why-nithronos)
- [Architecture](#architecture)
- [Quickstart (Dev)](#quickstart-dev)
- [First Boot & Auth](#first-boot--auth)
- [Updates & Rollback](#updates--rollback)
- [ISO Build](#iso-build)
- [Firewall (LAN-only by default)](#firewall-lan-only-by-default)
- [Repository Structure](#repository-structure)
- [Roadmap](#roadmap)
- [Docs](#docs)
- [Contributing](#contributing)
- [Licensing & Commercial Use](#licensing--commercial-use)
- [Branding](#branding)
- [Contact](#contact)

---

## Why NithronOS?
- **Local-first, privacy-first** — admin UI served on your LAN by default; remote access is opt-in.
- **Btrfs-first** — snapshots, send/receive; optional ZFS via DKMS*.
- **One-click apps** — Docker/Compose (Plex/Jellyfin, Nextcloud, Immich, etc.).
- **Real safety features** — dry-run plans, pre-update snapshots, easy rollback.
- **Clean UX** — React dashboard, clear health/status, sensible defaults.

---

## Architecture
- **`nosd`** (Go): REST API for disks, pools, snapshots, shares, jobs.
- **`nos-agent`** (Go, root): allow-listed helper for privileged actions.
- **Web UI** (React + TypeScript): talks to `nosd` via OpenAPI client.
- **Reverse proxy** (Caddy): serves UI and proxies API. Pre-alpha default is HTTP-only on LAN (no TLS) with security headers; backend bound to loopback. Browsers will show “Not secure” — acceptable for local preview.
- **Jobs**: systemd timers for snapshots/prune & scheduled maintenance.

**Related docs:**  
API/versioning & typed errors → [docs/api/versioning-and-errors.md](docs/api/versioning-and-errors.md)  
Config & safe hot reload → [docs/dev/config-and-reload.md](docs/dev/config-and-reload.md)  
Recovery mode (admin access) → [docs/dev/recovery-mode.md](docs/dev/recovery-mode.md)  
Pre-Alpha Recovery Checklist → [RECOVERY-CHECKLIST.md](RECOVERY-CHECKLIST.md)  
Storage pools (create/import/encrypt & device ops) → [docs/storage/pools.md](docs/storage/pools.md)  
Storage health (SMART, scrub, schedules) → [docs/storage/health.md](docs/storage/health.md)
Observability → [docs/dev/observability.md](docs/dev/observability.md) (scrape combined metrics via `/metrics/all`)

---

## Quickstart (Dev)
**Requires:** Go 1.23+, Node 20+, npm, make (optional), Docker (for app catalog dev)

    # clone
    git clone https://github.com/NotTekk/NithronOS.git
    cd NithronOS

    # web deps
    cd web && npm install && cd ..

    # backend (terminal 1)
    make api-dev

    # agent (terminal 2, optional)
    make agent-dev

    # web (terminal 3)
    make web-dev

- Backend: http://127.0.0.1:9000  
- Web dev (Vite): http://127.0.0.1:5173

> Security: keep `nosd` bound to loopback in dev. If you expose remotely, enforce 2FA + rate limits and use the LAN-only firewall by default.

---

## First Boot & Auth
On first boot, `nosd` generates a **one-time OTP**, logs it, and prints it to the console (`StandardOutput=journal+console`). The UI calls `/api/setup/state` and routes to `/setup` if required.

1. **OTP** — enter the 6-digit code (15-minute TTL).  
2. **Admin** — create the first admin (strong password with real-time strength indicator).  
3. **2FA (optional)** — TOTP QR + recovery codes.  
4. **Done** — sign in at `/login`.

After the first admin is created, `/api/setup/*` returns **410 Gone** and normal login applies.

**Frontend Features**
- **Smart error handling** — Specific messages for rate limiting, invalid credentials, TOTP required
- **Session management** — Automatic token refresh on 401, no manual re-login needed
- **Global error banner** — Backend unreachable detection with help link
- **Password strength meter** — Real-time feedback during account creation
- **Remember me** — Optional persistent sessions using localStorage

**Where things live**
- Users DB: `/etc/nos/users.json` (remove to rerun setup).  
- Secret key: `/etc/nos/secret.key` (`0600`).  
- First-boot state: `/var/lib/nos/state/firstboot.json`.

**Security defaults**
- Passwords: **Argon2id** (PHC).  
- Sessions: httpOnly cookies (`nos_session` ~15m, optional `nos_refresh` ~7d) + CSRF double-submit (`X-CSRF-Token`).  
- 2FA: TOTP (XChaCha20-Poly1305-encrypted secrets), recovery codes hashed.  
- Guardrails: rate limits per IP/username, generic auth errors, temporary lockout.

📚 For detailed implementation: [Authentication Guide](docs/frontend/authentication.md) | [Migration Guide](docs/frontend/auth-migration-guide.md)

---

## Updates & Rollback
Pre-update snapshots and rollback are **built-in and enabled**.

- Config file: `/etc/nos/snapshots.yaml` (dev default: `./devdata/snapshots.yaml`)
- Targets include Btrfs subvols (**ro snapshots**) or generic paths (**tarball snapshots**).
- A transaction index is appended at `/var/lib/nos/snapshots/index.json`.
- Retention: newest **N=5** per target by default (daily prune timer).

See [docs/updates.md](docs/updates.md) for workflow and CLI.

---

## ISO Build
Build a bootable ISO (Debian live) with NithronOS preinstalled.

    # 1) Build local .debs (to dist/deb)
    bash packaging/build-all.sh

    # 2) Build ISO (stages local debs automatically)
    sudo bash packaging/iso/build.sh packaging/iso/local-debs

    # Output -> dist/iso/nithronos-<tag>-<date>-amd64.iso

> **UEFI / Secure Boot:** The preview ISO is not shim-signed yet. **Disable Secure Boot** in your VM/UEFI (e.g., Hyper-V Gen2 → uncheck *Enable Secure Boot*). If your platform can’t disable it, use a legacy/BIOS VM for now.

At first boot the system prints the UI URL + one-time OTP to the console.

Boot menu provides **NithronOS Live**, **Install NithronOS (Debian Installer)**, and **failsafe** entries. The installer is functional but lightly branded; full nOS installer UX will come later.

> **Hyper-V:** Use “NithronOS Live”. The “Kernel fallback (no ACPI)” entry is **not** for Hyper-V; use “safe graphics” instead if you only need to disable GPU modesetting.

---

## Firewall (LAN-only by default)
NithronOS ships an **nftables** policy exposing the web UI (443) to LAN (RFC1918) only.

    sudo bash deploy/firewall/apply.sh
    sudo systemctl enable --now nftables.service

This:
- Loads `deploy/firewall/nos.nft`
- Sets **default-deny** on input
- Allows loopback + established/related
- Allows TCP **443** (web UI) and **22** (SSH) **from LAN only**  
  (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`)

> **Remote access is opt-in.** For Internet access (VPN/Tunnel/Direct), use the **Remote** wizard (coming soon): enforce 2FA, add rate limits, update firewall atomically, keep rollback backups.

Revert manually:

    sudo nft flush ruleset
    sudo systemctl restart nftables.service

**Optional brute-force protection (fail2ban)**  
Filters/jails under `deploy/fail2ban/`; see comments inside for enabling.

---

## Repository Structure

    /backend/nosd          # Go API server
    /agent/nos-agent       # Privileged helper (Unix socket)
    /web                   # React + TypeScript dashboard
    /packaging/deb         # Debian packaging for nosd/agent/web
    /packaging/iso/debian  # Debian live-build profile (installer ISO)
    /scripts               # CI/build/release tools, helpers
    /docs                  # Architecture notes, guides

---

## Roadmap

### Pre-M1 (Hardening & Quality)
- [x] Run nosd as system user + hardened systemd unit (least-privilege).
- [x] Robust auth storage (atomic writes / optional SQLite).
- [x] Session/refresh hardening (server-side session IDs, rotate refresh, reuse detection).
- [x] Rate limits persisted across restarts + proxy awareness (X-Forwarded-For when trusted).
- [x] API versioning `/api/v1`, typed error shape, initial OpenAPI spec.
- [x] Config system `/etc/nos/config.yaml` + env overrides + safe hot reload (SIGHUP).
- [x] Observability: request IDs, structured logs, `/metrics` (Prometheus), gated `/debug/pprof`.
- [x] Security headers (CSP, HSTS, Referrer-Policy, X-Content-Type-Options, COOP/COEP).
- [x] Frontend resilience & a11y (CSRF/refresh guard, toasts, focus, contrast).
- [x] CI upgrades (linters, govulncheck, TS check, stronger ISO smoke).
- [x] Packaging polish (sysusers/tmpfiles/postinst/postrm, console OTP visibility).
- [x] Agent ↔ daemon trust bootstrap (token/mTLS, rotation).
- [x] Recovery paths (console/TUI: reset admin/2FA, `nos.recovery=1`).
- [x] Threat model doc + fuzz/property tests.

### Milestones to v1
- [x] **M1 — Storage Foundation (Btrfs + Health)**: create/import, SMART, scrub/repair, schedules, device ops, destroy, support bundle. (complete)
- [x] **M2 — Shares & Permissions**: SMB/NFS with simple ACLs, guest toggle, recycle bin, Time Machine (fruit). (complete)
- [x] **M3 — App Catalog v1 (Docker/Compose)**: one-click apps, lifecycle, health checks, pre-snapshot + rollback. (complete)
- [x] **M4 — Networking & Remote**: Remote Access Wizard (LAN-only, WireGuard, reverse tunnel), HTTPS (LE), plan/apply/rollback firewall with 2FA for non-LAN. (complete)
- [ ] **M5 — Updates & Releases**: signed packages, channels (stable/beta), atomic upgrades (snapshot safety net).
- [ ] **M6 — Installer & First-boot++**: guided disk install (Btrfs subvols), hostname/timezone/network, telemetry opt-in.
- [ ] **M7 — Backup & Replication**: scheduled snapshots + retention; send/receive (SSH), rclone, restore wizard.
- [ ] **M8 — Monitoring & Alerts**: dashboard (CPU/RAM/IO), SMART/temps, scrubs, service health, notifications (email/webhook/ntfy).
- [ ] **M9 — Security Hardening (User-facing)**: account mgmt UI, password reset, audit log UI, session list & revoke.
- [ ] **M10 — Extensibility & API**: `nosctl` CLI, scoped API tokens, app-template SDK.
- [ ] **M11 — QA, CI, Docs (v1 Gate)**: ISO boot + HTTP/SSH/Btrfs E2E, UI E2E (Playwright), N-1→N upgrade tests, full docs site.

---

## Docs

### User Guides
- Shares & Permissions → [docs/user/shares-permissions.md](docs/user/shares-permissions.md)

### Administration
- API versioning & typed errors → [docs/api/versioning-and-errors.md](docs/api/versioning-and-errors.md)  
- App Catalog user guide → [docs/apps/catalog.md](docs/apps/catalog.md)  
- App runtime architecture → [docs/apps/runtime.md](docs/apps/runtime.md)  
- Certificates & HTTPS configuration → [docs/admin/certificates.md](docs/admin/certificates.md)  
- Network shares (SMB/NFS/Time Machine) → [docs/admin/shares.md](docs/admin/shares.md)  
- Networking & Remote Access → [docs/networking.md](docs/networking.md)  
- Config & hot reload → [docs/dev/config-and-reload.md](docs/dev/config-and-reload.md)  
- Recovery mode → [docs/dev/recovery-mode.md](docs/dev/recovery-mode.md)  
- Updates & rollback → [docs/updates.md](docs/updates.md)  
- Storage pools (device add/remove/replace, destroy, mount options) → [docs/storage/pools.md](docs/storage/pools.md)  
- Storage health (SMART alerts & thresholds, schedules, fstrim) → [docs/storage/health.md](docs/storage/health.md)  
- Pre-Alpha Recovery Checklist → [RECOVERY-CHECKLIST.md](RECOVERY-CHECKLIST.md)

---

## Contributing
We welcome issues and PRs! Please read:
- [CONTRIBUTING.md](CONTRIBUTING.md)
- [SECURITY.md](SECURITY.md)

All contributions are made under the **NithronOS Community License (NCL)**; see [LICENSE](LICENSE).

---

## Licensing & Commercial Use
- **Source code:** [LICENSE](LICENSE) (NithronOS Community License — non-commercial, source-available)  
- **Commercial & MSP terms:** [COMMERCIAL.md](COMMERCIAL.md)  
- **Official builds (ISOs/packages/updates):** [BINARIES-EULA.md](BINARIES-EULA.md)  
- **Trademarks:** [TRADEMARK_POLICY.md](TRADEMARK_POLICY.md)

> TL;DR — read/modify/contribute freely; selling/hosting/redistributing binaries requires a commercial agreement.

---

## Branding
Colors align with Nithron’s palette (dark UI, electric blue `#2D7FF9`, lime `#A4F932`).  
“Nithron”, “NithronOS”, and “nOS” are trademarks of Nithron — see [TRADEMARK_POLICY.md](TRADEMARK_POLICY.md).

---

## Contact
General: hello@nithron.com  
Commercial: licensing@nithron.com  
Security: security@nithron.com
