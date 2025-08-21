# CI

## Workflows
- Lint: golangci-lint, staticcheck, govulncheck, web typecheck + audit
- Build & Test: go build/test, web build, fuzz/property tests
- Packaging: build .deb, ISO
- Smoke: QEMU boot, health check; always upload logs

## Triggers
- Push to `main`
- Pull Request to `main`
- Tag (release): produce artifacts and GitHub Release

## Artifacts & logs
- Upload `/tmp/nos-serial.log` and `/tmp/qemu.log` for smoke
- Package artifacts under workflow run artifacts
