# Example Deployments

Two sample applications for testing and demonstrating `1ctl` against a local or remote SatuSky backend.

| App | Stack | Port | CPU | Memory | Build |
|-----|-------|------|-----|--------|-------|
| `backend` | Go HTTP API | 8080 | 0.5 | 256Mi | Cloud build |
| `frontend` | Nginx static site | 80 | 0.25 | 128Mi | Cloud build |

---

## Prerequisites

- `1ctl` installed (`go build -o bin/1ctl-dev ./cmd/...` from repo root)
- Backend running at `http://localhost:8080` (`task dev.debug` in `satusky-core_backend`)
- Valid API token
- `kubectl` access to the cluster (for verification)

---

## 1. Authenticate

> The installed `1ctl` (Homebrew v0.6.0) does **not** have `profile create`.
> Use `SATUSKY_API_URL` to point at local — it works with any binary version.

```bash
# Simplest — works with the Homebrew release binary
export SATUSKY_API_URL=http://localhost:8080/v1/cli
1ctl auth login --token eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3ODM0ODM1NzUsImlhdCI6MTc3NTcwNzU3NSwianRpIjoiNjgyZTg4YWItY2VhZi00NjkwLWE0MjgtNWRlODQ0NTEwMzU1Iiwic3ViIjoiN2FlYjFjMjQtYjdmZC00NmQ0LWJlN2EtYTE4YjQzY2RkNWQyIiwidHlwZSI6ImFwaV9rZXkifQ.NxrE1ugYINXqhj-5rJgok79fUhX3T677iS2FBAjw-gc
1ctl auth status
```

```bash
# Alternative — named profiles (requires dev binary: 1ctl-dev)
# Build and install once:
#   go build -ldflags "-X 1ctl/internal/config.defaultAPIURL=http://localhost:8080/v1/cli" \
#     -o bin/1ctl-dev ./cmd/... && sudo cp bin/1ctl-dev /usr/local/bin/1ctl-dev
1ctl-dev profile create --url http://localhost:8080/v1/cli local
1ctl-dev profile use local
1ctl-dev auth login --token <token>
```

Expected:
```
✅ Authenticated with Satusky
User Email: mingerz.k@gmail.com
Organization: org3
```

---

## 2. Find a Machine

```bash
# All machines (owned)
1ctl machine list

# Monetized machines available for rent
1ctl machine available

# Cluster zones
1ctl cluster zones
```

`compute-main-01` is the amd64 production node. Use it explicitly with `--machine compute-main-01` when deploying images built by cloud build (which produces amd64 images on this host).

---

## 3. Deploy backend-api (pre-built image)

```bash
cd examples/backend

# Deploy using a pre-built multi-arch image — no cloud build needed
1ctl deploy --cpu 0.5 --memory 256Mi --port 8080 --image nginx:alpine --machine compute-main-01
```

Expected output:
```
💡 Using pre-built image: nginx:alpine
Step 2/5: Creating/updating deployment backend-api ✓
Step 3/5: Configuring services backend-api ✓
Step 4/5: Setting up environment and storage backend-api ✓
Step 5/5: Configuring ingress and dependencies backend-api ✓
✅ 🚀 Deployment for backend-api is successful! Your app is live at: https://backend-api.satusky.com
Deployment ID: <id>
```

The `satusky.toml` is updated automatically with the new `deployment_id`.

---

## 4. Deploy frontend (cloud build with arch detection)

```bash
cd examples/frontend

# Cloud build — backend builds the image, detects arch, sets nodeSelector
1ctl deploy --cpu 0.25 --memory 128Mi --port 80 --machine compute-main-01
```

Expected output (key lines):
```
💡 Build queued (ID: <build-id>)
  [build] Docker build completed
  [build] Image pushed: registry.satusky.com/satusky-container-registry/frontend:<build-id>
✅ Cloud build complete: registry.satusky.com/satusky-container-registry/frontend:<build-id>
💡 Image architecture: amd64
✅ 🚀 Deployment for frontend is successful! Your app is live at: https://frontend.satusky.com
```

`Image architecture: amd64` confirms the arch was detected. The backend sets `nodeSelector: {"kubernetes.io/arch":"amd64"}` so K8s only schedules the pod on amd64 nodes.

**Verify** (optional):
```bash
kubectl -n org3-b322955e get deployment frontend -o jsonpath='{.spec.template.spec.nodeSelector}'
# {"kubernetes.io/arch":"amd64"}

kubectl -n org3-b322955e get pods -l app=frontend -o wide
# NAME                       READY   STATUS    NODE
# frontend-xxx-yyy           1/1     Running   compute-main-01
```

