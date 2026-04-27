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

- Backend running: `sudo task dev.debug > logs.txt 2>&1` in `satusky-core_backend`
- Build from source (see below)

### Can I use the Homebrew `1ctl` instead of building from source?

**No.** The Homebrew release (v0.6.0) is missing commands added in this development branch:
`env unset`, `secret unset`, `--wait`, `--output json`, `logs stream --config`, `org switch <name>`, and all the bug fixes (env/secret first-time create, token lifecycle, delete handlers).

You must build from source:

```bash
cd /path/to/1ctl
go build -o bin/1ctl-dev ./cmd/...
```

No `-ldflags` needed. The binary defaults to the prod API URL, but `SATUSKY_API_URL` always overrides it at runtime (see below).

Optional — install to PATH so you can call it without `./bin/`:
```bash
sudo cp bin/1ctl-dev /usr/local/bin/1ctl-dev
```

---

## 1. Authenticate

```bash
# Point at the local backend — required every session (or add to ~/.zshrc)
export SATUSKY_API_URL=http://localhost:8080/v1/cli

# Returning user — profile already exists, just activate and confirm
1ctl-dev profile use local
1ctl-dev auth status
```

```bash
# First time on this machine — create the local profile, then log in
1ctl-dev profile create --url http://localhost:8080/v1/cli local
1ctl-dev profile use local
1ctl-dev auth login --token <your-api-token>
```

> **`SATUSKY_API_URL` is the master switch.** It overrides the active profile URL for every
> command. Set it in your shell and forget about it. The `--api-url` flag does the same
> thing per-command if you prefer not to export.

---

## 2. Initialize a project

```bash
cd my-project
1ctl-dev init
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
1ctl-dev deploy --config satusky.toml --image nginx:alpine --machine compute-main-01

# Block until pods are Running
1ctl-dev deploy --config satusky.toml --image nginx:alpine --machine compute-main-01 --wait
```

Expected (with `--wait`):
```
Step 2/5: Creating/updating deployment backend-api ✓
...
✅ 🚀 Deployment for backend-api is successful!
💡 Waiting for deployment to become healthy...
✅ Deployment is healthy — pods Running
```

---

## 4. Deploy frontend (cloud build)

```bash
cd examples/frontend

1ctl-dev deploy --config satusky.toml --machine compute-main-01 --wait
```

Expected:
```
💡 Build queued (ID: ...)
  [build] Docker build completed
💡 Image architecture: amd64
✅ 🚀 Deployment for frontend is successful!
✅ Deployment is healthy — pods Running
```

---

## 5. Check status

```bash
# All of these resolve by name — no ID needed
1ctl-dev deploy status --config examples/backend/satusky.toml
1ctl-dev deploy get    --config examples/backend/satusky.toml

# Machine-readable output for scripting
1ctl-dev --output json deploy list | jq '.[].image'
1ctl-dev -o json deploy get --config examples/backend/satusky.toml | jq '.image'

# Direct ID still works
1ctl-dev deploy get --deployment-id <id>
```

---

## 6. Environment variables

```bash
cd examples/backend

# Create (first call creates, subsequent calls merge)
1ctl-dev env create --config satusky.toml --env APP_ENV=production --env LOG_LEVEL=info
1ctl-dev env create --config satusky.toml --env LOG_LEVEL=debug    # updates existing key

# Remove a specific key without touching others
1ctl-dev env unset --config satusky.toml --key LOG_LEVEL

# List
1ctl-dev env list

# Machine-readable
1ctl-dev -o json env list | jq '.[0].key_values'
```

Verify in K8s:
```bash
kubectl -n org3-b322955e get configmap backend-api-environments -o jsonpath='{.data}'
```

---

## 7. Secrets

```bash
cd examples/backend

1ctl-dev secret create --config satusky.toml --kv DB_PASS=s3cret --kv API_KEY=key123
1ctl-dev secret create --config satusky.toml --kv NEW_KEY=added   # merges

# Remove a specific key
1ctl-dev secret unset --config satusky.toml --key DB_PASS

1ctl-dev secret list
```

Use `--kv` for secrets (`--env` is a backward-compatible alias).

---

## 8. Logs

```bash
# Stored logs (Loki)
1ctl-dev logs --config examples/backend/satusky.toml

# Live stream — also accepts --config now
1ctl-dev logs stream --config examples/backend/satusky.toml
1ctl-dev logs stream -d <deployment-id>    # direct ID also works
```

