# Deploying a Node.js Express API to SatuSky

**Who this is for**: Full-stack JS developers deploying their first app to SatuSky.  
**What we're building**: A Node.js + Express REST API that connects to a Neon.tech PostgreSQL database, uses JWT auth, and exposes a JSON API.  
**What you'll learn**: Cloud build, secrets for credentials, env vars for config, live log streaming, rolling back a bad deploy.

---

## 1. Project structure

Your repo looks like this:

```
my-api/
├── src/
│   └── index.js
├── package.json
├── package-lock.json
├── Dockerfile
└── satusky.toml
```

---

## 2. Dockerfile

Multi-stage build keeps the final image small. Dependencies are installed in the builder stage; only production artifacts land in the runtime image.

```dockerfile
# syntax=docker/dockerfile:1
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci --omit=dev

FROM node:20-alpine AS runtime
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app/node_modules ./node_modules
COPY src/ ./src/
COPY package.json ./
EXPOSE 3000
CMD ["node", "src/index.js"]
```

---

## 3. satusky.toml

```toml
[app]
  name   = "my-api"
  port   = 3000
  cpu    = "0.5"
  memory = "512Mi"
```

That's it — no org, no deployment IDs. The platform resolves by `name` at runtime.

---

## 4. First deploy

SatuSky builds the image for you in the cloud — no local Docker daemon needed.

```bash
cd my-api
1ctl deploy --config satusky.toml --wait
```

`--wait` blocks until pods are Running and healthy. You'll see output like:

```
💡 Build queued (ID: ...)
  [build] Docker build completed
💡 Image architecture: amd64
Step 2/5: Creating/updating deployment my-api ✓
Step 3/5: ...
💡 Generated new domain: happyotter-x3k9m2.satusky.com
✅ 🚀 Deployment for my-api is successful! Your app is live at: https://happyotter-x3k9m2.satusky.com
💡 Waiting for deployment to become healthy...
✅ Deployment is healthy — pods Running
```

If you skip `--wait` and check status immediately, the deployment may still be in progress:

```bash
1ctl deploy status --config satusky.toml
# Workload: Pending  (image pull in progress)
```

Wait a few seconds and run it again, or just use `--wait` from the start.

---

## 5. Set secrets

Database URL and JWT secret are credentials — store them as secrets, not env vars. Secrets are encrypted at rest and injected into the container at startup.

```bash
1ctl secret create \
  --config satusky.toml \
  --kv DB_URL=postgresql://alice:s3cr3t@ep-cool-cloud-123456.us-east-2.aws.neon.tech/mydb?sslmode=require \
  --kv JWT_SECRET=hs256-super-long-random-string-change-me-in-prod
```

`secret create` merges — existing keys are updated, unrelated keys are left alone.

To remove a secret you no longer need:

```bash
1ctl secret unset --config satusky.toml --key JWT_SECRET
```

---

## 6. Set environment variables

Non-sensitive config lives in env vars. These are visible in deploy metadata, so keep credentials in secrets.

```bash
1ctl env create \
  --config satusky.toml \
  --env CORS_ORIGIN=https://app.example.com \
  --env NODE_ENV=production
```

Changes take effect on the next restart:

```bash
1ctl deploy restart --config satusky.toml
```

---

## 7. Stream live logs

In a second terminal, tail the log stream while you test:

```bash
1ctl logs stream --config satusky.toml
```

You'll see structured output like:

```
2026-06-12T14:03:01Z [my-api-7d9f4b6c8-xk2p9] Server listening on port 3000
2026-06-12T14:03:12Z [my-api-7d9f4b6c8-xk2p9] GET /api/users 200 14ms
2026-06-12T14:03:18Z [my-api-7d9f4b6c8-xk2p9] POST /api/auth/login 200 38ms
```

Press `Ctrl+C` to stop the stream.

---

## 8. Verify with curl

