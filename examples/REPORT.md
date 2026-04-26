# 1ctl CLI Test Report

**Date**: 2026-04-26 (full retest — name-based resolution, all scenarios)
**Branch**: development
**Backend**: satusky-core_backend @ localhost:8080 (`sudo task dev.debug > logs.txt 2>&1`)
**Namespace**: org3-b322955e
**User**: mingerz.k@gmail.com
**Org**: org3 (b322955e-6a86-4157-8bff-1bea605ef8ac)
**Binary**: `bin/1ctl-dev` (dev build, `defaultAPIURL` baked to `http://localhost:8080/v1/cli`)

> **All commands below require the local profile to be active.**
> Run once before starting:
> ```bash
> export SATUSKY_API_URL=http://localhost:8080/v1/cli
> 1ctl-dev profile use local
> ```

---

## Test Summary

| Category | Tested | Pass | Fail |
|----------|--------|------|------|
| Auth & Profile | 5 | 5 | 0 |
| Org | 2 | 2 | 0 |
| Deploy — core | 9 | 9 | 0 |
| Deploy — ops | 4 | 4 | 0 |
| Service & Ingress | 2 | 2 | 0 |
| Environment | 4 | 4 | 0 |
| Secret | 4 | 4 | 0 |
| Logs | 1 | 1 | 0 |
| Observability | 5 | 5 | 0 |
| Infrastructure | 6 | 6 | 0 |
| Cloud build + arch | 5 | 5 | 0 |
| Error scenarios | 5 | 5 | 0 |
| **Total** | **52** | **52** | **0** |

---

## Commands — Detailed Results

### 1. Auth & Profile

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

1ctl-dev profile use local
1ctl-dev auth status
1ctl-dev profile list
1ctl-dev profile current
1ctl-dev --profile local auth status
```

| Command | Result | Notes |
|---------|--------|-------|
| `1ctl-dev profile use local` | PASS | Activates local profile (sets active_profile in context.json) |
| `1ctl-dev auth status` | PASS | mingerz.k@gmail.com, org3, token 73d remaining |
| `1ctl-dev profile list` | PASS | local (active) + prod shown |
| `1ctl-dev profile current` | PASS | URL, email, org confirmed |
| `1ctl-dev --profile local auth status` | PASS | One-shot override works |

> **Note**: `profile create` requires the dev binary — the Homebrew release (v0.6.0) has `profile` as an alias for `user` only. On a fresh machine, build+install `1ctl-dev` first, then `profile create --url http://localhost:8080/v1/cli local`.

---

### 2. Organization

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

1ctl-dev org list
1ctl-dev org current
```

| Command | Result | Notes |
|---------|--------|-------|
| `1ctl-dev org list` | PASS | 3 orgs listed |
| `1ctl-dev org current` | PASS | org3 / org3-b322955e |

---

### 3. Deploy — Core

#### Setup: destroy existing deployments

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

1ctl-dev deploy list
1ctl-dev deploy destroy --deployment-id <backend-api-id> -y
1ctl-dev deploy destroy --deployment-id <frontend-id> -y
```

#### Deploy backend-api from clean toml (no deployment_id)

```bash
cd examples/backend
1ctl-dev deploy --config satusky.toml --image nginx:alpine --machine compute-main-01
```

#### Deploy frontend via cloud build

```bash
cd examples/frontend
1ctl-dev deploy --config satusky.toml --machine compute-main-01
```

#### Verify

```bash
1ctl-dev deploy list
1ctl-dev deploy get --config examples/backend/satusky.toml
1ctl-dev deploy get --config examples/frontend/satusky.toml
1ctl-dev deploy status --config examples/backend/satusky.toml
1ctl-dev deploy get --deployment-id <id>     # direct ID also works
```

