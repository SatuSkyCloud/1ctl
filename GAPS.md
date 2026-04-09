# 1ctl — Gap Analysis & Development Roadmap

Competitive analysis against flyctl (Fly.io), Railway CLI, Heroku CLI, and Render CLI.
Last updated: 2026-04-02.

---

## Summary

1ctl has genuine strengths in infrastructure depth — multi-cluster active-active deployment, Kubernetes-native HA (HPA/VPA/PDB), domain purchase, machine marketplace, and Talos Linux config management. None of these exist in flyctl.

However, for developer-experience (DX), flyctl leads significantly. The gaps below are what prevent 1ctl from being a first-choice CLI for developers deploying containerized apps.

---

## Section 1 — Critical / Table-Stakes Gaps

Issues that make 1ctl feel incomplete to any developer evaluating it against alternatives.

| ID | Gap | File / Location | Effort |
|----|-----|-----------------|--------|
| G-01 | **Streaming logs broken** — WebSocket handler fully commented out | `internal/commands/deploy.go:561`, `internal/api/logs.go` | Medium |
| G-02 | **No SSH / exec into containers** — no equivalent to `fly ssh console` | Not implemented | High |
| G-03 | **No config file** — `--cpu`, `--memory`, `--domain` etc. must be retyped on every deploy; no `satusky.toml` equivalent | Not implemented | Medium |
| G-04 | **UUID-only identification** — all commands require `--deployment-id=<uuid>`; no friendly app names | All commands with `--deployment-id` | Medium |
| G-05 | **No rollback command** — `deploy rollback` was scaffolded then removed; no way to re-deploy a previous image | Was in `internal/commands/deploy.go` | Low |
| G-06 | **No destroy/delete deployment** — can't tear down a deployment from the CLI | Not implemented | Low |

---

## Section 2 — Developer Experience (DX) Gaps

Issues that slow down day-to-day use for developers who have already onboarded.

| ID | Gap | File / Location | Effort |
|----|-----|-----------------|--------|
| D-01 | **Delete commands pass nil payload** — `DeleteService`, `DeleteIngress`, `DeleteEnvironment` all pass `nil` instead of fetching the resource first | `internal/commands/service.go:138`, `ingress.go:176`, `environment.go:136` | Low |
| D-02 | **No `1ctl open`** — no command to open the deployed app URL in the default browser | Not implemented | Low |
| D-03 | **No `1ctl restart`** — no rolling restart without a full redeploy | Not implemented | Low |
| D-04 | **No health check flag** — can't configure liveness/readiness probe at deploy time | Deploy flags in `internal/commands/deploy.go` | Low |
| D-05 | **List output uses `Key: Value` lines** — `deploy list`, `service list` etc. print status lines instead of aligned tables; hard to scan | `internal/utils/` printers | Medium |
| D-06 | **No interactive launch wizard** — `fly launch` detects runtime, suggests CPU/memory, walks setup; 1ctl requires knowing all flags upfront | Not implemented | High |
| D-07 | **No directory-scoped app context** — flyctl remembers the app name from `fly.toml` in the current directory; 1ctl has no equivalent | Not implemented | Medium |
| D-08 | **Error messages are not actionable** — errors surface the raw message but rarely suggest a fix or next command | Throughout `internal/commands/` | Medium |
| D-09 | **No confirmation on destructive commands** — `secret delete`, `token delete` etc. execute immediately with no `--yes` / interactive prompt | Throughout delete handlers | Low |

---

## Section 3 — Feature Gaps vs Competitors

Capabilities present in flyctl / Railway / Heroku that 1ctl lacks.

| ID | Gap | Notes | Effort |
|----|-----|-------|--------|
| F-01 | **No managed databases** | flyctl has Postgres, Redis, MySQL via extensions; 1ctl has S3 but no managed DB | High |
| F-02 | **No port forwarding / internal proxy** | `fly proxy 5432` tunnels to an internal service; useful for DB access during debugging | High |
| F-03 | **No one-off command execution** | `fly run`, `heroku run` — run a command inside a deployed container (migrations, scripts) | Medium |
| F-04 | **No release history** | flyctl lists all releases with image SHA, date, status; lets you see what changed | Low |
| F-05 | **No `scale` shorthand** | `fly scale count 3` — 1ctl requires a full redeploy with `--replicas 3` | Low |
| F-06 | **No env var diff / preview** | flyctl shows which env vars will change before applying; prevents accidental overwrites | Medium |
| F-07 | **No `--watch` on deploy** | flyctl streams deployment progress in real-time; 1ctl only has `deploy status --watch` as a separate step | Low |

---

## Section 4 — Technical Debt

Internal code issues that affect reliability, correctness, or test confidence.

