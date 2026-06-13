# Example Deployments

Two sample applications for demonstrating and testing `1ctl` against a SatuSky backend.

| App | Stack | Port | CPU | Memory |
|-----|-------|------|-----|--------|
| `backend` | Go HTTP API | 8080 | 0.5 | 256Mi |
| `frontend` | Nginx static site | 80 | 0.25 | 128Mi |

---

## How satusky.toml works

`satusky.toml` contains only static app config — no generated IDs, no org. The `name` field is the app identifier, resolved to a deployment at runtime (like `fly.toml`).

```toml
[app]
  name   = "backend-api"   # identifier — resolved via API at runtime
  port   = 8080            # required
  cpu    = "0.5"           # optional — platform default 0.5
  memory = "256Mi"         # optional — platform default 256Mi
```

**What's omitted intentionally:**
- `org` — taken from the active auth context
- `deployment_id` — resolved at runtime via `GET /deployments/namespace/:ns/app/:name`
- `replicas`, `domain`, `dockerfile` — platform defaults apply when not set

**Bare minimum that works:**
```toml
[app]
  port = 8080
```
Name = dirname, cpu/memory = platform defaults.

CLI flags always override toml values for a single invocation. The file is never modified by any command.

---

## Prerequisites

- A local API server reachable on `http://localhost:8080` (or whichever endpoint your `local` profile points at)
- Build from source (see below)

### Can I use the Homebrew `1ctl` instead of building from source?

**No.** The Homebrew release (v0.6.0) is missing commands added in this development branch:
`env unset`, `secret unset`, `--wait`, `--output json`, `logs stream --config`, `org switch <name>`, and all the bug fixes (env/secret first-time create, token lifecycle, delete handlers).

You must build from source:

```bash
cd /path/to/1ctl
go build -o bin/1ctl ./cmd/...
```

No `-ldflags` needed. The binary defaults to the prod API URL, but `SATUSKY_API_URL` always overrides it at runtime (see below).

Optional — install to PATH so you can call it without `./bin/`:
```bash
sudo cp bin/1ctl /usr/local/bin/1ctl
```

---

## 1. Authenticate

```bash
# Point at the local backend — required every session (or add to ~/.zshrc)
export SATUSKY_API_URL=http://localhost:8080/v1/cli

# Returning user — profile already exists, just activate and confirm
1ctl profile use local
1ctl auth status
```

```bash
# First time on this machine — create the local profile, then log in
1ctl profile create --url http://localhost:8080/v1/cli local
1ctl profile use local
1ctl auth login --token <your-api-token>
```

> **`SATUSKY_API_URL` is the master switch.** It overrides the active profile URL for every
> command. Set it in your shell and forget about it. The `--api-url` flag does the same
> thing per-command if you prefer not to export.

---

## 2. Initialize a project

```bash
cd my-project
1ctl init
```

Creates a minimal `satusky.toml` — only the fields you actually need:
```toml
[app]
  name = "my-project"
  port = 8080
```

No empty fields, no noise.

---

## 3. Deploy backend-api

```bash
cd examples/backend

# Uses satusky.toml for cpu/memory/port; --image and --machine are explicit
1ctl deploy --config satusky.toml --image nginx:alpine --machine compute-main-01

# Block until pods are Running
1ctl deploy --config satusky.toml --image nginx:alpine --machine compute-main-01 --wait
```

Expected (with `--wait`):
```
Step 2/5: Creating/updating deployment backend-api ✓
...
💡 Generated new domain: sleepytiger-z8w02g4.satusky.com
✅ 🚀 Deployment for backend-api is successful! Your app is live at: https://sleepytiger-z8w02g4.satusky.com
💡 Waiting for deployment to become healthy...
✅ Deployment is healthy — pods Running
```

> **Auto-assigned domain**: the backend generates a unique `adjective+animal-XXXXXXX.satusky.com` name (same format as web-dashboard deployments). The domain shown in the CLI output is what K8s ingress actually uses — there is no mismatch.

---

## 4. Deploy frontend (cloud build)

```bash
cd examples/frontend

1ctl deploy --config satusky.toml --machine compute-main-01 --wait
```

Expected:
```
💡 Build queued (ID: ...)
  [build] Docker build completed
💡 Image architecture: amd64
💡 Generated new domain: fastfalcon-bf2xkd9.satusky.com
✅ 🚀 Deployment for frontend is successful! Your app is live at: https://fastfalcon-bf2xkd9.satusky.com
✅ Deployment is healthy — pods Running
```

---

## 5. Check status

```bash
# All of these resolve by name — no ID needed
1ctl deploy status --config examples/backend/satusky.toml
1ctl deploy get    --config examples/backend/satusky.toml

# Machine-readable output for scripting
1ctl --output json deploy list | jq '.[].image'
1ctl -o json deploy get --config examples/backend/satusky.toml | jq '.image'

# Direct ID still works
1ctl deploy get --deployment-id <id>
```

