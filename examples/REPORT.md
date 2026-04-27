# 1ctl CLI Test Report

**Date**: 2026-04-27 (post gap-fixes retest)
**Branch**: development
**Backend**: satusky-core_backend @ localhost:8080 (`sudo task dev.debug > logs.txt 2>&1`)
**Namespace**: org3-b322955e
**User**: mingerz.k@gmail.com
**Org**: org3 (b322955e-6a86-4157-8bff-1bea605ef8ac)
**Binary**: `bin/1ctl-dev` (built from source — see build instructions below)
**CLI version**: `dev`

> **Can I use the Homebrew `1ctl` instead?**
> **No.** v0.6.0 is missing `env unset`, `secret unset`, `--wait`, `-o json`,
> `logs stream --config`, `org switch <name>`, and all bug fixes made in this sprint.
> You must build from source.

> **Build (one-time):**
> ```bash
> cd /path/to/1ctl
> go build -o bin/1ctl-dev ./cmd/...        # no -ldflags needed
> sudo cp bin/1ctl-dev /usr/local/bin/1ctl-dev   # optional: add to PATH
> ```

> **Setup — run once per shell session:**
> ```bash
> export SATUSKY_API_URL=http://localhost:8080/v1/cli   # points at local backend
> 1ctl-dev profile use local                             # activates stored credentials
> ```
> `SATUSKY_API_URL` overrides the active profile URL. The `--api-url` flag does
> the same per-command. Add the export to `~/.zshrc` to avoid repeating it.

---

## Test Summary

| Category | Tested | Pass | Fail | Notes |
|---|---|---|---|---|
| auth | login, logout, status | 3 | 0 | |
| profile | create, use, current, list, delete | 5 | 0 | |
| org | list, current, switch (flag + positional) | 4 | 0 | **Gap 6 fixed** |
| init | init (clean toml) | 1 | 0 | **Gap 2 fixed** |
| deploy — core | list, get, status, deploy (toml+flags+defaults), --wait, --output json | 9 | 0 | **Gap 3+4 fixed** |
| deploy — ops | restart, releases, rollback | 3 | 0 | |
| service | list, delete | 2 | 0 | |
| ingress | list, delete | 2 | 0 | |
| env | create (first+merge), list, unset | 4 | 0 | **Gap 5 fixed** |
| secret | create (first+merge), list, unset | 4 | 0 | **Gap 5 fixed** |
| logs | stored, stream (--config + -d) | 3 | 0 | **Gap 1 fixed** |
| notifications | list, count, read --all | 3 | 0 | |
| user | me, permissions | 2 | 0 | |
| token | list, get, create, disable, enable, delete | 6 | 0 | |
| marketplace | list, get | 2 | 0 | |
| audit | list, get | 2 | 0 | |
| credits | balance, transactions, usage | 3 | 0 | |
| pricing | list, lookup | 2 | 0 | no data in dev (expected) |
| storage | list | 1 | 0 | |
| cluster | zones, list | 2 | 0 | |
| machine | list (--output json), available, usage | 3 | 0 | |
| issuer | list | 1 | 0 | |
| completion | zsh, bash | 2 | 0 | |
| **Total** | **69** | **69** | **0** | |

---

## New Features — Detailed Results

### Gap 1: `logs stream --config`

Previously `logs stream` only accepted `-d`/`--deployment-id`. Now accepts `--config` like every other deployment-scoped command.

```bash
# Before (only way):
1ctl-dev logs stream -d 7f1fab9e-5f87-4612-b306-3da846b95d18

# After (both work):
1ctl-dev logs stream --config examples/backend/satusky.toml
1ctl-dev logs stream -d <deployment-id>
```

| Command | Result |
|---------|--------|
| `logs stream --config satusky.toml` | PASS — `Streaming logs for org3-b322955e/backend-api` |
| `logs stream -d <id>` | PASS — direct ID still works |

---

### Gap 2: `init` produces clean toml

