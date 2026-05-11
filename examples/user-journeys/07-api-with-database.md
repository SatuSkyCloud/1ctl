# User Journey 7: Deploying an API with a PostgreSQL Database

**Who this is for**: A backend developer deploying a Go (or Python) API that reads its database connection string from the environment.

**Goal**: Deploy an API, wire up `DATABASE_URL` as a secret, tune connection pooling via env vars, and verify everything is healthy.

---

## CLI Coverage

> ⚠️ **Mostly covered** — all database connection commands work. One gap with the
> optional Postgres addon.

> **Gap: `--postgres` addon flag does not exist**
> The guide mentions `--postgres` as an optional built-in addon. That flag does not
> exist in the current CLI. **Workaround:** use an external managed database
> (Neon, Supabase, PlanetScale, Railway Postgres, etc.) and store the full
> `DATABASE_URL` connection string as a secret:
> ```bash
> 1ctl secret create --config satusky.toml \
>   --kv DATABASE_URL=postgres://user:pass@host/db?sslmode=require
> ```
> All other commands in this guide — `deploy restart`, `env unset`, `logs stream`,
> `deploy rollback` — work fully.

---

## Overview

Secrets (like `DATABASE_URL`) and environment variables (like pool tuning) are kept separate on purpose. Secrets are encrypted at rest and never appear in plain text in `deploy list` output. Env vars are visible and suitable for non-sensitive configuration. Both are injected into the container at runtime — the app sees them as ordinary environment variables.

---

## The Application

A Go API that reads `DATABASE_URL` from the environment:

```go
// main.go (excerpt)
dbURL := os.Getenv("DATABASE_URL")
if dbURL == "" {
    log.Fatal("DATABASE_URL is not set")
}
db, err := sql.Open("pgx", dbURL)
```

---

## Dockerfile

Multi-stage build targeting a small final image:

```dockerfile
# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Final stage
FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
```

---

## satusky.toml

```toml
name   = "go-api"
port   = 8080
cpu    = "0.5"
memory = "256Mi"
```

---

## Step 1: Deploy the App (It Will Crash — That Is Expected)

```bash
1ctl deploy --config satusky.toml --wait
```

The deploy will complete from the platform's perspective (image pushed, container started), but the app itself will crash immediately because `DATABASE_URL` is not set yet. This is fine — the next steps fix it.

---

## Step 2: See the Crash in Logs

```bash
1ctl logs stream --config satusky.toml
```

You will see something like:

```
2026-04-26T10:01:05Z  go-api  FATAL: DATABASE_URL is not set
2026-04-26T10:01:05Z  go-api  exit status 1
2026-04-26T10:01:06Z  go-api  container restarting...
```

This confirms the app code is running but missing its secret. Exactly what you expected.

---

## Step 3: Add the Database Secret

```bash
1ctl secret create --config satusky.toml \
  --kv DATABASE_URL=postgres://api-user:strongpassword@db.internal:5432/myapp?sslmode=require
```

Secrets are merged — you can run `secret create` again later to update or add other secrets without disturbing existing ones.

---

## Step 4: Add Connection Pool Env Vars

These are non-sensitive tuning values, so they go in env, not secrets:

```bash
1ctl env create --config satusky.toml \
  --env DB_MAX_CONNECTIONS=25 \
  --env DB_POOL_TIMEOUT=30s
```

---

## Step 5: Restart the Deployment

Secrets and env vars are picked up on the next start. Trigger a rolling restart (zero downtime):

```bash
1ctl deploy restart --config satusky.toml
```

The platform replaces containers one by one, routing traffic to healthy instances throughout.

---

## Step 6: Verify the App Is Healthy

Stream logs again to confirm a clean startup:

```bash
1ctl logs stream --config satusky.toml
```

Expected output:

```
2026-04-26T10:05:10Z  go-api  connected to database in 42ms
2026-04-26T10:05:10Z  go-api  server listening on :8080
```

Or check status in JSON for scripting:

```bash
1ctl -o json deploy status --config satusky.toml
```

```json
{
  "name": "go-api",
  "status": "running",
  "replicas": { "ready": 1, "desired": 1 },
  "last_deployed": "2026-04-26T10:05:00Z"
}
```

`"status": "running"` and `ready == desired` means the app is healthy.

---

## Step 7: Remove a Tuning Variable (Go Back to App Defaults)

If you decide to let the app manage its own pool timeout instead of reading from env:

```bash
1ctl env unset --config satusky.toml --key DB_POOL_TIMEOUT
```

Then restart to apply:

```bash
1ctl deploy restart --config satusky.toml
```

`env unset` removes exactly one key. The rest of your env vars (`DB_MAX_CONNECTIONS`, etc.) are untouched.

---

## Step 8: Rotate the Database Password

When you rotate credentials, update the secret in place and restart:

```bash
1ctl secret create --config satusky.toml \
  --kv DATABASE_URL=postgres://api-user:newstrongpassword@db.internal:5432/myapp?sslmode=require

1ctl deploy restart --config satusky.toml
```

`secret create` merges, so only `DATABASE_URL` is updated. No other secrets are affected.

---

## Tips

- Never put `DATABASE_URL` in `satusky.toml` or in your Dockerfile. TOML files are committed to source control; use `secret create` every time.
- Use `1ctl -o json deploy status` in a health-check script after every deploy to assert the app is actually running before sending traffic.
- If the rolling restart stalls (one replica never becomes healthy), `logs stream` will show the crash reason immediately — no need to guess.
- To remove a secret key entirely (not just update it), use `1ctl secret unset --config satusky.toml --key DATABASE_URL`. Use with care — the next restart will crash until you add the secret back.
