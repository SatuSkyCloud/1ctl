# 1ctl CLI Test Report

**Date**: 2026-04-27 (full retest post-fixes)
**Branch**: development
**Backend**: satusky-core_backend @ localhost:8080 (`sudo task dev.debug > logs.txt 2>&1`)
**Namespace**: org3-b322955e
**User**: mingerz.k@gmail.com
**Org**: org3 (b322955e-6a86-4157-8bff-1bea605ef8ac)
**Binary**: `bin/1ctl-dev` (built from source, `defaultAPIURL=http://localhost:8080/v1/cli`)
**CLI version**: `dev`

> **Setup — run once before any section:**
> ```bash
> export SATUSKY_API_URL=http://localhost:8080/v1/cli
> 1ctl-dev profile use local
> ```

---

## Test Summary

| Category | Tested | Pass | Fail | Notes |
|---|---|---|---|---|
| auth | login, logout, status | 3 | 0 | |
| profile | create, use, current, list, delete | 5 | 0 | |
| org | list, current, switch | 3 | 0 | |
| init | init | 1 | 0 | see gap note |
| deploy — core | list, get, status, deploy (toml+flags), redeploy, destroy+restore | 8 | 0 | |
| deploy — ops | restart, releases, rollback | 3 | 0 | |
| service | list, delete | 2 | 0 | **BUG-A fixed** |
| ingress | list, delete | 2 | 0 | **BUG-A fixed** |
| env | create (first+merge), list | 3 | 0 | delete removed (wrong semantics) |
| secret | create (first+merge), list | 3 | 0 | delete removed (wrong semantics) |
| logs | stored, stream | 2 | 0 | stats/delete removed |
| notifications | list, count, read --all | 3 | 0 | |
| user | me, permissions | 2 | 0 | **BUG-C fixed** |
| token | list, get, create, disable, enable, delete | 6 | 0 | **BUG-D fixed** |
| marketplace | list, get | 2 | 0 | |
| audit | list, get | 2 | 0 | export removed |
| credits | balance, transactions, usage | 3 | 0 | topup/invoices/auto-topup removed |
| pricing | list, lookup | 2 | 0 | no data in dev (expected) |
| storage | list | 1 | 0 | |
| cluster | zones, list | 2 | 0 | |
| machine | list, available, usage | 3 | 0 | |
| issuer | list | 1 | 0 | |
| completion | zsh, bash | 2 | 0 | |
| **Total** | **63** | **63** | **0** | |

---

## Commands Tested

### 1. auth

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

1ctl-dev auth status
1ctl-dev auth logout
1ctl-dev auth login --token eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

| Command | Result |
|---------|--------|
| `auth status` | PASS — mingerz.k@gmail.com, org3, token 72d remaining |
| `auth logout` | PASS — `✅ Successfully logged out` |
| `auth login --token <jwt>` | PASS — `✅ Logged in successfully` |

---

### 2. profile

```bash
1ctl-dev profile list
1ctl-dev profile current
1ctl-dev profile create --url http://localhost:8080/v1/cli local
1ctl-dev profile use local
1ctl-dev profile delete <name>
```

| Command | Result |
|---------|--------|
| `profile list` | PASS — local (active) + prod |
| `profile current` | PASS — URL, email confirmed |
| `profile create --url ... test` | PASS — `✅ Profile 'test' created` |
| `profile use test` | PASS — `✅ Switched to profile 'test'` |
| `profile delete test` | PASS — `✅ Profile 'test' deleted` |

> `profile` subcommands require the dev binary. The Homebrew release (v0.6.0) maps `profile` as an alias for `user`.

---

### 3. org

```bash
1ctl-dev org list
1ctl-dev org current
1ctl-dev org switch --org-id 690839ba-3aed-47ea-a8ec-0cd019e4d180
1ctl-dev org switch --org-id b322955e-6a86-4157-8bff-1bea605ef8ac
```

| Command | Result |
|---------|--------|
| `org list` | PASS — 3 orgs listed |
| `org current` | PASS — org3 / org3-b322955e |
| `org switch --org-id <id>` | PASS — switches and back |

> `org switch` requires `--org-id` or `--org-name` flag — positional arg does not work.

---

### 4. init

```bash
mkdir /tmp/myapp && cd /tmp/myapp
1ctl-dev init
```

| Command | Result |
|---------|--------|
| `init` | PASS — creates `satusky.toml` with `name` from directory |

> **Gap**: `init` writes empty fields (`cpu = ""`, `memory = ""`, `replicas = 0`, `domain = ""`). Only `name` and `port` should be written; empty/zero fields should be omitted.

---

### 5. deploy — core

#### satusky.toml (minimal — only what's app-specific)

```toml
[app]
  name   = "backend-api"
  port   = 8080
  cpu    = "0.5"
  memory = "256Mi"
```