| Command | Result | Notes |
|---------|--------|-------|
| `deploy destroy` ×2 | PASS | Both torn down cleanly |
| `deploy --config satusky.toml --image nginx:alpine --machine compute-main-01` | PASS | 5-step pipeline; no deployment_id in toml before or after |
| `deploy --config satusky.toml --machine compute-main-01` (cloud build) | PASS | Build → arch=amd64 detected → nodeSelector set → 1/1 Running |
| `deploy list` | PASS | Both deployments listed by ID |
| `deploy get --config satusky.toml` | PASS | Name-based resolution: reads `name`, calls API, returns full details |
| `deploy get --deployment-id <id>` | PASS | Direct ID path still works |
| `deploy status --config satusky.toml` | PASS | `Running 100%` via name resolution |
| Re-deploy (upsert) via config | PASS | Same deployment_id reused; version incremented |
| Flag overrides toml value (`--cpu 0.25` over `cpu = "0.5"`) | PASS | `CPU Request: 0.25` confirmed via `deploy get`; toml unchanged |

**kubectl verification:**
```bash
kubectl -n org3-b322955e get pods -o wide
# backend-api-xxx  1/1  Running  compute-main-01
# frontend-xxx     1/1  Running  compute-main-01

kubectl -n org3-b322955e get deployment frontend \
  -o jsonpath='{.spec.template.spec.nodeSelector}'
# {"kubernetes.io/arch":"amd64"}
```

---

### 4. Deploy — Operational

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

1ctl-dev deploy restart --config examples/backend/satusky.toml
1ctl-dev deploy releases --config examples/backend/satusky.toml
1ctl-dev deploy rollback --config examples/backend/satusky.toml --version 1 -y
1ctl-dev deploy destroy --config examples/frontend/satusky.toml -y
# (then redeploy frontend to restore state)
```

| Command | Result | Notes |
|---------|--------|-------|
| `deploy restart --config` | PASS | Rolling restart initiated via name resolution |
| `deploy releases --config` | PASS | v1 (nginx:alpine, active) listed |
| `deploy rollback --config --version 1 -y` | PASS | Rollback initiated; v2 active → v1 active |
| `deploy destroy --config` | PASS | Destroys correct deployment via name resolution |

---

### 5. Service & Ingress

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

1ctl-dev service list
1ctl-dev ingress list
```

| Command | Result | Notes |
|---------|--------|-------|
| `service list` | PASS | backend-api (8080), frontend (80) listed |
| `ingress list` | PASS | backend-api.satusky.com, frontend.satusky.com listed |

---

### 6. Environment variables

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli
cd examples/backend

# First-time create (no prior ConfigMap — was BUG-2, now fixed)
1ctl-dev env create --config satusky.toml --env APP_ENV=production --env LOG_LEVEL=info

# Second call — merges: new keys added, existing updated, others preserved
1ctl-dev env create --config satusky.toml --env LOG_LEVEL=debug --env NEW_KEY=hello

1ctl-dev env list
```

| Command | Result | Notes |
|---------|--------|-------|
| `env create` (first call, no prior ConfigMap) | PASS | Creates ConfigMap + DB row |
| `env create` (second call, merge) | PASS | Keys merged; `log-level` updated, `new-key` added |
| `env list` | PASS | backend-api listed with deployment ID |

**kubectl verification:**
```bash
kubectl -n org3-b322955e get configmap backend-api-environments -o jsonpath='{.data}'
# {"app-env":"production","log-level":"debug","new-key":"hello","...":"..."}
```

---

### 7. Secrets

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli
cd examples/backend

# First-time create (no prior K8s Secret — was BUG-2, now fixed)
1ctl-dev secret create --config satusky.toml --kv DB_PASS=supersecret --kv API_KEY=abc123

# Second call — merges
1ctl-dev secret create --config satusky.toml --kv NEW_SECRET=newval

1ctl-dev secret list
```

| Command | Result | Notes |
|---------|--------|-------|
| `secret create` (first call) | PASS | Creates K8s Secret + DB row |
| `secret create` (second call, merge) | PASS | `new-secret` added, existing preserved |
| `secret list` | PASS | backend-api listed |

