# NithronOS (nOS)

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

---

## Quickstart (Dev)
> Requires: Go 1.22+, Node 20+, npm, make (optional), Docker (for app catalog dev)

~~~bash
# clone
git clone https://github.com/<you>/<repo>.git
cd <repo>

# backend (in one terminal)
cd backend/nosd
go run ./...

# web (in another terminal)
cd web
npm install
npm run dev
~~~

- Backend default: `http://127.0.0.1:9000`  
- Web dev server: `http://127.0.0.1:5173` (proxied to the API in dev)

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

Follow issues & discussions for up-to-date progress.

---

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