---

## 5. Environment variables

```bash
cd examples/backend

# First-time create (no prior ConfigMap — works now, was BUG-2 previously)
1ctl env create --config satusky.toml --env APP_NAME=backend-api --env LOG_LEVEL=info

# Update / merge more keys
1ctl env create --config satusky.toml --env LOG_LEVEL=debug --env VERSION=2.0

# List
1ctl env list
```

**Verify:**
```bash
kubectl -n org3-b322955e get configmap backend-api-environments -o jsonpath='{.data}'
# {"app-name":"backend-api","log-level":"debug","version":"2.0"}
```

---

## 6. Secrets

```bash
cd examples/backend

# First-time create (no prior K8s Secret — works now, was BUG-2 previously)
1ctl secret create --config satusky.toml --kv DB_PASS=supersecret --kv API_KEY=abc123

# Update / merge new key
1ctl secret create --config satusky.toml --kv DB_PASS=newpassword

# List
1ctl secret list
```

Use `--kv` for secrets. `--env` is a backward-compatible alias.

---

## 7. Operational commands

```bash
# Status (Running / Progress %)
1ctl deploy status --deployment-id fe7b53a5-80d8-4ddd-81b0-4a530767c723

# Rolling restart
1ctl deploy restart --deployment-id fe7b53a5-80d8-4ddd-81b0-4a530767c723

# Release history
1ctl deploy releases --deployment-id fe7b53a5-80d8-4ddd-81b0-4a530767c723

# Rollback to version 1
1ctl deploy rollback --deployment-id fe7b53a5-80d8-4ddd-81b0-4a530767c723 --version 1 -y

# Logs
1ctl logs --deployment-id fe7b53a5-80d8-4ddd-81b0-4a530767c723

# Tear down
1ctl deploy destroy --deployment-id fe7b53a5-80d8-4ddd-81b0-4a530767c723 -y
1ctl deploy destroy --deployment-id 38ab5d6b-c3cc-45c4-b0ef-ceab41cc9207 -y
```

---

## 8. Full cluster state check

```bash
1ctl deploy list
1ctl service list
1ctl ingress list
1ctl env list
1ctl secret list
1ctl notifications list
1ctl credits balance
1ctl audit list
1ctl cluster list
1ctl machine list
1ctl machine available
1ctl user me
1ctl token list
```

---

## Architecture Routing Notes

Cloud builds run on the backend server. On this host (macOS arm64 + Podman), builds produce `linux/amd64` images because Podman resolves to the amd64 base variant of multi-arch images like `nginx:alpine`.

The CLI now:
1. Gets `image_arch` from the build status response (`docker inspect` on the backend)
2. Filters the auto-select machine pool to only machines matching that arch
3. Passes `target_arch` to the deploy request
4. The backend sets `nodeSelector: {"kubernetes.io/arch": <arch>}` on the pod spec

This means:
- **amd64 images** → route to `compute-main-01` (amd64) or other amd64 nodes
- **arm64 images** → route to arm64 workers (if any are online and owned)
- **Multi-arch images** (manifest lists like `nginx:alpine`) → no arch filter, any node works

If a wrong-arch machine is selected, the pod stays **Pending** with a clear K8s scheduling error instead of starting and crashing with `exec format error`.

**DB requirement:** Each machine's `metadata` JSONB column must have `cpu_arch` set:
```sql
-- For arm64 workers (run once per cluster onboarding)
UPDATE machines SET metadata = metadata || '{"cpu_arch": "arm64"}'::jsonb
WHERE machine_id IN ('<id1>', '<id2>', ...);
```

---

## satusky.toml Reference

```toml
[app]
name = "backend-api"       # K8s app label (used for service/ingress/env/secret naming)
org = "org3"               # Organization name
port = 8080                # Container port
dockerfile = "Dockerfile"
cpu = "0.5"                # CPU cores
memory = "256Mi"           # Memory request and limit
replicas = 1
domain = ""                # Custom domain (empty = auto *.satusky.com)
deployment_id = "fe7b53a5-80d8-4ddd-81b0-4a530767c723"
```

Current deployment IDs:
| App | Deployment ID |
|-----|--------------|
| backend-api | `fe7b53a5-80d8-4ddd-81b0-4a530767c723` |
| frontend | `38ab5d6b-c3cc-45c4-b0ef-ceab41cc9207` |
