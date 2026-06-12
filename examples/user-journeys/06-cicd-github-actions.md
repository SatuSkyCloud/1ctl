# CI/CD with GitHub Actions

**Who this is for**: A DevOps engineer automating deployments from GitHub Actions.

**Goal**: On every push to `main`, deploy to production. On every push to `develop`, deploy to staging. Emergency rollback via workflow dispatch.

---

## CLI Coverage

> ⚠️ **Mostly covered** — the workflow logic is fully supported. One gap with the binary name.

> **Gap: no released `1ctl` binary for CI**
> The dev binary (`1ctl`) is not published to GitHub Releases — only the prod
> `1ctl` binary is (via `install.sh`). In CI, install the prod binary and point it at
> your backend using `SATUSKY_API_URL`:
> ```bash
> curl -sSL https://raw.githubusercontent.com/SatuSkyCloud/1ctl/main/install.sh | bash
> export SATUSKY_API_URL=https://api.satusky.com/v1/cli
> ```

---

## Overview

`1ctl` is a single static binary. In CI you install it in seconds, authenticate via an environment variable (`SATUSKY_API_KEY`), and use `-o json` to capture deployment metadata for downstream steps.

---

## Step 1: Store Your API Key

In your repository go to **Settings → Secrets and variables → Actions**:

| Secret name        | Value                  |
|--------------------|------------------------|
| `SATUSKY_API_KEY`  | your SatuSky API key   |

---

## Step 2: Install `1ctl` in CI

```yaml
- name: Install 1ctl
  run: |
    curl -fsSL https://get.satusky.com/1ctl/install.sh | sh
    echo "$HOME/.local/bin" >> $GITHUB_PATH
```

---

## Step 3: Full Workflow

```yaml
name: Deploy

on:
  push:
    branches: [main, develop]
  workflow_dispatch:
    inputs:
      environment:
        description: "Environment to roll back"
        required: true
        default: staging

env:
  SATUSKY_API_KEY: ${{ secrets.SATUSKY_API_KEY }}
  SATUSKY_API_URL: https://api.satusky.com/v1/cli

jobs:
  deploy-staging:
    if: github.ref == 'refs/heads/develop'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install 1ctl
        run: |
          curl -fsSL https://get.satusky.com/1ctl/install.sh | sh
          echo "$HOME/.local/bin" >> $GITHUB_PATH
      - name: Deploy to staging
        run: 1ctl deploy --config staging --wait
        timeout-minutes: 10
      - name: Capture staging URL
        run: |
          APP_URL=$(1ctl -o json deploy get --config staging | jq -r '.domain')
          echo "staging_url=$APP_URL" >> $GITHUB_OUTPUT

  deploy-production:
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install 1ctl
        run: |
          curl -fsSL https://get.satusky.com/1ctl/install.sh | sh
          echo "$HOME/.local/bin" >> $GITHUB_PATH
      - name: Deploy to production
        run: 1ctl deploy --config satusky.toml --wait
        timeout-minutes: 15
      - name: Capture production URL
        run: |
          APP_URL=$(1ctl -o json deploy get --config satusky.toml | jq -r '.domain')
          echo "app_url=$APP_URL" >> $GITHUB_OUTPUT

  rollback:
    if: github.event_name == 'workflow_dispatch'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install 1ctl
        run: |
          curl -fsSL https://get.satusky.com/1ctl/install.sh | sh
          echo "$HOME/.local/bin" >> $GITHUB_PATH
      - name: Roll back
        run: |
          ENV="${{ github.event.inputs.environment }}"
          if [ "$ENV" = "production" ]; then
            1ctl deploy rollback --config satusky.toml -y
          else
            1ctl deploy rollback --config staging -y
          fi
```

---

## Step 4: Capture the URL for Downstream Steps

```bash
1ctl deploy --config satusky.toml --wait
APP_URL=$(1ctl -o json deploy get --config satusky.toml | jq -r '.domain')
echo "Live at: $APP_URL"
```

The `domain` field contains the backend-assigned URL (e.g. `https://cleverpanda-a1b2c3d.satusky.com`).

---

## Step 5: Handling Timeouts

`--wait` polls until Running or fails. Set a step timeout:

```yaml
- name: Deploy to production
  run: 1ctl deploy --config satusky.toml --wait
  timeout-minutes: 15
```

If it times out, diagnose with:

```bash
1ctl logs stream --config satusky.toml
```

---

## Step 6: No Profile Needed in CI

Set two env vars and every command picks them up automatically:

```yaml
env:
  SATUSKY_API_KEY: ${{ secrets.SATUSKY_API_KEY }}
  SATUSKY_API_URL: https://api.satusky.com/v1/cli
```

No `login`, no profile creation needed.

---

## Tips

- Use `SATUSKY_API_URL` to point at different API endpoints per environment.
- Trigger rollback: **Actions → Deploy → Run workflow**, pick environment, click **Run**.
- Use `1ctl -o json deploy list` in a post-deploy step to assert deployments are `completed`.
- Pin the `1ctl` version in the install URL to avoid unexpected CI upgrades.

---

## Live Verification (2026-06-12)

CI/CD workflow commands verified against live instance. GitHub Actions YAML is declarative; individual CLI commands within it are all tested.

| # | Command | Exit |
|---|---------|------|
| 1 | `1ctl deploy --config staging --wait` (staged deploy) | ✅ 0 |
| 2 | `1ctl deploy --config satusky.toml --wait` (prod deploy) | ✅ 0 |
| 3 | `1ctl -o json deploy get --config satusky.toml \| jq -r '.domain'` | ✅ 0 |
| 4 | `1ctl -o json deploy list` | ✅ 0 |
| 5 | `1ctl deploy rollback --config satusky.toml --version N -y` | ✅ 0 |
| 6 | `1ctl deploy destroy --config satusky.toml -y` | ✅ 0 |
| 7 | `SATUSKY_API_KEY` + `SATUSKY_API_URL` env vars (no profile) | ✅ works |

**jq extraction verified**: `jq -r '.domain'` returns `https://quickpenguin-c7wwvsp.satusky.com`.

**Rollback flag verified**: `--yes/-y` exists for non-interactive CI. `--wait` does NOT exist on rollback.
