# CLAUDE.md

## Project

1ctl is the CLI client for SatuSky Cloud Platform. It communicates with the backend API at `https://api.satusky.com/v1/cli`.

## Build & Test

```bash
# Build
go build -o 1ctl ./cmd/...

# Run tests
go test ./...

# Run specific test
go test ./internal/commands/ -run TestDeploy

# Lint (gosec is enforced in CI)
gosec ./...
```

## Architecture

- `cmd/1ctl/main.go` — entrypoint
- `internal/commands/` — CLI command handlers (one file per command group)
- `internal/api/` — API client functions and response structs (one file per resource)
- `internal/api/client.go` — shared HTTP client, `makeRequest()`, `LoginCLI()`, auth headers
- `internal/deploy/` — deployment orchestrator (build, push, upsert deployment/service/ingress)
- `internal/config/` — config loading (`SATUSKY_API_URL`, `SATUSKY_DOCKER_API_URL`)
- `internal/context/` — persisted auth state (`~/.satusky/context.json`)
- `internal/docker/` — Docker image build and push
- `internal/validator/` — Dockerfile validation plus health-path validation (`ValidateURLPath()`)
- `internal/commands/doctor.go` — `1ctl doctor` diagnostic command (auth, zones, clusters, deployments, smoke)

## Key Patterns

- **API response format**: Backend wraps responses as `{"error": false, "data": {...}}`. Parse via `apiResponse.Data`. Exception: `/users/profile` returns flat (no `data` wrapper).
- **Paginated responses**: Some endpoints return `{"notifications": [...], "total": N}` inside `data`. Try paginated struct first, fall back to flat array.
- **Auth**: All requests send `x-satusky-api-key` header via `makeRequest()`. `LoginCLI()` also sends it.
- **Safe int conversions**: Use `api.SafeInt32()` for all `int` to `int32` conversions (gosec G115).
- **Upsert pattern**: Deploy/service/ingress use upsert endpoints, not separate create/update.
- **CLI framework**: `github.com/urfave/cli/v2`.
- **Post-deploy smoke testing**: After `deploy --wait` reports workload healthy, `reportDeployResult()` probes the public URL. Default (no `--health-path`): 401/403/404 are accepted as proof of platform reachability (DNS/TLS/routing worked). With `--health-path` set: only 2xx/3xx pass. Non-strict smoke failures are warnings; strict failures block the deploy. `health_path` can be set in `satusky.toml` `[app]` or via `--health-path` flag. Path validated by `ValidateURLPath()` — must start with `/`.
- **Logs metadata**: `GetStoredLogs` returns `(*DeploymentLogsMeta, error)` alongside logs. Check `meta.Degraded` for Loki unavailability; surface `FallbackReason`/`FallbackSource` to the user. The `logs` command prints a warning when Loki is degraded and response fell back to stored deployment logs.
- **Doctor command**: `--deployment-id`/`--config` = targeted mode (always runs smoke). `--smoke` = opt-in for namespace-wide smoke. `--health-path` enforces strict 2xx/3xx in doctor smoke too. Smoke failures in doctor are always warnings, never hard errors.

## Backend API Contract

Routes are at `/v1/cli/*`. The backend uses Fiber with these param conventions:
- Route params: `:orgId`, `:userId`, `:tokenId` (camelCase)
- `helpers.ParseUUID(c, "orgId")` — param name must match route exactly (case-sensitive)
- `InternalAccessMiddleware` requires `x-satusky-api-key` header for remote access

## Releasing

Push a semver tag to trigger GoReleaser:

```bash
git tag v0.X.Y && git push origin v0.X.Y
```

Update `RELEASE_NOTES.md` and the GitHub release description after.

Every tag produces a single `1ctl` binary family. Defaults to `https://api.satusky.com/v1/cli`. Published to brew (`satuctl`).

Non-prod environments are reached via named profiles, not a separate binary:

```bash
1ctl profile create --url <env-api-url> staging
1ctl profile use staging
```

Resolution order at runtime (highest wins): `--api-url` flag → `SATUSKY_API_URL` env → active profile URL → baked-in default.

## Gotchas

- `github` command was removed in v0.5.3 — don't re-add it
- HPA + VPA with mode `Auto` conflict — validate and reject in CLI
- `deploy create` subcommand was removed in v0.2.0 — use `deploy` directly
- Backend struct field names often differ from what you'd expect (e.g., `user_email` not `email`, `token_id` not `id`) — always check backend model JSON tags before adding/changing structs
- `--wait` flag does NOT exist on `deploy rollback` — use `-y`/`--yes` for non-interactive rollback
- `secret list` does NOT accept `--config` flag
- `-o json` is NOT wired for `ingress list` or `service list` (works for deploy, env, secret, machine, token)
