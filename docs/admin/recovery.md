# Recovery (Console-only)

Recovery mode enables local-only administrative actions when you have console access.

## Enable recovery
- Add kernel arg `nos.recovery=1` (or set env `NOS_RECOVERY=1`) and reboot.
- In recovery mode, endpoints under `/api/v1/recovery/*` are bound to localhost (127.0.0.1/::1).

## Endpoints
- Reset password:
  ```bash
  curl -sS -X POST http://127.0.0.1:9000/api/v1/recovery/reset-password \
    -H 'Content-Type: application/json' \
    -d '{"username":"admin","password":"NewStrongPassword123!"}'
  ```
- Disable 2FA:
  ```bash
  curl -sS -X POST http://127.0.0.1:9000/api/v1/recovery/disable-2fa \
    -H 'Content-Type: application/json' \
    -d '{"username":"admin"}'
  ```
- Generate one-time setup OTP:
  ```bash
  curl -sS -X POST http://127.0.0.1:9000/api/v1/recovery/generate-otp
  ```

## Safety notes
- Physical access implies high trust; anyone with console can use recovery.
- Remove `nos.recovery=1` after use, rotate credentials as needed, and review audit logs.
