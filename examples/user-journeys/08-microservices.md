# User Journey 8: Deploying Microservices with Inter-Service Communication

**Who this is for**: An architect deploying two services — `user-service` (authentication) and `order-service` (business logic). The order-service calls the user-service's internal URL.

**Goal**: Deploy both services independently, wire up inter-service communication via env vars, and manage secrets per service.

---

## Overview

Each service is its own deployable unit with its own `satusky.toml`. They are deployed separately, scaled separately, and updated independently. The only coupling is a URL that order-service reads from its environment — which you set after user-service is live.

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
name   = "user-service"
port   = 8081
cpu    = "0.5"
memory = "256Mi"
```

**`services/order-service/satusky.toml`**

```toml
name   = "order-service"
port   = 8082
cpu    = "0.5"
memory = "256Mi"
```

---

## Step 1: Deploy user-service First

Always deploy the dependency before the dependent service.

```bash
1ctl-dev deploy --config services/user-service/satusky.toml --wait
```

`--wait` blocks until user-service is Running. Only then will you have a stable URL to hand to order-service.

---

## Step 2: Find the user-service Ingress URL

Use `-o json` to extract the URL programmatically rather than eyeballing the output:

```bash
1ctl-dev -o json ingress list | jq '.[] | select(.service == "user-service")'
```

Example output:

```json
{
  "service": "user-service",
  "host": "user-service.satusky.com",
  "tls": true,
  "port": 443
}
```

The public (and internal) URL is `https://user-service.satusky.com`.

You can extract just the URL in one line for scripting:

```bash
USER_SERVICE_URL=$(1ctl-dev -o json ingress list \
  | jq -r '.[] | select(.service == "user-service") | "https://" + .host')
echo "$USER_SERVICE_URL"
# https://user-service.satusky.com
```

---

## Step 3: Wire the URL into order-service

Set the URL as an env var in order-service. Use the variable you captured above:

```bash
1ctl-dev env create \
  --config services/order-service/satusky.toml \
  --env USER_SERVICE_URL=https://user-service.satusky.com
```

Or, using the shell variable:

```bash
1ctl-dev env create \
  --config services/order-service/satusky.toml \
  --env USER_SERVICE_URL="$USER_SERVICE_URL"
```

---

## Step 4: Set Secrets per Service Independently

Each service has its own secret store keyed by its deployment name.

```bash
# user-service needs a JWT signing secret
1ctl-dev secret create \
  --config services/user-service/satusky.toml \
  --kv JWT_SECRET=supersecretjwtkey123

# order-service needs its own database
1ctl-dev secret create \
  --config services/order-service/satusky.toml \
  --kv DATABASE_URL=postgres://orders-user:pass@db.internal:5432/orders
```

There is no shared secret namespace between services — setting a secret for user-service has zero effect on order-service and vice versa.

---

## Step 5: Deploy order-service

```bash
1ctl-dev deploy --config services/order-service/satusky.toml --wait
```

---

## Step 6: Verify Both Services Are Running

```bash
1ctl-dev -o json deploy list
```

Example output:

```json
[
  {
    "name": "user-service",
    "status": "running",
    "image": "registry.satusky.com/user-service:d4e5f6a",
    "created_at": "2026-04-26T11:00:00Z"
  },
  {
    "name": "order-service",
    "status": "running",
    "image": "registry.satusky.com/order-service:b7c8d9e",
    "created_at": "2026-04-26T11:05:00Z"
  }
]
```

Both services appear by their `name` field from their respective TOML files.

---

## Step 7: Update user-service Without Touching order-service

This is the core benefit of independent deployments. Push a new version of user-service:

```bash
1ctl-dev deploy --config services/user-service/satusky.toml --wait
```

order-service keeps running uninterrupted. The URL stays the same — `user-service.satusky.com` — so no reconfiguration of order-service is needed.

---

## Step 8: Stream Logs per Service

Debug user-service independently:

```bash
1ctl-dev logs stream --config services/user-service/satusky.toml
```

Debug order-service independently:

```bash
1ctl-dev logs stream --config services/order-service/satusky.toml
```

---

## Step 9: Unset an Env Var When It Changes

If user-service moves to a new domain, update order-service:

```bash
# Overwrite the existing value (env create merges)
1ctl-dev env create \
  --config services/order-service/satusky.toml \
  --env USER_SERVICE_URL=https://auth.satusky.com

1ctl-dev deploy restart --config services/order-service/satusky.toml
```

Or remove the variable entirely and let order-service fall back to its own default:

```bash
1ctl-dev env unset \
  --config services/order-service/satusky.toml \
  --key USER_SERVICE_URL

1ctl-dev deploy restart --config services/order-service/satusky.toml
```

---

## Tips

- Always deploy dependencies (`--wait`) before dependents. A crashed order-service on first boot — because user-service wasn't ready — is a common mistake.
- Use `1ctl-dev -o json ingress list` in a deploy script to automatically extract and inject URLs rather than hardcoding them.
- Keep each service's `satusky.toml` in its own subdirectory so `--config` paths are unambiguous: `--config services/user-service/satusky.toml`.
- Use `1ctl-dev -o json deploy list` as a quick health dashboard for your entire mesh — pipe it through `jq '[.[] | {name, status}]'` to get a clean status table.
- Secret keys are scoped per deployment name, not per TOML file path. If you ever rename a service, create its secrets fresh — the old store does not carry over.
