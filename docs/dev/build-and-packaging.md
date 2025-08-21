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

Build `.deb` outputs under `dist/deb`:
```bash
bash packaging/build-all.sh
```

## ISO build
```bash
sudo bash packaging/iso/build.sh packaging/iso/local-debs
```

## System user and permissions
- `nosd` runs as `nos` system user
- Systemd unit uses hardened settings and restricts write paths to `/etc/nos` and `/var/lib/nos`
