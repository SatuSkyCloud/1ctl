# Example Deployments

Two sample applications for demonstrating and testing `1ctl` against a SatuSky backend.

| App | Stack | Port | CPU | Memory |
|-----|-------|------|-----|--------|
| `backend` | Go HTTP API | 8080 | 0.5 | 256Mi |
| `frontend` | Nginx static site | 80 | 0.25 | 128Mi |

---

## How satusky.toml works

`satusky.toml` contains only static app config — no generated IDs, no org. It works like `fly.toml`.

```toml
[app]
  name   = "backend-api"   # app identifier (resolved to deployment at runtime)
  port   = 8080            # required
  cpu    = "0.5"           # optional — defaults to 0.5
  memory = "256Mi"         # optional — defaults to 256Mi
```

**What's omitted intentionally:**
- `org` — taken from the active auth context
- `deployment_id` — resolved at runtime via `GET /deployments/namespace/:ns/app/:name`
- `replicas`, `domain`, `dockerfile` — platform defaults apply

**Bare minimum** (everything else defaults):
```toml
[app]
  port = 8080
```

CLI flags always override toml values for a single invocation; the file is never modified.

---

## Prerequisites

- Backend running: `sudo task dev.debug > logs.txt 2>&1` in `satusky-core_backend`
- Dev binary: `go build -ldflags "-X 1ctl/internal/config.defaultAPIURL=http://localhost:8080/v1/cli" -o bin/1ctl-dev ./cmd/...`

---

## 1. Authenticate

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli

# Returning user — activate the existing profile
1ctl-dev profile use local
1ctl-dev auth status    # confirms you're already logged in
```

```bash
# First time on this machine
1ctl-dev profile create --url http://localhost:8080/v1/cli local
1ctl-dev profile use local
1ctl-dev auth login --token eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3ODM0ODM1NzUsImlhdCI6MTc3NTcwNzU3NSwianRpIjoiNjgyZTg4YWItY2VhZi00NjkwLWE0MjgtNWRlODQ0NTEwMzU1Iiwic3ViIjoiN2FlYjFjMjQtYjdmZC00NmQ0LWJlN2EtYTE4YjQzY2RkNWQyIiwidHlwZSI6ImFwaV9rZXkifQ.NxrE1ugYINXqhj-5rJgok79fUhX3T677iS2FBAjw-gc
```

> `profile` subcommands require the dev binary. The Homebrew release (v0.6.0) only supports `SATUSKY_API_URL`.

---

## 2. Deploy backend-api

```bash
cd examples/backend

# Uses satusky.toml for cpu/memory/port; --image and --machine are explicit
1ctl-dev deploy --config satusky.toml --image nginx:alpine --machine compute-main-01
```

Expected:
```
💡 Using pre-built image: nginx:alpine
Step 2/5: Creating/updating deployment backend-api ✓
Step 3/5: Configuring services backend-api ✓
Step 4/5: Setting up environment and storage backend-api ✓
Step 5/5: Configuring ingress and dependencies backend-api ✓
✅ 🚀 Deployment for backend-api is successful!
Deployment ID: <generated — not written to satusky.toml>
```

---

## 3. Deploy frontend (cloud build)

```bash
cd examples/frontend

# No --image: backend builds the image, detects arch, sets nodeSelector
1ctl-dev deploy --config satusky.toml --machine compute-main-01
```

Expected:
```
💡 Build queued (ID: <build-id>)
  [build] Docker build completed
  [build] Image pushed: registry.satusky.com/...
✅ Cloud build complete: ...
💡 Image architecture: amd64
✅ 🚀 Deployment for frontend is successful!
```

Verify nodeSelector was set:
```bash
kubectl -n org3-b322955e get deploy frontend \
  -o jsonpath='{.spec.template.spec.nodeSelector}'
# {"kubernetes.io/arch":"amd64"}
```

---

## 4. Status checks

```bash
# All of these resolve the deployment via name — no ID needed
1ctl-dev deploy status --config examples/backend/satusky.toml
1ctl-dev deploy get    --config examples/backend/satusky.toml
1ctl-dev deploy status --config examples/frontend/satusky.toml