Previously `init` wrote `cpu = ""`, `memory = ""`, `replicas = 0`, `domain = ""` — noise fields with zero values. Now only writes fields that have non-default values.

```bash
mkdir /tmp/myapp && cd /tmp/myapp
1ctl-dev init
cat satusky.toml
```

Output:
```toml
[app]
  name = "myapp"
  port = 8080
```

| Command | Result |
|---------|--------|
| `init` | PASS — only `name` and `port` written |

---

### Gap 3: `--output json` global flag

All list and get commands now accept `--output json` / `-o json` for machine-readable output. The flag is global — apply it before any subcommand.

```bash
# Deploy list as JSON
1ctl-dev --output json deploy list | python3 -c "
import json,sys; d=json.load(sys.stdin)
for x in d: print(x['image'], x['deployment_id'][:8])"
# nginx:alpine           7f1fab9e
# registry.satusky.com/... c16a1454

# Pipe into jq for selective fields
1ctl-dev -o json deploy list | jq '.[].image'
1ctl-dev -o json env list | jq '.[0].key_values'
1ctl-dev -o json machine list | jq '.[] | select(.status == "online") | .machine_name'
1ctl-dev -o json token list | jq '.[] | {name, is_active}'
```

| Command | Result |
|---------|--------|
| `--output json deploy list` | PASS — valid JSON array of deployments |
| `--output json deploy get` | PASS — valid JSON object |
| `--output json deploy status` | PASS — valid JSON object |
| `-o json env list` | PASS — valid JSON array |
| `-o json secret list` | PASS — valid JSON array |
| `-o json machine list` | PASS — valid JSON array |
| `-o json token list` | PASS — valid JSON array |

**Commands with `--output json` wired in (confirmed):**
`deploy list/get/status`, `env list`, `secret list`, `machine list`, `token list`

**Not yet wired (`-o json` silently falls back to table):**
`service list`, `ingress list`, `audit list`, `notifications list`, `cluster list`

---

### Gap 4: `--wait` / `-w` on deploy

After the 5-step pipeline completes, `--wait` polls until pods are Running (5-minute timeout).

```bash
1ctl-dev deploy --config examples/backend/satusky.toml \
  --image nginx:alpine --machine compute-main-01 --wait
```

Output:
```
Step 2/5: Creating/updating deployment backend-api ✓
...
Step 5/5: Configuring ingress and dependencies backend-api ✓
✅ 🚀 Deployment for backend-api is successful! Your app is live at: https://backend-api.satusky.com
Deployment ID: 7f1fab9e-5f87-4612-b306-3da846b95d18
💡 Waiting for deployment to become healthy...
✅ Deployment is healthy — pods Running
```

| Command | Result |
|---------|--------|
| `deploy ... --wait` | PASS — blocks until `Running`, then prints healthy |
| `deploy ... -w` | PASS — alias works |

---

### Gap 5: `env unset` and `secret unset`

Per-key removal. Previously there was no way to remove a specific key without deleting the whole resource. `env delete` and `secret delete` were removed as dangerous ("delete all") — this replaces them with the correct per-key primitive.

```bash
cd examples/backend

# Add a key
1ctl-dev env create --config satusky.toml --env TEMP_KEY=remove_me
# → ConfigMap: {..., "temp-key": "remove_me"}

# Remove just that key
1ctl-dev env unset --config satusky.toml --key TEMP_KEY
# → ConfigMap: {...}  (TEMP_KEY gone, others untouched)

# Same for secrets
1ctl-dev secret create --config satusky.toml --kv TEMP_SECRET=gone
1ctl-dev secret unset  --config satusky.toml --key TEMP_SECRET
```

| Command | Result |
|---------|--------|
| `env unset --config satusky.toml --key TEMP_KEY` | PASS — `✅ Key "TEMP_KEY" removed from environment` |
| `secret unset --config satusky.toml --key TEMP_SECRET` | PASS — `✅ Key "TEMP_SECRET" removed from secrets` |
| K8s ConfigMap after unset | PASS — key absent, all other keys preserved |

---