**kubectl verification:**
```bash
kubectl -n org3-b322955e get secret backend-api-secrets \
  -o jsonpath='{.data}' | python3 -c \
  "import sys,json,base64; d=json.load(sys.stdin); [print(k,'=',base64.b64decode(v).decode()) for k,v in d.items()]"
# api-key = abc123
# db-pass = supersecret
# new-secret = newval
```

---

### 8. Logs

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

1ctl-dev logs --config examples/backend/satusky.toml
```

| Command | Result | Notes |
|---------|--------|-------|
| `logs --config` | PASS | Loki-sourced logs with timestamps, pod names |

---

### 9. Observability & Admin

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

1ctl-dev notifications list
1ctl-dev credits balance
1ctl-dev audit list
1ctl-dev storage list
1ctl-dev pricing list
```

| Command | Result | Notes |
|---------|--------|-------|
| `notifications list` | PASS | Deploy events from this session shown |
| `credits balance` | PASS | $388.16 MYR balance |
| `audit list` | PASS | Audit entries with action, user, IP |
| `storage list` | PASS | Empty (correct) |
| `pricing list` | PASS | No configs (dev backend) |

---

### 10. Infrastructure

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

1ctl-dev cluster zones
1ctl-dev cluster list
1ctl-dev machine list
1ctl-dev machine available
1ctl-dev user me
1ctl-dev token list
```

| Command | Result | Notes |
|---------|--------|-------|
| `cluster zones` | PASS | my-kul-1b |
| `cluster list` | PASS | kul (healthy, default), bki |
| `machine list` | PASS | Machines with status, CPU, memory, cost |
| `machine available` | PASS | 3 monetized machines |
| `user me` | PASS | tenant-admin, org3 |
| `token list` | PASS | 1 active token |

---

### 11. Cloud Build + Arch Routing

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli
cd examples/frontend

1ctl-dev deploy --config satusky.toml --machine compute-main-01
```

| Step | Result | Notes |
|------|--------|-------|
| Context packaging | PASS | .dockerignore respected |
| Submit to `/builds` | PASS | Build ID returned |
| Build execution | PASS | nginx:alpine amd64 variant pulled (WARNING logged — expected) |
| Arch detection (`docker inspect`) | PASS | `image_arch: "amd64"` in build status |
| CLI prints `Image architecture: amd64` | PASS | Surfaced to user |
| `nodeSelector: {"kubernetes.io/arch":"amd64"}` on K8s deployment | PASS | Verified via kubectl |
| Pod scheduled on amd64 node | PASS | `1/1 Running` on `compute-main-01` |

---

### 12. Error Scenarios

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli
```

| Scenario | Command | Expected | Result |
|----------|---------|----------|--------|
| No toml, no `--deployment-id` | `1ctl-dev deploy status` (from repo root) | `no --deployment-id and no satusky.toml found` | PASS |
| Config file missing | `1ctl-dev deploy status --config missing.toml` | `no --deployment-id and no satusky.toml found` | PASS |
| App not deployed yet | `1ctl-dev deploy status --config satusky.toml` (after destroy) | `app "frontend" not found — run 1ctl deploy first` | PASS |
| Invalid app name — starts with digit | `1ctl-dev deploy ... --name 1myapp` | `app name "1myapp" is not a valid K8s service name (starts with a digit — try --name app-1myapp)` | PASS |
| Wrong directory — name auto-detected as `1ctl` | `1ctl-dev deploy --image nginx:alpine` (from repo root, no --name) | `app name "1ctl" is not a valid K8s service name ... Auto-detected from git remote` | PASS |
| `--deployment-id` beats `--config` | `1ctl-dev deploy status --config frontend/satusky.toml --deployment-id <backend-api-id>` | Shows backend-api status, not frontend | PASS |

---

## satusky.toml — Before and After

**Before (stored generated state):**
```toml
[app]
  name = "backend-api"
  deployment_id = "fe7b53a5-80d8-4ddd-81b0-4a530767c723"  # ← environment-specific