---

## 9. Operational commands

```bash
# Rolling restart
1ctl-dev deploy restart --config examples/backend/satusky.toml

# Release history
1ctl-dev deploy releases --config examples/backend/satusky.toml

# Roll back
1ctl-dev deploy rollback --config examples/backend/satusky.toml --version 1 -y

# Service and ingress management
1ctl-dev service list
1ctl-dev service delete --service-id <id> -y

1ctl-dev ingress list
1ctl-dev ingress delete --ingress-id <id> -y

# Tear down
1ctl-dev deploy destroy --config examples/backend/satusky.toml -y
```

---

## 10. Token management

```bash
1ctl-dev token list
1ctl-dev token create --name "ci-token"
1ctl-dev token disable <token-id>
1ctl-dev token enable  <token-id>
1ctl-dev token delete  <token-id> -y

# Machine-readable
1ctl-dev -o json token list | jq '.[] | {name, is_active}'
```

---

## 11. Organisation switching

```bash
# Positional arg — name or UUID both work
1ctl-dev org switch org2
1ctl-dev org switch b322955e-6a86-4157-8bff-1bea605ef8ac

# Flags still work
1ctl-dev org switch --org-name org2
1ctl-dev org switch --org-id   b322955e-6a86-4157-8bff-1bea605ef8ac
```

---

## 12. Full cluster state

```bash
1ctl-dev deploy list
1ctl-dev service list
1ctl-dev ingress list
1ctl-dev env list
1ctl-dev secret list
1ctl-dev notifications list
1ctl-dev notifications read --all
1ctl-dev credits balance
1ctl-dev credits transactions
1ctl-dev audit list
1ctl-dev cluster list
1ctl-dev cluster zones
1ctl-dev machine list
1ctl-dev machine available
1ctl-dev user me
1ctl-dev user permissions
1ctl-dev token list
1ctl-dev marketplace list
```

---

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
1ctl-dev --output json deploy list | jq '.[].image'
1ctl-dev -o json env list | jq '.[0].key_values'
1ctl-dev -o json machine list | jq '.[] | select(.status == "online") | .machine_name'
1ctl-dev -o json token list | jq '.[] | {name, is_active}'
```

---

## Error cases

| Situation | Command | Error |
|---|---|---|
| No toml, no `--deployment-id` | `1ctl-dev deploy status` (from repo root) | `no --deployment-id and no satusky.toml found` |
| App not deployed | `1ctl-dev deploy status --config satusky.toml` (after destroy) | `app "backend-api" not found — run 1ctl deploy first` |
| Wrong directory | `1ctl-dev deploy --image nginx:alpine` (from repo root) | `app name "1ctl" is not a valid K8s service name ... Auto-detected from git remote` |
| Invalid name | `1ctl-dev deploy` (from `/tmp/1bad`) | `app name "1bad" is not a valid K8s service name (starts with digit — try --name app-1bad)` |
| `--deployment-id` beats `--config` | `1ctl-dev deploy status --config frontend/satusky.toml --deployment-id <backend-id>` | Shows backend status (ID wins) |

---

## Architecture notes

Cloud builds run on the backend server (macOS arm64 + Podman). Builds produce `linux/amd64` images because Podman resolves to the amd64 base variant of multi-arch images like `nginx:alpine`. The CLI:

1. Gets `image_arch` from build status (`docker inspect`)
2. Filters owner machines to those matching the arch
3. Sets `nodeSelector: {"kubernetes.io/arch": <arch>}` on the pod spec

Multi-arch images used with `--image` (e.g. `nginx:alpine`) have no arch filter — the kubelet picks the right variant automatically.

---

## satusky.toml reference

```toml
[app]
  name       = "backend-api"   # K8s app label — identifier for all commands
  port       = 8080            # container port (required)
  dockerfile = "Dockerfile"    # path to Dockerfile (default: ./Dockerfile)
  cpu        = "0.5"           # CPU cores — platform default 0.5 if omitted
  memory     = "256Mi"         # memory — platform default 256Mi if omitted
  replicas   = 1               # replica count — default 1 if omitted
  domain     = ""              # custom domain — empty = auto *.satusky.com
```

**Not in the file:** `org` (auth context), `deployment_id` (runtime lookup). Safe to commit as-is.
