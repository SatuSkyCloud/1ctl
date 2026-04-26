# Example Deployments

Two sample applications for testing and demonstrating `1ctl` against a local or remote SatuSky backend.

| App | Stack | Port | CPU | Memory | Build |
|-----|-------|------|-----|--------|-------|
| `backend` | Go HTTP API | 8080 | 0.5 | 256Mi | Pre-built image |
| `frontend` | Nginx static site | 80 | 0.25 | 128Mi | Cloud build |

---

## How app identity works

`satusky.toml` stores only static config. The `name` field is the app identifier — like `fly.toml`. No generated IDs are stored in the file.

```toml
[app]
  name = "backend-api"   # ← this is the identifier
  org  = "org3"
  port = 8080
  cpu  = "0.5"
  memory = "256Mi"
```

When you run `1ctl deploy status --config satusky.toml`, the CLI reads `name = "backend-api"`, calls the backend to resolve it to a deployment ID, then proceeds. Nothing environment-specific ever touches the committed file.

`--deployment-id` always wins over `--config` if both are provided.

---

## Prerequisites

- Backend running: `sudo task dev.debug > logs.txt 2>&1` in `satusky-core_backend`
- Dev binary built: `go build -ldflags "-X 1ctl/internal/config.defaultAPIURL=http://localhost:8080/v1/cli" -o bin/1ctl-dev ./cmd/...`
- Authenticated (see below)

---

## 1. Authenticate

```bash
# Returning user — activate the existing profile then check status
export SATUSKY_API_URL=http://localhost:8080/v1/cli
1ctl-dev profile use local
1ctl-dev auth status
```

```bash
# First time on this machine — create profile, activate, log in
go build -ldflags "-X 1ctl/internal/config.defaultAPIURL=http://localhost:8080/v1/cli" \
  -o bin/1ctl-dev ./cmd/...
sudo cp bin/1ctl-dev /usr/local/bin/1ctl-dev

1ctl-dev profile create --url http://localhost:8080/v1/cli local
1ctl-dev profile use local
1ctl-dev auth login --token <your-api-token>
```

Expected:
```
✅ Authenticated with Satusky
User Email: mingerz.k@gmail.com
Organization: org3
```

---

## 2. Find a machine

```bash
1ctl-dev machine list        # your machines (owned)
1ctl-dev machine available   # monetized machines
1ctl-dev cluster zones       # available zones
```

`compute-main-01` is the amd64 production node used in these examples.

---

## 3. Deploy backend-api

```bash
cd examples/backend

# Deploy using toml for cpu/memory/port + explicit image and machine
1ctl-dev deploy --config satusky.toml --image nginx:alpine --machine compute-main-01
```

Expected:
```
💡 Using pre-built image: nginx:alpine
Step 2/5: Creating/updating deployment backend-api ✓
Step 3/5: Configuring services backend-api ✓
Step 4/5: Setting up environment and storage backend-api ✓
Step 5/5: Configuring ingress and dependencies backend-api ✓
✅ 🚀 Deployment for backend-api is successful! Your app is live at: https://backend-api.satusky.com
Deployment ID: <generated>
```

The Deployment ID is printed but **not written back to satusky.toml**.

---

## 4. Deploy frontend (cloud build)

```bash
cd examples/frontend

# Cloud build — backend builds the image and detects arch automatically
1ctl-dev deploy --config satusky.toml --machine compute-main-01
```

Expected key lines:
```
💡 Build queued (ID: <build-id>)
  [build] Docker build completed
  [build] Image pushed: registry.satusky.com/satusky-container-registry/frontend:<build-id>
✅ Cloud build complete: ...
💡 Image architecture: amd64
✅ 🚀 Deployment for frontend is successful!
```

Verify nodeSelector was set:
```bash
kubectl -n org3-b322955e get deployment frontend \
  -o jsonpath='{.spec.template.spec.nodeSelector}'
# {"kubernetes.io/arch":"amd64"}
```

---

## 5. Check status

```bash
# Via config (name-based resolution — no deployment_id needed)
1ctl-dev deploy status --config examples/backend/satusky.toml
1ctl-dev deploy status --config examples/frontend/satusky.toml

# Via direct ID (always works as override)
1ctl-dev deploy status --deployment-id <id>

# Full details
1ctl-dev deploy get --config examples/backend/satusky.toml
```