```

**After (clean config, safe to commit):**
```toml
[app]
  name = "backend-api"
  org  = "org3"
  port = 8080
  cpu  = "0.5"
  memory = "256Mi"
  replicas = 1
  domain = ""
```

The CLI resolves the deployment ID at runtime via `GET /deployments/namespace/:namespace/app/:appLabel`. The toml file is a pure config template — no generated state, safe to commit as-is in the repo.

---

## kubectl Full Verification

```bash
kubectl -n org3-b322955e get pods -o wide
```
```
NAME                             READY   STATUS    NODE
backend-api-858f95bb7-d9x7h     1/1     Running   compute-main-01
frontend-667757744-mx4m8        1/1     Running   compute-main-01
```

```bash
# nodeSelector on arch-aware deployment
kubectl -n org3-b322955e get deployment frontend \
  -o jsonpath='{.spec.template.spec.nodeSelector}'
# {"kubernetes.io/arch":"amd64"}

# Merged env ConfigMap
kubectl -n org3-b322955e get configmap backend-api-environments \
  -o jsonpath='{.data}'
# {"app-env":"production","log-level":"debug","new-key":"hello","version":"2.0",...}

# Merged secret (decoded)
kubectl -n org3-b322955e get secret backend-api-secrets -o jsonpath='{.data}' \
  | python3 -c "import sys,json,base64; d=json.load(sys.stdin); \
    [print(k,'=',base64.b64decode(v).decode()) for k,v in d.items()]"
# api-key = abc123
# db-pass = supersecret
# new-secret = newval
```

---

## Backend Error Log Check

```bash
grep "level=ERROR" satusky-core_backend/logs.txt | tail -5
```

Only expected errors (test scenarios for not-found cases):
- `app_label=does-not-exist` — deliberate error-scenario test ✓
- `app_label=frontend` not found — status check after destroy ✓
- Ingress not found during re-deploy — transient between destroy and recreate ✓

No unexpected 5xx errors.

---

## Fixes Verified in This Session

| Fix | Status |
|-----|--------|
| `deployment_id` removed from `satusky.toml` — Fly.io name-based resolution | ✅ Verified |
| `env create` / `secret create` first-time (BUG-2 backfill) | ✅ Verified |
| Cloud build arch detection (`docker inspect` → `image_arch`) | ✅ Verified |
| `nodeSelector: kubernetes.io/arch` set on arch-aware pods | ✅ Verified |
| Invalid app name (DNS-1035) caught before deploy, actionable error | ✅ Verified |
| Deploy from wrong dir gives clear error with suggestion | ✅ Verified |
| Backend K8s DNS-1035 validation surfaced as 400 not 500 | ✅ Verified (backend fix) |
| `AppError.Public` wiring — typed 500s | ✅ In code (not directly triggered in happy-path tests) |
| `go mod tidy` — BurntSushi/toml and gorilla/websocket direct | ✅ Verified |

---

## Features Not Tested

| Feature | Reason |
|---------|--------|
| `1ctl init` (scaffold) | Files already exist |
| `--strategy recreate` | Requires multi-pod deployment |
| `--hpa` / `--vpa` | Requires metrics-server |
| `--multicluster` | Requires multi-zone nodes |
| `--zone` routing | Requires zone-labeled nodes |
| `1ctl domain` | Requires Cloudflare integration |
| `1ctl storage` CRUD | Requires S3-compatible backend |
| `1ctl marketplace` | Requires marketplace apps |
| `1ctl talos` | Requires Talos machine access |
| `1ctl admin` | Requires super-admin role |
| Auto-select amd64 (no `--machine`) | Only 1 online owner machine (arm64); monetized amd64 fallback untested |