---

## 6. Environment variables

```bash
cd examples/backend

# Create (first call creates, subsequent calls merge)
1ctl env create --config satusky.toml --env APP_ENV=production --env LOG_LEVEL=info
1ctl env create --config satusky.toml --env LOG_LEVEL=debug    # updates existing key

# Remove a specific key without touching others
1ctl env unset --config satusky.toml --key LOG_LEVEL

# List
1ctl env list

# Machine-readable
1ctl -o json env list | jq '.[0].key_values'
```

`env unset` removes the key from both the ConfigMap and the Deployment pod spec. Pods stay Running — no `CreateContainerConfigError` on next restart.

Verify in K8s:
```bash
kubectl -n org3-b322955e get configmap backend-api-environments -o jsonpath='{.data}'
# Deployment env is also clean:
kubectl -n org3-b322955e get deployment backend-api \
  -o jsonpath='{.spec.template.spec.containers[0].env}'
```

---

## 7. Secrets

```bash
cd examples/backend

1ctl secret create --config satusky.toml --kv DB_PASS=s3cret --kv API_KEY=key123
1ctl secret create --config satusky.toml --kv NEW_KEY=added   # merges

# Remove a specific key
1ctl secret unset --config satusky.toml --key DB_PASS

1ctl secret list
```

Use `--kv` for secrets (`--env` is a backward-compatible alias). Like `env unset`, `secret unset` also removes the `valueFrom.secretKeyRef` entry from the Deployment pod spec so pods don't crash on the next restart.

---

## 8. Logs

```bash
# Stored logs (Loki)
1ctl logs --config examples/backend/satusky.toml

# Live stream — also accepts --config now
1ctl logs stream --config examples/backend/satusky.toml
1ctl logs stream -d <deployment-id>    # direct ID also works
```

---

## 9. Operational commands

```bash
# Rolling restart
1ctl deploy restart --config examples/backend/satusky.toml

# Release history
1ctl deploy releases --config examples/backend/satusky.toml

# Roll back
1ctl deploy rollback --config examples/backend/satusky.toml --version 1 -y

# Service and ingress management
1ctl service list
1ctl service delete --service-id <id> -y

1ctl ingress list
1ctl ingress delete --ingress-id <id> -y

# Tear down
1ctl deploy destroy --config examples/backend/satusky.toml -y
```

---

## 10. Token management

```bash
1ctl token list
1ctl token create --name "ci-token"
1ctl token disable <token-id>
1ctl token enable  <token-id>
1ctl token delete  <token-id> -y

# Machine-readable
1ctl -o json token list | jq '.[] | {name, is_active}'
```

---

## 11. Organisation switching

```bash
# Positional arg — name or UUID both work
1ctl org switch org2
1ctl org switch b322955e-6a86-4157-8bff-1bea605ef8ac

# Flags still work
1ctl org switch --org-name org2
1ctl org switch --org-id   b322955e-6a86-4157-8bff-1bea605ef8ac
```

---

## 12. Full cluster state

```bash
1ctl deploy list
1ctl service list
1ctl ingress list
1ctl env list
1ctl secret list
1ctl notifications list
1ctl notifications read --all
1ctl credits balance
1ctl credits transactions
1ctl audit list
1ctl cluster list
1ctl cluster zones
1ctl machine list
1ctl machine get <machine>
1ctl machine update <machine>
1ctl machine visibility <machine> owner|organisation|public
1ctl machine labels list <machine>
1ctl machine labels set <machine> key=value ...
1ctl machine labels remove <machine> key ...
1ctl machine inspect <machine>
1ctl machine logs <machine>
1ctl machine events <machine>
1ctl machine available
1ctl machine usage list
1ctl user me
1ctl user permissions
1ctl token list
1ctl marketplace list
```

---

## Machine Visibility & Labels

Control who can see and deploy to your machines:

```bash
# Set visibility — who can see and use this machine
1ctl machine visibility <machine> owner         # only the owner
1ctl machine visibility <machine> organisation  # owner + org members
1ctl machine visibility <machine> public        # anyone (monetized machines auto-public)

# Manage labels — used by --machine-label and --machine-tag on deploy
1ctl machine labels list <machine>
1ctl machine labels set <machine> env=production gpu=true
1ctl machine labels remove <machine> gpu
```

Deploy with label targeting:

```bash
# Deploy to machines with env=production AND gpu labels
1ctl deploy --machine-label env=production --machine-label gpu ...

# Shorthand: --machine-tag production → satusky.com/production exists
1ctl deploy --machine-tag production ...

# OR semantics: deploy to machines matching ANY of these
1ctl deploy --machine-label-any zone=kul --machine-label-any zone=bki ...

# Combine AND + OR: env=prod AND (zone=kul OR zone=bki)
1ctl deploy --machine-label env=prod \
            --machine-label-any zone=kul --machine-label-any zone=bki ...
```