### Gap 6: `org switch` positional arg

Previously required `--org-id` or `--org-name` flags. Now accepts a positional argument — UUID is treated as org-id, any other string as org-name.

```bash
# Before (only way):
1ctl-dev org switch --org-id 690839ba-3aed-47ea-a8ec-0cd019e4d180
1ctl-dev org switch --org-name org2

# After (all work):
1ctl-dev org switch org2
1ctl-dev org switch 690839ba-3aed-47ea-a8ec-0cd019e4d180
1ctl-dev org switch --org-id 690839ba-3aed-47ea-a8ec-0cd019e4d180   # flags still work
```

| Command | Result |
|---------|--------|
| `org switch org2` (name, positional) | PASS — `✅ Switched to organization: org2` |
| `org switch b322955e-...` (UUID, positional) | PASS — `✅ Switched to organization: org3` |

---

## All Commands — Current State

### auth

```bash
1ctl-dev auth status
1ctl-dev auth logout
1ctl-dev auth login --token <jwt>
```

| Command | Result |
|---------|--------|
| `auth status` | PASS |
| `auth logout` | PASS |
| `auth login --token` | PASS |

### profile

```bash
1ctl-dev profile list
1ctl-dev profile current
1ctl-dev profile create --url http://localhost:8080/v1/cli local
1ctl-dev profile use local
1ctl-dev profile delete <name>
```

All PASS. `profile` subcommands require the dev binary.

### org

```bash
1ctl-dev org list
1ctl-dev org current
1ctl-dev org switch org2          # positional name
1ctl-dev org switch <uuid>        # positional UUID
1ctl-dev org switch --org-id <id> # flag still works
```

All PASS.

### init

```bash
mkdir /tmp/test && cd /tmp/test
1ctl-dev init
# Produces:
# [app]
#   name = "test"
#   port = 8080
```

PASS — clean toml, no empty fields.

### deploy — core

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

# From toml
cd examples/backend
1ctl-dev deploy --config satusky.toml --image nginx:alpine --machine compute-main-01

# With --wait
1ctl-dev deploy --config satusky.toml --image nginx:alpine --machine compute-main-01 --wait

# No toml — port only, cpu/memory default to 0.5/256Mi
1ctl-dev deploy --port 8080 --image nginx:alpine --machine compute-main-01

# Cloud build with arch detection
cd examples/frontend
1ctl-dev deploy --config satusky.toml --machine compute-main-01 --wait

# JSON output
1ctl-dev -o json deploy list
1ctl-dev -o json deploy get --config examples/backend/satusky.toml
1ctl-dev -o json deploy status --config examples/backend/satusky.toml
```

| Command | Result |
|---------|--------|
| deploy from toml | PASS |
| deploy with `--wait` | PASS — polls until Running |
| deploy with no toml (defaults) | PASS — cpu=0.5, memory=256Mi |
| cloud build + arch detection | PASS — `Image architecture: amd64`, nodeSelector set |
| `--output json deploy list` | PASS — valid JSON |
| re-deploy (upsert) | PASS — same deployment ID reused |
| flag overrides toml (`--cpu 0.25`) | PASS — one-shot, file unchanged |

### deploy — ops

```bash
1ctl-dev deploy restart  --config examples/backend/satusky.toml
1ctl-dev deploy releases --config examples/backend/satusky.toml
1ctl-dev deploy rollback --config examples/backend/satusky.toml --version 1 -y
1ctl-dev deploy destroy  --config examples/backend/satusky.toml -y
```

All PASS.

### service / ingress

```bash
1ctl-dev service list
1ctl-dev service delete --service-id <id> -y

1ctl-dev ingress list
1ctl-dev ingress delete --ingress-id <id> -y
```

All PASS.

### env

```bash
cd examples/backend

