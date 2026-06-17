# Fullstack API — Complete SatuSky Deployment Example

A production-grade Go REST API that exercises every feature of the SatuSky
platform: persistent volumes, environment variables, secrets, horizontal
autoscaling, pod disruption budgets, rolling update strategy, TCP wait-for
dependencies, health checks, and multi-environment configs.

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                    fullstack-api (Go 1.24)                    │
│                                                              │
│  /health          → Platform smoke check (always available)  │
│  /                → API index + docs                         │
│  /api/tasks       → CRUD (PostgreSQL-backed)                 │
│  /api/upload      → File upload (persistent volume)          │
│  /api/files/*     → Serve uploaded files                     │
│  /api/config      → Show active environment variables        │
│  /api/secrets-info → Confirm secrets are injected (no values)│
│  /api/stats       → Task count + uptime                      │
│                                                              │
│  Volume:     /data/uploads  (1 Gi persistent block storage)  │
│  Database:   PostgreSQL (via env vars, graceful degradation) │
│  Secrets:    DB_PASSWORD, API_KEY, SMTP_PASSWORD, JWT_SECRET │
└──────────────────────────────────────────────────────────────┘
```

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/health` | No | Platform smoke check — always returns 200 |
| `GET` | `/` | No | API index |
| `GET` | `/api/config` | No | Active env vars (safe — no secrets) |
| `GET` | `/api/tasks` | No | List tasks |
| `POST` | `/api/tasks` | No | Create task `{"title":"..."}` |
| `GET` | `/api/tasks/:id` | No | Get task by ID |
| `DELETE` | `/api/tasks/:id` | No | Delete task |
| `POST` | `/api/upload` | No | Upload file (multipart, field: `file`, max 10 MB) |
| `GET` | `/api/files/:name` | No | Serve uploaded file |
| `GET` | `/api/stats` | No | Task count + uptime |
| `GET` | `/api/secrets-info` | No | Which secrets are set (never reveals values) |

The app is **resilient to database absence** — if PostgreSQL is unreachable, it
still serves `/health` (critical for platform smoke checks) and provides a
degraded in-memory task list.

## satusky.toml Features Exercised

| Section | Fields | Notes |
|---------|--------|-------|
| `[app]` | `name`, `port`, `cpu_request`, `cpu_limit`, `memory`, `replicas`, `domain`, `zone`, `organization` | App identity + compute resources |
| `[build]` | `dockerfile`, `fast_build` | How the container image is built |
| `[checks]` | `health_path` | Post-deploy smoke check path |
| `[deploy]` | `strategy`, `rolling_max_surge`, `rolling_max_unavailable`, `machine_tag`, `wait_for` | Deployment strategy + scheduling |
| `[volume]` | `size`, `mount` | Persistent 1 Gi block storage at `/data/uploads` |
| `[hpa]` | `enabled`, `min_replicas`, `max_replicas`, `cpu_target`, `memory_target` | Auto-scale 2–10 pods |
| `[vpa]` | `enabled`, `mode`, `min_cpu`, `max_cpu`, `min_memory`, `max_memory` | Vertical autoscaling (disabled by default) |
| `[pdb]` | `enabled`, `type` | Pod disruption budget (auto mode = 50% min available) |
| `[multicluster]` | `enabled`, `mode`, `backup_enabled`, `backup_schedule`, `backup_retention`, `backup_priority_cluster` | Multi-cluster HA (disabled by default) |

## Quick Start

```bash
# 1. Authenticate
export SATUSKY_API_URL=http://localhost:8080/v1/cli
1ctl auth login --token <your-token>

# 2. Deploy (cloud build + volume + env vars + wait)
cd examples/fullstack-api
1ctl deploy --config satusky.toml --wait \
  --env APP_ENV=production \
  --env LOG_LEVEL=debug \
  --env DB_HOST=postgres.example.com \
  --env DB_PORT=5432 \
  --env DB_USER=app \
  --env DB_NAME=fullstack

# 3. Inject secrets
1ctl secret create --config satusky.toml --wait \
  --kv DB_PASSWORD=your-db-password \
  --kv API_KEY=sk-live-xxxx

# 4. Upload a file (exercises the volume mount)
curl -F "file=@README.md" https://<your-domain>/api/upload

# 5. Create a task
curl -X POST https://<your-domain>/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"Deploy to production"}'

# 6. Check health + stats
curl https://<your-domain>/health
curl https://<your-domain>/api/stats

# 7. Run diagnostics
1ctl doctor --config satusky.toml
1ctl doctor --smoke                          # namespace-wide with smoke
1ctl logs --config satusky.toml --tail 20
```

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `PORT` | No | `8080` | HTTP listen port |
| `APP_ENV` | No | `development` | Environment name |
| `LOG_LEVEL` | No | `info` | Log verbosity |
| `DB_HOST` | No | `""` | PostgreSQL host (empty = in-memory mode) |
| `DB_PORT` | No | `5432` | PostgreSQL port |
| `DB_USER` | No | `postgres` | Database user |
| `DB_PASSWORD` | **Yes** (for DB) | `""` | Database password — **use secrets, not env vars** |
| `DB_NAME` | No | `app` | Database name |
| `UPLOAD_DIR` | No | `/data/uploads` | Where uploaded files are stored (matched to volume mount) |
| `FEATURE_X_ENABLED` | No | `false` | Feature flag example |

## Secrets

| Secret | Purpose | Best Practice |
|--------|---------|---------------|
| `DB_PASSWORD` | PostgreSQL authentication | Use `1ctl secret create --kv` |
| `API_KEY` | External API authentication | Rotate regularly |
| `SMTP_PASSWORD` | Email delivery | Use app-specific passwords |
| `JWT_SECRET` | Token signing | Minimum 256-bit random value |

Secrets are injected as K8s Secrets → mounted as environment variables.
Never put secrets in `satusky.toml` or CLI `--env` flags.

## Multi-Environment Deploy

```bash
# Production (uses satusky.toml — full resources)
1ctl deploy --config satusky.toml --wait

# Staging (uses satusky.staging.toml — reduced resources, no HPA)
1ctl deploy --config staging --wait
```

Staged configs inherit from the base `satusky.toml`. Fields set in
`satusky.staging.toml` override the base. See the file for details.

## Volume Workflow

```bash
# 1. Create a deployment with a volume (already in satusky.toml)
1ctl deploy --config satusky.toml --wait

# 2. Upload a file — stored on the persistent volume
curl -F "file=@main.go" https://<domain>/api/upload
# {"filename":"main.go","size_bytes":12300,"path":"/data/uploads/main.go"}

# 3. Serve it back
curl https://<domain>/api/files/main.go

# 4. Detach volume (data preserved, mount removed)
1ctl volumes detach <volume-id> -y

# 5. Destroy volume permanently
1ctl volumes delete <volume-id> -y
```

## Local Testing (without SatuSky)

```bash
# Run without a database — uses in-memory fake tasks
go run .
# → fullstack-api starting on :8080
# → config: env=development db= log=info uploads=/data/uploads
# → WARNING: database unavailable (will serve health endpoints anyway)

# Test endpoints
curl http://localhost:8080/health
curl http://localhost:8080/api/tasks
curl -X POST http://localhost:8080/api/tasks -d '{"title":"test"}'
```

## satusky.toml Field Map

| TOML Field | CLI Flag | API Field | Default |
|-----------|----------|-----------|---------|
| `app.name` | `--name` | `name` | dirname |
| `app.port` | `--port` | `port` | (required) |
| `app.cpu_request` | `--cpu-request` | `cpu_request` | `"0.5"` |
| `app.cpu_limit` | `--cpu-limit` | `cpu_limit` | equals cpu_request |
| `app.memory` | `--memory` | `memory` | `"256Mi"` |
| `app.replicas` | `--replicas` | `replicas` | `1` |
| `app.domain` | `--domain` | `domain` | auto-generated |
| `app.zone` | `--zone` | `zone` | auto |
| `build.dockerfile` | `--dockerfile` | `dockerfile_path` | `Dockerfile` |
| `build.fast_build` | `--fast-build` | `fast_build` | `false` |
| `checks.health_path` | `--health-path` | `smoke_path` | tries `/health` then `/` |
| `deploy.strategy` | `--strategy` | `strategy` | `"rolling"` |
| `deploy.rolling_max_surge` | `--rolling-max-surge` | `rolling_max_surge` | `"25%"` |
| `deploy.rolling_max_unavailable` | `--rolling-max-unavailable` | `rolling_max_unavailable` | `"25%"` |
| `deploy.machine_tag` | — | `machine_tag` | — |
| `deploy.wait_for` | — | `wait_for` | — |
| `volume.size` | `--volume-size` | `volume.size` | — |
| `volume.mount` | `--volume-mount` | `volume.mount_path` | — |
| `hpa.*` | `--hpa`, `--hpa-min-replicas`, etc. | `hpa_config.*` | disabled |
| `vpa.*` | `--vpa`, `--vpa-mode`, etc. | `vpa_config.*` | disabled |
| `pdb.*` | `--pdb`, `--pdb-type`, etc. | `pdb_config.*` | disabled |
| `multicluster.*` | `--multicluster` etc. | `multicluster_*` | disabled |

> **v3 migration note**: `dockerfile`, `health_path`, `strategy`, `rolling_max_surge`,
> `rolling_max_unavailable`, `machine_tag`, and `wait_for` were moved from `[app]`
> to `[build]`, `[checks]`, and `[deploy]` sections. Old `[app]` placement still
> works for backward compatibility — values are auto-migrated to the new sections.

---

## Ingress Architecture (Gateway API Migration)

> **Hybrid routing model** — the platform is migrating from traditional Ingress
> to Kubernetes Gateway API. As of this writing, both coexist:

| Domain Type | Resource | Controller | TLS |
|---|---|---|---|
| Primary (`.satusky.com`) | **HTTPRoute** (Gateway API) | `satusky-gateway` in `gateway-system` | Cloudflare (platform) |
| Custom (`*.flyingchicken.xyz`) | **Ingress** (legacy) | `ingress-nginx-my-kul-1a` | Let's Encrypt (`cert-manager`) |

Both route to the same `Service` (e.g. `fullstack-api:8080`). Verify with:

```bash
# Gateway API (primary domains)
kubectl -n <namespace> get httproute

# Legacy Ingress (custom domains)
kubectl -n <namespace> get ingress

# TLS certs for custom domains
kubectl -n <namespace> get secret <app>-custom-letsencrypt-tls
```

---

## 1ctl Implementation — UP (Full Deployment & Verification)

> **Live-verified 2026-06-17** against local backend + production OpenProvider DNS.
> All commands below are the exact invocations that succeeded.

### Phase 0 — Pre-flight Checks

```bash
# 1. Confirm auth is valid
1ctl auth status
# ✅ Authenticated with Satusky
# User Email: <your-email>
# Organization: org123
# Namespace: org123-c0bee423

# 2. Check existing deployments
1ctl deploy list

# 3. Check existing Postgres clusters
1ctl postgres list

# 4. Check available machines
1ctl machine list

# 5. Check existing domains
1ctl domain list
```

### Phase 1 — Create Managed Postgres Cluster

```bash
# 6. List available storage classes
1ctl postgres storage-classes
# ceph-block (default), ceph-block-noreplica, ceph-filesystem, ...

# 7. Create a Postgres cluster.
#    NOTE: --storage-class must come BEFORE the cluster name in urfave/cli v2.
1ctl postgres create --storage-class ceph-block fullstack-db \
  --database fullstack \
  --user app

# 8. Watch the cluster become healthy
1ctl postgres status fullstack-db
# Output: initializing → Waiting for instances → healthy (1/1 ready)

# 9. Get connection credentials
1ctl postgres credentials fullstack-db
# Username: app
# Password: <32-char-hex>
# Database: fullstack_db       ← NB: backend may append _db suffix
# Host: 211.25.36.186 (external) / fullstack-db-pg-rw.<ns>.svc.cluster.local (internal)
# Port: 31560 (external) / 5432 (internal)
```

### Phase 2 — Deploy fullstack-api

```bash
cd examples/fullstack-api

# 10. Deploy with reduced resources for local dev.
#     Use the INTERNAL postgres hostname (runs inside the cluster).
#     --cpu-request/--memory overrides the satusky.toml values for this run.
#     --hpa=false --pdb=false disables features that need multi-replica.
1ctl deploy --config satusky.toml --wait \
  --env APP_ENV=production \
  --env LOG_LEVEL=debug \
  --env DB_HOST=fullstack-db-pg-rw.org123-c0bee423.svc.cluster.local \
  --env DB_PORT=5432 \
  --env DB_USER=app \
  --env DB_NAME=fullstack_db \
  --cpu-request 250m \
  --cpu-limit 1 \
  --memory 256Mi \
  --replicas 1 \
  --hpa=false \
  --pdb=false

# Step 1/5: Building image (cloud)       ← multi-arch amd64+arm64 via buildx
# Step 2/5: Creating/updating deployment
# Step 3/5: Configuring services
# Step 4/5: Setting up environment and storage
# Step 5/5: Configuring ingress and dependencies
# ✅ 🚀 Deployment successful!
# Auto-generated domain: <adjective-animal-XXXXXXX>.satusky.com
```

### Phase 3 — Inject Secrets

```bash
# 11. Create secrets for the deployment.
#     Use the DB_PASSWORD from step 9 (postgres credentials output).
1ctl secret create --config satusky.toml \
  --kv DB_PASSWORD=<password-from-step-9> \
  --kv API_KEY=sk-test-fullstack-api-key \
  --kv SMTP_PASSWORD=test-smtp-password \
  --kv JWT_SECRET=super-secret-jwt-key-at-least-256-bits-long
# ✅ Secret fullstack-api created successfully
# NOTE: The K8s Secret is created but the Deployment spec may not be
# auto-patched to reference it (known gap during Gateway API migration).
# If /api/secrets-info returns false for all keys, manually patch:
#   kubectl -n <ns> patch deployment fullstack-api --type='json' -p='[...]'

# 12. Rolling restart to pick up secrets
1ctl deploy restart --config satusky.toml
# 💡 Initiating rolling restart... ✅ Rolling restart initiated.
```

### Phase 4 — Verify Deployment State

```bash
# 13. Confirm deployment is Running
1ctl deploy list
1ctl deploy status --config satusky.toml
# Workload: Running  |  Progress: 100%  |  Secrets: attached  |  Volume: attached

# 14. Check the persistent volume
1ctl volumes list --config satusky.toml
# Shows volume ID, name, PVC claim, PVC status, mount path, destroy policy

# 15. List all secrets
1ctl secret list

# 16. Check logs (Loki may be unavailable in dev — falls back to stored logs)
1ctl logs --config satusky.toml --tail 20

# 17. Run platform diagnostics
1ctl doctor --config satusky.toml

# 18. Full namespace smoke test
1ctl doctor --smoke
```

### Phase 5 — Set Up Custom Domain

```bash
# 19. Add custom domain (flag --app BEFORE the domain value).
#     --custom-dns treats the hostname as externally-managed.
#     --no-wait returns immediately (DNS takes time to propagate).
1ctl domains add --app fullstack-api --custom-dns --no-wait \
  fullstack.flyingchicken.xyz
# ✅ Domain fullstack.flyingchicken.xyz attached to app fullstack-api
# Shows: App, Namespace, DNS status, TLS status, Required DNS Records

# 20. Create the DNS A record (requires OpenProvider credentials in backend):
1ctl domains dns create --type A --name fullstack \
  --data 211.25.36.186 flyingchicken.xyz
# ✅ DNS record created

# 21. Verify DNS records for the domain
1ctl domains dns list flyingchicken.xyz

# 22. Verify the domain is fully configured
1ctl domain list
# fullstack.flyingchicken.xyz    fullstack-api    custom    letsencrypt

# 23. Check domain health (DNS, TLS, HTTP reachability)
1ctl domains check fullstack.flyingchicken.xyz --probe
# Backend: attached   Route: Ingress/ingress-fullstack-api-custom
# DNS: resolved - 211.25.36.186   TLS: active, expires YYYY-MM-DD

# 24. Show exact DNS setup instructions
1ctl domains setup fullstack.flyingchicken.xyz

# 25. Verify via kubectl:
kubectl -n org123-c0bee423 get httproute fullstack-api-route
kubectl -n org123-c0bee423 get ingress ingress-fullstack-api-custom
kubectl -n org123-c0bee423 get secret fullstack-api-custom-letsencrypt-tls
```

### Phase 6 — DB Migration (if needed)

```bash
# The `app` user created by CNPG may lack CREATE TABLE permissions.
# If /api/tasks returns "relation does not exist", run the migration
# as the superuser via kubectl exec:

kubectl -n org123-c0bee423 exec -it fullstack-db-pg-1 -- \
  psql -U postgres -d fullstack_db -c "
    CREATE TABLE IF NOT EXISTS tasks (
      id SERIAL PRIMARY KEY,
      title TEXT NOT NULL,
      status TEXT NOT NULL DEFAULT 'pending',
      created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );
    GRANT ALL PRIVILEGES ON TABLE tasks TO app;
    GRANT USAGE, SELECT ON SEQUENCE tasks_id_seq TO app;
  "
# CREATE TABLE  |  GRANT  |  GRANT
```

### Phase 7 — End-to-End API Verification

> Replace `<AUTO_DOMAIN>` with the `*.satusky.com` domain and
> `<CUSTOM_DOMAIN>` with `fullstack.flyingchicken.xyz`.

```bash
# Health check (both domains)
curl -sk https://<AUTO_DOMAIN>/health | jq
curl -sk https://<CUSTOM_DOMAIN>/health | jq
# {"status":"ok","timestamp":"...","version":"2.0.0","uptime":"..."}

# API index
curl -sk https://<CUSTOM_DOMAIN>/ | jq
# {"name":"fullstack-api","version":"2.0.0","docs":"/api/tasks",...}

# Active environment variables (no secrets exposed)
curl -sk https://<CUSTOM_DOMAIN>/api/config | jq
# {"app_env":"production","db_host":"fullstack-db-pg-rw...",...}

# Confirm secrets are injected (presence only, values hidden)
curl -sk https://<CUSTOM_DOMAIN>/api/secrets-info | jq
# {"api_key_set":true,"db_password_set":true,"jwt_secret_set":true,"smtp_password_set":true}

# Stats (task count + uptime + DB connection)
curl -sk https://<CUSTOM_DOMAIN>/api/stats | jq

# CRUD — Create a task
curl -sk -X POST https://<CUSTOM_DOMAIN>/api/tasks \
  -H "Content-Type: application/json" \
  -d '{"title":"Verify 1ctl deployment"}' | jq

# CRUD — List tasks
curl -sk https://<CUSTOM_DOMAIN>/api/tasks | jq

# CRUD — Get task by ID
curl -sk https://<CUSTOM_DOMAIN>/api/tasks/1 | jq

# CRUD — Delete a task
curl -sk -X DELETE https://<CUSTOM_DOMAIN>/api/tasks/2 | jq

# Volume — Upload a file (exercises /data/uploads persistent mount)
curl -sk -F "file=@README.md" https://<CUSTOM_DOMAIN>/api/upload | jq
# {"filename":"README.md","size_bytes":15706,"path":"/data/uploads/README.md"}

# Volume — Serve the uploaded file back
curl -sk https://<CUSTOM_DOMAIN>/api/files/README.md | head -5

# NOTE: If DNS hasn't propagated locally yet, use --resolve to bypass:
# curl -sk --resolve <CUSTOM_DOMAIN>:443:211.25.36.186 https://<CUSTOM_DOMAIN>/health
```

### Phase 8 — Feature-Specific Checks

```bash
# 1ctl doctor (deployment-level diagnostics)
1ctl doctor --config satusky.toml

# Verify volume is attached
1ctl volumes list --config satusky.toml

# Verify secrets exist
1ctl secret list

# Rolling restart works
1ctl deploy restart --config satusky.toml

# Ingress architecture verification
kubectl -n org123-c0bee423 get httproute   # Gateway API routes
kubectl -n org123-c0bee423 get ingress     # Legacy Ingress routes
kubectl -n org123-c0bee423 get gateway     # Gateway resources (cluster-level)
```

---

## 1ctl Implementation — DOWN (Full Teardown)

Run these commands in order to clean up every resource created above.

```bash
# Work from the project directory so --app resolves
cd examples/fullstack-api

# 1. Remove the DNS record (if created via OpenProvider)
1ctl domains dns list flyingchicken.xyz
1ctl domains dns delete -y flyingchicken.xyz <RECORD_ID>

# 2. Remove the custom domain (--app flag resolves the deployment)
1ctl domains delete --app fullstack-api fullstack.flyingchicken.xyz -y

# 3. Unset all secrets (--app flag replaces --config)
1ctl secret unset --app fullstack-api --key DB_PASSWORD
1ctl secret unset --app fullstack-api --key API_KEY
1ctl secret unset --app fullstack-api --key SMTP_PASSWORD
1ctl secret unset --app fullstack-api --key JWT_SECRET

# 4. Delete the persistent volume
1ctl volumes list --app fullstack-api
1ctl volumes delete -y <VOLUME_ID>

# 5. Delete the deployment (cascade: also cleans up service, ingress, env)
1ctl deploy delete --app fullstack-api -y

# 6. Delete the Postgres cluster
1ctl postgres delete fullstack-db   # confirm with 'y'

# 7. Confirm everything is gone
1ctl deploy list
1ctl postgres list
1ctl domain list | grep fullstack
```

> **v3 note**: `-y` / `--yes` can now go anywhere — interspersed flags work natively in urfave/cli v3.
> Use `--app <name>` instead of `--config satusky.toml` for deployment-targeting commands.

---

## Known Gaps (Integration Merge — 2026-06-17)

These were discovered during live testing of the merged feature branches:

| # | Gap | Workaround |
|---|---|---|
| 1 | `secret create` makes the K8s Secret but doesn't auto-patch the Deployment to add `secretKeyRef` env vars | Manual `kubectl patch deployment` or redeploy |
| 2 | `CreateVolume` creates a DB record but PVC reconciliation is async — PVC shows "missing" initially | Wait for reconciliation job (~1-2 min) |
| 3 | New Postgres `app` user lacks `CREATE TABLE` permission — app auto-migration fails silently | Run migration as superuser via `kubectl exec` (Phase 6) |

## urfave/cli v3 Migration Notes (2026-06-17)

The CLI was migrated from urfave/cli v2 → v3. Key user-facing changes:

| Change | Before (v2) | After (v3) |
|---|---|---|
| Primary delete command | `destroy` / `remove` | `delete` (old names kept as aliases) |
| App name resolution | `--config satusky.toml` only | `--app <name>` works everywhere |
| Flag position | `-y` must be before positional arg | `-y` can go anywhere (interspersed flags) |
| JSON output | Some commands lacked `-o json` | All list/get commands support `-o json` |
| Config structure | Flat `[app]` section | `[build]`, `[checks]`, `[deploy]` sections |
| Cascade preview | Generic "delete all" prompt | Enumerates Ingress, Volume, Service before confirming |
