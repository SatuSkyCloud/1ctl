# 1ctl CLI Test Report

**Date**: 2026-04-26 (retest — all prior bugs addressed)
**Branch**: development
**Backend**: satusky-core_backend @ localhost:8080
**Namespace**: org3-b322955e
**User**: mingerz.k@gmail.com
**Org**: org3 (b322955e-6a86-4157-8bff-1bea605ef8ac)

> **All commands below target the local backend.**
>
> The installed `1ctl` (Homebrew, v0.6.0) does **not** have `profile create`.
> Use one of these approaches before any section:
>
> **Option A — env var (works with any binary):**
> ```bash
> export SATUSKY_API_URL=http://localhost:8080/v1/cli
> 1ctl auth login --token <token>
> ```
>
> **Option B — dev binary (profile subcommands available):**
> ```bash
> # Build once from repo root:
> go build -ldflags "-X 1ctl/internal/config.defaultAPIURL=http://localhost:8080/v1/cli" \
>   -o bin/1ctl-dev ./cmd/...
> sudo cp bin/1ctl-dev /usr/local/bin/1ctl-dev
> # Then use 1ctl-dev everywhere instead of 1ctl
> ```
>
> Commands in this report use bare `1ctl` for readability. Prefix with
> `SATUSKY_API_URL=http://localhost:8080/v1/cli` or use `1ctl-dev` if the
> env var isn't already set in your shell.

---

## Test Summary

| Category | Commands Tested | Pass | Fail | Notes |
|----------|----------------|------|------|-------|
| Auth & Profile | 8 | 8 | 0 | |
| Org | 3 | 3 | 0 | |
| Deploy (core) | 7 | 7 | 0 | Destroy + fresh redeploy verified |
| Deploy (ops) | 3 | 3 | 0 | |
| Service | 1 | 1 | 0 | |
| Ingress | 1 | 1 | 0 | |
| Environment | 3 | 3 | 0 | **BUG-2 FIXED** — first-time create works |
| Secret | 2 | 2 | 0 | **BUG-2 FIXED** — first-time create works |
| Logs | 1 | 1 | 0 | |
| Notifications | 1 | 1 | 0 | |
| Credits | 1 | 1 | 0 | |
| Storage | 1 | 1 | 0 | |
| Audit | 1 | 1 | 0 | |
| Cluster | 2 | 2 | 0 | |
| Machine | 2 | 2 | 0 | |
| Token | 1 | 1 | 0 | |
| User | 1 | 1 | 0 | |
| Pricing | 1 | 1 | 0 | |
| **Total** | **41** | **41** | **0** | **All clear** |

---

## Commands Tested — Detailed Results

### 1. Auth & Profile

> `profile create/list/use/current` require the dev binary (`1ctl-dev`).
> The Homebrew release (v0.6.0) only understands `SATUSKY_API_URL`.

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

1ctl auth login --token eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
1ctl auth status

# Profile subcommands — dev binary only:
1ctl-dev profile create --url http://localhost:8080/v1/cli local
1ctl-dev profile list
1ctl-dev profile current
1ctl-dev profile use local
1ctl-dev --profile local auth status
```

| Command | Binary | Result | Notes |
|---------|--------|--------|-------|
| `export SATUSKY_API_URL=http://localhost:8080/v1/cli` | any | PASS | Highest priority; works with Homebrew release |
| `1ctl auth login --token <JWT>` | any | PASS | Token stored; email, org, namespace returned |
| `1ctl auth status` | any | PASS | email mingerz.k@gmail.com, org3, token 73d expiry |
| `1ctl-dev profile create --url ... local` | dev | PASS | Creates local profile (not in Homebrew v0.6.0) |
| `1ctl-dev profile list` | dev | PASS | local (active) + prod profiles shown |
| `1ctl-dev profile current` | dev | PASS | URL, auth, org confirmed |
| `1ctl-dev profile use local` | dev | PASS | Profile switch confirmed |
| `1ctl-dev --profile local auth status` | dev | PASS | One-shot override works |

