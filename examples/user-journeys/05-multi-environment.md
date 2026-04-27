# User Journey 5: Multi-Environment Deployments (Staging + Production)

**Who this is for**: A team lead setting up staging and production environments for a Node.js API.

**Goal**: Deploy the same codebase twice — staging uses lighter resources and verbose logging; production uses more resources and error-only logging. Each environment has its own database credentials.

---

## Overview

SatuSky uses named TOML files to represent environments. The default config file is `satusky.toml` (production). Passing `--config staging` resolves to `satusky.staging.toml` in the same directory. You never touch a shared file — each environment is fully independent.

---

## Step 1: Initialize the Production Config

Start in your project root:

```bash
cd ~/projects/my-api
1ctl-dev init
```

This creates `satusky.toml`. Edit it for production:

```toml
# satusky.toml  (production)
name    = "my-api"
port    = 3000
cpu     = "1"
memory  = "512Mi"
```

---

## Step 2: Initialize the Staging Config

```bash
1ctl-dev init --config staging
```

This creates `satusky.staging.toml` in the same directory. Edit it for staging:

```toml
# satusky.staging.toml  (staging)
name    = "my-api-staging"
port    = 3000
cpu     = "0.25"
memory  = "256Mi"
```

Both files live side by side:

```
my-api/
  satusky.toml           ← production
  satusky.staging.toml   ← staging
  Dockerfile
  src/
```

---

## Step 3: Set Environment Variables per Environment

Staging gets verbose logging; production gets errors only:

```bash
# Staging
1ctl-dev env create --config staging --env LOG_LEVEL=debug --env NODE_ENV=staging

# Production
1ctl-dev env create --env LOG_LEVEL=error --env NODE_ENV=production
```

---

## Step 4: Set Database Secrets per Environment

Secrets are isolated per deployment name. Because staging is named `my-api-staging` and production is `my-api`, each gets its own secret store.

```bash
# Staging — points at a test database
1ctl-dev secret create --config staging --kv DATABASE_URL=postgres://test-user:test-pass@staging-db.internal:5432/myapp_staging

# Production — points at the real database
1ctl-dev secret create --kv DATABASE_URL=postgres://prod-user:secret-pass@prod-db.internal:5432/myapp
```

---

## Step 5: Deploy Both Environments

Deploy staging first and confirm it works before touching production.

```bash
# Deploy staging, wait until it is Running
1ctl-dev deploy --config staging --wait

# Deploy production
1ctl-dev deploy --config satusky.toml --wait
```

The `--wait` flag blocks until the deployment reaches a Running (healthy) state, so your terminal gives you a clear green light before you move on.

---

## Step 6: Verify Both Deployments

List all deployments and check they both appear:

```bash
1ctl-dev -o json deploy list
```

Example output:

```json
[
  {
    "name": "my-api-staging",
    "status": "running",
    "image": "registry.satusky.com/my-api-staging:a1b2c3d",
    "created_at": "2026-04-26T10:00:00Z"
  },
  {
    "name": "my-api",
    "status": "running",
    "image": "registry.satusky.com/my-api:a1b2c3d",
    "created_at": "2026-04-26T10:05:00Z"
  }
]
```

Both deployments are identified by the `name` field from their respective TOML files.

---

## Step 7: Promote a Staging Image to Production

After QA passes on staging, promote the exact image that was tested rather than building a new one.

First, find the image tag that passed QA:

```bash
1ctl-dev deploy releases --config staging
```

Example output:

```
RELEASE   IMAGE TAG   DEPLOYED AT              STATUS
r-003     a1b2c3d     2026-04-26T09:50:00Z     active
r-002     9f8e7d6     2026-04-25T14:20:00Z     superseded
r-001     5c4b3a2     2026-04-24T08:10:00Z     superseded
```

Copy the image tag (`a1b2c3d`) and deploy it to production:

```bash
1ctl-dev deploy --config satusky.toml --image registry.satusky.com/my-api:a1b2c3d --wait
```

Production now runs the identical artifact that passed staging QA — no rebuild, no drift.

---

## Step 8: Stream Logs per Environment

To tail staging logs in real time:

```bash
1ctl-dev logs stream --config satusky.staging.toml
```

To tail production logs:

```bash
1ctl-dev logs stream --config satusky.toml
```

---

## Tips

- Keep both TOML files committed to source control. They contain no secrets — only shape (CPU, memory, port, name).
- Run `1ctl-dev deploy releases --config staging` before every production promotion to get the exact tag. Never guess.
- If staging and production ever diverge in environment variables, run `1ctl-dev -o json deploy list` and compare the `env` fields to spot differences.
- You can add more environments (e.g., QA) by creating `satusky.qa.toml` and using `--config qa`.
