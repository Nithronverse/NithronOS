# NithronOS (nOS)
[![CI](https://github.com/NotTekk/NithronOS/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/NotTekk/NithronOS/actions/workflows/ci.yml)

**Open-source Linux-based OS for NAS & homelabs.**  
Local-first storage management (Btrfs/ZFS*), snapshots, shares, backups, and a modern web dashboard with an optional app catalog — all without cloud lock-in.

> **Status:** Pre-alpha. Expect rapid changes.

---

## Why NithronOS?
- **Local-first, privacy-first** — admin UI served on your LAN by default; remote access is opt-in.
- **Btrfs-first** (snapshots, send/receive); optional ZFS via DKMS*.
- **One-click apps** via Docker/Compose (Plex/Jellyfin, Nextcloud, Immich, etc.).
- **Real safety features** — dry-run plans for destructive ops, pre-update snapshots, easy rollback.
- **Clean UX** — modern React dashboard, clear health/status, sensible defaults.

\* ZFS availability depends on platform licensing constraints.

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
- [ ] Disk discovery & health (lsblk, smartctl)
- [ ] Btrfs pool create/import, snapshots, send/receive
- [ ] SMB/NFS shares with simple ACLs
- [ ] App catalog (Docker/Compose) with one-click install
- [ ] Snapshot-before-update & rollback
- [ ] Installable ISO (Debian base), first-boot wizard
- [ ] Remote Access Wizard & Firewall Toggle — plan → confirm → apply → rollback; modes: **lan-only** (default), **vpn-only**, **tunnel**, **direct**. Require 2FA for non–lan-only, back up current ruleset before apply, and expose controls under **Settings → Remote**.

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
