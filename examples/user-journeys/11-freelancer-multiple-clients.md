# Freelancer Managing Multiple Client Projects

**Who this is for**: Freelance developers managing several client projects, each under a different SatuSky org.

**Goal**: Set up named profiles so you can switch between clients cleanly and never accidentally push client A's code to client B's account.

---

## CLI Coverage

> ✅ **Fully covered** — `profile create`, `profile use`, `--profile` one-shot flag,
> `org switch` with positional arg, and `SATUSKY_PROFILE` env var all work.

---

## Overview

Each profile stores a backend URL and auth token. Profiles are named so you can switch with a single command or reference them inline with `--profile`.

---

## Step 1: Create Profiles

```bash
# Client A
1ctl profile create --url https://api.satusky.com/v1/cli client-a

# Client B
1ctl profile create --url https://api.satusky.com/v1/cli client-b

# Local dev
1ctl profile create --url http://localhost:8080/v1/cli local
```

Confirm:

```bash
1ctl profile list
```

```
Profiles
────────
* local
  API URL: http://localhost:8080/v1/cli
  Auth: mingerz.k@gmail.com
  Org: org123
---
  client-a
  API URL: https://api.satusky.com/v1/cli
  Auth: client-a@example.com
  Org: client-a
---
  client-b
  API URL: https://api.satusky.com/v1/cli
  Auth: client-b@example.com
  Org: client-b
```

---

## Step 2: Project Layout

```
~/projects/
├── client-a-dashboard/
│   ├── satusky.toml   ← name = "dashboard"
│   └── Dockerfile
├── client-b-shop/
│   ├── satusky.toml   ← name = "shop-api"
│   └── Dockerfile
└── my-personal-tool/
    ├── satusky.toml   ← name = "dev-tool"
    └── Dockerfile
```

```toml
[app]
  name   = "dashboard"
  port   = 3000
  cpu    = "0.5"
  memory = "256Mi"
```

---

## Step 3: Daily Workflow

**Morning — Client A:**

```bash
1ctl profile use client-a
cd ~/projects/client-a-dashboard
1ctl deploy --config satusky.toml --wait
```

**Afternoon — Client B:**

```bash
1ctl profile use client-b
cd ~/projects/client-b-shop
1ctl deploy --config satusky.toml --wait
```

---

## Step 4: One-Shot Deploy (No Permanent Switch)

```bash
# Check Client B's status while staying on Client A
1ctl --profile client-b deploy status \
  --config ~/projects/client-b-shop/satusky.toml
```

The active profile stays as `client-a`.

---

## Step 5: Switch Org Within a Profile

```bash
1ctl profile use client-a

# Deploy to staging org
1ctl org switch client-a-staging
1ctl deploy --config satusky.toml --wait

# Promote to prod org
1ctl org switch client-a-prod
1ctl deploy --config satusky.toml --wait
```

---

## Step 6: Use SATUSKY_PROFILE in a Makefile

```makefile
deploy-staging:
	SATUSKY_PROFILE=client-b 1ctl org switch client-b-staging && \
	SATUSKY_PROFILE=client-b 1ctl deploy --config satusky.toml --wait

deploy-prod:
	SATUSKY_PROFILE=client-b 1ctl org switch client-b-prod && \
	SATUSKY_PROFILE=client-b 1ctl deploy --config satusky.toml --wait
```

---

## Step 7: Quick Status Across All Clients

```bash
1ctl --profile client-a -o json deploy list | jq '.[] | {app_label, status}'
# {"app_label": "dashboard", "status": "completed"}

1ctl --profile client-b -o json deploy list | jq '.[] | {app_label, status}'
# {"app_label": "shop-api", "status": "completed"}
```

---

## Step 8: Recover from Wrong-Client Deploy

```bash
# Destroy from the wrong profile
1ctl --profile client-b deploy destroy \
  --config ~/projects/client-a-dashboard/satusky.toml -y

# Deploy to the correct profile
1ctl profile use client-a
cd ~/projects/client-a-dashboard
1ctl deploy --config satusky.toml --wait
```

---

## Tips

- Run `1ctl profile list` at the start of a new terminal session.
- `satusky.toml` files contain no credentials — the profile handles auth.
- Use `SATUSKY_PROFILE` in CI so parallel jobs can't stomp on each other.

---

## Live Verification (2026-06-12)

Multi-profile and org-switching workflow verified against live instance.

| # | Command | Exit |
|---|---------|------|
| 1 | `1ctl profile list` | ✅ 0 |
| 2 | `1ctl profile use local` | ✅ 0 |
| 3 | `1ctl --profile local deploy list` (one-shot) | ✅ 0 |
| 4 | `1ctl org current` | ✅ 0 |
| 5 | `1ctl org switch --help` (positional + flags) | ✅ 0 |
| 6 | `1ctl -o json deploy list` | ✅ 0 |
| 7 | `1ctl --profile local -o json deploy list` | ✅ 0 |
| 8 | `SATUSKY_PROFILE=local 1ctl deploy list` (env var) | ✅ 0 |

**Profile list output**: Shows `*` next to active profile, URL, auth email, org.
**One-shot flag verified**: `--profile X` applies to single command without changing active.
**Destroy across profiles**: `deploy destroy --config <path> -y` works with `--profile` override.