No `org` (from auth context). No `deployment_id` (resolved at runtime). No `replicas`, `domain`, `dockerfile` (platform defaults).

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

# Deploy from toml
cd examples/backend
1ctl-dev deploy --config satusky.toml --image nginx:alpine --machine compute-main-01

# Cloud build (no --image)
cd examples/frontend
1ctl-dev deploy --config satusky.toml --machine compute-main-01

# Deploy with no toml — port only, all else defaults
1ctl-dev deploy --port 8080 --image nginx:alpine --machine compute-main-01
# name = basename(cwd), cpu = 0.5, memory = 256Mi (platform defaults)

# Verify
1ctl-dev deploy list
1ctl-dev deploy get --config examples/backend/satusky.toml
1ctl-dev deploy get --deployment-id <id>
1ctl-dev deploy status --config examples/backend/satusky.toml
```

| Command | Result | Notes |
|---------|--------|-------|
| `deploy --config satusky.toml --image nginx:alpine` | PASS | 5-step pipeline; toml unchanged after deploy |
| `deploy --config satusky.toml` (cloud build) | PASS | Build → `Image architecture: amd64` → nodeSelector set → 1/1 Running |
| `deploy --port 8080 --image ...` (no toml) | PASS | Name from dirname, cpu=0.5/memory=256Mi defaults |
| Re-deploy (upsert) via config | PASS | Backend detects existing by name → updates in place |
| `--cpu 0.25` overrides toml `cpu = "0.5"` | PASS | One-shot flag override; toml file unchanged |
| `deploy list` | PASS | Both deployments listed |
| `deploy get --config satusky.toml` | PASS | Name-based resolution → full details |
| `deploy get --deployment-id <id>` | PASS | Direct ID path |
| `deploy status --config satusky.toml` | PASS | `Running 100%` |
| `deploy destroy --config -y` + redeploy | PASS | Status returns "not found" after destroy; redeploy succeeds |

**kubectl verify:**
```bash
kubectl -n org3-b322955e get pods -l "app in (backend-api,frontend)" -o wide
# backend-api-xxx  1/1  Running  compute-main-01
# frontend-xxx     1/1  Running  compute-main-01

kubectl -n org3-b322955e get deploy frontend \
  -o jsonpath='{.spec.template.spec.nodeSelector}'
# {"kubernetes.io/arch":"amd64"}
```

---

### 6. deploy — ops

```bash
1ctl-dev deploy restart --config examples/backend/satusky.toml
1ctl-dev deploy releases --config examples/backend/satusky.toml
1ctl-dev deploy rollback --config examples/backend/satusky.toml --version 1 -y
```

| Command | Result |
|---------|--------|
| `deploy restart --config` | PASS — rolling restart initiated |
| `deploy releases --config` | PASS — version history listed |
| `deploy rollback --config --version 1 -y` | PASS — rollback initiated |

---

### 7. service

```bash
1ctl-dev service list
1ctl-dev service delete --service-id <id> -y
```

| Command | Result | Notes |
|---------|--------|-------|
| `service list` | PASS | backend-api (8080), frontend (80) |
| `service delete --service-id <id> -y` | **PASS** *(was BUG-A)* | `✅ Service ... deleted successfully` — backend now looks up by ID, no body needed |

---

### 8. ingress

```bash
1ctl-dev ingress list
1ctl-dev ingress delete --ingress-id <id> -y
```

| Command | Result | Notes |
|---------|--------|-------|
| `ingress list` | PASS | backend-api.satusky.com, frontend.satusky.com |
| `ingress delete --ingress-id <id> -y` | **PASS** *(was BUG-A)* | `✅ Ingress ... deleted successfully` |

---

### 9. env (environment)

> `env delete` was **removed** — it deleted all keys at once with no per-key granularity. Replace with `env unset KEY` (future sprint). `deploy destroy` handles full cleanup.

```bash
cd examples/backend

# First-time create (no prior ConfigMap)
1ctl-dev env create --config satusky.toml --env APP_ENV=production --env LOG_LEVEL=info

# Second call — merges: new keys added, existing updated
1ctl-dev env create --config satusky.toml --env LOG_LEVEL=debug --env VERSION=1.0

1ctl-dev env list
```

| Command | Result |
|---------|--------|
| `env create` (first, no prior ConfigMap) | PASS — creates ConfigMap + DB row |
| `env create` (second, merge) | PASS — keys merged; existing updated |
| `env list` | PASS — backend-api listed |

**kubectl verify:**
```bash
kubectl -n org3-b322955e get configmap backend-api-environments -o jsonpath='{.data}'
# {"app-env":"production","log-level":"debug","version":"1.0",...}
```

---

### 10. secret

> `secret delete` was **removed** — same reason as `env delete`. Use `secret unset KEY` (future sprint).

```bash
cd examples/backend