## --output json

`--output json` / `-o json` is wired into these commands:

| Command | `-o json` |
|---------|-----------|
| `deploy list/get/status` | ✅ |
| `env list` | ✅ |
| `secret list` | ✅ |
| `machine list` | ✅ |
| `token list` | ✅ |
| `service list`, `ingress list`, `audit list`, `notifications list` | ❌ (table only — future sprint) |

```bash
# Pipe into jq
1ctl --output json deploy list | jq '.[].image'
1ctl -o json env list | jq '.[0].key_values'
1ctl -o json machine list | jq '.[] | select(.status == "online") | .machine_name'
1ctl -o json token list | jq '.[] | {name, is_active}'
```

---

## Error cases

| Situation | Command | Error |
|---|---|---|
| No toml, no `--deployment-id` | `1ctl deploy status` (from repo root) | `no --deployment-id and no satusky.toml found` |
| App not deployed | `1ctl deploy status --config satusky.toml` (after destroy) | `app "backend-api" not found — run 1ctl deploy first` |
| Wrong directory | `1ctl deploy --image nginx:alpine` (from repo root) | `app name "1ctl" is not a valid K8s service name ... Auto-detected from git remote` |
| Invalid name | `1ctl deploy` (from `/tmp/1bad`) | `app name "1bad" is not a valid K8s service name (starts with digit — try --name app-1bad)` |
| `--deployment-id` beats `--config` | `1ctl deploy status --config frontend/satusky.toml --deployment-id <backend-id>` | Shows backend status (ID wins) |
| No machines match label selectors | `1ctl deploy --machine-label nonexistent=true ...` | `no visible online machines matched the machine label selectors` |
| Invalid visibility value | `1ctl machine visibility <m> invalid` | `machine_visibility must be one of: owner, organisation, public` |

---

## Architecture notes

Cloud builds run on the backend server (macOS arm64 + Podman). Builds produce `linux/amd64` images because Podman resolves to the amd64 base variant of multi-arch images like `nginx:alpine`. The CLI:

1. Gets `image_arch` from build status (`docker inspect`)
2. Filters owner machines to those matching the arch
3. Sets `nodeSelector: {"kubernetes.io/arch": <arch>}` on the pod spec

Multi-arch images used with `--image` (e.g. `nginx:alpine`) have no arch filter — the kubelet picks the right variant automatically.

### Machine label targeting

When `--machine-label`, `--machine-label-any`, or `--machine-tag` is used (or their
TOML equivalents), the CLI calls `POST /machines/label-query` on the backend. This
endpoint queries K8s node labels directly and cross-references with the user's
visible machines. Only online machines are returned. The selected machine IDs are
set as `requiredDuringSchedulingIgnoredDuringExecution` node affinity on the pod
spec via `machine.satusky.com/id`.

---

## satusky.toml reference

```toml
[app]
  name                = "backend-api"    # K8s app label — identifier for all commands
  port                = 8080             # container port (required)
  dockerfile          = "Dockerfile"     # path to Dockerfile (default: ./Dockerfile)
  cpu_request         = "250m"           # guaranteed CPU reservation per replica
  cpu_limit           = "1"              # maximum burst CPU per replica
  memory              = "256Mi"          # memory allocation (default 256Mi)
  replicas            = 1                # replica count — default 1 if omitted
  domain              = ""               # custom domain — empty = auto backend-assigned
  zone                = "my-kul-1b"      # target deployment zone
  organization        = "my-org"         # organization slug
  health_path         = "/healthz"       # HTTP health check path
  strategy            = "rolling"        # rollout strategy: rolling or recreate
  rolling_max_surge   = "25%"            # max surge during rolling update
  rolling_max_unavailable = "25%"        # max unavailable during rolling update
  machine_tag         = "production"     # deploy to machines with satusky.com/production label
  machine_labels      = ["env=prod", "gpu"]       # AND label selectors
  machine_label_any   = ["zone=kul", "zone=bki"]  # OR label selectors
  wait_for            = ["postgres:5432"]          # TCP dependencies

[volume]
  size                = "10Gi"
  mount               = "/data"

[hpa]
  enabled             = true
  min_replicas        = 1
  max_replicas        = 10
  cpu_target          = 80
  memory_target       = 0

[vpa]
  enabled             = false
  mode                = "Off"
  min_cpu             = "100m"
  max_cpu             = "4"
  min_memory          = "128Mi"
  max_memory          = "8Gi"

[pdb]
  enabled             = false
  type                = "auto"
  min_available       = 1
  percent             = 0

[multicluster]
  enabled             = false
  mode                = "active-passive"
  backup_enabled      = true
  backup_schedule     = "daily"
  backup_retention    = "168h"
  backup_priority_cluster = 1
```

**Not in the file:** `org` (auth context), `deployment_id` (runtime lookup). Safe to commit as-is.
