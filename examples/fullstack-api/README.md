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
| `[app]` | `name`, `port`, `dockerfile`, `cpu_request`, `cpu_limit`, `memory`, `replicas`, `health_path`, `strategy`, `rolling_max_surge`, `rolling_max_unavailable`, `zone`, `machine_tag`, `wait_for` | Full compute + networking + scheduling |
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
1ctl volumes detach --deployment-id <id> --volume-id <vid>

# 5. Re-attach (data comes back)
1ctl volumes attach --deployment-id <id> --volume-id <vid>

# 6. Destroy volume permanently
1ctl volumes destroy --deployment-id <id> --volume-id <vid> -y
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
| `app.dockerfile` | `--dockerfile` | `dockerfile_path` | `Dockerfile` |
| `app.cpu_request` | `--cpu-request` | `cpu_request` | `"0.5"` |
| `app.cpu_limit` | `--cpu-limit` | `cpu_limit` | equals cpu_request |
| `app.memory` | `--memory` | `memory` | `"256Mi"` |
| `app.replicas` | `--replicas` | `replicas` | `1` |
| `app.domain` | `--domain` | `domain` | auto-generated |
| `app.health_path` | `--health-path` | `smoke_path` | tries `/health` then `/` |
| `app.zone` | `--zone` | `zone` | auto |
| `app.machine_tag` | — | `machine_tag` | — |
| `app.strategy` | `--strategy` | `strategy` | `"rolling"` |
| `app.rolling_max_surge` | `--rolling-max-surge` | `rolling_max_surge` | `"25%"` |
| `app.rolling_max_unavailable` | `--rolling-max-unavailable` | `rolling_max_unavailable` | `"25%"` |
| `app.wait_for` | — | `wait_for` | — |
| `volume.size` | `--volume-size` | `volume.size` | — |
| `volume.mount` | `--volume-mount` | `volume.mount_path` | — |
| `hpa.*` | `--hpa`, `--hpa-min-replicas`, etc. | `hpa_config.*` | disabled |
| `vpa.*` | `--vpa`, `--vpa-mode`, etc. | `vpa_config.*` | disabled |
| `pdb.*` | `--pdb`, `--pdb-type`, etc. | `pdb_config.*` | disabled |
| `multicluster.*` | `--multicluster` etc. | `multicluster_*` | disabled |
