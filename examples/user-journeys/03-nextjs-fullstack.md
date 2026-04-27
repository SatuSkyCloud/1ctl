# Deploying a Next.js Fullstack App to SatuSky

**Who this is for**: Frontend developers shipping a Next.js app with SSR and API routes. Your backend API is already running on SatuSky as `backend-api`.  
**What we're building**: A Next.js app that communicates with `backend-api.satusky.com` via an env var, deployed with cloud build.  
**What you'll learn**: Multi-stage Next.js Docker builds, passing build-time env vars, linking to an existing deployed service, and scripting with `-o json`.

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

Next.js benefits from a multi-stage build. The `builder` stage runs `next build`; the `runner` stage is a minimal image that just executes the compiled output.

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
# NEXT_PUBLIC_* vars are baked into the client bundle at build time.
# The cloud build context passes these as --build-arg or from env vars
# set on the deployment (see section 5).
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

Enable Next.js standalone output in `next.config.js`:

```js
/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'standalone',
};

module.exports = nextConfig;
```

---

## 3. satusky.toml

```toml
name   = "my-nextjs-app"
port   = 3000
cpu    = "0.5"
memory = "512Mi"
```

---

## 4. First deploy

```bash
cd my-nextjs-app
1ctl-dev deploy --config satusky.toml --wait
```

```
Building image...  done (1m 48s)
Pushing image...   done (15s)
Creating deployment my-nextjs-app...
Waiting for pods to be Running...
  my-nextjs-app-8b7c3d5e1-np6q4   Running   ✓
Deploy complete. App is live.
```

Next.js builds take longer than other frameworks because `next build` runs during the image build step. This is expected.

---

## 5. Point the frontend at the backend

`NEXT_PUBLIC_API_URL` needs to be available **at build time** (it's embedded in the client bundle) and **at runtime** (for server-side fetch calls). Set it as an env var before deploying so the cloud build context picks it up:

```bash
1ctl-dev env create \
  --config satusky.toml \
  --env NEXT_PUBLIC_API_URL=https://backend-api.satusky.com
```

Because `NEXT_PUBLIC_API_URL` is baked into the bundle during `next build`, you need to redeploy for the change to take effect:

```bash
1ctl-dev deploy --config satusky.toml --wait
```

Now your client-side code can reference `process.env.NEXT_PUBLIC_API_URL` and server components can use it for SSR data fetching.

---

## 6. Verify the deployment

```bash
1ctl-dev deploy status --config satusky.toml
```

```
NAME              STATUS    VERSION   REPLICAS   URL
my-nextjs-app     running   2         1          https://my-nextjs-app.satusky.com
```

Do a quick smoke test:

```bash
curl -I https://my-nextjs-app.satusky.com
# HTTP/2 200
# content-type: text/html; charset=utf-8
```

---

## 7. See all your deployed apps

You have both the frontend and backend running. List them together:

```bash
1ctl-dev deploy list -o json
```

```json
[
  {
    "name": "backend-api",
    "status": "running",
    "version": 7,
    "cpu": "0.5",
    "memory": "512Mi",
    "url": "https://backend-api.satusky.com",
    "deployed_at": "2026-04-25T18:44:01Z"
  },
  {
    "name": "my-nextjs-app",
    "status": "running",
    "version": 2,
    "cpu": "0.5",
    "memory": "512Mi",
    "url": "https://my-nextjs-app.satusky.com",
    "deployed_at": "2026-04-26T10:15:33Z"
  }
]
```

Extract just the frontend URL in a script:

```bash
FRONTEND_URL=$(1ctl-dev deploy list -o json | jq -r '.[] | select(.name=="my-nextjs-app") | .url')
echo "Frontend live at $FRONTEND_URL"
```

---

## 8. Stream live logs

Watch SSR requests and API route calls in real time:

```bash
1ctl-dev logs stream --config satusky.toml
```

```
2026-04-26T10:17:02Z [my-nextjs-app] ready - started server on 0.0.0.0:3000
2026-04-26T10:17:15Z [my-nextjs-app] GET / 200 in 312ms
2026-04-26T10:17:16Z [my-nextjs-app] GET /_next/static/chunks/main-app.js 200 in 5ms
2026-04-26T10:17:31Z [my-nextjs-app] GET /api/health 200 in 8ms
2026-04-26T10:17:45Z [my-nextjs-app] GET /dashboard 200 in 287ms
```

If your SSR data fetches are failing, you'll see the backend errors inline here, which is handy for diagnosing cross-service issues without jumping between dashboards.

---

## 9. Scripting deploys in CI

A minimal CI step that deploys, waits, and then smoke-tests the app URL:

```bash
#!/bin/bash
set -euo pipefail

# Deploy and wait for healthy pods
1ctl-dev deploy --config satusky.toml --wait

# Grab the URL from JSON output
APP_URL=$(1ctl-dev deploy get --config satusky.toml -o json | jq -r '.url')

# Smoke test
HTTP_STATUS=$(curl -o /dev/null -s -w "%{http_code}" "$APP_URL")
if [ "$HTTP_STATUS" != "200" ]; then
  echo "Smoke test failed — HTTP $HTTP_STATUS"
  exit 1
fi
echo "Deploy OK — $APP_URL returned 200"
```

---

## 10. A note on build-time vs runtime env vars in Next.js

`NEXT_PUBLIC_*` variables are statically inlined at `next build` time. This means:

- Setting them after a build has no effect until you rebuild and redeploy.
- Non-`NEXT_PUBLIC_` variables (used only in server components or API routes) are read at runtime from the container environment, so `env create` + `deploy` is enough — no full rebuild needed.

If you change `NEXT_PUBLIC_API_URL`, always follow it with a full `deploy` to trigger a new build.

---

## Summary

| Task | Command |
|---|---|
| Deploy (cloud build) | `1ctl-dev deploy --config satusky.toml --wait` |
| Set env vars (incl. build-time) | `1ctl-dev env create --config satusky.toml --env KEY=VAL` |
| Remove an env var | `1ctl-dev env unset --config satusky.toml --key KEY` |
| Check deploy status | `1ctl-dev deploy status --config satusky.toml` |
| List all apps (JSON) | `1ctl-dev deploy list -o json` |
| Get app details (JSON) | `1ctl-dev deploy get --config satusky.toml -o json` |
| Live logs | `1ctl-dev logs stream --config satusky.toml` |
