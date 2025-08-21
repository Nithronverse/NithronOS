# Branching & Release

## Trunk-based flow
- Feature branches → PR to `main` (protected)
- CI runs lint, tests, packaging, smoke; artifacts on PR
- Merge to `main` produces fresh artifacts
- Tag from `main` creates a Release with packaged artifacts
- No long-lived dev branches