1ctl-dev secret create --config satusky.toml --kv DB_PASS=s3cret --kv API_KEY=key123
1ctl-dev secret create --config satusky.toml --kv NEW_KEY=added
1ctl-dev secret list
```

| Command | Result |
|---------|--------|
| `secret create` (first) | PASS — K8s Secret + DB row created |
| `secret create` (second, merge) | PASS — new key added, existing preserved |
| `secret list` | PASS |

---

### 11. logs

> `logs stats` and `logs delete` were **removed** — analytics and bulk deletion belong in the dashboard.

```bash
1ctl-dev logs --config examples/backend/satusky.toml
1ctl-dev logs stream -d <deployment-id>
```

| Command | Result |
|---------|--------|
| `logs --config` | PASS — Loki-sourced lines with timestamps and pod name |
| `logs stream -d <id>` | PASS — live kubectl-style stream; `Ctrl+C` to stop |

> **Gap**: `logs stream` only accepts `-d`/`--deployment-id`, not `--config`. Inconsistent with all other subcommands.

---

### 12. notifications

```bash
1ctl-dev notifications list
1ctl-dev notifications count
1ctl-dev notifications read --all
```

| Command | Result |
|---------|--------|
| `notifications list` | PASS — deployment events shown |
| `notifications count` | PASS — `Count: 0` after read --all |
| `notifications read --all` | PASS — `✅ All notifications marked as read` |

---

### 13. user

```bash
1ctl-dev user me
1ctl-dev user permissions
```

| Command | Result | Notes |
|---------|--------|-------|
| `user me` | PASS | ID, email, name, org |
| `user permissions` | **PASS** *(was BUG-C)* | Lists all permissions as `{name, description}` rows |

---

### 14. token

```bash
1ctl-dev token list
1ctl-dev token get <token-id>
1ctl-dev token create --name "my-token"
1ctl-dev token disable <token-id>
1ctl-dev token enable <token-id>
1ctl-dev token delete <token-id> -y
```

| Command | Result | Notes |
|---------|--------|-------|
| `token list` | PASS | Active tokens with status, last used |
| `token get <id>` | PASS | Full token details |
| `token create --name <name>` | **PASS** *(was BUG-D)* | Token created; ID returned |
| `token disable <id>` | **PASS** *(was BUG-D)* | `✅ Token disabled successfully` |
| `token enable <id>` | **PASS** *(was BUG-D)* | `✅ Token enabled successfully` |
| `token delete <id> -y` | **PASS** *(was BUG-D)* | `✅ Token deleted successfully` |

---

### 15. marketplace

```bash
1ctl-dev marketplace list
1ctl-dev marketplace get <id>
```

| Command | Result |
|---------|--------|
| `marketplace list` | PASS — uptime-kuma, vaultwarden, gitea, nextcloud, mongodb, … |
| `marketplace get <id>` | PASS — name, description, status |

---

### 16. audit

> `audit export` was **removed** — compliance export belongs in the web dashboard.

```bash
1ctl-dev audit list
1ctl-dev audit get <log-id>
```

| Command | Result |
|---------|--------|
| `audit list` | PASS — actions with user, resource type, IP |
| `audit get <id>` | PASS — full detail for single entry |

---

### 17. credits / billing

> `credits topup`, `credits invoices`, `credits auto-topup`, `credits notifications` were **removed** — billing configuration belongs in the web UI.

```bash
1ctl-dev credits balance
1ctl-dev credits transactions
1ctl-dev credits usage
```

| Command | Result |
|---------|--------|
| `credits balance` | PASS — $388.16 MYR |
| `credits transactions` | PASS — usage charges listed |
| `credits usage` | PASS — `No machine usage found for the last 7 days` |

---

### 18. pricing

```bash
1ctl-dev pricing list
1ctl-dev pricing lookup --region my-kul-1b --type standard --sla standard
```

| Command | Result |
|---------|--------|
| `pricing list` | PASS — `No pricing configurations found` (none in dev) |
| `pricing lookup ...` | PASS — `Pricing configuration not found` (none in dev) |

---

### 19. storage

```bash
1ctl-dev storage list
```

| Command | Result |
|---------|--------|
| `storage list` | PASS — `No storage configurations found` (correct) |

---

### 20. cluster

```bash
1ctl-dev cluster zones
1ctl-dev cluster list
```

| Command | Result |
|---------|--------|
| `cluster zones` | PASS — `my-kul-1b` (KUL-1B) |
| `cluster list` | PASS — kul (healthy, default ★), bki |

---

### 21. machine

```bash
1ctl-dev machine list
1ctl-dev machine available
1ctl-dev machine usage list
```

| Command | Result |
|---------|--------|
| `machine list` | PASS — all machines with status, CPU, memory, cost |
| `machine available` | PASS — 3 monetized machines with scores |
| `machine usage list` | PASS — `No machine usage records found` |

---

### 22. issuer

```bash
1ctl-dev issuer list
```

| Command | Result |
|---------|--------|
| `issuer list` | PASS — `No certificate issuers found` |

---

### 23. completion & version

```bash
1ctl-dev completion zsh
1ctl-dev completion bash
1ctl-dev --version
```

| Command | Result |
|---------|--------|
| `completion zsh` | PASS — zsh completion script generated |
| `completion bash` | PASS — bash completion script generated |
| `--version` | PASS — `1ctl version dev` |

To install zsh completion:
```bash
source <(1ctl-dev completion zsh)
# or persist:
1ctl-dev completion zsh > ~/.zsh/completions/_1ctl
```

---

## Fixes Applied (since last report)

| Bug | Fix | Status |
|-----|-----|--------|
| BUG-A: service delete → 500 Invalid payload | Backend `DeleteService` now looks up by path ID (no body required) | ✅ PASS |
| BUG-A: ingress delete → 500 Invalid payload | Backend `DeleteIngress` now looks up by path ID | ✅ PASS |
| BUG-A: env delete → 500 Invalid payload | **Command removed** — wrong "delete all" semantics |
| BUG-A: secret delete → 500 Invalid payload | **Command removed** — wrong "delete all" semantics |
| BUG-C: user permissions JSON unmarshal | `UserPermission` struct updated to match `{permission_name, description, resource_type, action}` | ✅ PASS |
| BUG-D: token create → 404 | Backend route `create/:userId` → `create/:userId/:orgId`; CLI path aligned | ✅ PASS |
| BUG-D: token disable/enable → 404 | Backend route `state/:tokenId` → `state/:userId/:tokenId`; CLI path aligned | ✅ PASS |

## Commands Removed (not bugs — wrong place for them in a CLI)

| Removed | Reason |
|---------|--------|
| `logs stats`, `logs delete` | Analytics + bulk delete belong in dashboard |
| `audit export` | Compliance export belongs in dashboard |
| `credits topup`, `invoices`, `auto-topup`, `notifications` | Billing configuration belongs in web UI |
| `env delete` | Deletes ALL keys at once — dangerous; replace with `env unset KEY` (future) |
| `secret delete` | Same reason as env delete |

---

## satusky.toml — Current Design

```toml
[app]
  name   = "backend-api"   # identifier — resolved at runtime via API (no deployment_id stored)
  port   = 8080            # required (can't be guessed)
  cpu    = "0.5"           # optional — platform default 0.5 if omitted
  memory = "256Mi"         # optional — platform default 256Mi if omitted