| ID | Issue | File / Location | Effort |
|----|-------|-----------------|--------|
| T-01 | **3× nil payload on delete** — same as D-01; backend may reject or silently succeed | `service.go:138`, `ingress.go:176`, `environment.go:136` | Low |
| T-02 | **Let's Encrypt email header missing** — `x-satusky-user-email` not sent; custom domain TLS may silently fail | `internal/api/client.go` (TODO comment) | Low |
| T-03 | **No command handler tests** — `internal/commands/` has ~23 files; most lack unit tests | `internal/commands/` | High |
| T-04 | **Storage `create`, `upload`, `download` missing from CLI** — API functions exist (`internal/api/storage.go`) but no command handlers are wired up | `internal/commands/storage.go` | Medium |
| T-05 | **Deployment log streaming dead code** — commented-out `handleDeploymentLogs` blocks future contributors from knowing the intent | `internal/commands/deploy.go:561-581` | Low |

---

## Prioritized Roadmap

### Phase 1 — Quick wins (low effort, high visibility)

Target: ship as a point release.

- [ ] **G-05** Add `1ctl deploy rollback [deployment-id]` — re-deploy the previous image SHA
- [ ] **G-06** Add `1ctl deploy destroy [deployment-id]` — delete a deployment with confirmation
- [ ] **D-01 / T-01** Fix three nil-payload delete calls (fetch resource first, then delete)
- [ ] **D-02** Add `1ctl open` — open app URL in browser (`open` / `xdg-open` / `start`)
- [ ] **D-03** Add `1ctl restart [deployment-id]` — trigger rolling restart without redeploy
- [ ] **D-09** Add `--yes` / interactive `y/N` confirmation on all destructive commands
- [ ] **F-04** Add `1ctl deploy releases [deployment-id]` — list release history
- [ ] **F-05** Add `1ctl scale [deployment-id] --replicas N` shorthand
- [ ] **T-02** Send `x-satusky-user-email` header for Let's Encrypt
- [ ] **T-05** Either fully implement or remove `handleDeploymentLogs` dead code

### Phase 2 — DX overhaul (medium effort)

Target: ship as a minor release.

- [ ] **G-01** Unblock streaming logs — complete WebSocket integration once backend endpoint is ready
- [ ] **G-03** Add `satusky.toml` — persist `cpu`, `memory`, `domain`, `port`, `org`, `dockerfile` per project directory
- [ ] **G-04** Support friendly app names — resolve `--app myapp` to deployment ID via API
- [ ] **D-05** Replace `Key: Value` status lines with aligned table output for all list commands
- [ ] **D-07** Directory context — auto-load `satusky.toml` from current directory (ties into G-03)
- [ ] **D-08** Actionable error messages — add `hint:` lines suggesting the next command to run
- [ ] **F-06** Env var diff preview before apply
- [ ] **T-04** Wire up `storage create`, `storage upload`, `storage download` command handlers
- [ ] **T-03** Add unit tests for deploy, service, secret, ingress command handlers

### Phase 3 — Major features (high effort)

Target: ship as a major minor release.

- [ ] **G-02** SSH / exec into containers — requires backend proxy/tunnel support
- [ ] **F-03** One-off command execution (`1ctl run [deployment-id] -- migrate`)
- [ ] **D-06** Interactive launch wizard (`1ctl launch`) — detect runtime, suggest resources, walk setup
- [ ] **F-02** Port forwarding (`1ctl proxy [deployment-id] 5432:postgres:5432`)
- [ ] **F-01** Managed database add-ons (Postgres, Redis)

---

## Competitive Position Summary

| Dimension | 1ctl | flyctl | Railway | Heroku |
|-----------|------|--------|---------|--------|
| Multi-cluster active-active | ✅ | ❌ | ❌ | ❌ |
| Kubernetes HA (HPA/VPA/PDB) | ✅ | ❌ | ❌ | ❌ |
| Domain purchase + DNS | ✅ | ❌ | ❌ | ❌ |
| Machine marketplace | ✅ | ❌ | ❌ | ❌ |
| Talos Linux config | ✅ | ❌ | ❌ | ❌ |
| Streaming logs | ❌ | ✅ | ✅ | ✅ |
| SSH / exec into containers | ❌ | ✅ | ❌ | ✅ |
| Config file (persist flags) | ❌ | ✅ | ✅ | ✅ |
| Friendly app names | ❌ | ✅ | ✅ | ✅ |
| Rollback | ❌ | ✅ | ✅ | ✅ |
| Delete deployment | ❌ | ✅ | ✅ | ✅ |
| Table output | ❌ | ✅ | ✅ | ✅ |
| Interactive wizard | ❌ | ✅ | ✅ | ❌ |
| Managed databases | ❌ | ✅ | ✅ | ✅ |
| Port forwarding | ❌ | ✅ | ❌ | ❌ |
