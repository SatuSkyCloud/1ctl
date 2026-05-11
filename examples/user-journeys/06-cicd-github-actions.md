# User Journey 6: CI/CD with GitHub Actions

**Who this is for**: A DevOps engineer automating deployments from GitHub Actions.

**Goal**: On every push to `main`, build the image and deploy to production. On every push to `develop`, deploy to staging. Emergency rollback is a single click via workflow dispatch.

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
> No `profile create` is needed — `SATUSKY_API_URL` and the token are sufficient.
> Every other command in this guide (`deploy --wait`, `-o json`, `rollback`) works
> identically on the prod binary.

---

## Overview

`1ctl` is a single static binary. In CI you install it in seconds, authenticate via an environment variable (`SATUSKY_API_KEY`), and use `-o json` to capture deployment metadata for downstream steps. No profile setup, no interactive login.

---

## Step 1: Store Your API Key as a GitHub Secret

In your repository go to **Settings → Secrets and variables → Actions** and add:

| Secret name        | Value                          |
|--------------------|--------------------------------|
| `SATUSKY_API_KEY`  | your SatuSky API key           |

---

## Step 2: Install `1ctl` in CI

Add an install step before any deploy command. The binary is available via a install script:

```yaml
- name: Install 1ctl
  run: |
    curl -fsSL https://get.satusky.com/1ctl/install.sh | sh
    echo "$HOME/.local/bin" >> $GITHUB_PATH
```

Or, if you prefer pinning via Go toolchain:

```yaml
- name: Install 1ctl
  run: go install github.com/satusky/1ctl/cmd/1ctl@latest
```

---

## Step 3: The Full Workflow File

Create `.github/workflows/deploy.yml` in your repository:

```yaml
name: Deploy

on:
  push:
    branches:
      - main
      - develop
  workflow_dispatch:
    inputs:
      environment:
        description: "Environment to roll back (staging or production)"
        required: true
        default: staging

env:
  SATUSKY_API_KEY: ${{ secrets.SATUSKY_API_KEY }}
  SATUSKY_API_URL: http://localhost:8080/v1/cli   # swap for https://api.satusky.com/v1/cli in real usage

jobs:
  deploy-staging:
    name: Deploy to Staging
    if: github.ref == 'refs/heads/develop'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install 1ctl
        run: |
          curl -fsSL https://get.satusky.com/1ctl/install.sh | sh
          echo "$HOME/.local/bin" >> $GITHUB_PATH

      - name: Deploy to staging
        id: deploy
        run: |
          1ctl deploy --config staging --wait
        timeout-minutes: 10

      - name: Capture staging info
        run: |
          APP_URL=$(1ctl -o json deploy get --config satusky.staging.toml | jq -r '.domain')
          echo "Staging URL: $APP_URL"
          echo "staging_url=$APP_URL" >> $GITHUB_OUTPUT

  deploy-production:
    name: Deploy to Production
    if: github.ref == 'refs/heads/main'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install 1ctl
        run: |
          curl -fsSL https://get.satusky.com/1ctl/install.sh | sh
          echo "$HOME/.local/bin" >> $GITHUB_PATH

      - name: Deploy to production
        id: deploy
        run: |
          1ctl deploy --config satusky.toml --wait
        timeout-minutes: 15

      - name: Capture production info
        run: |
          APP_URL=$(1ctl -o json deploy get --config satusky.toml | jq -r '.domain')
          echo "Production URL: $APP_URL"
          echo "app_url=$APP_URL" >> $GITHUB_OUTPUT

  rollback:
    name: Emergency Rollback
    if: github.event_name == 'workflow_dispatch'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install 1ctl
        run: |
          curl -fsSL https://get.satusky.com/1ctl/install.sh | sh
          echo "$HOME/.local/bin" >> $GITHUB_PATH

      - name: Roll back environment
        run: |
          ENV="${{ github.event.inputs.environment }}"
          if [ "$ENV" = "production" ]; then
            1ctl deploy rollback --config satusky.toml --wait
          else
            1ctl deploy rollback --config staging --wait
          fi
```

---

## Step 4: Capture the Deployment URL for Downstream Steps

After `deploy --wait` completes, use `deploy get -o json` to get the URL. The `domain` field contains the backend-assigned URL (e.g. `https://cleverpanda-a1b2c3d.satusky.com`):

```bash
1ctl deploy --config satusky.toml --wait
APP_URL=$(1ctl -o json deploy get --config satusky.toml | jq -r '.domain')
echo "Live at: $APP_URL"
```

You can pass `APP_URL` to a smoke-test step, a Slack notification, or a GitHub deployment status update.

---

## Step 5: Handling Deployment Timeout in CI

`--wait` polls until the deployment is Running or fails. GitHub Actions has its own job timeout. If the deployment takes longer than expected, the step exits non-zero and the job fails — which is the correct behavior.

To set an explicit timeout on the step itself (independent of the job-level timeout):

```yaml
- name: Deploy to production
  run: 1ctl deploy --config satusky.toml --wait
  timeout-minutes: 15
```

If the step times out, diagnose with:

```bash
# Run locally or in a debug job
1ctl logs stream --config satusky.toml
```

Common causes: image pull taking too long, health check failing, or a missing secret causing the app to crash on startup.

---

## Step 6: No Profile Needed in CI

On a developer machine, `1ctl` reads credentials from `~/.config/satusky/config`. In CI, skip that entirely — just set two environment variables:

```yaml
env:
  SATUSKY_API_KEY: ${{ secrets.SATUSKY_API_KEY }}
  SATUSKY_API_URL: https://api.satusky.com/v1/cli
```

Every `1ctl` command in the same job picks these up automatically. No `login`, no profile creation.

---

## Tips

- Use `SATUSKY_API_URL` to point at different API endpoints per environment without changing any command flags.
- Trigger rollback manually: go to **Actions → Deploy → Run workflow**, pick the environment, and click **Run**. The `deploy rollback` command swaps to the previous release instantly.
- Use `1ctl -o json deploy list` in a post-deploy step to assert both staging and production deployments are in `running` state before marking the workflow green.
- Pin the `1ctl` version in the install URL to avoid unexpected behavior from automatic upgrades in CI.
