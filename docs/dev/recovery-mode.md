## Recovery Mode (Admin Access)

Enable by kernel arg `nos.recovery=1` (or environment `NOS_RECOVERY=1`).
In recovery mode, `nosd` exposes localhost-only APIs under `/api/v1/recovery/*`:

- POST `/api/v1/recovery/reset-password` { username, password }
- POST `/api/v1/recovery/disable-2fa` { username }
- POST `/api/v1/recovery/generate-otp` → { otp }

All endpoints require localhost (127.0.0.1/::1) and should be invoked from the console.

Example commands:

```bash
curl -sS -X POST http://127.0.0.1:9000/api/v1/recovery/reset-password \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"NewStrongPassword123!"}'

curl -sS -X POST http://127.0.0.1:9000/api/v1/recovery/disable-2fa \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin"}'

curl -sS -X POST http://127.0.0.1:9000/api/v1/recovery/generate-otp
```

Warnings:
- Physical access implies high trust; anyone with console can use recovery.
- Remove `nos.recovery=1` after use.
- Consider rotating credentials and reviewing audit logs afterwards.


