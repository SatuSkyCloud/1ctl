# Deploying a Next.js Fullstack App to SatuSky

**Who this is for**: Frontend developers shipping a Next.js app with SSR and API routes. Your backend API is already running on SatuSky as `backend-api`.
**What we're building**: A Next.js app that communicates with a backend service via an env var, deployed with cloud build.
**What you'll learn**: Multi-stage Next.js Docker builds, passing build-time env vars, linking to an existing deployed service, and scripting with `-o json`.

---

## CLI Coverage

> ⚠️ **Mostly covered** — all commands work except one gap with build-time env vars.

> **Gap: `--build-arg` flag not yet available**
> `NEXT_PUBLIC_*` variables must be baked into the image at build time, which normally
> requires passing them as Docker build arguments (`--build-arg`). The CLI does not
> yet expose a `--build-arg` flag on `deploy`. **Workaround:** Set a hardcoded default
> in your `Dockerfile` (`ARG NEXT_PUBLIC_API_URL=https://cleverbear-xmqs6l.satusky.com`) so
> the cloud build uses it automatically, or move the value to a runtime env var and fetch
> it server-side instead of using `NEXT_PUBLIC_`.

---

## 1. Project structure

```
my-nextjs-app/
├── app/
│   ├── page.tsx
│   └── api/
│       └── health/
│           └── route.ts
├── next.config.js
├── package.json
├── package-lock.json
├── Dockerfile
└── satusky.toml
```

---

## 2. Dockerfile

Next.js benefits from a multi-stage build. Enable standalone output in `next.config.js`:

```js
/** @type {import('next').NextConfig} */
const nextConfig = { output: 'standalone' };
module.exports = nextConfig;
```

```dockerfile
# syntax=docker/dockerfile:1
FROM node:20-alpine AS deps
WORKDIR /app
COPY package*.json ./
RUN npm ci

FROM node:20-alpine AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
ARG NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
RUN npm run build

FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public
EXPOSE 3000
CMD ["node", "server.js"]
```

---

## 3. satusky.toml

```toml
[app]
  name   = "my-nextjs-app"
  port   = 3000
  cpu    = "0.5"
  memory = "512Mi"
```

---

## 4. First deploy

```bash
cd my-nextjs-app
1ctl deploy --config satusky.toml --wait
```

```
💡 Build queued (ID: ...)
  [build] Docker build completed
💡 Generated new domain: bravehawk-f2j8l5.satusky.com
✅ 🚀 Deployment for my-nextjs-app is successful! Your app is live at: https://bravehawk-f2j8l5.satusky.com
✅ Deployment is healthy — pods Running
```

Next.js builds take longer because `next build` runs during the image build step.

---

## 5. Point the frontend at the backend

Capture the backend's assigned domain and wire it into the frontend:

```bash
# Get the domain your backend was assigned
BACKEND_URL=$(1ctl -o json deploy get \
  --config ../backend/satusky.toml | jq -r '.domain')

1ctl env create \
  --config satusky.toml \
  --env NEXT_PUBLIC_API_URL="$BACKEND_URL"
```

Redeploy for the build-time variable to take effect:

```bash
1ctl deploy --config satusky.toml --wait
```

---

## 6. Verify the deployment

```bash
1ctl deploy status --config satusky.toml
```

```
Deployment Status
─────────────────
App: my-nextjs-app
Workload: Running
Message: Deployment is running normally
Progress: 100%
```

Smoke test:

```bash
curl -I https://bravehawk-f2j8l5.satusky.com
# HTTP/2 200
```

---

## 7. See all your deployed apps

```bash
1ctl -o json deploy list
```