### 2. Organization

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

1ctl org list
1ctl org current
```

| Command | Result | Notes |
|---------|--------|-------|
| `1ctl org list` | PASS | 3 orgs listed with IDs, names, dates |
| `1ctl org current` | PASS | org3 with namespace org3-b322955e |

### 3. Deploy — Core

Deployment IDs from this session:
- backend-api: `fe7b53a5-80d8-4ddd-81b0-4a530767c723`
- frontend: `38ab5d6b-c3cc-45c4-b0ef-ceab41cc9207`

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

# Destroy existing deployments
1ctl deploy destroy --deployment-id 982d638e-f518-4450-b637-832ef4663e72 -y
1ctl deploy destroy --deployment-id c73e3fe4-4ebf-4dd8-bd45-59097ded3bc4 -y

# Redeploy backend-api (pre-built multi-arch image)
cd examples/backend
1ctl deploy --cpu 0.5 --memory 256Mi --port 8080 --image nginx:alpine --machine compute-main-01

# Redeploy frontend (cloud build — arch detection + nodeSelector)
cd examples/frontend
1ctl deploy --cpu 0.25 --memory 128Mi --port 80 --machine compute-main-01

# Verify
1ctl deploy list
1ctl deploy get --deployment-id fe7b53a5-80d8-4ddd-81b0-4a530767c723
1ctl deploy status --deployment-id fe7b53a5-80d8-4ddd-81b0-4a530767c723
```

| Command | Result | Notes |
|---------|--------|-------|
| `1ctl deploy destroy ... -y` (×2) | PASS | Both deployments torn down cleanly |
| `1ctl deploy --image nginx:alpine --machine compute-main-01` | PASS | 5-step pipeline: build skip, deployment, service, env/storage, ingress |
| `1ctl deploy --cpu 0.25 --memory 128Mi --port 80 --machine compute-main-01` (cloud build) | PASS | Cloud build → arch=amd64 detected → nodeSelector set → 1/1 Running on compute-main-01 |
| `1ctl deploy list` | PASS | Both deployments listed |
| `1ctl deploy get --deployment-id <id>` | PASS | Full details including version, port, CPU, memory |
| `1ctl deploy status --deployment-id <id>` | PASS | Status: Running, Progress: 100% |
| `1ctl deploy get --config satusky.toml` | PASS | Resolves deployment_id from toml |

### 4. Deploy — Operational

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

1ctl deploy restart --deployment-id fe7b53a5-80d8-4ddd-81b0-4a530767c723
1ctl deploy releases --deployment-id fe7b53a5-80d8-4ddd-81b0-4a530767c723
1ctl deploy rollback --deployment-id fe7b53a5-80d8-4ddd-81b0-4a530767c723 --version 1 -y
```

| Command | Result | Notes |
|---------|--------|-------|
| `1ctl deploy restart` | PASS | Rolling restart initiated, pods replaced |
| `1ctl deploy releases` | PASS | Version 1 (nginx:alpine, active) listed |
| `1ctl deploy rollback --version 1 -y` | PASS | Rollback to v1 initiated |

### 5. Service & Ingress

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

1ctl service list
1ctl ingress list
```

| Command | Result | Notes |
|---------|--------|-------|
| `1ctl service list` | PASS | backend-api (8080) and frontend (80) |
| `1ctl ingress list` | PASS | backend-api.satusky.com and frontend.satusky.com |

### 6. Environment & Secrets — BUG-2 FIXED

Both commands now work on **fresh deployments** with no prior ConfigMap/Secret.

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

# Env: first-time create (no prior ConfigMap) — was BUG-2
cd examples/backend
1ctl env create --config satusky.toml --env APP_NAME=backend-api --env LOG_LEVEL=info

# Env: second call (update path, merges new keys)
1ctl env create --config satusky.toml --env APP_NAME=backend-api --env LOG_LEVEL=debug --env VERSION=2.0

