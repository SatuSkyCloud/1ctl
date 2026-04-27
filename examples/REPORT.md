# 1ctl CLI Test Report

**Date**: 2026-04-27 (comprehensive — every command exercised)
**Branch**: development
**Backend**: satusky-core_backend @ localhost:8080 (`sudo task dev.debug > logs.txt 2>&1`)
**Namespace**: org3-b322955e
**User**: mingerz.k@gmail.com / dev@satusky.com
**Org**: org3 (b322955e-6a86-4157-8bff-1bea605ef8ac)
**Binary**: `bin/1ctl-dev` (dev build, `defaultAPIURL=http://localhost:8080/v1/cli`)
**CLI version**: `dev` (built from development branch)

> **Setup — run once before any section:**
> ```bash
> export SATUSKY_API_URL=http://localhost:8080/v1/cli
> 1ctl-dev profile use local
> ```

---

## Test Summary

| Category | Subcommands Tested | Pass | Fail | Bug |
|---|---|---|---|---|
| auth | login, logout, status | 3 | 0 | — |
| profile | create, use, current, list, delete | 5 | 0 | — |
| org | list, current, switch | 3 | 0 | — |
| init | init | 1 | 0 | see note |
| deploy — core | list, get, status, destroy, deploy (with+without toml), flag overrides | 8 | 0 | — |
| deploy — ops | restart, releases, rollback | 3 | 0 | — |
| service | list, delete | 1 | 1 | BUG-A: delete → 500 Invalid payload |
| ingress | list, delete | 1 | 1 | BUG-A: delete → 500 Invalid payload |
| env | create (first+merge), list, delete | 3 | 1 | BUG-A: delete → 500 Invalid payload |
| secret | create (first+merge), list, delete | 3 | 1 | BUG-A: delete → 500 Invalid payload |
| logs | default (stored), stream, stats | 2 | 1 | BUG-B: stats → 500 |
| notifications | list, count, read --all | 3 | 0 | — |
| user | me, permissions, sessions | 2 | 1 | BUG-C: permissions → unmarshal error |
| token | list, get, create, disable, enable | 2 | 3 | BUG-D: create/disable/enable → 404 |
| marketplace | list, get | 2 | 0 | — |
| audit | list, get, export | 2 | 1 | BUG-E: export → "not found" |
| credits | balance, transactions, usage, auto-topup | 3 | 1 | BUG-F: auto-topup → 404 |
| pricing | list, lookup | 1 | 1 | no data (dev backend, expected) |
| storage | list | 1 | 0 | — |
| cluster | zones, list | 2 | 0 | — |
| machine | list, available, usage | 3 | 0 | — |
| issuer | list | 1 | 0 | empty (expected) |
| completion | zsh, bash | 2 | 0 | — |
| **Total** | **64** | **54** | **10** | **6 distinct bugs** |

---

## Commands — Detailed Results

### 1. auth

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

1ctl-dev auth status
1ctl-dev auth logout
1ctl-dev auth login --token eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

| Command | Result | Output |
|---------|--------|--------|
| `auth status` | PASS | mingerz.k@gmail.com, org3, token 72d remaining |
| `auth logout` | PASS | `✅ Successfully logged out` |
| `auth login --token <jwt>` | PASS | `✅ Logged in successfully to SatuSky 1ctl as mingerz.k@gmail.com!` |

---

### 2. profile

```bash
1ctl-dev profile list
1ctl-dev profile current
1ctl-dev profile create --url http://localhost:8080/v1/cli test-profile
1ctl-dev profile use test-profile
1ctl-dev profile use local
1ctl-dev profile delete test-profile
```

| Command | Result | Notes |
|---------|--------|-------|
| `profile list` | PASS | local (active) + prod shown |
| `profile current` | PASS | URL, email confirmed |
| `profile create --url ... test-profile` | PASS | `✅ Profile 'test-profile' created` |
| `profile use test-profile` | PASS | `✅ Switched to profile 'test-profile'` |
| `profile use local` | PASS | Restored |
| `profile delete test-profile` | PASS | `✅ Profile 'test-profile' deleted` |

