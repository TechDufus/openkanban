# CI/CD Workflows

## Release Flow

```
Push to main ─► auto-release.yaml ─► release.yaml ─► GitHub Release + Homebrew
                    │                     │
                    │                     └── goreleaser builds binaries
                    │
                    └── Creates tag (v0.0.X), triggers release.yaml
```

### Triggers

| Workflow | Trigger | Action |
|----------|---------|--------|
| `auto-release.yaml` | Push to `main` with `*.go`, `go.mod`, or `go.sum` changes | Auto-increment patch version, create tag, trigger release |
| `release.yaml` | Tag push `v*` / workflow_dispatch | Build and publish release via goreleaser |

### Manual Release

For minor/major bumps, run `release.yaml` manually via Actions tab:
- Select version type (patch/minor/major)
- Workflow creates tag and publishes release

### Files That Trigger Auto-Release

- `**.go`
- `go.mod`
- `go.sum`

Changes to docs, workflows, README, etc. do **not** trigger a release.
