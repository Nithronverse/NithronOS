# NithronOS (nOS)
![NithronOS](./assets/brand/nithronos-readme-banner.svg)

[![CI](https://github.com/NotTekk/NithronOS/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/NotTekk/NithronOS/actions/workflows/ci.yml)
[![Snapshots On Update](https://img.shields.io/badge/Snapshots%20On%20Update-Enabled%20by%20default-2D7FF9)](docs/updates.md)
[![Release](https://img.shields.io/badge/NithronOS-v0.1.0--pre--alpha-yellow)](https://github.com/OWNER/REPO/releases/tag/v0.1.0-pre-alpha)

**Open-source Linux-based OS for NAS & homelabs.**  
Local-first storage management (Btrfs/ZFS*), snapshots, shares, backups, and a modern web dashboard with an optional app catalog — all without cloud lock-in.

> **Status:** v0 public preview.

---

## Why NithronOS?
- **Local-first, privacy-first** — admin UI served on your LAN by default; remote access is opt-in.
- **Btrfs-first** (snapshots, send/receive); optional ZFS via DKMS*.
- **One-click apps** via Docker/Compose (Plex/Jellyfin, Nextcloud, Immich, etc.).
- **Real safety features** — dry-run plans for destructive ops, pre-update snapshots, easy rollback.
- **Clean UX** — modern React dashboard, clear health/status, sensible defaults.

* ZFS availability depends on platform licensing constraints.

---

## High-level Architecture
- **`nosd`** (Go): REST/gRPC API for disks, pools, snapshots, shares, jobs.
- **`nos-agent`** (Go, root): tiny allowlisted helper for privileged actions.
- **Web UI** (React + TypeScript): talks to `nosd` via OpenAPI client.
- **Reverse proxy** (Caddy): TLS, headers, rate limits; backend bound to loopback.
- **Jobs**: systemd timers + lightweight queue for scrubs/snapshots/replication.

**Remote access (opt-in):**
1) VPN (WireGuard/Tailscale) — recommended  
2) Cloudflare Tunnel — no port-forward, requires 2FA  
3) Direct port-forward — forces 2FA + rate limits

> WARNING: Remote modes (Tunnel or Direct) MUST enforce strong 2FA on the admin UI and apply rate limits. Exposing the dashboard without 2FA is unsupported and unsafe.

---

## Quickstart (Dev)
> Requires: Go 1.22+, Node 20+, npm, make (optional), Docker (for app catalog dev)

~~~bash
# clone
git clone https://github.com/<you>/<repo>.git
cd <repo>

# install deps
cd web && npm install && cd ..

# backend (in one terminal)
make api-dev

# agent (optional, separate terminal)
make agent-dev

# web (separate terminal)
make web-dev
~~~

- Backend default: `http://127.0.0.1:9000`  
- Web dev server: `http://127.0.0.1:5173` (Vite)

---

## First Boot & Auth (Setup)

On first boot, `nosd` generates a one-time 6‑digit OTP, logs it, and prints it to the console (unit uses `StandardOutput=journal+console`). The UI calls `/api/setup/state` on load and routes to `/setup` when required.

Setup steps:
- **Step 1: OTP** — Enter the OTP (valid 15 minutes). A temporary setup token is issued; it is kept only in memory by the UI.
- **Step 2: Admin** — Create the first admin (username + strong password). Optionally choose to enable 2FA now.
- **Step 3: 2FA (optional)** — Scan the TOTP QR (or use manual secret), enter a 6‑digit code to verify, and save the shown recovery codes.
- **Step 4: Done** — Sign in at `/login`.

After the first admin is created, all `/api/setup/*` routes return `410 Gone` and the normal login applies.

### Credentials location and reset
- Users database: `/etc/nos/users.json` (versioned schema).  
  Remove this file to rerun the first-boot setup on next start (recovery).  
  Secret key for encryption/cookies: `/etc/nos/secret.key` (32 bytes; `0600`).
- First‑boot state: `/var/lib/nos/state/firstboot.json` (stores the temporary OTP and usage).

### Security defaults
- Passwords: **Argon2id** (PHC format) with safe defaults.
- Sessions: httpOnly cookies (`nos_session` 15m, optional `nos_refresh` 7d) and CSRF cookie (`nos_csrf`) with double‑submit (`X‑CSRF‑Token`).
- 2FA: TOTP enrollment/verification with encrypted secrets (XChaCha20‑Poly1305); recovery codes stored as SHA‑256 hashes only.
- Guardrails: in‑memory rate limiting per IP and username, generic auth errors, and temporary account lockout after repeated failures.

---

## Local Dev

~~~bash
# run API with live reload (air/reflex if installed, else plain go)
make api-dev

# run agent (optional)
make agent-dev

# run web dev server
make web-dev

# run both nosd and web concurrently
bash scripts/dev-up.sh

# package .debs (Debian toolchain required)
make package
~~~

> Security: keep `nosd` bound to loopback in dev. For any remote exposure, enforce 2FA and rate limits, and apply the LAN-only firewall by default.

---

## Snapshot retention

NithronOS keeps a rolling set of the newest snapshots per target.

- Agent provides a prune endpoint that keeps the newest N snapshots (default 5) for each target and removes older ones (Btrfs subvolumes and `.tar.gz` snapshots).
- A systemd timer (`nos-snapshot-prune.timer`) runs daily and triggers the prune service.
- You can trigger pruning manually from the API:

```bash
curl -sS --unix-socket /run/nos-agent.sock -X POST http://localhost/v1/snapshot/prune -H 'Content-Type: application/json' -d '{"keep_per_target":5}'
```

Future versions will surface retention settings in the UI.

---

## Updates & Rollback

Pre-update snapshots and rollback are built-in and enabled by default.

- Configuration file: `/etc/nos/snapshots.yaml` (dev default: `./devdata/snapshots.yaml`).
  - Schema (v1):
    - `version: 1`
    - `targets[]`: entries with `{ id, path (absolute), type: "btrfs"|"auto"|"tar", stop_services?: [names...] }`
  - Example defaults in dev: `/etc/nos` (tar), `/opt/nos/apps` (auto), `/srv/apps` (auto if present).
- What gets snapshotted:
  - For `type: auto`, the agent detects if the path is on Btrfs; it creates a read-only subvolume snapshot under `path/.snapshots/<ts>-pre-update`. Otherwise a tarball is created under `/var/lib/nos/snapshots/<slug(path)>/<ts>-pre-update.tar.gz`.
  - Optional `stop_services` can be used to briefly stop services during snapshot (restarted after).
- Apply updates:
  - The backend takes pre-update snapshots for configured targets, then applies Debian updates (`apt-get install -y <pkgs>` or `apt-get upgrade -y`).
  - A transaction record (`tx_id`, packages, targets, result) is appended to `/var/lib/nos/snapshots/index.json`.
- Rollback behavior:
  - For each target in the transaction, the agent restores from the corresponding pre-update snapshot:
    - Btrfs: replaces the current subvolume with a writable clone of the snapshot.
    - Tar: extracts the tarball over the target path (a safety backup of current content is created first).
- Retention:
  - By default, the newest 5 snapshots are kept per target. Older snapshots are pruned daily by `nos-snapshot-prune.timer`.
  - You can trigger pruning manually via the API (see below) or from the UI (Settings → Updates).

See `docs/updates.md` for CLI examples and troubleshooting.

---

## ISO build

You can build a bootable ISO (Debian live) with NithronOS preinstalled.

Prereqs: Debian/Ubuntu with `live-build`.

```bash
cd packaging/iso/debian
sudo ./auto/config
sudo lb build
```

Place local `.deb` artifacts in `config/includes.chroot/root/debs/` to include `nosd`, `nos-agent`, and `nos-web` during build. On first boot, a first-boot service generates TLS certs, enables required services, and prints the UI URL + one-time OTP to the console.

---

## Repository Structure (planned)
~~~text
/backend/nosd          # Go API server (REST/gRPC, OpenAPI)
/agent/nos-agent       # Privileged helper (Unix socket)
/web                   # React + TypeScript dashboard (shadcn/ui)
/packaging/deb         # Debian packaging for nosd/agent/web
/packaging/iso/debian  # Debian live-build profile (installer ISO)
/scripts               # CI/build/release tools, support bundle
/docs                  # Architecture notes, ADRs, guides
~~~

---

## Roadmap (early milestones)

- [x] Disk discovery & health (lsblk, smartctl)
- [x] Btrfs pool create/import & snapshots — basic; send/receive pending
- [x] SMB/NFS shares with simple ACLs — UI wizard shipped; SMB user mgmt included
- [x] App catalog (Docker/Compose) with one-click install
- [x] Snapshot-before-update & rollback
- [x] Installable ISO (Debian base), first-boot wizard
- [x] Remote Access Wizard & Firewall Toggle — plan → confirm → apply → rollback; modes: lan-only (default), vpn-only, tunnel, direct. Require 2FA for non–lan-only, back up current ruleset before apply, controls under Settings → Remote.

## Updates & Rollback

NithronOS takes a **pre-update snapshot** of key data before applying updates, then lets you **roll back** if something breaks.

- Config file: `snapshots.yaml` (system: `/etc/nos/snapshots.yaml`, dev: `./devdata/snapshots.yaml`)
- Modes: **btrfs** read-only snapshot (preferred) or **tar.gz** fallback when not on Btrfs
- Where they go:
  - Btrfs: `<target>/.snapshots/<timestamp>-pre-update`
  - Tar: `/var/lib/nos/snapshots/<target-id>/<timestamp>-pre-update.tar.gz`
- Retention: keep the **last 5** per target (a daily prune timer runs automatically)

**How to use (UI):** Settings → **Updates** → Check → (toggle *Snapshot before update*) → **Apply Updates**.  
To revert: pick a prior transaction and click **Rollback**.

**CLI / API (advanced):**
- Check: `GET /api/updates/check`
- Apply with snapshots: `POST /api/updates/apply { snapshot:true, confirm:"yes" }`
- Rollback: `POST /api/updates/rollback { tx_id, confirm:"yes" }`


Follow issues & discussions for up-to-date progress.

---

## Firewall (LAN-only by default)

NithronOS ships with an **nftables** policy that exposes the web UI (443) only to LAN subnets (RFC1918). To apply it on a fresh Debian install:

~~~bash
sudo bash deploy/firewall/apply.sh
sudo systemctl enable --now nftables.service
~~~

This:
- Loads `deploy/firewall/nos.nft`
- Sets **default-deny** on input
- Allows loopback and established/related
- Allows TCP **443** (web UI) and **22** (SSH) **from LAN only** (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`)

> **Remote access is opt-in.** When you later enable Internet access (VPN/Tunnel/Direct), use the **Remote** wizard in the UI (coming soon). It will enforce 2FA, add rate limiting, update firewall rules safely, and keep a rollback backup.

To revert the ruleset manually:

~~~bash
sudo nft flush ruleset
sudo systemctl restart nftables.service
~~~

### Brute-force protection (fail2ban)

Enable request rate-limiting and fail2ban for auth endpoints in production:

- Caddy logs JSON to `/var/log/caddy/access.log` and rate-limits `/api/auth/*` and the SPA `/login` route.
- fail2ban jail and filter are provided:
  - `deploy/fail2ban/filter.d/caddy-nithronos.conf`
  - `deploy/fail2ban/jail.d/nithronos.local`

Apply and enable fail2ban:

~~~bash
sudo install -d -o caddy -g caddy /var/log/caddy
sudo systemctl daemon-reload
sudo systemctl enable --now caddy
sudo cp -r deploy/fail2ban/* /etc/fail2ban/
sudo systemctl enable --now fail2ban
sudo fail2ban-client reload
sudo fail2ban-client status caddy-nithronos
~~~

## Contributing
We welcome issues and PRs! Please read:
- [`CONTRIBUTING.md`](CONTRIBUTING.md)
- [`SECURITY.md`](SECURITY.md)

All contributions are made under the **NithronOS Community License (NCL)**; see `LICENSE`.

---

## Licensing & Commercial Use
- **Source code:** [`LICENSE`](LICENSE) (NithronOS Community License — non-commercial, source-available)  
- **Commercial & MSP terms:** [`COMMERCIAL.md`](COMMERCIAL.md)  
- **Official builds (ISOs/packages/updates):** [`BINARIES-EULA.md`](BINARIES-EULA.md)  
- **Trademarks:** [`TRADEMARK_POLICY.md`](TRADEMARK_POLICY.md)

> TL;DR: you can read/modify/contribute freely; selling, hosting, or redistributing binaries requires a commercial agreement.

---

## Branding
Colors align with Nithron’s palette (dark UI, electric blue `#2D7FF9`, lime accent `#A4F932`).  
“Nithron”, “NithronOS”, and “nOS” are trademarks of Nithron — see `TRADEMARK_POLICY.md`.

---

## Contact
General: **hello@nithron.com**  
Commercial licensing: **licensing@nithron.com**  
Security: **security@nithron.com**
