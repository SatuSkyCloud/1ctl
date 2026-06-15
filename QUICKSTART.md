# 1ctl Quickstart

Deploy your first application to SatuSky Cloud in under 5 minutes.

## Prerequisites

- `1ctl` installed — see [README](README.md#installation) (Homebrew, `install.sh`, or build from source)
- A SatuSky Cloud account and API token from `https://cloud.satusky.com/<org-id>/token`

---

## 1. Set up a profile

A profile stores your API endpoint and credentials. You only need to do this once.

```bash
# Create a profile pointing at the SatuSky Cloud API
1ctl profile create --url https://api.satusky.com/v1/cli prod

# Activate it
1ctl profile use prod

# Authenticate with your API token
1ctl auth login --token <your-api-token>

# Verify
1ctl profile current
```

---

## Local development (against a local backend)

When running a local SatuSky backend (e.g. at `http://localhost:8080`), use the built-in `local` profile:

```bash
# Build the CLI from source
go build -o 1ctl ./cmd/...

# Create a profile pointing at the local backend (or it already exists)
./1ctl profile create --url http://localhost:8080/v1/cli local

# Switch to local profile
./1ctl profile use local

# Authenticate (same API token works against local backend)
./1ctl auth login --token <your-api-token>

# Verify you're logged in
./1ctl profile current
```

Expected output:
```
Active Profile
──────────────
Profile: local
API URL: http://localhost:8080/v1/cli
Auth:    mingerz.k@gmail.com
```

---

## 2. Initialize your project (optional)

Running `1ctl init` in any project directory creates a `satusky.toml` that persists your deploy settings so you don't have to repeat flags every time.

```bash
cd my-app
1ctl init
```

The generated `satusky.toml` looks like:
```toml
[app]
name = "my-app"
port = 8080
cpu = "0.5"
memory = "256Mi"
dockerfile = "Dockerfile"
```

Once a `satusky.toml` exists, you can deploy with just:
```bash
1ctl deploy
```

---

## Examples

The `examples/` directory contains two ready-to-deploy apps:

| App | Language | Port | Description |
|-----|----------|------|-------------|
| `examples/backend` | Go | 8080 | JSON REST API with `/health` and `/api/message` endpoints |
| `examples/frontend` | nginx + HTML | 80 | Static frontend that calls the backend API |

Each example has a `satusky.toml` with all deploy settings pre-configured.

---

## Deploy the Backend API

```bash
cd examples/backend

# Deploy — reads cpu/memory/port from satusky.toml, no flags needed
./1ctl deploy
```

Or with explicit flags:
```bash
./1ctl deploy --cpu-request 250m --cpu-limit 1 --memory 256Mi --port 8080
```

What happens:
1. `1ctl` packages your source directory into a `.tar.gz` (respects `.dockerignore`)
2. Context is uploaded to `POST /v1/cli/builds`
3. The backend runs a cloud build — builds your Dockerfile and pushes the image to `registry.satusky.com/satusky-container-registry/backend-api:<build-id>`
4. `1ctl` streams build logs until the image is ready
5. The image ref is handed to the deployment orchestrator

After a successful deploy you'll see:
```
✅ 🚀 Deployment for backend-api is successful! Your app is live at: https://backend-api.satusky.com
Deployment ID: <uuid>
```

Test it:

```bash
curl https://backend-api.satusky.com/health
# {"status":"ok","timestamp":"2026-04-09T...","version":"1.0.0"}

curl https://backend-api.satusky.com/api/message
# {"message":"Greetings from the SatuSky backend API!","from":"backend-api"}
```

---

## Deploy the Frontend

```bash
cd examples/frontend

# Deploy — reads settings from satusky.toml
./1ctl deploy
```

Output:
```
✅ 🚀 Deployment for frontend is successful! Your app is live at: https://frontend.satusky.com
Deployment ID: <uuid>
```

Open `https://frontend.satusky.com` in your browser and click **Call Backend API** to see the two apps communicate.

> **Connect frontend → backend**: Edit `examples/frontend/nginx.conf`, uncomment the `location /api/` block, and set the `proxy_pass` URL to your backend domain. Then redeploy with `1ctl deploy`.

---

## Managing your deployments

```bash
# List all your deployments
1ctl deploy list

# Check status
1ctl deploy status --deployment-id <uuid>

# Watch status until it's running
1ctl deploy status --deployment-id <uuid> --watch

# View release history
1ctl deploy releases --deployment-id <uuid>

# Roll back to the previous release
1ctl deploy rollback --deployment-id <uuid>

# Rolling restart (without a new build)
1ctl deploy restart --deployment-id <uuid>

# Tear down a deployment and all its resources
1ctl deploy destroy --deployment-id <uuid> --yes
```

---

## Environment variables

Pass secrets and config to your app at deploy time:

```bash
1ctl deploy --cpu 0.5 --memory 256Mi --port 8080 \
  --env DATABASE_URL=postgres://... \
  --env API_SECRET=my-secret \
  --env LOG_LEVEL=info
```

Or manage them separately after deployment:

```bash
1ctl env list   --deployment-id <uuid>
1ctl env set    --deployment-id <uuid> KEY=VALUE
1ctl env delete --deployment-id <uuid> KEY
```

---

## Secrets

For sensitive values use `1ctl secret` — secrets are stored encrypted and injected at runtime:

```bash
1ctl secret create --deployment-id <uuid> --key DB_PASSWORD --value supersecret
1ctl secret list   --deployment-id <uuid>
```

---

## Custom domain

```bash
1ctl deploy --cpu 0.5 --memory 256Mi --port 8080 --domain api.mycompany.com
```

Then point your DNS `CNAME` to `backend-api.satusky.com`.

---

## Using a pre-built image

If you already have an image in the SatuSky registry (or any accessible registry), skip the cloud build entirely:

```bash
1ctl deploy --cpu 0.5 --memory 256Mi --port 8080 \
  --image registry.satusky.com/satusky-container-registry/backend-api:abc1234
```

---

## Multi-environment workflow

```bash
# Create a staging profile
1ctl profile create --url https://api.satusky.com/v1/cli staging
1ctl profile use staging
1ctl auth login --token <staging-token>

# Deploy to staging
1ctl --profile staging deploy --cpu 0.5 --memory 256Mi --port 8080

# Deploy to prod without switching profiles
1ctl --profile prod deploy --cpu 1 --memory 512Mi --port 8080
```

---

## How cloud builds work

When you run `1ctl deploy`:

1. **Context packaged** — your project directory is compressed into a `.tar.gz` (respects `.dockerignore`)
2. **Uploaded to SatuSky** — the context is sent to `POST /v1/cli/builds`
3. **Image built** — the backend runs a cloud build that builds your Dockerfile and pushes the image to `registry.satusky.com/satusky-container-registry/<project>:<sha>`
4. **CLI polls** — `1ctl` streams build logs until the image is ready
5. **Deployed** — the image ref is handed to the deployment orchestrator which creates/updates K8s Deployment, Service, Ingress, and Environment resources

No local Docker installation required.

---

## Local end-to-end test (verified 2026-04-09)

The following steps were run against a local API server (`http://localhost:8080`) to verify the full flow:

```bash
# 1. Build CLI from source
go build -o 1ctl ./cmd/...

# 2. Start the local API server in another terminal
#    (refer to the server's own README for the exact command)

# 3. Switch to local profile and authenticate
./1ctl profile use local
./1ctl auth login --token <your-api-token>
# ✅ Logged in successfully to SatuSky 1ctl as mingerz.k@gmail.com!

# 4. List existing deployments
./1ctl deploy list

# 5. Destroy old deployments (if any)
./1ctl deploy destroy --deployment-id <uuid> --yes

# 6. Deploy backend from examples/backend/ using satusky.toml (no flags needed)
cd examples/backend
../../1ctl deploy
# ✅ Cloud build complete: registry.satusky.com/satusky-container-registry/backend-api:<build-id>
# ✅ 🚀 Deployment for backend-api is successful! Your app is live at: https://backend-api.satusky.com

# 7. Deploy frontend from examples/frontend/ using satusky.toml
cd ../frontend
../../1ctl deploy
# ✅ Cloud build complete: registry.satusky.com/satusky-container-registry/frontend:<build-id>
# ✅ 🚀 Deployment for frontend is successful! Your app is live at: https://frontend.satusky.com

# 8. Verify management commands
./1ctl deploy list
./1ctl deploy status --deployment-id <uuid>
./1ctl deploy releases --deployment-id <uuid>
./1ctl deploy restart --deployment-id <uuid>
```

**What was verified (full 5-step pipeline):**
- Cloud builds succeed for both Go and nginx/static apps
- App name from `satusky.toml` (`backend-api`, `frontend`) is used correctly in the image tag
- CPU/memory/port defaults are loaded from `satusky.toml` — no CLI flags required
- Float CPU values (e.g. `0.5`) are accepted by the validator
- All 5 deploy steps complete: image build → deployment → service → environment → ingress
- Idempotent upsert: re-deploying over existing K8s resources (from prior runs) works correctly
- Management commands: `list`, `status`, `releases`, `restart`, `rollback` all verified

**Known local behaviour:** After a successful deploy, `deploy status` will show `NotReady` / "Deployment does not have minimum availability" if the local K8s cluster has no worker nodes with sufficient capacity to schedule pods. This is expected — the deploy pipeline succeeded and K8s resources were created. The pods will start when capacity is available.

---

## CPU format reference

All of these are valid CPU values:

| Format | Example | Meaning |
|--------|---------|---------|
| Float  | `0.5`   | 0.5 cores (500m) |
| Integer | `1`    | 1 core |
| Millicores | `500m` | 500 millicores (0.5 cores) |

`--cpu-request` is the guaranteed scheduler reservation. `--cpu-limit` is the burst ceiling. The default shared tier is `250m` request with `1` vCPU burst.

---

## Next steps

- Browse available machines: `1ctl machine list`
- View logs: `1ctl logs --deployment-id <uuid>`
- Set up autoscaling: `1ctl deploy --hpa --hpa-min-replicas 2 --hpa-max-replicas 10 --cpu-request 250m --cpu-limit 1 --memory 512Mi --port 8080`
- Explore all commands: `1ctl --help`
- Full docs: [docs.satusky.com/cli](https://docs.satusky.com/cli)
