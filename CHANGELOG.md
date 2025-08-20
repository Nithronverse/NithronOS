# Changelog

## v0.1.0 — First public preview
- Disk discovery & SMART
- Btrfs: create/import, snapshots; basic usage reporting
- Shares: SMB/NFS with simple ACLs + UI wizard, SMB user mgmt
- App catalog: Docker/Compose one-click installs
- Firewall: LAN-only default; Remote Access wizard (vpn/tunnel/direct) with 2FA + rollback/backup
- Updates: snapshot-before-update, rollback UI/API, retention (keep 5)
- ISO build workflow + QEMU smoketest
- Tests across backend/agent/web

Known limits: Btrfs send/receive; rootfs rollback; per-app pre/post hooks; advanced retention policies.