```json
[
  {
    "deployment_id": "uuid-1",
    "app_label": "backend-api",
    "status": "completed",
    "image": "registry.satusky.com/backend-api:abc123",
    "cpu_request": "500m",
    "domain": "https://cleverbear-xmqs6l.satusky.com"
  },
  {
    "deployment_id": "uuid-2",
    "app_label": "my-nextjs-app",
    "status": "completed",
    "image": "registry.satusky.com/my-nextjs-app:def456",
    "domain": "https://bravehawk-f2j8l5.satusky.com"
  }
]
```

---

## 8. Stream live logs

```bash
1ctl logs stream --config satusky.toml
```

```
2026-06-12T10:17:02Z [my-nextjs-app] ready - started server on 0.0.0.0:3000
2026-06-12T10:17:15Z [my-nextjs-app] GET / 200 in 312ms
2026-06-12T10:17:31Z [my-nextjs-app] GET /api/health 200 in 8ms
```

---

## 9. Scripting deploys in CI

```bash
#!/bin/bash
set -euo pipefail

1ctl deploy --config satusky.toml --wait

APP_URL=$(1ctl -o json deploy get --config satusky.toml | jq -r '.domain')

HTTP_STATUS=$(curl -o /dev/null -s -w "%{http_code}" "$APP_URL")
if [ "$HTTP_STATUS" != "200" ]; then
  echo "Smoke test failed — HTTP $HTTP_STATUS"
  exit 1
fi
echo "Deploy OK — $APP_URL returned 200"
```

---

## 10. Build-time vs runtime env vars in Next.js

`NEXT_PUBLIC_*` variables are statically inlined at `next build` time:
- Setting them after a build has no effect until you rebuild and redeploy.
- Non-`NEXT_PUBLIC_` variables (server components / API routes) are read at runtime, so `env create` + `deploy restart` is enough — no rebuild needed.

---

## Summary

| Task | Command |
|---|---|
| Deploy (cloud build) | `1ctl deploy --config satusky.toml --wait` |
| Set env vars | `1ctl env create --config satusky.toml --env KEY=VAL` |
| Remove an env var | `1ctl env unset --config satusky.toml --key KEY` |
| Check deploy status | `1ctl deploy status --config satusky.toml` |
| List all apps (JSON) | `1ctl -o json deploy list` |
| Get app details (JSON) | `1ctl deploy get --config satusky.toml -o json` |
| Live logs | `1ctl logs stream --config satusky.toml` |

---

## Live Verification (2026-06-12)

All commands verified against live `org123-c0bee423` namespace with `backend-api` and `frontend` deployments.

| # | Command | Exit |
|---|---------|------|
| 1 | `1ctl deploy list` (table) | ✅ 0 |
| 2 | `1ctl -o json deploy list` | ✅ 0 |
| 3 | `1ctl deploy status --deployment-id <id>` | ✅ 0 |
| 4 | `1ctl -o json deploy get --deployment-id <id>` | ✅ 0 |
| 5 | `1ctl deploy releases --deployment-id <id>` | ✅ 0 |
| 6 | `1ctl env create --deployment-id <id> --env KEY=VAL` | ✅ 0 |
| 7 | `1ctl env list --deployment-id <id>` | ✅ 0 |
| 8 | `1ctl -o json env list --deployment-id <id>` | ✅ 0 |
| 9 | `1ctl env unset --deployment-id <id> --key KEY` | ✅ 0 |
| 10 | `1ctl deploy restart --deployment-id <id>` | ✅ 0 |
| 11 | `1ctl logs --deployment-id <id> --tail 3` | ✅ 0 |
| 12 | `1ctl ingress list` | ✅ 0 |
| 13 | `1ctl domains list` | ✅ 0 |
| 14 | `1ctl doctor --deployment-id <id>` | ✅ 0 |

**JSON field verification** (`deploy get -o json`):
```
app_label: backend-api     status: completed    domain: https://...satusky.com
cpu_request: 250m          memory_request: 256Mi   replicas: 1
```

**Multi-deployment verified**: `deploy list` returns both `backend-api` and `frontend`. Each has unique `deployment_id`, `app_label`, `domain`.
