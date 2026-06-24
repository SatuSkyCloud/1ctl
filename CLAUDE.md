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

- `cmd/1ctl/main.go` — entrypoint, command grouping, global flags
- `internal/commands/cmd.go` — re-exports all command sub-packages for a flat API surface (`commands.PricingCommand()`, etc.)
- `internal/commands/<group>/` — each command group is an isolated sub-package with three files:
  - `command.go` — flag constants, input structs, CLI wiring (urfave/cli v3)
  - `handlers.go` — action/handler logic, delegates to `internal/api/` or `internal/deploy/`
  - `command_test.go` — unit tests for the command tree structure
- `internal/api/` — API client functions and response structs (one file per resource)
- `internal/api/client.go` — shared HTTP client, `makeRequest()`, `LoginCLI()`, auth headers
- `internal/deploy/` — deployment orchestrator (build, push, upsert deployment/service/ingress); also `tagexpr.go` for `--machine-tag` expression evaluation
- `internal/config/` — config loading (`SATUSKY_API_URL`, `SATUSKY_DOCKER_API_URL`), `ProjectConfig` with v2 sections
- `internal/context/` — persisted auth state (`~/.satusky/context.json`)
- `internal/docker/` — Docker image build and push
- `internal/validator/` — Dockerfile validation plus health-path validation (`ValidateURLPath()`)
- `internal/utils/` — shared utilities (error types, output formatting)

## Key Patterns

- **API response format**: Backend wraps responses as `{"error": false, "data": {...}}`. Parse via `apiResponse.Data`. Exception: `/users/profile` returns flat (no `data` wrapper).
- **Paginated responses**: Some endpoints return `{"notifications": [...], "total": N}` inside `data`. Try paginated struct first, fall back to flat array.
- **Auth**: All requests send `x-satusky-api-key` header via `makeRequest()`. `LoginCLI()` also sends it.
- **Safe int conversions**: Use `api.SafeInt32()` for all `int` to `int32` conversions (gosec G115).
- **Upsert pattern**: Deploy/service/ingress use upsert endpoints, not separate create/update.
- **CLI framework**: `github.com/urfave/cli/v3` (migrated June 2026). See TEST_REPORT.md for migration verification.
  - `cli.App` → `cli.Command` with `EnableShellCompletion: true`
  - Actions/handlers take `(ctx context.Context, cmd *cli.Command) error`
  - `Before` returns `(context.Context, error)` — must pass ctx through
  - `EnvVars` → `Sources: cli.EnvVars(...)`
  - `Destination` binding used on string/int flags for direct population
  - `IsSet` no longer available; track user-set flags manually or use `cmd.String()`
- **Sub-package pattern**: Every command group lives in `internal/commands/<name>/` with `command.go` (CLI wiring) + `handlers.go` (logic) + `command_test.go`. `internal/commands/cmd.go` re-exports all public constructors (e.g., `AppCommand()`, `PostgresCommand()`). Import the sub-package directly only if you need its internal types.
- **Positional args**: 24 commands use `cmd.Args().First()` instead of `--id`/`--name` flags. Pattern: `1ctl <resource> <action> <target> [flags]`. UUIDs are auto-detected via `looksLikeUUID()`; names resolve via app-label lookup. Fallback `--deployment-id`/`--id` flags exist for backward compatibility.
- **Command grouping**: `main.go` uses a `cat()` helper to assign each command to a category (Core workflow, Applications, Data, Infrastructure, Catalog, Account, Billing & operations). External/internal commands (service, ingress, issuer) are ungrouped/hidden.
- **Post-deploy smoke testing**: After `deploy --wait` reports workload healthy, `reportDeployResult()` probes the public URL. Default (no `--health-path`): 401/403/404 are accepted as proof of platform reachability (DNS/TLS/routing worked). With `--health-path` set: only 2xx/3xx pass. Non-strict smoke failures are warnings; strict failures block the deploy. `health_path` should be set in `satusky.toml` `[checks]` section or via `--health-path` flag. Legacy `[app].health_path` still works (auto-migrated by `Normalize()`). Path validated by `ValidateURLPath()` — must start with `/`.
- **Config v2 sections**: `satusky.toml` has been restructured:
  - `[app]` — identity and resources (name, port, cpu, memory, domain, zone)
  - `[build]` — Dockerfile path and `fast_build` opt-in
  - `[checks]` — `health_path` for smoke testing
  - `[deploy]` — strategy, rolling update params, `machine_tag`, `wait_for`
  - `[env]` — non-sensitive environment variables (secrets are managed via `1ctl secret`)
  - `Normalize()` auto-migrates legacy `[app].*` fields to the new sections on load. Downstream code always reads from the canonical v2 sections.
- **Machine tag expressions**: `--machine-tag` accepts Go AST expressions via `EvaluateTagExpr()` in `internal/deploy/tagexpr.go`. Syntax: `key` (label exists), `key=value` (exact match), `&` (AND), `|` (OR), `(group)`. Keys without `satusky.com/` prefix are automatically prefixed.
- **`DeployInput` struct**: All deploy flags are captured into a single `DeployInput` struct with `Destination` binding. `mergeConfig()` combines CLI flags with `satusky.toml` config, tracking which flags were user-set via `UserSetFlags` map.
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
- -o json — most list commands support it. Exceptions (table-only):
  - marketplace list, user me, user permissions, issuer list, credits usage
  - pricing list -o json returns error while table mode succeeds
- --app flag — supported on deploy subcommands, volumes list, logs, doctor, secret
  - NOW also supported on env list / env unset (via `ResolveDeploymentID`)
- user info subcommand does not exist — use user me
- notifications delete does not support -y flag (deletes immediately with positional `<id>`)
- Full JWT token exposed in token list -o json output — security concern
- `app` and `deploy` share the same handler code in `internal/commands/deploy/`. `AppCommand()` wires the same handlers as `DeployCommand()` but with a different top-level name. New subcommands should be added to both trees.
- New command sub-packages must follow the three-file pattern: `command.go` (CLI wiring), `handlers.go` (logic), `command_test.go`. Re-export via `internal/commands/cmd.go`.
- `urfave/cli/v3` `IsSet` is not available — use `cmd.String("flag")` to check if a flag was provided (returns empty string if not set for string flags). For deploy merge logic, track user-set flags manually via `UserSetFlags`.
- See TEST_REPORT.md for full test results and known issues