# List
1ctl env list

# Secret: first-time create (no prior K8s Secret) — was BUG-2
1ctl secret create --config satusky.toml --kv DB_PASS=supersecret --kv API_KEY=abc123

# Secret: second call (merges keys)
1ctl secret create --config satusky.toml --kv DB_PASS=newpassword

# List
1ctl secret list
```

| Command | Result | Notes |
|---------|--------|-------|
| `1ctl env create --env ...` (first call, fresh deployment) | **PASS** | Previously returned "No existing environment found". Now creates ConfigMap + DB row. |
| `1ctl env create --env ...` (second call, update) | PASS | Merges new keys; CONFIG MAP shows `app-name`, `log-level=debug`, `version=2.0` |
| `1ctl env list` | PASS | backend-api env listed with deployment ID |
| `1ctl secret create --kv ...` (first call, fresh deployment) | **PASS** | Previously returned "No existing secret found". Now creates K8s Secret + DB row. |
| `1ctl secret create --kv ...` (second call, update) | PASS | Merges keys; K8s Secret contains api-key + db-pass (updated) |
| `1ctl secret list` | PASS | backend-api secret listed |

**Verification** (`kubectl -n org3-b322955e get configmap backend-api-environments -o jsonpath='{.data}'`):
```json
{"app-name":"backend-api","log-level":"debug","version":"2.0"}
```

### 7. Logs

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

1ctl logs --deployment-id fe7b53a5-80d8-4ddd-81b0-4a530767c723
```

| Command | Result | Notes |
|---------|--------|-------|
| `1ctl logs --deployment-id <id>` | PASS | Loki-sourced log lines with timestamps, pod names |

### 8. Observability & Admin

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

1ctl notifications list
1ctl credits balance
1ctl storage list
1ctl audit list
1ctl pricing list
```

| Command | Result | Notes |
|---------|--------|-------|
| `1ctl notifications list` | PASS | Deployment events including this session's creates/deletes |
| `1ctl credits balance` | PASS | $388.16 MYR balance |
| `1ctl storage list` | PASS | No storage configs (correct) |
| `1ctl audit list` | PASS | Audit entries with action, user, resource, IP |
| `1ctl pricing list` | PASS | No configs (dev backend) |

### 9. Infrastructure

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

1ctl cluster zones
1ctl cluster list
1ctl machine list
1ctl machine available
1ctl user me
1ctl token list
```

| Command | Result | Notes |
|---------|--------|-------|
| `1ctl cluster zones` | PASS | 1 zone: my-kul-1b |
| `1ctl cluster list` | PASS | kul (healthy, default) and bki |
| `1ctl machine list` | PASS | Machines listed with status, CPU, memory, cost |
| `1ctl machine available` | PASS | 3 monetized machines with scores |
| `1ctl user me` | PASS | Profile: tenant-admin, org3 |
| `1ctl token list` | PASS | 1 active token |

---

## Cloud Build + Arch Routing

Full end-to-end cloud build pipeline with architecture detection verified.

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

