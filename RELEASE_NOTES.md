# Release Notes

## Version 0.8.3 (14-05-2026)

Deploy defaults patch for minimal project configs.

### Bug Fixes

* **`deploy` no longer requires CPU/memory in `satusky.toml`**: `1ctl deploy` now honors the built-in `--cpu 0.5` and `--memory 256Mi` defaults instead of printing help when those fields are omitted from the project config.
  - A minimal config with `[app] name` and `port` can deploy without uncommenting resource defaults.
  - Explicit CLI flags and active `satusky.toml` values still keep the same precedence.

### Tests

* Added regression coverage for deploy help guard behavior with default CPU/memory values.

## Version 0.8.2 (13-05-2026)

Deploy reliability patch for platform-managed apps.

### Bug Fixes

* **Backend-reported DNS readiness before "live" output**: `1ctl deploy` now waits for the control plane's ingress DNS-status endpoint before saying a `*.satusky.com` deployment is live.
  - The CLI carries the ingress ID returned by the deploy flow and polls `/ingresses/{ingress_id}/dns-status`.
  - If DNS is still propagating after the timeout, deploy still succeeds, but the output warns that the app URL is not live yet instead of claiming immediate availability.
  - DNS readiness is now based on backend state instead of the operator workstation's resolver.
* **Multi-arch build output no longer becomes a Kubernetes arch selector**: cloud build results such as `linux/amd64,linux/arm64` now normalize to an empty `target_arch`, avoiding invalid `kubernetes.io/arch` nodeSelector values.
  - Single-arch `amd64`, `arm64`, `linux/amd64`, and `linux/arm64` outputs still map to the expected Kubernetes label values.

### Tests

* Added regression coverage for target-arch normalization.
* Added deploy flow coverage for returning the ingress ID used by DNS readiness polling.

## Version 0.8.1 (12-05-2026)

Internal refactor. No user-visible behaviour change; CLI surface, on-disk file layout, and command exit codes are all identical to v0.8.0.

### Refactor

* **Closes #9** — replace `internal/context` package-level globals (`configDir`, `profileOverride`, `cachedCtx`) with a `*Store` type holding the same state as fields.
  - `Store` is constructed once in package `init()` and exposed as `context.Default()`.
  - All package-level functions (`GetToken`, `SetCurrentNamespace`, `UseProfile`, etc.) are now thin shims around `Default().X()` — the ~80 existing call sites across the codebase keep working unchanged.
  - `SetConfigDir` (the test-only escape hatch that was exported on the public API) is **removed**. Tests now call `context.SetDefault(context.NewTestStore(tempDir))` instead.
  - The race-condition concerns in the original issue are resolved architecturally: each `*Store` carries its own `sync.RWMutex` for cache access; tests that construct their own Store via `NewTestStore` can run in parallel without touching shared state.

### Quality gates

* `task lint` — 0 issues
* `task test` — all packages PASS
* `go test -race ./...` — clean (race detector confirms concurrency safety)
* `gosec ./...` — 0 issues across 66 files
* `task format` / `go vet ./...` — clean

### Migration

None for end users. For contributors:

* Tests that previously called `context.SetConfigDir(dir)` should now call `context.SetDefault(context.NewTestStore(dir))`.
* New test code that wants total isolation can construct a `*Store` via `NewTestStore` and call methods on it directly (`store.GetToken()`) without touching `Default()`.

## Version 0.8.0 (11-05-2026)

