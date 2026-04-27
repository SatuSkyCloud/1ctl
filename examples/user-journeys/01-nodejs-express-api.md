# Deploying a Node.js Express API to SatuSky

**Who this is for**: Full-stack JS developers deploying their first app to SatuSky.  
**What we're building**: A Node.js + Express REST API that connects to a Neon.tech PostgreSQL database, uses JWT auth, and exposes a JSON API.  
**What you'll learn**: Cloud build, secrets for credentials, env vars for config, live log streaming, rolling back a bad deploy.

---

## CLI Coverage

> ✅ **Fully covered** — every command in this guide works with the current CLI.
> No gaps.

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
name    = "my-api"
port    = 3000
cpu     = "0.5"
memory  = "512Mi"
```

That's it — no org, no deployment IDs. The platform resolves by `name` at runtime.

---

## 4. First deploy

SatuSky builds the image for you in the cloud — no local Docker daemon needed.

```bash
cd my-api
1ctl-dev deploy --config satusky.toml --wait
```

`--wait` blocks until pods are Running and healthy. You'll see output like:

```
Building image...  done (42s)
Pushing image...   done (8s)
Creating deployment my-api...
Waiting for pods to be Running...
  my-api-7d9f4b6c8-xk2p9   Running   ✓
Deploy complete. App is live.
```

If you skip `--wait` and check status immediately, the deployment may still be in progress:

```bash
# Checking too early — pods may show Pending or ContainerCreating
1ctl-dev deploy status --config satusky.toml
# STATUS: Pending  (image pull in progress)
```

Wait a few seconds and run it again, or just use `--wait` from the start to avoid the guesswork.

---

## 5. Set secrets

Database URL and JWT secret are credentials — store them as secrets, not env vars. Secrets are encrypted at rest and injected into the container at startup.

```bash
1ctl-dev secret create \
  --config satusky.toml \
  --kv DB_URL=postgresql://alice:s3cr3t@ep-cool-cloud-123456.us-east-2.aws.neon.tech/mydb?sslmode=require \
  --kv JWT_SECRET=hs256-super-long-random-string-change-me-in-prod
```

`secret create` merges — existing keys are updated, unrelated keys are left alone.

To remove a secret you no longer need:

```bash
1ctl-dev secret unset --config satusky.toml --key JWT_SECRET
```

---

## 6. Set environment variables

Non-sensitive config lives in env vars. These are visible in deploy metadata, so keep credentials in secrets.

```bash
1ctl-dev env create \
  --config satusky.toml \
  --env CORS_ORIGIN=https://app.example.com \
  --env NODE_ENV=production
```

Changes take effect on the next deploy. Trigger a redeploy to pick them up:

```bash
1ctl-dev deploy --config satusky.toml --wait
```

---

## 7. Stream live logs

In a second terminal, tail the log stream while you test:

```bash
1ctl-dev logs stream --config satusky.toml
```

You'll see structured output like:

```
2026-04-26T14:03:01Z [my-api] Server listening on port 3000
2026-04-26T14:03:12Z [my-api] GET /api/users 200 14ms
2026-04-26T14:03:18Z [my-api] POST /api/auth/login 200 38ms
2026-04-26T14:03:25Z [my-api] GET /api/users/42 401 3ms
```

Press `Ctrl+C` to stop the stream.

---

## 8. Verify with curl

```bash
# Health check
curl https://my-api.satusky.com/health
# {"status":"ok","db":"connected"}

# Auth endpoint
curl -X POST https://my-api.satusky.com/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"hunter2"}'
# {"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."}
```

---

## 9. Re-deploy after a code change

Edit `src/index.js`, commit, and deploy again. The platform upserts — it creates a new version and keeps the old one in history.

```bash
git add src/index.js
git commit -m "fix: return 404 when user not found"
1ctl-dev deploy --config satusky.toml --wait
```

Output shows the new version number:

```
Building image...  done (39s)
Deploy complete. Version: 4
```

---

## 10. View release history

```bash
1ctl-dev deploy releases --config satusky.toml
```

```
VERSION  STATUS    DEPLOYED AT           MESSAGE
4        active    2026-04-26 14:22:11   fix: return 404 when user not found
3        inactive  2026-04-26 13:55:04   feat: add /api/users/:id route
2        inactive  2026-04-26 12:10:47   chore: update dependencies
1        inactive  2026-04-26 11:30:00   initial deploy
```

---

## 11. Roll back to a previous version

Version 4 broke something — let's roll back to version 3 immediately.

```bash
1ctl-dev deploy rollback --config satusky.toml --version 3
```

```
Rolling back my-api to version 3...
Waiting for pods to be Running...
  my-api-6c8d9b5f2-tz7n1   Running   ✓
Rollback complete. Now running version 3.
```

No image rebuild needed — the platform reruns the already-built image from version 3.

---

## Summary

| Task | Command |
|---|---|
| First deploy (cloud build) | `1ctl-dev deploy --config satusky.toml --wait` |
| Set credentials | `1ctl-dev secret create --config satusky.toml --kv KEY=VAL` |
| Remove a secret | `1ctl-dev secret unset --config satusky.toml --key KEY` |
| Set config vars | `1ctl-dev env create --config satusky.toml --env KEY=VAL` |
| Live logs | `1ctl-dev logs stream --config satusky.toml` |
| Release history | `1ctl-dev deploy releases --config satusky.toml` |
| Roll back | `1ctl-dev deploy rollback --config satusky.toml --version N` |
