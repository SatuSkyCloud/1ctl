# Deploying an API with a PostgreSQL Database

**Who this is for**: A backend developer deploying a Go (or Python) API that reads its database connection string from the environment.

**Goal**: Deploy an API, wire up `DATABASE_URL` as a secret, tune connection pooling via env vars, and verify everything is healthy.

---

## CLI Coverage

> ⚠️ **Mostly covered** — all database connection commands work. One gap.

> **Gap: `--postgres` addon flag does not exist**
> The guide mentions `--postgres` as an optional built-in addon. That flag does not
> exist in the current CLI. **Workaround:** use an external managed database
> (Neon, Supabase, PlanetScale, etc.) and store the full `DATABASE_URL` as a secret:
> ```bash
> 1ctl secret create --config satusky.toml \
>   --kv DATABASE_URL=postgres://user:pass@host/db?sslmode=require
> ```

---

## Overview

Secrets (like `DATABASE_URL`) and env vars (like pool tuning) are kept separate. Secrets are encrypted at rest; env vars are visible in deploy metadata.

---

## The Application

```go
dbURL := os.Getenv("DATABASE_URL")
if dbURL == "" { log.Fatal("DATABASE_URL is not set") }
db, err := sql.Open("pgx", dbURL)
```

---

## Dockerfile

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
```

---

## satusky.toml

```toml
[app]
  name   = "go-api"
  port   = 8080
  cpu    = "0.5"
  memory = "256Mi"
```

---

## Step 1: Deploy (It Will Crash — Expected)

```bash
1ctl deploy --config satusky.toml --wait
```

The app crashes because `DATABASE_URL` is not set. This is fine — the next steps fix it.

---

## Step 2: See the Crash in Logs

```bash
1ctl logs stream --config satusky.toml
```

```
2026-06-12T10:01:05Z [go-api] FATAL: DATABASE_URL is not set
```

---

## Step 3: Add the Database Secret

```bash
1ctl secret create --config satusky.toml \
  --kv DATABASE_URL=postgres://api-user:strongpassword@db.internal:5432/myapp?sslmode=require
```

---

## Step 4: Add Connection Pool Env Vars

```bash
1ctl env create --config satusky.toml \
  --env DB_MAX_CONNECTIONS=25 \
  --env DB_POOL_TIMEOUT=30s
```

---

## Step 5: Restart to Apply

```bash
1ctl deploy restart --config satusky.toml
```

Platform replaces containers one by one, routing traffic to healthy instances throughout.

---

## Step 6: Verify

```bash
1ctl logs stream --config satusky.toml
# [go-api] connected to database in 42ms
# [go-api] server listening on :8080
```

Or check status:

```bash
1ctl deploy status --config satusky.toml
```

```
Workload: Running
Message: Deployment is running normally
Progress: 100%
```

---

## Step 7: Remove a Tuning Variable

```bash
1ctl env unset --config satusky.toml --key DB_POOL_TIMEOUT
1ctl deploy restart --config satusky.toml
```

---

## Step 8: Rotate the Database Password

```bash
1ctl secret create --config satusky.toml \
  --kv DATABASE_URL=postgres://api-user:newpassword@db.internal:5432/myapp?sslmode=require

1ctl deploy restart --config satusky.toml
```

---

## Tips

- Never put `DATABASE_URL` in `satusky.toml` — use `secret create`.
- Use `1ctl deploy status` after every deploy to confirm the app is healthy.
- To remove a secret entirely: `1ctl secret unset --config satusky.toml --key DATABASE_URL`.

---

## Live Verification (2026-06-12)

Secret injection and restart workflow verified against live `backend-api` deployment.

| # | Command | Exit |
|---|---------|------|
| 1 | `1ctl secret create --deployment-id <id> --kv DATABASE_URL=postgres://...` | ✅ 0 |
| 2 | `1ctl secret list` | ✅ 0 |
| 3 | `1ctl secret unset --deployment-id <id> --key DATABASE_URL` | ✅ 0 |
| 4 | `1ctl env create --deployment-id <id> --env DB_MAX_CONNECTIONS=25` | ✅ 0 |
| 5 | `1ctl env unset --deployment-id <id> --key DB_POOL_TIMEOUT` | ✅ 0 |
| 6 | `1ctl deploy restart --deployment-id <id>` | ✅ 0 |
| 7 | `1ctl logs --deployment-id <id> --tail 3` | ✅ 0 |
| 8 | `1ctl deploy status --deployment-id <id>` | ✅ 0 |

**Restart output**: `💡 Initiating rolling restart... ✅ Rolling restart initiated.`

**Secret scoping verified**: `secret list` (no `--config` flag) returns secrets globally. `secret create/unset` accept `--deployment-id`.
