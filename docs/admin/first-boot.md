### First-boot OTP and Recovery

During initial setup, the system generates a one-time setup OTP stored at `/var/lib/nos/state/firstboot.json`. The backend exposes `/api/setup/*` endpoints during first-boot. Once an admin user exists, these endpoints return 410 Gone with a typed error `{ "error": { "code": "setup.complete", "message": "Setup already completed" } }`.

At service start, if in first-boot and the OTP exists and is not expired (15 minutes), the code is logged to the journal and console as:

`First-boot OTP: <code> (valid 15m)`

### HTTPS on first boot (self-signed)

On first boot, a local ECDSA certificate is generated with SANs for `127.0.0.1`, the primary IPv4, and `nithronos.local`. Caddy serves HTTPS using this cert. Browsers will warn because it is self-signed; this is expected. The certificate and key live under `/etc/nithronos/tls/cert.pem` and `/etc/nithronos/tls/key.pem`.

The console shows a banner after services start:

`NithronOS UI → https://<this-ip>/  (self-signed)`

Use the OTP printed above to proceed through Setup.

To regenerate the OTP:

```
rm -f /var/lib/nos/state/firstboot.json
systemctl restart nosd
```

Where the local cert lives: `/etc/nithronos/tls`.

#### Recovery endpoint (localhost only)

For recovery (admin/local only), you can reset first-boot state via:

`POST /api/setup/recover`

Body:

```
{ "confirm": "yes", "delete_users": false }
```

- `confirm` is required and must be `yes`.
- If `delete_users` is true, `/etc/nos/users.json` is removed. Use with care.

Access is restricted to localhost (127.0.0.1/::1).

#### Setup state rules

- Setup is considered complete only if the users database loads and contains at least one user (admin).
- If the users file is missing, empty, or invalid, setup remains in first-boot.
- OTP is considered required if `firstboot.json` contains a valid non-expired code.

# First Boot (Admin)

On first boot, `nosd` ensures a one-time 6‑digit OTP exists and prints it to both console and logs on every `nosd` start while in first‑boot mode (unit uses `StandardOutput=journal+console`). The web UI detects setup state and routes to `/setup`.

## Flow
1) OTP: Enter the OTP (valid ~15 minutes). The UI receives a temporary setup token (memory only).
2) Create Admin: Choose a username and a strong password.
3) Optional 2FA: Enroll TOTP, confirm a 6‑digit code, and save recovery codes securely.
4) Finish: You can now sign in at `/login`.

After the first admin is created, `/api/setup/*` endpoints return `410 Gone`.

Regenerate OTP:
- Delete `/var/lib/nos/state/firstboot.json` and restart `nosd`. A new OTP will be generated, saved atomically, and printed again on startup.

## Data paths
- Users: `/etc/nos/users.json`
- Secrets: `/etc/nos/secret.key` (32 bytes, 0600)
- First‑boot state: `/var/lib/nos/state/firstboot.json`

## Reset (dev/recovery)
To rerun first‑boot on next start, remove the users DB:
```bash
sudo rm -f /etc/nos/users.json
```
Reboot `nosd` and repeat the OTP + admin creation.