---

## 6. Environment variables

```bash
cd examples/backend

# First-time create (no prior ConfigMap)
1ctl-dev env create --config satusky.toml --env APP_ENV=production --env LOG_LEVEL=info

# Second call merges — keys are added or updated, existing keys preserved
1ctl-dev env create --config satusky.toml --env LOG_LEVEL=debug --env NEW_KEY=hello

# List
1ctl-dev env list
```

Verify merged result:
```bash
kubectl -n org3-b322955e get configmap backend-api-environments -o jsonpath='{.data}'
# {"app-env":"production","log-level":"debug","new-key":"hello",...}
```

---

## 7. Secrets

```bash
cd examples/backend

# First-time create (no prior K8s Secret)
1ctl-dev secret create --config satusky.toml --kv DB_PASS=supersecret --kv API_KEY=abc123

# Second call merges
1ctl-dev secret create --config satusky.toml --kv DB_PASS=newpassword

# List
1ctl-dev secret list
```

Use `--kv` for secrets (`--env` is a backward-compatible alias).

---

## 8. Operational commands

```bash
# Rolling restart
1ctl-dev deploy restart --config examples/backend/satusky.toml

# Release history
1ctl-dev deploy releases --config examples/backend/satusky.toml

# Roll back to a previous version
1ctl-dev deploy rollback --config examples/backend/satusky.toml --version 1 -y

# Logs
1ctl-dev logs --config examples/backend/satusky.toml

# Destroy
1ctl-dev deploy destroy --config examples/backend/satusky.toml -y
1ctl-dev deploy destroy --config examples/frontend/satusky.toml -y
```

---

## 9. Full cluster state

```bash
1ctl-dev deploy list
1ctl-dev service list
1ctl-dev ingress list
1ctl-dev env list
1ctl-dev secret list
1ctl-dev notifications list
1ctl-dev credits balance
1ctl-dev audit list
1ctl-dev cluster list
1ctl-dev machine list
1ctl-dev machine available
1ctl-dev user me
1ctl-dev token list
```

---

## Error cases

| Situation | Command | Error shown |
|---|---|---|
| Wrong directory, no toml | `1ctl-dev deploy status` | `no --deployment-id and no satusky.toml found` |
| Config file missing | `1ctl-dev deploy status --config missing.toml` | `no --deployment-id and no satusky.toml found` |
| App name not deployed | `1ctl-dev deploy status --config satusky.toml` (after destroy) | `app "backend-api" not found — run 1ctl deploy first` |
| App name invalid (starts with digit) | `1ctl-dev deploy --name 1myapp ...` | `app name "1myapp" is not a valid K8s service name (starts with a digit — try --name app-1myapp)` |
| Deploy from repo root | `1ctl-dev deploy ...` (no --name) | `app name "1ctl" is not a valid K8s service name ... Auto-detected from git remote` |

---

## Architecture notes

Cloud builds run on the backend server. On this host (macOS arm64 + Podman), builds pull the amd64 base variant of `nginx:alpine`, so images are `linux/amd64`. The CLI:

1. Gets `image_arch` from the build status (`docker inspect` on the backend)
2. Filters the auto-select machine pool to only machines matching that arch
3. Passes `target_arch` to the deploy request
4. Backend sets `nodeSelector: {"kubernetes.io/arch": <arch>}` on the pod spec

Multi-arch images (manifest lists like `nginx:alpine` used directly with `--image`) have no arch filter — any node works.

---

## satusky.toml reference

```toml
[app]
  name       = "backend-api"   # App label — used as K8s resource name and identifier
  org        = "org3"          # Organization name (used to derive namespace)
  port       = 8080            # Container port
  dockerfile = "Dockerfile"    # Relative path to Dockerfile (for cloud builds)
  cpu        = "0.5"           # CPU cores (request + limit)
  memory     = "256Mi"         # Memory (request + limit)
  replicas   = 1               # Replica count
  domain     = ""              # Custom domain (empty = auto *.satusky.com)
```

**No `deployment_id` field.** The CLI resolves the deployment at runtime using `name` + `org`. The file is safe to commit as-is.

CLI flags override toml values: `--cpu 0.25` overrides `cpu = "0.5"` for that one invocation, with no change to the file.
