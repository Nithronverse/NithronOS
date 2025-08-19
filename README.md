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

```bash
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