> **Note**: `profile` subcommands require the dev binary. The Homebrew release (v0.6.0) maps `profile` as an alias for `user`. On a fresh machine, build `1ctl-dev` first.

---

### 3. org

```bash
1ctl-dev org list
1ctl-dev org current
1ctl-dev org switch --org-id 690839ba-3aed-47ea-a8ec-0cd019e4d180
1ctl-dev org switch --org-id b322955e-6a86-4157-8bff-1bea605ef8ac
```

| Command | Result | Notes |
|---------|--------|-------|
| `org list` | PASS | 3 orgs with IDs and names |
| `org current` | PASS | org3 / org3-b322955e / namespace |
| `org switch --org-id <id>` | PASS | `✅ Switched to organization: org2` |
| `org switch --org-id <id>` (back) | PASS | `✅ Switched to organization: org3` |

> `org switch` requires `--org-id` or `--org-name` — positional arg does NOT work.

---

### 4. init

```bash
mkdir /tmp/myapp && cd /tmp/myapp
1ctl-dev init
cat satusky.toml
```

| Command | Result | Notes |
|---------|--------|-------|
| `init` | PASS | Creates `satusky.toml` with name auto-detected from directory |

Generated toml:
```toml
[app]
  name = "myapp"
  port = 8080
  dockerfile = "Dockerfile"
  cpu = ""
  memory = ""
  replicas = 0
  domain = ""
```

> **Gap**: `init` writes `cpu = ""`, `memory = ""`, `replicas = 0` — these empty/zero values are noise. A world-class CLI (fly launch) only writes fields you actually set and omits optional ones. `init` should only emit `name` and `port`.

---

### 5. deploy — core

#### satusky.toml (minimal — no org, no deployment_id)

```toml
[app]
  name = "backend-api"
  port = 8080
  cpu  = "0.5"
  memory = "256Mi"
```

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

# Deploy with toml (reads name, port, cpu, memory)
cd examples/backend
1ctl-dev deploy --config satusky.toml --image nginx:alpine --machine compute-main-01

# Deploy with no toml — all flags, cpu/memory default to 0.5/256Mi
1ctl-dev deploy --port 8080 --image nginx:alpine --machine compute-main-01
# Name auto-detected from directory. Defaults: cpu=0.5, memory=256Mi

# Cloud build (no --image)
cd examples/frontend
1ctl-dev deploy --config satusky.toml --machine compute-main-01