Closes 21 of 23 open issues on the repo (#10–#30) plus the in-scope subset of #3. See PR #31 for the full diff.

**Server compatibility:** this release is backward-compatible with the previous API server version. A coordinated server release (`v0.66.0`) ships alongside; the new server is only required to take advantage of the `--zone` wire-format cleanup and admin-controlled signup gating on alternate deployments.

### Breaking Changes

- **Single `1ctl` binary**: the separate `1ctl-dev` variant is retired. Per-environment isolation is now handled by named profiles, not by binary variant. If you previously ran `1ctl-dev`, configure a profile in your `~/.satusky/` directory (`1ctl profile create --url <server-url> <name>`). Existing `~/.satusky-development/` data is not auto-migrated.
- **Memory validator requires `Mi`/`Gi` suffix**: `--memory 512` is now rejected; use `--memory 512Mi`. Bare numbers were being interpreted by Kubernetes as bytes (causing silent OOMKills).
- **`auth login` errors on org-less tokens**: previously wrote a poisoned context silently; now fails at login with a clear message.
- **`org switch` errors on empty namespace**: previously fell back to the org name (an invalid k8s namespace); now fails fast.
- **Implicit owner-machine selection removed**: `1ctl deploy` defaults to managed cloud even when you have registered machines. Pass `--machine <name>` or `--machine-tag <tag>` to target BYOA.
- **`1ctl service` / `1ctl ingress` upsert / `1ctl issuer` hidden from `--help`**: commands still work for scripts; the public surface for these workflows is now `1ctl domains`.

### New commands

- **`1ctl launch`**: interactive wizard. Detects project runtime (Go, Node.js/Bun, Python, Python-Poetry, Rust, Ruby, Java-Maven, Java-Gradle, PHP), suggests CPU/memory/port defaults, writes a minimal `satusky.toml`. `--non-interactive` accepts all defaults.
- **`1ctl domains`** (alias: `domain`): custom-domain management without UUIDs.
  - `1ctl domains add <domain> --app <app>` — auto-picks Let's Encrypt TLS for non-`*.satusky.com` hosts
  - `1ctl domains list`
  - `1ctl domains remove <domain> --app <app> [--yes]` — refuses cross-app removal even when the domain matches
  - `1ctl domains check <domain>` — shows DNS/TLS state from the ingress record
- **`1ctl deploy open`**: opens the deployment's primary URL in the default browser (`open` / `xdg-open` / `cmd /c start`).
- **`1ctl deploy scale --replicas N`**: replica-count shorthand. Refuses to scale HPA/VPA-managed deployments.

### New flags

- **`--name` on `deploy`**: explicit override; precedence is `--name` > `satusky.toml [app] name` > git-remote auto-detect.
- **`--machine-tag <tag>` on `deploy`**: BYOA targeting by Kubernetes node label `satusky.com/<tag>`. The CLI resolves owned-machine IDs client-side via the existing labels API.

### `satusky.toml` v2 schema

Major expansion of the project manifest. Old `[app]` files still parse identically.

- `[app]` adds: `zone`, `organization`, `strategy`, `rolling_max_surge`, `rolling_max_unavailable`, `machine_tag`, `wait_for`.
- `[app]` fixes: `dockerfile` and `replicas` were declared in v1 but silently ignored; they now wire through.
- New sub-sections: `[volume]`, `[hpa]`, `[vpa]`, `[pdb]`, `[multicluster]`. See `1ctl init` for the full template.

Resolution precedence per field: **CLI flag (explicit) > `satusky.toml` > flag `Value:` default**.

### Improvements

- **`profile use` / `profile current`** now show the resolved API URL (full precedence chain) rather than the raw profile field.
- **`deploy get`** no longer panics on untagged images.
- **`deploy list`** gains a `NAME` column as the first column (sourced from `app_label`); `TYPE` column dropped.
- **`WaitForDeployment`** treats unknown statuses as non-terminal — keeps `--wait` forward-compatible with new backend status strings.
- **CleanupManager** now tracks env (auto-cleaned on partial-deploy failure) and ingress (auto-cleaned) and registers volumes for audit logs (manual PVC cleanup still required pending backend DELETE support).
- **`UpsertEnvironment` / `CreateSecret`** stop overriding caller-supplied namespace — `--organization` finally routes resources to the right place.
- **Context loads cached** with `sync.Once`; a typical deploy now reads `~/.satusky/...` once instead of ~30 times.
- **`x-satusky-config` header dropped** from API requests; backend `K8sClientMiddleware` already had a DB fallback in production.
- **Repo cleaned up for open-source release**: alternate-deployment URLs removed from `.goreleaser.yml`, `CLAUDE.md`, `CONTRIBUTING.md`, and the user-journey example docs. Anyone running their own SatuSky deployment configures access via a named profile pointing at their server.

### Tests

12 new test files / suites totalling 50+ new test cases. Notable:

- `TestParseV2Schema_*` (8) — toml v2 every section + v1 backwards compat
- `TestBuildStrategyConfig` — explicit-defaults preservation (issue #27 sub-5)
- `TestCaptureUserSetFlags_NotPoisonedByCSet` — locks down the c.Set/c.IsSet trap that broke `RollingFlagsExplicit` pre-review
- `TestDetectRuntime` — all 9 runtimes + unknown-stack fallback + first-match-wins ordering
- `TestMachineHasTag` — label-match predicate for `--machine-tag`

### Migration

1. Drop `1ctl-dev`. A single `1ctl` binary covers every environment. To target a non-production server, configure a profile once: `1ctl profile create --url <server-url> <name>` then `1ctl profile use <name>`.
2. Run `1ctl deploy` with `--memory <N>Mi` not bare numbers.
3. If you relied on implicit owner-machine selection: pass `--machine-tag <tag>` or `--machine <name>`, or add `machine_tag = "..."` to `satusky.toml`.
4. Existing `~/.satusky/context.json` files load unchanged (the deprecated `user_config_key` field is silently dropped).

### Deferred

- **#9 Store struct refactor** — separate follow-up PR.
- **G-01 streaming logs** — needs backend WS endpoint.
- **F-03 sibling-pod `1ctl run`** — needs backend Job spawn.
- **D-05 table output across all list commands, F-06 env diff, T-04 storage handlers** — separate DX-focused PRs.

### Permanently out (Cloud Run security stance)

- **G-02 SSH/exec into containers**, **F-02 port-forwarding**, **F-01 managed databases**.

### Also bundled (development work between v0.7.3 and the bundle PR)

These commits had merged to the `development` branch between v0.7.3 and the cut of PR #31 but were never tagged for a release. They ship in 0.8.0:

- **`feat: implement 6 CLI gaps`** (5b0a215). `logs stream --config` flag; `init` writes only non-empty fields (cleaner template); new global `--output/-o json` flag for `deploy list/get/status`, `env list`, `secret list`, `machine list`, `token list`; new `--wait/-w` on `deploy` that polls until pods are Running; new `env unset` / `secret unset` for per-key removal (replacing the old wholesale `env delete` / `secret delete`).
- **`feat: arch-aware cloud build routing`** (7e3aadb). The build pipeline now detects the produced image's CPU architecture and the orchestrator filters owner machines whose `CPUArch` doesn't match. The backend sets `nodeSelector: {"kubernetes.io/arch": <arch>}` on the pod spec so cross-arch deploys can't land on incompatible nodes. Also fixes "no existing environment/secret found" on the first `env create` / `secret create` against a fresh deployment.
- **`feat(config): name-based deployment resolution (Fly.io style)`** (3e811b3). Removed `deployment_id` from `satusky.toml` — the app `name` is the identifier; the CLI resolves the deployment via `GET /deployments/namespace/:namespace/app/:appLabel` at command time. Add `internal/commands/resolve.go` as the single resolution helper.
- **`feat(config): org field removed; platform defaults for cpu/memory`** (6d9aea4). `org` no longer in `AppConfig` — the namespace comes from the auth context. `--cpu` / `--memory` get platform defaults (`0.5` / `256Mi`) instead of hard-erroring when neither flag nor toml supplied them.
- **`fix(deploy): DNS-1035 app name validation`** (ddf029d). `validateAppName` rejects names that wouldn't be valid Kubernetes Service names BEFORE any resource is created.
- **`fix+chore: remove analytics/billing/wholesale-delete commands`** (db4cf7a). Removed: `logs stats`, `logs delete`, `audit export`, `credits topup`, `credits invoices`, `credits auto-topup`, `credits notifications` (web-dashboard concerns), `env delete`, `secret delete` (wrong semantics — use `env unset` / `secret unset`). Also fixed two backend-route mismatches in `user permissions` and `token create/state` paths.
- **`fix(domain): delegate domain generation to backend`** (d43986b). Removed client-side domain generation; the backend is the single source of truth for `*.satusky.com` allocation. Avoids drift between CLI guesses and backend reality.
- **`fix(deploy): domain in deploy get -o json + configmap env dedup`** (8f49f04). Closes a JSON-output gap and a duplicate-env-var bug in the env ConfigMap upsert.
- **`chore(cli): remove admin/talos/domain/storage/machine vm for v1`** (f07d3f8). Top-level commands that didn't belong in a user-facing CLI surface for v1. Note: my v0.8.0 work brings `domains` back (different scope — custom-domain attachment, not the legacy purchase flow).
- **`feat: architectural hardening — HTTP safety, auth, deploy reliability`** (2e0809d). Enforces HTTPS for non-localhost API URLs (anti-token-leakage), tightens auth error paths, and tightens deploy retries / cleanup.
- **`feat: add 1ctl-dev build variant for dev backend testing`** (986b938). Note: this commit is **retired in the same release** by commit `0aa7a37` above — the dev binary lived for ~4 weeks on development before being replaced by named profiles. Net effect for end users: dev binary never existed in a tagged release.
- **Examples + user-journey docs** (2bccaaa, ff67f1f, 0415203, 9dd610c, 3d12e1d, f57df86, 2b6bd88, 5eb094f, 5ad215f). 12 user-journey guides under `examples/user-journeys/` covering real-world deployment scenarios, plus a comprehensive `REPORT.md` walking through 64 commands tested.

These commits live in the `development` branch already and will arrive in `main` together with the bundle when the standing `development → main` PR (#6) merges and `v0.8.0` is tagged. See `git log v0.7.3..v0.8.0 --oneline` after the tag is cut for the complete enumeration.

## Version 0.7.3 (16-04-2026)

> **Superseded:** the v0.7.3 dev-binary variant has been retired. See the [Unreleased] section above for the replacement (named profiles).

### New Features

- **Build variant for dev-backend testing**: shipped a second `1ctl` binary variant that baked an alternate API URL via `-ldflags`. Internal testing only; not published to Homebrew.
- **Credential isolation**: the dev variant used a separate config directory so dev and prod tokens didn't clobber each other.

### Improvements

- **Build-time URL overrides**: `defaultAPIURL` and `defaultDockerUploadURL` in `internal/config/config.go` are now `var` (not `const`) so GoReleaser can override them via `-ldflags`. Runtime `SATUSKY_API_URL` / `SATUSKY_DOCKER_API_URL` env vars still win for per-invocation overrides.
- **`--version` output**: now routes through `GetVersionInfo()` so commit hash and build date show consistently.

### Bug Fixes

- **README token URL**: fixed hallucinated `https://cloud.satusky.com/token` → correct path is `https://cloud.satusky.com/<org-id>/token`.

### Known Issues

- **Homebrew formula not refreshed for v0.7.3**: The `HOMEBREW_TAP_TOKEN` returned 401 during publish. Tarballs on the GitHub release are complete; brew users remain on v0.7.2 until the tap token is rotated and the formula re-published. Unrelated to the dev-binary work.

---

## Version 0.7.2 (06-04-2026)

### Bug Fixes

- **Reject `--multicluster` combined with custom domains**: Combining `--multicluster` with a `--domain` value that doesn't end in `.satusky.com` is now rejected at the client side with a friendly error before any backend round trip. Previously the deployment would succeed superficially but the platform's satusky-operator silently blocked replication of the custom-domain ingress to the secondary cluster, leaving the user with broken HA expectations.
  - The check is case-insensitive and tolerates a leading `*.` wildcard.
  - 9 unit test cases added in `internal/commands/deploy_test.go`.
  - The server (v0.47.2) and Control Panel (v0.11.1) enforce the same constraint.

---

## Version 0.7.1 (06-04-2026)

### Bug Fixes

- **Fix `1ctl cluster zones` and `1ctl cluster list`**: Both commands returned a JSON unmarshal error on v0.7.0 because the API client bypassed the shared `apiResponse` wrapper that the backend returns (`{"error": false, "data": [...]}`). Rewrote `internal/api/cluster.go` to follow the same pattern used by all other API calls.
- **Fix `%!s(MISSING)` in cluster command error output**: `internal/commands/cluster.go` called `utils.NewError("...%s", err)` but `NewError(message, err)` takes a message string and an error, not format args. Fixed to use `fmt.Sprintf`.

### Improvements

- **Comprehensive shell completion refresh**: `1ctl completion {bash,zsh,fish,powershell}` now reflects the full, current command inventory:
  - Added `cluster` command with `zones` and `list` subcommands
  - Added `--zone` flag completion on `deploy` and `marketplace deploy` (pulls live zones from `1ctl cluster zones`)
  - Added missing top-level commands: `domain`, `pricing`
  - Added missing subcommand sets: `domain` (11 subs), `credits` (7 subs), `logs` (3 subs), `pricing` (4 subs), `machine vm` (7 subs)
  - Added flag value completions: `--backup-schedule`, `--backup-retention`, `--pdb-type`, `--vpa-mode`, `--pricing-tier`
  - Removed stale `github` command (removed in v0.5.3) and its stale subcommand completions
  - Removed stale `get`/`create`/`upload`/`download` subcommand completions that no longer exist
  - Fixed `machine` subcommands to match actual CLI (`list`, `available`, `vm`, `usage`)
  - Added a maintenance note on `CompletionCommand` reminding contributors to update all four shell templates

---

## Version 0.7.0 (05-04-2026)

### New Features

- **Multi-cluster zone targeting**: New `--zone` flag on `1ctl deploy` to target a specific deployment zone (e.g., `--zone my-bki-1a` to deploy to BKI cluster). When omitted, the backend auto-selects from all available machines.
  - `1ctl deploy --zone my-bki-1a --cpu 2 --memory 1Gi` deploys to BKI cluster
  - `1ctl deploy --multicluster --zone my-kul-1b` deploys to KUL and replicates to other clusters
  - `1ctl deploy --multicluster` deploys to default cluster (KUL) and replicates to all
- **Cluster management commands**: New `1ctl cluster` command group for viewing cluster and zone information:
  - `1ctl cluster zones` lists all available deployment zones with their cluster mappings
  - `1ctl cluster list` shows all enabled clusters with health status, endpoints, and priority

---

## Version 0.6.1 (01-04-2026)

### 🐛 Bug Fixes

- **Remove hardcoded SG region**: Deployments no longer force `Region: "SG"` and `Zone: "sg-sgp-1"`. Previously, this caused 500 errors when no machines existed in the SG region. The region fields are now left empty so the backend calls `GetMonetizedMachines()` and auto-selects from all available machines regardless of region.

---

## Version 0.6.0 (30-03-2026)

### ✨ New Features

- **`--wait-for` flag for dependency readiness**: Declare TCP dependencies that must be reachable before your app starts. The platform injects init containers server-side, eliminating crash-restart loops when dependencies (postgres, redis, etc.) are slow to start.
  - `1ctl deploy --wait-for postgres:5432 --wait-for redis:6379 --cpu 500m --memory 512Mi --image myapp:latest`
  - Supports multiple `--wait-for` flags for multiple dependencies
  - Validates `host:port` format with port range 1–65535
  - Mirrors flyctl's philosophy: declare what you need, platform handles the mechanism

---

## Version 0.5.13 (26-03-2026)

### ✨ New Features

- **`--image` flag for prebuilt images**: Skip local Docker build and push by supplying a prebuilt image reference directly
  - `1ctl deploy --image myregistry.io/myapp:v1.0 --cpu 100m --memory 256Mi`
  - Useful for CI/CD pipelines where images are built externally
  - When `--image` is provided, Dockerfile validation, Docker build, and Docker push steps are all skipped

- **Deployment ID output**: `deploy` now prints the deployment ID after a successful deploy, making it easier to reference in follow-up commands (`deploy status`, `logs`, etc.)

### 🐛 Bug Fixes

- **Deployment status polling**: Fixed handling of Kubernetes-style status values (`True`/`False`/`Unknown`) from the deployment status endpoint — previously only `Ready`/`NotReady` were recognized, causing status checks to hang or misreport
- **NotReady status handling**: `WaitForDeployment` now correctly detects and reports `NotReady` status instead of timing out silently

---

## Version 0.5.12 (16-03-2026)

### 🚀 Distribution

- **Homebrew tap**: `brew install SatuSkyCloud/tap/satuctl` (installs `1ctl` binary)
- **GitHub Action**: `SatuSkyCloud/setup-1ctl@v1` for CI/CD workflows
- **Install script**: `curl -sSL .../install.sh | bash` for quick installs
- **GoReleaser auto-publish**: Homebrew formula auto-updated on every release
- Simplified README installation and GitHub Actions sections
- Formula named `satuctl` because Homebrew can't handle names starting with a digit

---

## Version 0.5.11 (16-03-2026)

### 🔧 Fixes

- Fixed Homebrew formula class name (Ruby class can't start with a digit)

---

## Version 0.5.10 (16-03-2026)

### 🚀 Distribution (initial)

- Added Homebrew formula, GitHub Action, and install script
- GoReleaser brews config for auto-publish (class name fix pending)

---

## Version 0.5.9 (16-03-2026)

### 🔧 Improvements

- Internal maintenance release

---

## Version 0.5.8 (15-03-2026)

### 🐛 Bug Fixes

- **`org team list`**: Fix JSON field mapping — was showing blank Name/Email and zero UUID
- **`token list`**: Fix JSON field mapping — ID showed as zero UUID, status always "Disabled"
- **`user me`**: Fix endpoint path (`/auth/me` → `/users/profile`) and response struct
- **`notifications list`**: Handle paginated wrapper response, fix field names (`title` → `subject`)
- **`audit list`**: Handle paginated wrapper response, fix field names (`user_email` → `actor_email`)
- **`machine list`**: Fixed machine `owner_id` in prod DB (was nil UUID)

### 🔧 Requires

- Backend v0.44.4+ (fixes credit route parameter name mismatch)

---

## Version 0.5.7 (14-03-2026)

### 🐛 Bug Fixes

- **Fix CLI authentication**: Send `x-satusky-api-key` header in login requests. Backend's `InternalAccessMiddleware` (deployed in v0.44.0) requires this header for remote CLI access. Without it, `1ctl auth login` fails with 401.

---

## Version 0.5.6 (13-03-2026)

### 🔒 Security

- **Safe integer conversions**: All CLI flag value conversions (`int` → `int32`) now use `SafeInt32()` to prevent potential integer overflow (gosec G115)

### 🔧 Improvements

- Includes all features from v0.5.5 (PDB/HPA/VPA CLI flags)

---

## Version 0.5.5 (12-03-2026)

### ✨ New Features

- **PDB CLI Flags**: Configure PodDisruptionBudgets from the CLI
  - `--pdb` — Enable PDB (auto-enabled when replicas > 1)
  - `--pdb-type` — Strategy: `auto` (default), `fixed`, or `percent`
  - `--pdb-min-available` — Minimum available pods (for type=fixed)
  - `--pdb-percent` — Minimum available percentage (for type=percent)

- **HPA CLI Flags**: Configure HorizontalPodAutoscalers from the CLI
  - `--hpa` — Enable HPA
  - `--hpa-min-replicas` — Minimum replicas (default: 1)
  - `--hpa-max-replicas` — Maximum replicas (default: 10)
  - `--hpa-cpu-target` — Target CPU utilization % (default: 80)
  - `--hpa-memory-target` — Target memory utilization % (0 = disabled)

- **VPA CLI Flags**: Configure VerticalPodAutoscalers from the CLI
  - `--vpa` — Enable VPA
  - `--vpa-mode` — Update mode: `Off` (default), `Initial`, or `Auto`
  - `--vpa-min-cpu`, `--vpa-max-cpu` — CPU resource bounds
  - `--vpa-min-memory`, `--vpa-max-memory` — Memory resource bounds

- **Replica Count Override**: `--replicas` flag to set replica count manually instead of deriving from machine count

### 🔧 Improvements

- **Smart PDB Auto-Enable**: PDB automatically enabled when replicas > 1, even without `--pdb` flag
- **Validation**: HPA + VPA with mode `Auto` rejected (resource scaling conflict), HPA min/max bounds checked, PDB percent range validated (1-100)

### 📋 New Flags Summary

| Flag | Default | Description |
|------|---------|-------------|
| `--replicas` | auto | Manual replica count override |
| `--pdb` | false | Enable PodDisruptionBudget |
| `--pdb-type` | auto | PDB strategy (auto/fixed/percent) |
| `--hpa` | false | Enable HorizontalPodAutoscaler |
| `--hpa-min-replicas` | 1 | HPA minimum replicas |
| `--hpa-max-replicas` | 10 | HPA maximum replicas |
| `--hpa-cpu-target` | 80 | Target CPU % |
| `--vpa` | false | Enable VerticalPodAutoscaler |
| `--vpa-mode` | Off | VPA mode (Off/Initial/Auto) |

### 📚 New Command Usage

```bash
# Deploy with HPA auto-scaling
1ctl deploy --cpu 2 --memory 1Gi --hpa --hpa-max-replicas 5

# Deploy with VPA recommendations (read-only mode)
1ctl deploy --cpu 2 --memory 1Gi --vpa --vpa-mode Off

# Deploy with PDB for high availability
1ctl deploy --cpu 2 --memory 1Gi --replicas 3 --pdb --pdb-type percent --pdb-percent 60

# Combined: HPA + VPA in Off mode (recommendations only)
1ctl deploy --cpu 2 --memory 1Gi --hpa --vpa --vpa-mode Off
```

---

## Version 0.5.4 (12-03-2026)

### ✨ New Features

- **`domain purchase-status`**: New subcommand to poll the status of a pending domain purchase intent
- **`domain contact`**: New subcommand to view saved contact info from the last domain purchase

### 🔧 Improvements

- **Domain purchase flow**: `domain purchase` now initiates a Stripe Checkout session — returns a payment URL and intent ID instead of purchasing directly. Complete payment in browser, then use `domain purchase-status <intent-id>` to confirm.

---

## Version 0.5.3 (12-03-2026)

### 🔧 Improvements

- **GitHub integration removed**: `github` command removed — GitHub deployment feature is discontinued
- **`secret delete`**: Added missing `secret delete --secret-id` subcommand
- **`issuer delete`**: Added missing `issuer delete --issuer-id` subcommand
- **Ingress validation**: `--service-id` now gives a clear error when omitted instead of a confusing UUID parse error

### 🔒 Security

- Fixed gosec G204 false positive in Docker `SaveImage` (correct `// #nosec G204` directive)

---

## Version 0.5.2 (12-03-2026)

### ✨ New Features

- **Mac VM Agent Commands**: Full lifecycle control for Mac mini machines via the Mac agent
  - `machine vm status` — Show current VM state
  - `machine vm start/stop/reboot` — Power control commands
  - `machine vm stop --force` — Force stop with optional grace period
  - `machine vm resize --cpu N --memory N` — Resize VM resources
  - `machine vm apply-config --config-file PATH` — Apply Talos configuration (base64 encoded)
  - `machine vm console [--enable|--disable]` — Toggle console streaming

- **Domain Management**: Full lifecycle for external domain registration and DNS via OpenProvider
  - `domain list/get/create/delete/verify` — Domain lifecycle management
  - `domain check` — Check domain availability
  - `domain search` — Search available domains
  - `domain purchase` — Purchase a domain
  - `domain dns list/create/update/delete` — DNS record management

- **Machine Usage Tracking**: View and calculate usage costs for deployed machines
  - `machine usage list` — List all usage records for current user
  - `machine usage get --usage-id <id>` — Get usage record details
  - `machine usage cost --usage-id <id>` — Calculate cost for a usage record

- **Pricing Configuration**: Browse platform pricing information
  - `pricing list` — List all pricing configurations
  - `pricing get --id <id>` — Get specific pricing config
  - `pricing lookup --region <r> --type <t> --sla <s>` — Look up pricing by region/type/SLA
  - `pricing calculate --machine-ref-id <id> --machine-id <id>` — Calculate machine cost

- **Billing Settings**: Manage auto top-up and notification preferences
  - `credits auto-topup get` — View current auto top-up settings
  - `credits auto-topup set --enabled [--threshold N] [--amount N]` — Configure auto top-up
  - `credits notifications get` — View notification preferences
  - `credits notifications set --low-balance [--email] [--push] [--threshold N]` — Configure notifications

- **Live Log Streaming**: Real-time log streaming via WebSocket
  - `logs stream --deployment-id <id>` — Stream logs using deployment ID
  - `logs stream --namespace <ns> --app <label>` — Stream logs using namespace + app label
  - `logs stream --batch-size N` — Control log lines per batch

### 🔧 Technical Improvements

- **Backend CLI Route Expansion**: Added machine usage, pricing, and billing settings routes to CLI API
- **WebSocket Log Streaming**: Direct WebSocket connection replacing Loki log querying
- **Mac VM Agent Integration**: `POST /machines/:machineId/command` now exposed via CLI routes
- **Machine Model Enhancement**: Added `VMState` and `ConnectionMode` fields to Machine struct
- **New `gorilla/websocket` dependency**: WebSocket client for live log streaming

### 📋 New Commands Summary

| Category | Commands Added |
|----------|---------------|
| Machine VM | 7 subcommands |
| Domain | 9 subcommands + 4 DNS subcommands |
| Machine Usage | 3 subcommands |
| Pricing | 4 subcommands |
| Billing Settings | 4 subcommands |
| Logs Streaming | 1 subcommand |

---

## Version 0.5.1 (13-01-2026)

### ✨ New Features

- **Resource Exhausted Error Handling**: Enhanced error handling for resource quota exceeded scenarios
  - Beautiful formatted error display showing current tier limits
  - Clear guidance on which resources are exhausted (CPU, Memory, Pods, etc.)
  - Automatic tier upgrade suggestions when applicable
  - Support for displaying next tier limits and requirements

- **Tier Info Display**: Added tier information to credits commands
  - `credits balance` now shows current tier and limits
  - Display of highest achieved tier (peak tier)
  - Credits required to reach next tier
  - Current resource limits per tier

### 🔧 Technical Improvements

- **Resource Error Utilities**: New `internal/utils/resource_error.go` module
  - `ParseResourceExhaustedError()` - Parse API error responses for resource exhaustion
  - `FormatResourceExhaustedError()` - Beautiful CLI formatting for resource errors
  - Comprehensive test coverage with `resource_error_test.go`

- **Deploy Command Enhancement**: Improved error handling during deployment
  - Detects resource exhausted errors from API
  - Displays actionable upgrade guidance
  - Shows current vs required resources

- **Credits API Integration**: Enhanced credits balance endpoint integration
  - Added `TierInfo` struct with tier details
  - Support for tier limits display
  - Upgrade path information

---

## Version 0.5.0 (09-01-2026)

### ✨ New Features

- **Credits & Billing Management**: Complete billing and credits integration
  - `credits balance` - View organization credit balance
  - `credits transactions` - View transaction history
  - `credits usage` - View machine usage and costs
  - `credits topup` - Initiate credit top-up via Stripe
  - `credits invoices` - Manage invoices (list, get, download PDF, generate)

- **Storage Management (S3)**: Full S3-compatible object storage support
  - `storage list/get/create/delete` - Manage storage configurations
  - `storage buckets` - Bucket management operations
  - `storage files/upload/download` - File operations
  - `storage presign` - Generate presigned URLs
  - `storage usage` - View storage usage statistics

- **Logs Command**: Deployment log viewing and streaming
  - `logs --deployment-id` - View stored deployment logs
  - `logs --follow` - Stream logs in real-time (WebSocket)
  - `logs --stats` - View log statistics
  - `logs --tail` - Limit number of lines

- **GitHub Integration**: Complete GitHub OAuth and repository management
  - `github status` - Check GitHub connection status
  - `github connect/disconnect` - Manage GitHub account connection
  - `github repos` - List, sync, and get repository details
  - `github installation` - Manage GitHub App installation

- **Notifications**: In-app notification management
  - `notifications list` - List notifications with filtering
  - `notifications count` - Get unread notification count
  - `notifications read` - Mark notifications as read
  - `notifications delete` - Delete notifications

- **Marketplace**: Browse and deploy pre-configured applications
  - `marketplace list` - Browse available apps
  - `marketplace get` - Get app details
  - `marketplace deploy` - Deploy marketplace apps (WordPress, Immich, N8N, etc.)

- **User Profile Management**: Personal account management
  - `user me` - View current user profile
  - `user update` - Update profile (name, email)
  - `user password` - Change password (interactive)
  - `user permissions` - View role and permissions
  - `user sessions revoke` - Revoke all sessions

- **API Token Management**: Manage API tokens programmatically
  - `token list` - List all API tokens
  - `token create` - Create new tokens with expiry
  - `token get` - Get token details
  - `token enable/disable` - Toggle token state
  - `token delete` - Delete tokens

- **Audit Logs**: View organization audit trails
  - `audit list` - List audit logs with filtering
  - `audit get` - Get audit log details
  - `audit export` - Export logs to JSON/CSV

- **Talos Configuration**: Talos Linux machine configuration
  - `talos generate` - Generate Talos configuration
  - `talos apply` - Apply configuration to machines
  - `talos history` - View configuration history
  - `talos network` - View machine network info

- **Admin Operations**: Super-admin management tools
  - `admin usage` - Manage machine usage records
  - `admin credits` - Add/refund organization credits
  - `admin namespaces` - List all Kubernetes namespaces
  - `admin cluster-roles` - List cluster roles
  - `admin cleanup` - Cleanup resources by label

### 🔄 Enhanced Commands

- **Organization Command Enhancements**
  - `org list` - List all user organizations
  - `org create` - Create new organizations
  - `org delete` - Delete organizations
  - `org team list` - List team members
  - `org team add` - Add team members with role
  - `org team role` - Update member roles
  - `org team remove` - Remove team members
  - Enhanced `org switch` with `--org-name` support

### 🔧 Technical Improvements

- **Backend Route Expansion**: Added 70+ new CLI routes to backend
  - Full DI pattern implementation for CLI controllers
  - Split controllers for better separation of concerns
  - Enhanced authentication middleware

- **API Client Architecture**: Modular API client files
  - `api/credits.go` - Credits and billing operations
  - `api/storage.go` - S3 storage operations
  - `api/logs.go` - Log operations
  - `api/github.go` - GitHub integration
  - `api/notifications.go` - Notification operations
  - `api/marketplace.go` - Marketplace operations
  - `api/audit.go` - Audit log operations
  - `api/talos.go` - Talos configuration
  - `api/admin.go` - Admin operations
  - `api/user.go` - User profile operations
  - `api/token.go` - API token operations
  - `api/org.go` - Organization and team management

- **Command Structure**: 11 new command files with consistent patterns
  - Beautiful output formatting with status lines, headers, dividers
  - Comprehensive error handling
  - Flag validation and help text

### 📚 Documentation

- Updated README.md with all new commands and examples
- Added usage examples for:
  - Credits & billing operations
  - Storage management
  - Log streaming
  - GitHub integration
  - Marketplace deployment
  - Team management
  - API token management
  - Admin operations

### 📋 New Commands Summary

| Category | Commands Added |
|----------|---------------|
| Credits | 9 subcommands |
| Storage | 12 subcommands |
| Logs | 1 command with 4 flags |
| GitHub | 10 subcommands |
| Notifications | 5 subcommands |
| Marketplace | 3 subcommands |
| User | 5 subcommands |
| Token | 6 subcommands |
| Audit | 3 subcommands |
| Talos | 4 subcommands |
| Admin | 8 subcommands |
| Org (enhanced) | 7 new subcommands |

---

## Version 0.4.0 (18-12-2025)

### ✨ New Features

- **Multi-Cluster Deployment**: Deploy applications across multiple Kubernetes clusters for high availability
  - New `--multicluster` flag to enable multi-cluster replication
  - New `--multicluster-mode` flag to choose deployment strategy:
    - `active-active`: Both clusters serve traffic simultaneously with geo-routing (ideal for stateless apps)
    - `active-passive`: Primary serves traffic, secondary is standby with automatic failover (ideal for stateful apps)
  - New `--backup-schedule` flag to configure backup frequency: `hourly`, `daily`, `weekly`
  - New `--backup-retention` flag to configure backup retention: `24h`, `72h`, `168h`, `720h`

### 🔄 API Enhancements

- **MulticlusterConfig Model**: Added new configuration model for multi-cluster deployments

  - `Enabled`: Toggle multi-cluster replication
  - `Mode`: Active-Active or Active-Passive deployment strategy
  - `BackupEnabled`: Enable Velero backups for data protection
  - `BackupSchedule`: Cron-based backup scheduling
  - `BackupRetention`: Duration-based backup retention
  - `FailoverEnabled`: Automatic failover on primary cluster failure
  - `RestoreOnFailover`: Automatic restore from backup on failover

- **Deployment Model Update**: Extended `Deployment` struct with `MulticlusterConfig` field

### 🔧 Technical Improvements

- **Orchestrator Enhancement**: Updated deployment orchestration to build and send multi-cluster configuration

  - Automatic cron schedule conversion from friendly names (hourly/daily/weekly)
  - Smart defaults: Active-passive mode automatically enables backup, failover, and restore-on-failover
  - Seamless integration with existing deployment workflow

- **DeploymentOptions Update**: Added multi-cluster fields to deployment options struct
  - `MulticlusterEnabled`: Enable/disable multi-cluster
  - `MulticlusterMode`: Deployment strategy selection
  - `BackupSchedule`: User-friendly schedule selection
  - `BackupRetention`: Retention duration

### 📚 New Command Usage

```bash
# Deploy with multi-cluster enabled (active-passive with daily backups, 7-day retention)
1ctl deploy --cpu 100m --memory 256Mi --multicluster

# Deploy with active-active mode (geo-routing, no backups)
1ctl deploy --cpu 100m --memory 256Mi --multicluster --multicluster-mode active-active

# Deploy with custom backup configuration
1ctl deploy --cpu 100m --memory 256Mi --multicluster \
  --multicluster-mode active-passive \
  --backup-schedule hourly \
  --backup-retention 72h

# Full example with all options
1ctl deploy --cpu 500m --memory 1Gi \
  --machine my-machine-1 --machine my-machine-2 \
  --env DATABASE_URL=postgres://... \
  --multicluster \
  --multicluster-mode active-passive \
  --backup-schedule daily \
  --backup-retention 168h
```

### 📋 Multi-Cluster Configuration Reference

| Flag                  | Default          | Description                            |
| --------------------- | ---------------- | -------------------------------------- |
| `--multicluster`      | `false`          | Enable multi-cluster deployment        |
| `--multicluster-mode` | `active-passive` | Deployment strategy                    |
| `--backup-schedule`   | `daily`          | Backup frequency (hourly/daily/weekly) |
| `--backup-retention`  | `168h`           | Backup retention period                |

### Backup Schedule Mapping

| Schedule | Cron Expression | Description              |
| -------- | --------------- | ------------------------ |
| `hourly` | `0 * * * *`     | Every hour               |
| `daily`  | `0 0 * * *`     | Daily at midnight UTC    |
| `weekly` | `0 18 * * 6`    | Weekly (Sunday 2 AM MYT) |

### Retention Options

| Value  | Duration | Use Case                        |
| ------ | -------- | ------------------------------- |
| `24h`  | 1 Day    | Quick recovery, minimal storage |
| `72h`  | 3 Days   | Short-term retention            |
| `168h` | 7 Days   | Recommended for most apps       |
| `720h` | 30 Days  | Long-term retention             |

### 🔒 Security

- Multi-cluster configuration securely transmitted via existing API authentication
- Backup and failover operations handled by operator with proper RBAC
- Cloudflare Load Balancer hostname auto-generated by backend

---

## Version 0.3.0 (30-09-2025)

### ✨ New Features

- **Multi-Organization Support**: Full support for users belonging to multiple organizations
  - New `org current` command to view current organization context
  - New `org switch` command to switch between organizations
  - Enhanced `auth status` command to display organization ID and namespace
  - Organization context automatically saved and restored across sessions

### 🔄 API Enhancements

- **Multi-Tenant Authentication**: Updated authentication flow to support multi-tenant backend

  - Login now returns and stores complete organization information (ID, name, namespace)
  - Token validation includes organization context
  - All API calls properly scoped to user's current organization
  - Fixed `/api-tokens/list` endpoint to include organization ID parameter

- **Backend Compatibility**: Updated all endpoints to work with multi-tenant backend
  - Fixed `TokenValidate` model types (UUID → string) to match backend response
  - Added `GetUserProfile` API function for retrieving user organization details
  - Simplified login flow to use backend-provided organization data
  - Updated issuer endpoint to use upsert pattern (`/issuers/upsert`)

### 🔧 Technical Improvements

- **Enhanced Context Management**: Improved CLI context storage

  - Added `CurrentOrgID` and `CurrentOrgName` fields to context
  - New helper functions: `GetCurrentOrgID()`, `SetCurrentOrgID()`, `GetCurrentOrgName()`, `SetCurrentOrgName()`
  - New combined setter: `SetCurrentOrganization()` for atomic updates
  - Maintains backward compatibility with existing context files

- **Improved Organization Visibility**: Better user experience for multi-org environments

  - Auth status now shows: Email, Organization, Organization ID, Namespace, Token expiry
  - Deploy command properly uses current organization context by default
  - `--organization` flag allows deploying to specific organization/namespace

- **Comprehensive Test Coverage**: Added tests for all new features
  - 3 new test cases for organization context operations
  - 3 new test cases for org command structure
  - All tests passing with 100% coverage of new code
  - Updated existing tests to accommodate new fields

### 🛠️ Breaking Changes

- **TokenValidate Model Changes**: ID fields changed from `uuid.UUID` to `string` type
  - Affects: `UserID`, `TokenID`, `OrganizationID`
  - Added new fields: `Token`, `Namespace`, `Message`
  - Ensures compatibility with backend multi-tenant implementation

### 📚 New Commands

```bash
# View current organization
1ctl org current

# Switch to different organization
1ctl org switch --org-id <organization-id>

# Enhanced auth status
1ctl auth status
```

### 🔒 Security

- All operations properly scoped to user's organization
- Context file maintains secure 0600 permissions
- Organization switching validates user access

## Version 0.2.2 (08-08-2025)

### ✨ New Features

- **Enhanced Docker Image Upload**: Added support for large Docker image uploads (>500MB and <8GB)

### 🔄 API Enhancements

- **Docker Upload Infrastructure**: Migrated Docker image uploads to dedicated service
  - Previous: `/docker/images/upload` endpoint on main API
  - New: Direct upload to specialized Docker upload service
  - Optimized for handling large binary transfers
  - Supports file sizes from 500MB to 8GB

## Version 0.2.1 (20-06-2025)

### 🐛 Bug Fixes

- **Fixed ingress domain management**: Resolved issue where ingress upsert operations were generating new domains instead of reusing existing ones
  - Enhanced `upsertIngress` function to check for existing ingresses by deployment ID before creating new ones
  - Added proper existing domain name reuse logic to prevent unnecessary domain generation
  - Fixed Kubernetes error "ingress not found" by properly identifying and updating existing ingress resources
  - Improved ingress update flow to use existing domain names and preserve ingress IDs
  - Fixed ingress upsert response parsing to correctly handle backend's string response containing only the ingress ID

### 🔧 Technical Improvements

- **Enhanced Ingress API**: Added `GetIngressByDeploymentID` function to retrieve existing ingresses

  - Leverages existing backend route `/ingresses/deploymentId/:deploymentId`
  - Enables proper lookup of existing ingress resources during deployment updates
  - Simplifies client-side logic by leveraging backend's existing upsert capabilities

- **Improved Deployment Orchestration**: Enhanced ingress handling in deployment process
  - Better separation between new ingress creation and existing ingress updates
  - Consistent domain name handling across deployment scenarios
  - Improved logging and error messaging for ingress operations
  - Returns domain name from backend response to ensure consistency

### 🔄 API Enhancements

- **Ingress Management**: Streamlined ingress upsert operations to work seamlessly with backend logic
  - Client now properly leverages backend's existing ingress detection by namespace and app label
  - Reduced client-side complexity by relying on backend's robust upsert implementation
  - Enhanced error handling and response processing for ingress operations
  - Updated `UpsertIngress` to correctly unmarshal string ingress ID response from backend

## Version 0.2.0 (19-06-2025)

### ✨ Enhanced Command Structure

- **Simplified Deploy Command**: Removed `deploy create` subcommand and moved deployment flags directly to main `deploy` command

  - New usage: `1ctl deploy --cpu 100m --memory 20Mi --env HELLO=BYE --env SAY=WHAT`
  - Subcommands `list`, `get`, and `status` remain available
  - Enhanced validation with clear error messages for required flags

- **Simplified Service Command**: Removed `service create` subcommand and moved service flags directly to main `service` command

  - New usage: `1ctl service --deployment-id=123 --name=myservice --port=8080 --namespace=myorg`
  - Subcommands `list` and `delete` remain available
  - Required flags: `--deployment-id`, `--name`, `--port`

- **Simplified Ingress Command**: Removed `ingress create` subcommand and moved ingress flags directly to main `ingress` command
  - New usage: `1ctl ingress --deployment-id=123 --service-id=456 --app-label=myapp --namespace=myorg --domain=example.com`
  - Subcommands `list` and `delete` remain available
  - Required flags: `--deployment-id`, `--domain`, `--app-label`, `--namespace`

### 🔄 API Enhancements

- **Upsert Endpoints Migration**: Updated all resource creation to use upsert endpoints for idempotent operations

  - **Deployment**: `POST /deployments/upsert/:namespace/:appLabel` (namespace = organization, appLabel = app name)
  - **Service**: `POST /services/upsert/:namespace/:serviceName` (namespace = organization, serviceName = app name)
  - **Ingress**: `POST /ingresses/upsert/:namespace/:appLabel` (namespace = organization, appLabel = app name)

- **API Client Improvements**: Replaced create functions with upsert functions
  - `CreateDeployment` → `UpsertDeployment`
  - `CreateService` → `UpsertService`
  - `CreateIngress` → `UpsertIngress`
  - Maintained same payload structure for backward compatibility

### 🔧 Technical Improvements

- **Enhanced User Experience**: Streamlined command syntax eliminates unnecessary subcommand nesting

  - Direct flag access from main commands improves CLI ergonomics
  - Consistent command patterns across deploy, service, and ingress resources
  - Maintained backward compatibility for list/delete subcommands

- **Comprehensive Test Updates**: Updated all test suites to reflect new command structure

  - Mock API functions updated to use upsert patterns
  - Integration tests migrated to new endpoint structure
  - Command tests updated with new handler functions
  - Maintained test coverage across all functionality

- **Shell Completion Updates**: Updated all shell completion scripts for new command structure

  - **Bash**: Removed obsolete "create" subcommands, updated flag completions
  - **Zsh**: Enhanced completion with new command patterns
  - **Fish**: Updated subcommand and flag completion logic
  - **PowerShell**: Removed deprecated creation commands from completions

- **Deployment Orchestrator**: Updated orchestration logic to use upsert operations

  - Service creation now uses `UpsertService` for idempotent operations
  - Ingress creation now uses `UpsertIngress` for consistent updates
  - Dependency handling updated with new upsert patterns
  - Improved error messages with operation context

- **Enhanced Image Upload Reliability**: Added retry mechanism for Docker image uploads

  - Automatic retry up to 3 attempts on upload failures
  - Exponential backoff strategy (2, 4, 8 seconds between retries)
  - Clear progress indication and error reporting for each attempt
  - Improved resilience against temporary network issues or server errors

- **Enhanced Docker Validation**: Significantly improved Dockerfile validation to support modern Docker features
  - **Multistage Build Support**: Now properly validates `FROM image AS stage_name` syntax
  - **Line Continuation Handling**: Fixed parsing of multi-line commands using backslash (`\`) continuations
  - **Enhanced Image Name Validation**: Updated regex patterns to support registry URLs, namespaces, and case-insensitive matching
  - **COPY --from Syntax**: Added proper validation for `COPY --from=stage_name` commands in multistage builds
  - **Stage Name Validation**: Validates stage names allow alphanumeric characters, underscores, and hyphens
  - **Comprehensive Test Coverage**: Added extensive test cases for all multistage build scenarios

### 🛠️ Breaking Changes

- **Command Structure**: Removed `create` subcommands from `deploy`, `service`, and `ingress` commands
  - Old: `1ctl deploy create --cpu 100m --memory 20Mi`
  - New: `1ctl deploy --cpu 100m --memory 20Mi`
  - Old: `1ctl service create --deployment-id=123 --name=myservice --port=8080`
  - New: `1ctl service --deployment-id=123 --name=myservice --port=8080`
  - Old: `1ctl ingress create --deployment-id=123 --domain=example.com`
  - New: `1ctl ingress --deployment-id=123 --domain=example.com --app-label=myapp --namespace=myorg`

## Version 0.1.8 (17-06-2025)

### ✨ New Features

- **Machine Marketplace Discovery**: Added `machine available` command to browse and filter available machines for rent
  - Comprehensive filtering options: `--region`, `--zone`, `--min-cpu`, `--min-memory`, `--gpu`, `--recommended`, `--pricing-tier`
  - Enhanced display with pricing information, performance metrics, and recommendation indicators
  - Real-time availability status and resource specifications

### 🔧 Technical Improvements

- **Updated Machine Model**: Synchronized frontend Machine model with enhanced backend structure

  - Changed `MachineID` from `uuid.UUID` to `string` type
  - Added new fields: `ID`, `Status`, `LastHealthCheck`, `Recommended`, `ResourceScore`
  - Added performance metrics: `CPUUsagePercent`, `MemoryUsagePercent`, `StorageUsagePercent`, `NetworkUsageGbps`
  - Added hardware features: `HasGPU`, `HasHDD`, `HasNVME`, `NodeType`
  - Added pricing information: `PricingTier`, `HourlyCost`
  - Added reliability metrics: `UptimePercent`, `ResponseTimeMs`, `NetworkMetricsType`

- **Enhanced Hostname Mapping**: Updated deployment logic to use machine IDs instead of machine names

  - Both manual machine selection (`--machine` flag) and automatic selection now use machine IDs
  - Improved deduplication logic based on unique machine IDs
  - Ensures consistent backend integration with machine ID-based deployments

- **Improved Machine Information Display**: Enhanced machine listing with additional useful information
  - Added Status, Node Type, Pricing Tier, and Hourly Cost to machine details
  - Added conditional display of Resource Score and Uptime percentage
  - Separate optimized display format for available machines marketplace

### 🛠️ API Enhancements

- Added `GetAvailableMachines()` API function for fetching monetized machines
- Updated `MachineIDs` struct to use string array instead of UUID array

## Version 0.1.7 (12-06-2025)

### 🐛 Bug Fixes

- **Fixed hostname deduplication for monetized machines**: When multiple machines share the same hostname (e.g., "1"), the system now properly preserves the original hostname instead of incrementing it (e.g., "1" stays "1" instead of becoming "2")
  - Added hostname deduplication logic for owner's machines in automatic selection
  - Added hostname deduplication logic for manually specified machines via `--machine` flag
  - Ensures consistent hostname behavior across both user-owned and monetized machine deployments

### 🔧 Technical Improvements

- **Enhanced versioning system**: Fixed automatic version detection in build process
  - Updated `Taskfile.yml` to automatically detect version from Git tags instead of using hardcoded default
  - Added `task version` command to easily check current version information
  - Version now correctly reflects Git state with format like `v0.1.6-3-g94b0eb5` for commits ahead of tags
  - Improved build-time version injection with commit hash and build date

## Version 0.1.6 (22-03-2025)

### 🔧 Technical Improvements

- Version number fix only, no functional changes

## Version 0.1.5 (22-03-2025)

### 🔧 Technical Improvements

- Updated API endpoints for better resource management:
  - Fix secret and environment creation endpoints from `/create` to `upsert`
- Enforced minimum replica count for deployments (monetized).

## Version 0.1.4 (18-03-2025)

### 🔧 Technical Improvements

- Improved machine allocation system:
  - Automatic machine assignment if no specific hostnames provided
  - System now intelligently selects the most cost-effective machine based on resource requirements
- Enhanced hostname selection logic:
  - Prioritizes user-owned machines first
  - Falls back to monetized machines with automatic selection
  - Improved error handling for machine allocation

## Version 0.1.3 (17-01-2025)

### 🔧 Technical Improvements

- Introduced centralized error handling with `utils.NewError`
- Standardized error formatting across the codebase
- Improved error messages with better context and readability
- Added consistent error wrapping pattern
- Enhanced error handling in cleanup operations

## Version 0.1.2 (13-01-2025)

### 🔧 Technical Improvements

- Updated registry URL to use the new registry

## Version 0.1.1 (04-01-2025)

### 🔒 Security Improvements

- Added safe integer conversion handling to prevent overflows in port and replica configurations
- Enhanced path validation for file operations to prevent directory traversal attacks
- Improved Docker build input validation with tag format checking
- Implemented secure file permission handling (0750 for directories, 0600 for files)
- Added protection against command injection in Docker build operations
- Better error handling in cleanup operations for test utilities

### 🔧 Technical Improvements

- Introduced `SafeInt32` utility function for safe integer conversions
- Added path validation functions in Docker and context operations
- Enhanced error handling in file operations
- Improved input validation for Docker build options

## Version 0.1.0 (31-12-2024)

### 🎉 Genesis Release

First public release of the Satusky CLI (1ctl) with core functionality for managing containerized applications on Satusky Cloud Platform.

### ✨ Features

#### Authentication

- Login with API token support
- Automatic token management and validation
- Session status checking
- Secure logout functionality

#### Machine Management

- List and view machine details

#### Deployment Management

- Create new deployments with customizable resources
- List and view deployment details
- Support for multiple deployment environments
- Real-time deployment status tracking with progress indicators
- Automatic Dockerfile detection and validation

#### Container Management

- Automated Docker image building
- Secure registry authentication
- Support for custom Dockerfiles
- Automatic image versioning and tagging

#### Resource Configuration

- CPU and memory allocation management
- Environment variables management
- Volume management for persistent storage
- Custom domain configuration
- Service port configuration

#### Service Management

- Create and configure services
- Automatic service discovery
- Port mapping and exposure

#### Networking

- Ingress configuration and management
- Custom domain support
- SSL/TLS certificate management
- Domain validation and verification

### 🔧 Technical Improvements

- Color-coded CLI output for better user experience
- Progress spinners for long-running operations
- Structured error handling and user feedback
- Comprehensive input validation
- Secure credential management

### 📚 Documentation

- Basic usage documentation
- Command reference
- Installation instructions
- Resource limits documentation

### 🔒 Security

- Secure token storage
- Encrypted communication
- Input sanitization and validation

### 🐛 Known Issues

- Broken CI (both unit and integration tests are not completed yet)
- Darwin 386 builds are not supported

### 📋 Requirements

- Go 1.21 or higher
- Docker installed and running
- Verified Satusky Control Panel account

For detailed documentation, visit [https://docs.satusky.com/1ctl](https://docs.satusky.com/1ctl)