cd examples/frontend
1ctl deploy --cpu 0.25 --memory 128Mi --port 80 --machine compute-main-01
```

| Step | Result | Notes |
|------|--------|-------|
| Package build context | PASS | .dockerignore respected |
| Submit to `/builds` endpoint | PASS | Build ID returned immediately |
| Build execution (Podman/Docker) | PASS | nginx:alpine amd64 variant pulled; WARNING logged |
| `docker inspect` arch detection | **PASS** | `image_arch: "amd64"` returned in build status |
| Image push to registry | PASS | Pushed to `registry.satusky.com/satusky-container-registry/frontend:<build-id>` |
| Build log streaming | PASS | Live output streamed |
| CLI prints `Image architecture: amd64` | **PASS** | Arch surfaced to user |
| `nodeSelector: kubernetes.io/arch=amd64` on K8s deployment | **PASS** | Verified via `kubectl get deployment frontend -o jsonpath='{.spec.template.spec.nodeSelector}'` → `{"kubernetes.io/arch":"amd64"}` |
| Pod schedules on amd64 node | **PASS** | `frontend-9ccff8f88-zrkcm 1/1 Running compute-main-01` |

### Arch Routing — How It Works

Build host is macOS arm64 (Podman). During the frontend build, Podman pulled the **amd64 variant** of `nginx:alpine` (logged as a WARNING). The final image was therefore `linux/amd64`. `docker inspect` confirmed this and the CLI printed `Image architecture: amd64`.

With `--machine compute-main-01` (amd64 node), the deployment:
1. Set `target_arch = "amd64"` on the deploy request
2. Backend set `nodeSelector: {"kubernetes.io/arch": "amd64"}` in the pod spec
3. Pod scheduled on `compute-main-01` ✓

**What happens without `--machine` (auto-select):** After the DB fix (see below), the auto-select loop now filters owner machines by arch. The only online owner machine (`worker-efc2fd3e1276...`) is arm64 and is skipped for amd64 images, giving a clear warning: "No owner machines with arch amd64 are online — will use monetized machines."

**What was the original exec format error?** nodeSelector now prevents it at the K8s layer. If the wrong arch machine is selected, the pod stays Pending with a K8s scheduling error instead of starting and crashing.

### DB Fix Applied: arm64 Machine Metadata

The machine querier defaults `cpu_arch` to `"amd64"` when the DB metadata field is unset. Six arm64 worker nodes had no `cpu_arch` set in the `metadata` JSONB column. This caused the arch filter to incorrectly pass them as "amd64-compatible" machines.

**Fix applied:**
```sql
UPDATE machines
SET metadata = metadata || '{"cpu_arch": "arm64"}'::jsonb
WHERE machine_id IN (
    '07a3a8460746383ca552c3d48c7911d5',
    '2a8152f42d37779ddf81f1dbe76d82d8',
    '3f113bd539e6d4eb025d2462f9490103',
    '9462f5524146828123aa98c8b951fb30',
    'd45cf2d88d96cd5b27a99a32c4847e3a',
    'efc2fd3e1276d75d1d638049d1dfcbf7'
);
-- 6 rows updated
```

After the fix, arch filtering in the CLI auto-select loop correctly excludes arm64 machines when deploying amd64 images.

---

## Bugs Fixed (this session)

### BUG-2: Backend env/secret upsert 404 on first standalone call — FIXED

**Root cause:** `CreateOrUpdateConfigmap` and `CreateOrUpdateSecret` returned 404 when the K8s ConfigMap/Secret existed but no DB row was present (state drift). A backfill was added: when K8s state is ahead of DB, the handler now creates the missing DB row and falls through to the normal update path.

**Files changed:**
- `satusky-core_backend/controllers/environment_controller.go:189`
- `satusky-core_backend/controllers/secret_controller.go:263`

**Before:**
```
$ 1ctl env create --config satusky.toml --env FOO=bar
❌ failed to upsert environment: No existing environment found

$ 1ctl secret create --config satusky.toml --kv DB_PASS=secret
❌ failed to create secret: No existing secret found
```

**After:**
```
$ 1ctl env create --config satusky.toml --env APP_NAME=backend-api --env LOG_LEVEL=info
✅ Environment backend-api created successfully