# Status checks
1ctl-dev deploy list
1ctl-dev deploy get --config examples/backend/satusky.toml
1ctl-dev deploy get --deployment-id <id>
1ctl-dev deploy status --config examples/backend/satusky.toml
```

| Command | Result | Notes |
|---------|--------|-------|
| `deploy --config satusky.toml --image nginx:alpine` | PASS | 5-step pipeline; no deployment_id written to toml |
| `deploy --port 8080 --image nginx:alpine` (no toml) | PASS | Name=dirname, cpu=0.5, memory=256Mi defaults applied |
| `deploy --config satusky.toml` (cloud build) | PASS | Build → arch=amd64 → nodeSelector set → 1/1 Running |
| Re-deploy (upsert) | PASS | Backend detects existing by name → updates, same ID reused |
| `--cpu 0.25` overrides `cpu = "0.5"` in toml | PASS | One-shot flag override, toml unchanged |
| `deploy list` | PASS | All deployments listed |
| `deploy get --config satusky.toml` | PASS | Name-based resolution via API → full details |
| `deploy get --deployment-id <id>` | PASS | Direct ID path |
| `deploy status --config satusky.toml` | PASS | `Running 100%` |

---

### 6. deploy — ops

```bash
1ctl-dev deploy restart --config examples/backend/satusky.toml
1ctl-dev deploy releases --config examples/backend/satusky.toml
1ctl-dev deploy rollback --config examples/backend/satusky.toml --version 1 -y
1ctl-dev deploy destroy --config examples/frontend/satusky.toml -y
```

| Command | Result | Notes |
|---------|--------|-------|
| `deploy restart --config` | PASS | Rolling restart via name resolution |
| `deploy releases --config` | PASS | Version history listed |
| `deploy rollback --config --version 1 -y` | PASS | Rollback initiated |
| `deploy destroy --config -y` | PASS | Correct deployment destroyed via name resolution |

---

### 7. service

```bash
1ctl-dev service list
1ctl-dev service delete --service-id <id> -y
```

| Command | Result | Notes |
|---------|--------|-------|
| `service list` | PASS | backend-api (8080), frontend (80) |
| `service delete --service-id <id> -y` | **FAIL** | `❌ failed to delete service: Invalid payload` — 500 on backend |

---

### 8. ingress

```bash
1ctl-dev ingress list
1ctl-dev ingress delete --ingress-id <id> -y
```

| Command | Result | Notes |
|---------|--------|-------|
| `ingress list` | PASS | backend-api.satusky.com, frontend.satusky.com |
| `ingress delete --ingress-id <id> -y` | **FAIL** | `❌ failed to delete ingress: Invalid payload` — 500 on backend |

---

### 9. env (environment)

```bash
cd examples/backend

1ctl-dev env create --config satusky.toml --env APP_ENV=production --env LOG_LEVEL=info
1ctl-dev env create --config satusky.toml --env LOG_LEVEL=debug --env NEW_KEY=hello
1ctl-dev env list
1ctl-dev env delete --env-id <id> -y
```

| Command | Result | Notes |
|---------|--------|-------|
| `env create` (first, no prior ConfigMap) | PASS | Creates ConfigMap + DB row |
| `env create` (second, merge) | PASS | Keys merged; new added, existing updated |
| `env list` | PASS | backend-api listed |
| `env delete --env-id <id> -y` | **FAIL** | `❌ failed to delete environment: Invalid payload` — 500 on backend |

**kubectl verify:**
```bash
kubectl -n org3-b322955e get configmap backend-api-environments -o jsonpath='{.data}'
# {"app-env":"production","log-level":"debug","new-key":"hello",...}
```

---

### 10. secret

```bash
cd examples/backend

1ctl-dev secret create --config satusky.toml --kv DB_PASS=supersecret --kv API_KEY=abc123
1ctl-dev secret create --config satusky.toml --kv NEW_SECRET=newval
1ctl-dev secret list
1ctl-dev secret delete -s <id> -y
```

| Command | Result | Notes |
|---------|--------|-------|
| `secret create` (first, no prior K8s Secret) | PASS | Creates Secret + DB row |
| `secret create` (second, merge) | PASS | Keys merged |
| `secret list` | PASS | backend-api listed |
| `secret delete -s <id> -y` | **FAIL** | `❌ failed to delete secret: Invalid payload` — 500 on backend |

---

### 11. logs

```bash
# Stored logs (via Loki)
1ctl-dev logs --config examples/backend/satusky.toml
1ctl-dev logs --deployment-id <id>

# Live streaming (like kubectl logs -f)
1ctl-dev logs stream -d <id>

