# Deploying Microservices with Inter-Service Communication

**Who this is for**: An architect deploying two services — `user-service` (authentication) and `order-service` (business logic). The order-service calls the user-service.

**Goal**: Deploy both services independently, wire up inter-service communication via env vars, and manage secrets per service.

---

## CLI Coverage

> ⚠️ **Mostly covered** — one gap with JSON output on `ingress list`.

> **Gap: `-o json ingress list` is not yet wired**
> `--output json` works on `deploy list/get/status`, `env list`, `secret list`,
> `machine list`, and `token list` — but **not** on `ingress list`.
>
> **Domain is backend-assigned** (e.g. `cleverpanda-a1b2c3d.satusky.com`). Read it
> from `deploy get -o json`:
> ```bash
> USER_SERVICE_URL=$(1ctl -o json deploy get \
>   --config services/user-service/satusky.toml | jq -r '.domain')
> ```

---

## Project Layout

```
services/
  user-service/
    satusky.toml
    Dockerfile
    main.go
  order-service/
    satusky.toml
    Dockerfile
    main.go
```

---

## TOML Files

**`services/user-service/satusky.toml`**

```toml
[app]
  name   = "user-service"
  port   = 8081
  cpu    = "0.5"
  memory = "256Mi"
```

**`services/order-service/satusky.toml`**

```toml
[app]
  name   = "order-service"
  port   = 8082
  cpu    = "0.5"
  memory = "256Mi"
```

---

## Step 1: Deploy user-service First

```bash
1ctl deploy --config services/user-service/satusky.toml --wait
```

---

## Step 2: Capture the user-service URL

```bash
USER_SERVICE_URL=$(1ctl -o json deploy get \
  --config services/user-service/satusky.toml | jq -r '.domain')
echo "$USER_SERVICE_URL"
# https://cleverpanda-a1b2c3d.satusky.com
```

---

## Step 3: Wire URL into order-service

```bash
1ctl env create \
  --config services/order-service/satusky.toml \
  --env USER_SERVICE_URL="$USER_SERVICE_URL"
```

---

## Step 4: Set Secrets per Service

```bash
# user-service JWT secret
1ctl secret create \
  --config services/user-service/satusky.toml \
  --kv JWT_SECRET=supersecretjwtkey123

# order-service database
1ctl secret create \
  --config services/order-service/satusky.toml \
  --kv DATABASE_URL=postgres://orders:pass@db.internal:5432/orders
```

Secrets are scoped per deployment name — no cross-contamination.

---

## Step 5: Deploy order-service

```bash
1ctl deploy --config services/order-service/satusky.toml --wait
```

---

## Step 6: Verify Both

```bash
1ctl -o json deploy list
```

```json
[
  {
    "deployment_id": "uuid-1",
    "app_label": "user-service",
    "status": "completed",
    "image": "registry.satusky.com/user-service:d4e5f6a"
  },
  {
    "deployment_id": "uuid-2",
    "app_label": "order-service",
    "status": "completed",
    "image": "registry.satusky.com/order-service:b7c8d9e"
  }
]
```

---

## Step 7: Update user-service Independently

```bash
1ctl deploy --config services/user-service/satusky.toml --wait
```

order-service keeps running. The domain does not change across redeploys of the same deployment.

---

## Step 8: Stream Logs per Service

```bash
1ctl logs stream --config services/user-service/satusky.toml
1ctl logs stream --config services/order-service/satusky.toml
```

---

## Step 9: Update the Service URL

```bash
1ctl env create \
  --config services/order-service/satusky.toml \
  --env USER_SERVICE_URL=https://new-domain.satusky.com

1ctl deploy restart --config services/order-service/satusky.toml
```

Or remove it entirely:

```bash
1ctl env unset --config services/order-service/satusky.toml --key USER_SERVICE_URL
1ctl deploy restart --config services/order-service/satusky.toml
```

---

## Tips

- Deploy dependencies (`--wait`) before dependents.
- Use `1ctl -o json deploy list | jq '[.[] | {app_label, status}]'` for a quick health dashboard.
- Secrets are scoped per deployment name, not TOML path.

---

## Live Verification (2026-06-12)

Multi-service deployment and inter-service env wiring verified against live instance.

| # | Command | Exit |
|---|---------|------|
| 1 | `1ctl deploy list` (multiple apps in namespace) | ✅ 0 |
| 2 | `1ctl -o json deploy list \| jq '[.[] \| {app_label, status}]'` | ✅ 0 |
| 3 | `1ctl -o json deploy get --deployment-id <id> \| jq -r '.domain'` | ✅ 0 |
| 4 | `1ctl env create --deployment-id <id> --env SERVICE_URL=...` | ✅ 0 |
| 5 | `1ctl env unset --deployment-id <id> --key SERVICE_URL` | ✅ 0 |
| 6 | `1ctl deploy restart --deployment-id <id>` | ✅ 0 |
| 7 | `1ctl secret create --deployment-id <id> --kv KEY=VAL` | ✅ 0 |
| 8 | `1ctl logs stream --deployment-id <id>` | ✅ 0 |
| 9 | `1ctl ingress list` | ✅ 0 |

**Multi-deployment verified**: `deploy list` returns `backend-api` + `frontend` with separate `deployment_id` values.

**Domain stability verified**: Same deployment ID retains same domain across redeploys.

**Ingress `-o json` not wired**: `1ctl -o json ingress list` falls back to table output.