# Direct ID also always works
1ctl-dev deploy get --deployment-id <id>
```

---

## 5. Environment variables

```bash
cd examples/backend

# First-time create (no prior ConfigMap — works even on fresh deployments)
1ctl-dev env create --config satusky.toml --env APP_ENV=production --env LOG_LEVEL=info

# Second call merges: adds new keys, updates existing, preserves others
1ctl-dev env create --config satusky.toml --env LOG_LEVEL=debug --env VERSION=1.0

1ctl-dev env list
```

Verify:
```bash
kubectl -n org3-b322955e get configmap backend-api-environments -o jsonpath='{.data}'
# {"app-env":"production","log-level":"debug","version":"1.0"}
```

> No `env delete` — it would wipe all keys at once. Per-key removal (`env unset KEY`) is coming in a future release.

---

## 6. Secrets

```bash
cd examples/backend

1ctl-dev secret create --config satusky.toml --kv DB_PASS=s3cret --kv API_KEY=key123
1ctl-dev secret create --config satusky.toml --kv NEW_KEY=added   # merges
1ctl-dev secret list
```

Use `--kv` for secrets (`--env` is a backward-compatible alias).

---

## 7. Logs

```bash
# Stored logs (Loki)
1ctl-dev logs --config examples/backend/satusky.toml

# Live stream (like kubectl logs -f) — needs deployment-id
1ctl-dev logs stream -d <deployment-id>
```

---

## 8. Operational commands

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

# Tear down everything
1ctl-dev deploy destroy --config examples/backend/satusky.toml -y
1ctl-dev deploy destroy --config examples/frontend/satusky.toml -y
```

---

## 9. Token management

```bash
1ctl-dev token list
1ctl-dev token create --name "ci-token"
1ctl-dev token disable <token-id>
1ctl-dev token enable  <token-id>
1ctl-dev token get     <token-id>
1ctl-dev token delete  <token-id> -y
```

---

## 10. Full cluster state

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

## Error cases

| Situation | Command | Error |
|---|---|---|
| No toml, no `--deployment-id` | `1ctl-dev deploy status` (from repo root) | `no --deployment-id and no satusky.toml found` |
| App not deployed | `1ctl-dev deploy status --config satusky.toml` (after destroy) | `app "backend-api" not found — run 1ctl deploy first` |
| Wrong directory | `1ctl-dev deploy --image nginx:alpine` (from repo root) | `app name "1ctl" is not a valid K8s service name ... Auto-detected from git remote` |
| Invalid name (starts with digit) | `1ctl-dev deploy ... --image x` (from `/tmp/1bad`) | `app name "1bad" is not a valid K8s service name (starts with a digit — try --name app-1bad)` |
| `--deployment-id` beats `--config` | `1ctl-dev deploy status --config frontend/satusky.toml --deployment-id <backend-id>` | Shows backend status (ID wins) |

---

## Architecture notes

Cloud builds run on the backend server. The host is macOS arm64 (Podman). Images are built for `linux/amd64` because Podman resolves to the amd64 base variant of multi-arch images like `nginx:alpine`.

The CLI:
1. Gets `image_arch` from build status (`docker inspect` on the backend)
2. Filters owner machines to those matching the arch
3. Sets `nodeSelector: {"kubernetes.io/arch": <arch>}` on the pod spec

For multi-arch base images used with `--image` directly (e.g. `nginx:alpine`), no arch filter is applied — the kubelet selects the right variant automatically.

---

## Machine arch in DB

The machine `cpu_arch` metadata must be set for the arch filter to work:

```sql
-- Run once per cluster onboarding for arm64 workers
UPDATE machines
SET metadata = metadata || '{"cpu_arch": "arm64"}'::jsonb
WHERE machine_id IN ('<id1>', '<id2>', ...);
```

`compute-main-01` is already set to `amd64`. arm64 workers (`worker-efc2fd3e...`, etc.) have been patched.