$ 1ctl secret create --config satusky.toml --kv DB_PASS=supersecret --kv API_KEY=abc123
✅ Secret backend-api created successfully
```

### Arch mismatch (exec format error) — FIXED via nodeSelector + arch routing

**Root cause:** Cloud builds on an arm64 Mac produce arm64 images; deploying to amd64 nodes caused `exec format error`. No nodeSelector was being set.

**Fix:**
1. `build_controller.go`: `docker inspect` after build captures `image_arch`; returned in `/builds/:id/status`
2. `services/deployment/k8s.go`: `nodeSelector: {"kubernetes.io/arch": <arch>}` added to PodSpec when `TargetArch` is set
3. CLI: arch flows from build status → `DeploymentOptions.TargetArch` → `Deployment.TargetArch` in API request
4. DB: 6 arm64 worker machines patched with `cpu_arch=arm64` in metadata

**Result:** Wrong-arch images now get a scheduling error ("unmatched node selector") instead of starting and crashing with `exec format error`.

---

## Issues Fixed (previous session, still verified)

### ISSUE-2: AppError.Public wiring for typed 500s

`Response.InternalError` now checks for `AppError{Public: true}` and forwards the message to the client. Business-logic errors can opt in; structural errors still return a generic message. Backward-compatible.

**Files changed:**
- `satusky-core_backend/helpers/error_handler.go`: Added `Public bool` to `AppError`
- `satusky-core_backend/helpers/response_builder.go`: `InternalError` uses `errors.As` to check it

---

## kubectl Verification

```bash
kubectl -n org3-b322955e get deployments,pods -o wide
```

```
NAME                      READY   IMAGE
backend-api               1/1     nginx:alpine
frontend                  1/1     registry.satusky.com/satusky-container-registry/frontend:5dae849d

NAME                                       READY   NODE
backend-api-586cf67875-6b2x2               1/1     compute-main-01
frontend-9ccff8f88-zrkcm                   1/1     compute-main-01
```

Node selectors:
```bash
kubectl -n org3-b322955e get deployment frontend -o jsonpath='{.spec.template.spec.nodeSelector}'
# {"kubernetes.io/arch":"amd64"}

kubectl -n org3-b322955e get deployment backend-api -o jsonpath='{.spec.template.spec.nodeSelector}'
# (empty — nginx:alpine is multi-arch, no TargetArch sent)
```

ConfigMap contents:
```bash
kubectl -n org3-b322955e get configmap backend-api-environments -o jsonpath='{.data}'
# {"app-name":"backend-api","log-level":"debug","version":"2.0"}
```

---

## Verification Checklist

| Check | Result |
|-------|--------|
| `go build -o bin/1ctl-dev ./cmd/...` | PASS |
| `go test ./...` | PASS |
| `go mod tidy` | PASS — BurntSushi/toml and gorilla/websocket promoted to direct |
| Deploy list / get / status | PASS |
| Destroy both deployments | PASS |
| Redeploy backend-api (nginx:alpine, pre-built) | PASS |
| Redeploy frontend (cloud build, arch detected) | PASS — amd64 detected, nodeSelector=amd64 |
| Pod placement: both on compute-main-01 (amd64) | PASS |
| `nodeSelector: kubernetes.io/arch=amd64` on frontend | PASS |
| `env create` on fresh deployment (BUG-2) | **PASS** (was FAIL before) |
| `env create` second call (update, merge) | PASS |
| `secret create` on fresh deployment (BUG-2) | **PASS** (was FAIL before) |
| `secret create` second call (update, merge) | PASS |
| ConfigMap values correct in K8s | PASS |
| K8s Secret values correct (decoded) | PASS |
| DB arm64 machines patched | PASS — 6 rows updated |
| Arch filter skips arm64 machine for amd64 image | PASS (after DB fix) |

---

## Features NOT Tested

| Feature | Reason |
|---------|--------|
| `1ctl init` (scaffold) | File already exists |
| `--strategy recreate` | Requires multi-pod deployment |
| `--hpa` / `--vpa` | Requires metrics-server |
| `--multicluster` | Requires multi-zone nodes |
| `--zone` routing | Requires zone-labeled nodes |
| `1ctl domain` | Requires Cloudflare integration |
| `1ctl storage` CRUD | Requires S3-compatible backend |
| `1ctl marketplace` | Requires marketplace apps |
| `1ctl talos` | Requires Talos machine access |
| `1ctl admin` | Requires super-admin role |
| Auto-select amd64 without `--machine` | Only 1 online owner machine (arm64); monetized amd64 fallback untested |