```

No `org` (from auth context). No `deployment_id` (resolved at runtime). No `replicas`, `domain`, `dockerfile` unless overriding defaults. Bare minimum that works:

```toml
[app]
  port = 8080
```

---

## kubectl Final State

```bash
kubectl -n org3-b322955e get pods -l "app in (backend-api,frontend)" -o wide
```
```
NAME                           READY   STATUS    NODE
backend-api-7f96c4986b-ktplk   1/1     Running   compute-main-01
frontend-5bc456fc86-t6xqp      1/1     Running   compute-main-01
```

```bash
kubectl -n org3-b322955e get deploy frontend \
  -o jsonpath='{.spec.template.spec.nodeSelector}'
# {"kubernetes.io/arch":"amd64"}

kubectl -n org3-b322955e get configmap backend-api-environments -o jsonpath='{.data}'
# {"app-env":"production","log-level":"debug","version":"1.0",...}
```

No unexpected 5xx errors in backend logs.

---

## Remaining Gaps (for future sprints)

| Gap | Priority |
|-----|----------|
| `logs stream` doesn't accept `--config` (inconsistent with all other commands) | High |
| `init` writes empty/zero fields (`cpu=""`, `replicas=0`) | Medium |
| No `--output json` on any command (needed for scripting/CI) | Medium |
| No `--wait` on deploy (must poll `deploy status` manually) | Medium |
| `env unset KEY` / `secret unset KEY` (per-key removal) | Medium |
| `org switch` positional arg doesn't work (needs `--org-id`) | Low |

---

## Features Skipped (require infrastructure)

| Feature | Reason |
|---------|--------|
| `domain` (all subcommands) | Requires Cloudflare + DNS registrar |
| `machine vm` | Requires Mac agent machines |
| `marketplace deploy` | Requires marketplace apps in dev |
| `talos` | Requires Talos Linux machines |
| `admin` | Requires super-admin role |
| `storage` CRUD | Requires S3/Ceph backend |
| `issuer create` | Requires cert-manager |
| `--strategy recreate`, `--hpa`, `--vpa` | Requires multi-pod + metrics-server |
| `--multicluster` | Requires multi-zone nodes |
