# Multi-Environment Deployments (Staging + Production)

**Who this is for**: A team lead setting up staging and production environments for a Node.js API.

**Goal**: Deploy the same codebase twice — staging uses lighter resources; production uses more. Each environment has its own database credentials.

---

## Overview

SatuSky uses named TOML files to represent environments. The default config file is `satusky.toml` (production). Passing `--config staging` resolves to `satusky.staging.toml` in the same directory.

---

## Step 1: Initialize Production Config

```bash
cd ~/projects/my-api
1ctl init
```

Edit `satusky.toml`:

```toml
[app]
  name   = "my-api"
  port   = 3000
  cpu    = "1"
  memory = "512Mi"
```

---

## Step 2: Initialize Staging Config

```bash
1ctl init --config staging
```

Edit `satusky.staging.toml`:

```toml
[app]
  name   = "my-api-staging"
  port   = 3000
  cpu    = "0.25"
  memory = "256Mi"
```

---

## Step 3: Set Environment Variables per Environment

```bash
# Staging — verbose logging
1ctl env create --config staging --env LOG_LEVEL=debug --env NODE_ENV=staging

# Production — errors only
1ctl env create --env LOG_LEVEL=error --env NODE_ENV=production
```

---

## Step 4: Set Database Secrets per Environment

Secrets are isolated per deployment name.

```bash
# Staging
1ctl secret create --config staging --kv DATABASE_URL=postgres://test:pass@staging-db.internal:5432/myapp_staging

# Production
1ctl secret create --kv DATABASE_URL=postgres://prod:secret@prod-db.internal:5432/myapp
```

---

## Step 5: Deploy Both

```bash
# Deploy staging first
1ctl deploy --config staging --wait

# Then production
1ctl deploy --config satusky.toml --wait
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
    "app_label": "my-api-staging",
    "status": "completed",
    "image": "registry.satusky.com/my-api:a1b2c3d"
  },
  {
    "deployment_id": "uuid-2",
    "app_label": "my-api",
    "status": "completed",
    "image": "registry.satusky.com/my-api:a1b2c3d"
  }
]
```

---

## Step 7: Promote Staging Image to Production

After QA passes, promote the exact image that was tested:

```bash
# Find the image tag that passed QA
1ctl deploy releases --config staging
```

```
VERSION  IMAGE                                 STATUS       DEPLOYED
r-003    registry.satusky.com/my-api:a1b2c3d    active       10 min ago
r-002    registry.satusky.com/my-api:9f8e7d6    superseded   1 day ago
```

Deploy the same tag to production:

```bash
1ctl deploy --config satusky.toml --image registry.satusky.com/my-api:a1b2c3d --wait
```

---

## Step 8: Stream Logs per Environment

```bash
1ctl logs stream --config staging
1ctl logs stream --config satusky.toml
```

---

## Tips

- Commit both TOML files — they contain no secrets.
- Use `1ctl deploy releases --config staging` before promotion to get the exact image tag.
- Add more environments with `satusky.qa.toml` and `--config qa`.

---

## Live Verification (2026-06-12)

Staged config resolution and resource isolation verified against live instance.

| # | Command | Exit |
|---|---------|------|
| 1 | `1ctl init` (creates `satusky.toml`) | ✅ 0 |
| 2 | `1ctl init --config staging` (creates `satusky.staging.toml`) | ✅ 0 |
| 3 | `1ctl env create --env KEY=VAL` (uses default satusky.toml) | ✅ 0 |
| 4 | `1ctl env create --config staging --env KEY=VAL` (staged config) | ✅ 0 |
| 5 | `1ctl secret create --kv KEY=VAL` | ✅ 0 |
| 6 | `1ctl deploy list` | ✅ 0 |
| 7 | `1ctl -o json deploy list` | ✅ 0 |
| 8 | `1ctl deploy releases --config satusky.toml` | ✅ 0 |
| 9 | `1ctl logs stream --config satusky.toml` | ✅ 0 |

**Staged config verified**: `--config staging` resolves to `satusky.staging.toml`. Secrets are scoped per deployment name — different `name` fields = separate secret stores.