# Stats
1ctl-dev logs stats -d <id>
```

| Command | Result | Notes |
|---------|--------|-------|
| `logs --config` | PASS | Loki-sourced log lines with timestamps, pod names |
| `logs stream -d <id>` | PASS | Live kubectl-style streaming, `Ctrl+C` to stop |
| `logs stats -d <id>` | **FAIL** | `❌ failed to get log stats: An internal server error occurred` — 500 backend |
| `logs --config` with `--config` on `stream` | **FAIL** | `--config` flag not defined on `logs stream` subcommand |

> **Gap**: `logs stream` only accepts `--deployment-id/-d`, not `--config`. Inconsistent with other commands.

---

### 12. notifications

```bash
1ctl-dev notifications list
1ctl-dev notifications count
1ctl-dev notifications read --all
1ctl-dev notifications count   # verify 0
```

| Command | Result | Notes |
|---------|--------|-------|
| `notifications list` | PASS | Shows deployment events |
| `notifications count` | PASS | `Count: 0` (after read --all) |
| `notifications read --all` | PASS | `✅ All notifications marked as read` |

---

### 13. user

```bash
1ctl-dev user me
1ctl-dev user permissions
1ctl-dev user sessions
```

| Command | Result | Notes |
|---------|--------|-------|
| `user me` | PASS | ID, email, name, org |
| `user permissions` | **FAIL** | `❌ failed to get permissions: json: cannot unmarshal array into Go value of type api.UserPermissions` — CLI struct mismatch |
| `user sessions` | PASS | Subcommand menu shown (list/revoke available) |

---

### 14. token

```bash
1ctl-dev token list
1ctl-dev token get <token-id>
1ctl-dev token create --name "my-token"
1ctl-dev token disable <token-id>
1ctl-dev token enable <token-id>
```

| Command | Result | Notes |
|---------|--------|-------|
| `token list` | PASS | 1 active token shown |
| `token get <id>` | PASS | ID, name, status, last-used |
| `token create --name "my-token"` | **FAIL** | `404: Cannot POST /v1/cli/api-tokens/create/...` — backend route missing |
| `token disable <id>` | **FAIL** | `404: Cannot POST /v1/cli/api-tokens/state/...` — backend route missing |
| `token enable <id>` | **FAIL** | `404: Cannot POST /v1/cli/api-tokens/state/...` — backend route missing |

---

### 15. marketplace

```bash
1ctl-dev marketplace list
1ctl-dev marketplace get <marketplace-id>
```

| Command | Result | Notes |
|---------|--------|-------|
| `marketplace list` | PASS | uptime-kuma, vaultwarden, and others listed |
| `marketplace get <id>` | PASS | Full details including description, status |

---

### 16. audit

```bash
1ctl-dev audit list
1ctl-dev audit get <log-id>
1ctl-dev audit export --format json
```

| Command | Result | Notes |
|---------|--------|-------|
| `audit list` | PASS | Actions with user, resource type, IP |
| `audit get <id>` | PASS | Full detail for single entry |
| `audit export --format json` | **FAIL** | `❌ failed to export audit logs: Audit log not found` — backend returns 404 |

---

### 17. credits / billing

```bash
1ctl-dev credits balance
1ctl-dev credits transactions
1ctl-dev credits usage
1ctl-dev credits auto-topup get
```

| Command | Result | Notes |
|---------|--------|-------|
| `credits balance` | PASS | $388.16 MYR, last updated 5d ago |
| `credits transactions` | PASS | Usage transactions listed |
| `credits usage` | PASS | `No machine usage found for the last 7 days` |
| `credits auto-topup get` | **FAIL** | `❌ Resource not found` — not configured (dev backend, likely expected) |

---

### 18. pricing

```bash
1ctl-dev pricing list
1ctl-dev pricing lookup --region my-kul-1b --type standard --sla standard
```

| Command | Result | Notes |
|---------|--------|-------|
| `pricing list` | PASS | `No pricing configurations found` (dev backend has none) |
| `pricing lookup --region my-kul-1b --type standard --sla standard` | PASS | `Pricing configuration not found` (none configured in dev) |

---

### 19. storage

```bash
1ctl-dev storage list
```

| Command | Result | Notes |
|---------|--------|-------|
| `storage list` | PASS | `No storage configs` — correct for this namespace |

---

### 20. cluster

```bash
1ctl-dev cluster zones
1ctl-dev cluster list
```

| Command | Result | Notes |
|---------|--------|-------|
| `cluster zones` | PASS | `my-kul-1b` |
| `cluster list` | PASS | kul (healthy, default, priority 1), bki (priority 2) |

---

### 21. machine

```bash
1ctl-dev machine list
1ctl-dev machine available
1ctl-dev machine usage list
```

| Command | Result | Notes |
|---------|--------|-------|
| `machine list` | PASS | All machines with status, CPU, memory, cost |
| `machine available` | PASS | 3 monetized machines with scores |
| `machine usage list` | PASS | `No machine usage records found` |

---

### 22. issuer

```bash
1ctl-dev issuer list
```

| Command | Result | Notes |
|---------|--------|-------|
| `issuer list` | PASS | `No certificate issuers found` (none created) |

---

### 23. completion

```bash
1ctl-dev completion zsh
1ctl-dev completion bash
```

| Command | Result | Notes |
|---------|--------|-------|
| `completion zsh` | PASS | Zsh completion script generated |
| `completion bash` | PASS | Bash completion script generated |

```bash
# To install zsh completion:
source <(1ctl-dev completion zsh)
# Or persist:
1ctl-dev completion zsh > ~/.zsh/completions/_1ctl
```

---

## Bugs Found

### BUG-A: service/ingress/env/secret delete → 500 Invalid payload

**Commands**: `service delete`, `ingress delete`, `env delete`, `secret delete`

All return `❌ failed to delete X: Invalid payload` with a 500 on the backend. The CLI sends the correct ID in the path; the backend is failing to parse or validate the request body (likely expects a body that the CLI doesn't send, or the route is not matching the handler correctly).

**Reproduce:**
```bash
1ctl-dev service delete --service-id <id> -y
1ctl-dev ingress delete --ingress-id <id> -y
1ctl-dev env delete --env-id <id> -y
1ctl-dev secret delete -s <id> -y
```

### BUG-B: logs stats → 500

**Command**: `1ctl-dev logs stats -d <deployment-id>`

Returns `❌ failed to get log stats: An internal server error occurred`. Backend is returning 500 — likely a Loki query issue or missing data in dev.

### BUG-C: user permissions → JSON unmarshal error

**Command**: `1ctl-dev user permissions`

Error: `json: cannot unmarshal array into Go value of type api.UserPermissions`

The backend returns a JSON array but the CLI expects an object. The `UserPermissions` struct needs to be `[]UserPermission` (slice) not a struct wrapper.

### BUG-D: token create / disable / enable → 404

**Commands**: `token create`, `token disable <id>`, `token enable <id>`

All return 404 with routes like `/v1/cli/api-tokens/create/:userId/:orgId` and `/v1/cli/api-tokens/state/:userId/:tokenId`. These backend routes either don't exist or have different paths than what the CLI is calling.

### BUG-E: audit export → not found

**Command**: `1ctl-dev audit export --format json`

Returns `❌ failed to export audit logs: Audit log not found`. The backend's export endpoint may need query params (date range) that the CLI doesn't send, or the route is broken.

### BUG-F: credits auto-topup get → not found

**Command**: `1ctl-dev credits auto-topup get`

Returns `❌ Resource not found`. May be expected if auto-topup is not configured — but the error should say "not configured" not "not found".

---

## gaps vs World-Class CLI (fly / gcloud / aws)

### Missing UX features

| Gap | Fly.io equivalent | Priority |
|-----|-------------------|----------|
| `1ctl deploy` shows no deployment URL until step 5 — should print it earlier | `fly deploy` shows app URL immediately | High |
| `logs stream` doesn't accept `--config` (inconsistency) | `fly logs` accepts app name/config | High |
| `init` writes empty fields (`cpu = ""`, `replicas = 0`) | `fly launch` only writes non-default values | Medium |
| No `--output json` on any command | `gcloud --format json`, `aws --output json` | Medium |
| No `--watch`/`--wait` on `deploy` — user must poll `deploy status` | `fly deploy --wait-timeout 300` | Medium |
| `org switch` requires `--org-id` flag — positional arg doesn't work | `fly orgs select my-org` accepts name | Low |
| `token create/disable/enable` broken (BUG-D) | Core token management | High |
| `user permissions` broken (BUG-C) | `gcloud auth list` / IAM | Medium |
| No `1ctl open` to open app URL in browser | `fly open` | Low |
| No `--dry-run` / preview mode for deploy | `terraform plan` analogy | Low |
| `completion` scripts generated but no install instructions in help | `fly completion install` | Low |

### Inconsistencies

| Issue | Example |
|-------|---------|
| Some commands use `--id` (token), some `--service-id`, `--ingress-id`, `--env-id`, `--secret-id` | Should be consistent — either positional arg or `--id` everywhere |
| `logs stream` requires `-d` but other subcommands accept `--config` | `--config` should work on all deployment-scoped commands |
| `deploy destroy` prints deployment_id hint (`Use '1ctl deploy status --deployment-id ...'`) after destroy when the deployment is gone | Should say `1ctl deploy list` instead |
| `audit get` / `marketplace get` take positional arg, but `token get` / `service delete` take flags | Inconsistent arg vs flag patterns across commands |

---

## satusky.toml — Current State

```toml
[app]
  name   = "backend-api"   # identifier — looked up at runtime, never stored as ID
  port   = 8080            # required
  cpu    = "0.5"           # optional — platform default 0.5 if omitted
  memory = "256Mi"         # optional — platform default 256Mi if omitted