> The URL in the deploy output (e.g. `https://happyotter-x3k9m2.satusky.com`) is the actual domain — substitute it below.

```bash
# Health check
curl https://happyotter-x3k9m2.satusky.com/health
# {"status":"ok","db":"connected"}

# Auth endpoint
curl -X POST https://happyotter-x3k9m2.satusky.com/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"hunter2"}'
# {"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."}
```

---

## 9. Re-deploy after a code change

Edit `src/index.js`, commit, and deploy again. The platform creates a new release and keeps the old one in history.

```bash
git add src/index.js
git commit -m "fix: return 404 when user not found"
1ctl deploy --config satusky.toml --wait
```

---

## 10. View release history

```bash
1ctl deploy releases --config satusky.toml
```

```
VERSION  IMAGE                                   STATUS       DEPLOYED
4        registry.satusky.com/my-api:d4e5f6a      active       2 min ago
3        registry.satusky.com/my-api:c3b4a5c      superseded   2 hours ago
2        registry.satusky.com/my-api:b2a3c4d      superseded   1 day ago
1        registry.satusky.com/my-api:a1b2c3d      superseded   2 days ago
```

---

## 11. Roll back to a previous version

Version 4 broke something — roll back to version 3.

```bash
1ctl deploy rollback --config satusky.toml --version 3 -y
```

```
💡 Rolling back to release 3...
✅ Rollback to version 3 initiated
```

No image rebuild needed — the platform reruns the already-built image from version 3.

---

## Summary

| Task | Command |
|---|---|
| First deploy (cloud build) | `1ctl deploy --config satusky.toml --wait` |
| Set credentials | `1ctl secret create --config satusky.toml --kv KEY=VAL` |
| Remove a secret | `1ctl secret unset --config satusky.toml --key KEY` |
| Set config vars | `1ctl env create --config satusky.toml --env KEY=VAL` |
| Apply env/secret changes | `1ctl deploy restart --config satusky.toml` |
| Live logs | `1ctl logs stream --config satusky.toml` |
| Release history | `1ctl deploy releases --config satusky.toml` |
| Roll back | `1ctl deploy rollback --config satusky.toml --version N -y` |

---

## Live Verification (2026-06-12)

All commands verified against live `org123-c0bee423` namespace with `backend-api` deployment.

| # | Command | Exit |
|---|---------|------|
| 1 | `1ctl deploy list` | ✅ 0 |
| 2 | `1ctl -o json deploy list` | ✅ 0 |
| 3 | `1ctl deploy status --deployment-id <id>` | ✅ 0 |
| 4 | `1ctl -o json deploy get --deployment-id <id>` | ✅ 0 |
| 5 | `1ctl deploy releases --deployment-id <id>` | ✅ 0 |
| 6 | `1ctl secret create --deployment-id <id> --kv X=Y` | ✅ 0 |
| 7 | `1ctl secret list` | ✅ 0 |
| 8 | `1ctl secret unset --deployment-id <id> --key X` | ✅ 0 |
| 9 | `1ctl env create --deployment-id <id> --env X=Y` | ✅ 0 |
| 10 | `1ctl env list --deployment-id <id>` | ✅ 0 |
| 11 | `1ctl -o json env list --deployment-id <id>` | ✅ 0 |
| 12 | `1ctl env unset --deployment-id <id> --key X` | ✅ 0 |
| 13 | `1ctl deploy restart --deployment-id <id>` | ✅ 0 |
| 14 | `1ctl logs --deployment-id <id> --tail 3` | ✅ 0 |
| 15 | `1ctl doctor --deployment-id <id>` | ✅ 0 |

**JSON field verification** (`deploy get -o json`):
```
app_label: backend-api     status: completed    domain: https://...satusky.com
cpu_request: 250m          memory_request: 256Mi   replicas: 1
```

**Releases columns** (`deploy releases`): `VERSION  IMAGE  STATUS  DEPLOYED`

**Status values**: `active`, `superseded`, `rolled_back`
