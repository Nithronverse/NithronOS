# Updates & Rollback — Admin Guide

This guide explains how updates work in NithronOS, how snapshots are taken before updates, and how to roll back safely if needed. It also includes CLI equivalents for automation and troubleshooting tips.

## Configuration: snapshots.yaml

File: `/etc/nos/snapshots.yaml` (dev default: `./devdata/snapshots.yaml`)

Schema (v1):
```yaml
version: 1
targets:
  - id: etc
    path: /etc/nos
    type: tar           # btrfs | auto | tar
    stop_services: []   # optional; services to stop during snapshot
  - id: apps
    path: /opt/nos/apps
    type: auto          # auto → btrfs if possible, else tar
```

Rules:
- `path` must be absolute and exist; invalid or missing paths are skipped gracefully.
- `type: auto` will detect Btrfs (preferred) and fall back to tar.
- `stop_services` is optional; services are restarted after snapshot (success or failure).

## What gets snapshotted

- Btrfs target: a read-only snapshot is created at `path/.snapshots/<timestamp>-pre-update`.
- Tar target: a tarball is created at `/var/lib/nos/snapshots/<slug(path)>/<timestamp>-pre-update.tar.gz` using xattrs/ACLs when available.

## Applying updates

From the Web UI: Settings → Updates → Apply Updates.

CLI equivalent via API:
```bash
# Check
curl -sS -X GET http://127.0.0.1:9000/api/updates/check | jq

# Apply with snapshots (confirm required)
curl -sS -X POST http://127.0.0.1:9000/api/updates/apply \
  -H 'Content-Type: application/json' \
  -d '{"snapshot":true, "confirm":"yes"}' | jq
```

Agent (what happens under the hood on Debian):
- If packages list provided: `apt-get install -y <packages>`
- Else: `apt-get upgrade -y`
- A daily systemd timer prunes old snapshots (keep newest 5 per target).

## Rollback

From the Web UI: Settings → Updates → Previous updates → Rollback.

CLI equivalent via API:
```bash
# Find recent transactions
curl -sS -X GET http://127.0.0.1:9000/api/snapshots/recent | jq

# Rollback a transaction by tx_id (confirm required)
curl -sS -X POST http://127.0.0.1:9000/api/updates/rollback \
  -H 'Content-Type: application/json' \
  -d '{"tx_id":"<your-tx-id>", "confirm":"yes"}' | jq
```

Behavior:
- Btrfs: current subvolume is replaced with a writable clone of the pre-update snapshot.
- Tar: a safety backup is taken, then the tarball is extracted over the target path.

## Retention & pruning

Defaults:
- Keep newest 5 snapshots per target (Btrfs subvolumes and tarballs)
- Daily prune timer: `nos-snapshot-prune.timer`

Manual prune:
```bash
# Via backend (proxied to agent)
curl -sS -X POST http://127.0.0.1:9000/api/snapshots/prune -H 'Content-Type: application/json' -d '{"keep_per_target":5}' | jq

# Direct to agent (Unix socket)
curl --unix-socket /run/nos-agent.sock -sS -X POST http://localhost/v1/snapshot/prune \
  -H 'Content-Type: application/json' -d '{"keep_per_target":5}' | jq
```

## Troubleshooting

- Target not on Btrfs: `type: auto` will fall back to tar. Verify with:
  ```bash
  findmnt -n -o FSTYPE --target /path/to/target
  ```
- Snapshot create failed: check journald logs for nos-agent and filesystem permissions.
- Updates apply failed: see `/var/log/nithronos-updates.log` and apt logs (`/var/log/apt/`).
- Rollback failed: ensure the snapshot exists (`.snapshots` dir for Btrfs or the tarball path), and that there is enough free space.

## Notes
- Pre-update snapshots are best-effort when `snapshot` is enabled.
- For destructive operations or cross-version upgrades, always maintain external backups.
