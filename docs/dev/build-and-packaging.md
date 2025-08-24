# Build & Packaging

## Build (dev)
```bash
# backend
make api-dev
# agent
make agent-dev
# web
make web-dev
```

## Debian packages
- Sysusers and tmpfiles are included in `nosd` package
- `postinst` chowns `/etc/nos` and `/var/lib/nos` to `nos:nos`
- `postrm` removes state only on purge

### Web reverse proxy (Caddy)

For pre-alpha LAN preview, the default Caddyfile serves HTTP-only and proxies the API. TLS snippets remain commented for later.

Installed to `/etc/caddy/Caddyfile`:

```
{
  admin localhost:2019
  auto_https off
}

:80, nithron.os:80 {
  encode gzip zstd
  header {
    X-Content-Type-Options "nosniff"
    Referrer-Policy "no-referrer"
    Cross-Origin-Opener-Policy "same-origin"
    Cross-Origin-Embedder-Policy "require-corp"
  }
  @api path /api/*
  handle @api { reverse_proxy 127.0.0.1:9000 }
  handle {
    root * /usr/share/nithronos/web
    try_files {path} /index.html
    file_server
  }
}
```

Notes:
- Enable service: `systemctl enable --now caddy`
- Optional: add `nithron.os` to your hosts file pointing at the device IP

### Deps
- Runtime tools installed by default via meta `nithronos`:
  - btrfs-progs, smartmontools, cryptsetup, util-linux, coreutils, findutils
- Suggested (pulled on ISO, optional in package mgr):
  - mdadm, lvm2
- Recommended:
  - nvme-cli (NVMe SMART/health)
- On first boot, a oneshot unit logs detected versions to the journal.

### Runtime dependencies
- btrfs-progs: filesystem creation and management
- smartmontools: disk health (SMART)
- cryptsetup: LUKS encryption support
- util-linux: core block tools (lsblk, mount)
- coreutils, findutils: base utilities used by scripts
- nvme-cli (Recommended): extended NVMe health/maintenance

Build `.deb` outputs under `dist/deb`:
```bash
bash packaging/build-all.sh
```

## ISO build
```bash
sudo bash packaging/iso/build.sh packaging/iso/local-debs
```

### Device integration tests (optional, local-only)
- A gated test exists behind the `devdevice` build tag. It creates a sparse loopback device and formats/mounts it.
- Requirements: Linux, run as root, `losetup` and `mkfs.btrfs` installed, and environment `NOS_DEVICE_TESTS=1`.
- Example invocation:
```bash
cd backend/nosd
sudo env NOS_DEVICE_TESTS=1 go test -tags devdevice ./internal/storage -run LoopDeviceCreateSingle -v
```

## System user and permissions
- `nosd` runs as `nos` system user
- Systemd unit uses hardened settings and restricts write paths to `/etc/nos` and `/var/lib/nos`