1ctl-dev env create --config satusky.toml --env APP_ENV=production --env LOG_LEVEL=info
1ctl-dev env create --config satusky.toml --env LOG_LEVEL=debug     # merges
1ctl-dev env unset  --config satusky.toml --key LOG_LEVEL           # removes one key
1ctl-dev env list
1ctl-dev -o json env list
```

| Command | Result |
|---------|--------|
| `env create` (first — no prior ConfigMap) | PASS |
| `env create` (merge) | PASS |
| `env unset --key <key>` | PASS — single key removed, others preserved |
| `env list` | PASS |

**kubectl verify:**
```bash
kubectl -n org3-b322955e get configmap backend-api-environments -o jsonpath='{.data}'
```

### secret

```bash
cd examples/backend

1ctl-dev secret create --config satusky.toml --kv DB_PASS=s3cret --kv API_KEY=key123
1ctl-dev secret create --config satusky.toml --kv NEW_KEY=added     # merges
1ctl-dev secret unset  --config satusky.toml --key DB_PASS          # removes one key
1ctl-dev secret list
```

All PASS.

### logs

```bash
1ctl-dev logs --config examples/backend/satusky.toml
1ctl-dev logs stream --config examples/backend/satusky.toml  # --config now supported
1ctl-dev logs stream -d <deployment-id>
```

All PASS.

### notifications

```bash
1ctl-dev notifications list
1ctl-dev notifications count
1ctl-dev notifications read --all
```

All PASS.

### user / token

```bash
1ctl-dev user me
1ctl-dev user permissions

1ctl-dev token list
1ctl-dev -o json token list
1ctl-dev token create --name "ci-token"
1ctl-dev token disable <id>
1ctl-dev token enable  <id>
1ctl-dev token delete  <id> -y
```

All PASS.

### Other commands

```bash
1ctl-dev marketplace list
1ctl-dev marketplace get <id>
1ctl-dev audit list
1ctl-dev audit get <id>
1ctl-dev credits balance
1ctl-dev credits transactions
1ctl-dev credits usage
1ctl-dev pricing list
1ctl-dev storage list
1ctl-dev cluster zones
1ctl-dev cluster list
1ctl-dev machine list
1ctl-dev -o json machine list
1ctl-dev machine available
1ctl-dev machine usage list
1ctl-dev issuer list
1ctl-dev completion zsh
1ctl-dev completion bash
1ctl-dev --version
```

All PASS. (`pricing list` and `pricing lookup` return "no data" — expected for dev backend.)

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
kubectl -n org3-b322955ee get deploy frontend \
  -o jsonpath='{.spec.template.spec.nodeSelector}'
# {"kubernetes.io/arch":"amd64"}
```

No unexpected 5xx errors in backend logs.

---

## satusky.toml — Current Design

```toml
[app]
  name   = "backend-api"
  port   = 8080
  cpu    = "0.5"
  memory = "256Mi"
```

- No `org` — taken from auth context
- No `deployment_id` — resolved at runtime
- `cpu`/`memory` — optional (0.5 / 256Mi defaults)
- `init` only writes `name` and `port` — everything else is optional

---

## Removed Commands (dashboard features, not CLI features)

| Removed | Reason | Replacement |
|---------|--------|-------------|
| `logs stats` | Analytics → dashboard | — |
| `logs delete` | Bulk delete → dashboard | — |
| `audit export` | Compliance export → dashboard | — |
| `credits topup/invoices/auto-topup/notifications` | Billing config → web UI | — |
| `env delete` | Dangerous "delete all keys" | `env unset --key KEY` |
| `secret delete` | Dangerous "delete all keys" | `secret unset --key KEY` |

---

## Remaining Known Issues / Future Work

| Item | Notes |
|------|-------|
| `init` doesn't set cpu/memory — deploy prompts for them | Low priority: platform defaults kick in |
| `logs stream` requires `-d` when not using `--config` and there are multiple deployments in namespace | Expected behaviour |
| `--output json` not wired into every command (only list/get) | Medium: add to `service list`, `ingress list`, `audit list`, etc. |
| Auto-select amd64 machine (no `--machine`) on amd64 images | Only 1 online owner machine (arm64); monetized amd64 fallback untested |

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