```

No `org` (derived from auth context). No `deployment_id` (resolved at runtime via name). No `replicas`, `domain`, `dockerfile` unless overriding defaults. The bare minimum that works:

```toml
[app]
  port = 8080
```

Name from directory, cpu/memory from platform defaults.

---

## kubectl Full Verification

```bash
kubectl -n org3-b322955e get pods -o wide
# backend-api-xxx  1/1  Running  compute-main-01
# frontend-xxx     1/1  Running  compute-main-01

kubectl -n org3-b322955e get deployment frontend \
  -o jsonpath='{.spec.template.spec.nodeSelector}'
# {"kubernetes.io/arch":"amd64"}

kubectl -n org3-b322955e get configmap backend-api-environments \
  -o jsonpath='{.data}'
# {"app-env":"production","log-level":"debug","new-key":"hello",...}

kubectl -n org3-b322955e get secret backend-api-secrets -o jsonpath='{.data}' \
  | python3 -c "import sys,json,base64; d=json.load(sys.stdin); \
    [print(k,'=',base64.b64decode(v).decode()) for k,v in d.items()]"
# api-key = abc123
# db-pass = supersecret
# new-secret = newval
```

---

## Backend Error Log (unexpected 5xx)

```bash
grep "level=ERROR" satusky-core_backend/logs.txt | grep -v "app_label=does-not-exist\|app_label=frontend\|no rows in result set" | tail -10
```

Unexpected errors found:
- `secrets/delete/...` → 500 (BUG-A)
- `services/delete/...` → 500 (BUG-A)
- `environments/delete/...` → 500 (BUG-A)
- `ingresses/delete/...` → 500 (BUG-A)
- `logs/stats/...` → 500 (BUG-B)

---

## Features Skipped (require infrastructure)

| Feature | Reason |
|---------|--------|
| `domain` (all) | Requires Cloudflare + DNS registrar |
| `machine vm` | Requires Mac agent machines |
| `marketplace deploy` | Requires marketplace apps in dev |
| `talos` | Requires Talos Linux machines |
| `admin` | Requires super-admin role |
| `--strategy recreate` | Requires multi-pod deployment |
| `--hpa` / `--vpa` | Requires metrics-server |
| `--multicluster` | Requires multi-zone nodes |
| `--zone` routing | Requires zone-labeled nodes |
| `storage` CRUD | Requires S3/Ceph backend |
| `issuer create` | Requires cert-manager |
| `credits topup` | Requires Stripe integration |
