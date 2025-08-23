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
