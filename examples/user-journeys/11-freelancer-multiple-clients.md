# User Journey 11: Freelancer Managing Multiple Client Projects

**Who this is for**: Freelance developers managing several client projects, each under a different SatuSky org with different credentials.

**Goal**: Set up named profiles so you can switch between clients cleanly, deploy to the right org every time, and never accidentally push client A's code to client B's account.

---

## CLI Coverage

> ✅ **Fully covered** — every command in this guide works with the current CLI.
> `profile create`, `profile use`, `--profile` one-shot flag, `org switch` with
> positional arg, and `SATUSKY_PROFILE` env var all work. No gaps.

---

## Overview

Each SatuSky profile stores a backend URL and auth token. Profiles are named so you can switch with a single command or reference them inline with `--profile`. Three profiles cover the common freelance setup: two clients on the SatuSky cloud and one local dev server for personal projects.

---

## Step 1: Create the Profiles

```bash
# Client A — a startup on SatuSky cloud
1ctl profile create --url https://api.satusky.com/v1/cli client-a

# Client B — an agency on SatuSky cloud
1ctl profile create --url https://api.satusky.com/v1/cli client-b

# Local — your own dev environment
1ctl profile create --url http://localhost:8081/v1/cli local
```

Each command prompts for credentials and stores them under the given name. The URL is the only required flag; the profile name is the positional argument.

List your profiles to confirm:

```bash
1ctl profile list
```

```
NAME       URL                                    ACTIVE
client-a   https://api.satusky.com/v1/cli         no
client-b   https://api.satusky.com/v1/cli         no
local      http://localhost:8081/v1/cli            yes
```

---

## Step 2: Project Layout

Each client project lives in its own directory with its own `satusky.toml`. The TOML contains no credentials — only the app shape.

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

`client-a-dashboard/satusky.toml`:

```toml
name   = "dashboard"
port   = 3000
cpu    = "0.5"
memory = "256Mi"
```

`client-b-shop/satusky.toml`:

```toml
name   = "shop-api"
port   = 8080
cpu    = "0.5"
memory = "512Mi"
```

---

## Step 3: Daily Workflow — Switch Profile, Deploy, Switch Back

**Morning: working on Client A**

```bash
1ctl profile use client-a
```

```
Switched to profile: client-a
```

```bash
cd ~/projects/client-a-dashboard
1ctl deploy --config satusky.toml --wait
```

```
Building image...  done (31s)
Deploy complete. App is live.
```

**Afternoon: switching to Client B**

```bash
1ctl profile use client-b
cd ~/projects/client-b-shop
1ctl deploy --config satusky.toml --wait
```

Client B's deployment goes to Client B's org. No cross-contamination.

---

## Step 4: One-Shot Deploy Without Permanently Switching

When you need to check on Client B while you're in the middle of Client A work, use `--profile` inline so the active profile doesn't change:

```bash
1ctl --profile client-b deploy status --config ~/projects/client-b-shop/satusky.toml
```

```
NAME       STATUS    IMAGE TAG   AGE
shop-api   running   b3c9f1a     2h
```

Your active profile stays as `client-a`. The `--profile` flag applies only to that single command.

---

## Step 5: Switch Org Within a Profile (Staging vs Prod)

Client A has two orgs on SatuSky: `client-a-staging` and `client-a-prod`. After switching to the client-a profile, pick the right org before deploying:

```bash
1ctl profile use client-a

# Deploy to staging org
1ctl org switch client-a-staging
1ctl deploy --config satusky.toml --wait

# Promote to prod org
1ctl org switch client-a-prod
1ctl deploy --config satusky.toml --wait
```

Check which org is currently active at any time:

```bash
1ctl org current
```

```
client-a-prod
```

---

## Step 6: Use SATUSKY_PROFILE in a Makefile

For projects where teammates also run deploys, encode the profile in the Makefile so the right profile is always used without relying on whoever ran `profile use` last:

```makefile
# client-b-shop/Makefile

deploy-staging:
	SATUSKY_PROFILE=client-b 1ctl org switch client-b-staging && \
	SATUSKY_PROFILE=client-b 1ctl deploy --config satusky.toml --wait

deploy-prod:
	SATUSKY_PROFILE=client-b 1ctl org switch client-b-prod && \
	SATUSKY_PROFILE=client-b 1ctl deploy --config satusky.toml --wait
```

`SATUSKY_PROFILE` overrides whatever profile is currently active for the duration of that shell command.

---

## Step 7: Quick Status Across All Clients

Get a machine-readable snapshot of every deployment per profile:

```bash
# Client A
1ctl --profile client-a -o json deploy list

# Client B
1ctl --profile client-b -o json deploy list
```

Pipe into `jq` for a condensed view:

```bash
1ctl --profile client-a -o json deploy list | jq '.[] | {name, status}'
```

```json
{"name": "dashboard", "status": "running"}
```

```bash
1ctl --profile client-b -o json deploy list | jq '.[] | {name, status}'
```

```json
{"name": "shop-api", "status": "running"}
```

---

## Step 8: Recover from a Deploy to the Wrong Client

You ran `1ctl deploy` while `client-b` was active, but the code was Client A's. The deploy list shows `dashboard` appearing in Client B's org.

First, check what got deployed where:

```bash
1ctl --profile client-b -o json deploy list | jq '.[].name'
```

```
"shop-api"
"dashboard"
```

`dashboard` should not be there. Destroy it from Client B:

```bash
1ctl --profile client-b deploy destroy \
  --config ~/projects/client-a-dashboard/satusky.toml -y
```

```
Destroying deployment dashboard...  done
```

Now deploy it to the correct profile:

```bash
1ctl profile use client-a
cd ~/projects/client-a-dashboard
1ctl deploy --config satusky.toml --wait
```

---

## Tips

- Run `1ctl profile list` at the start of a new terminal session to confirm which profile is active before deploying.
- Keep `satusky.toml` files committed to source control. They contain no credentials — the profile handles auth.
- Use `SATUSKY_PROFILE` in CI/CD pipelines instead of calling `profile use`, so parallel jobs can't stomp on each other's active profile.
- If a client has both a staging and prod org, consider naming profiles `client-a-staging` and `client-a-prod` rather than switching orgs manually.

---

## Summary

| Task | Command |
|---|---|
| Create a profile | `1ctl profile create --url https://api.satusky.com/v1/cli <name>` |
| Switch active profile | `1ctl profile use <name>` |
| List profiles | `1ctl profile list` |
| One-shot deploy to another profile | `1ctl --profile <name> deploy --config satusky.toml --wait` |
| Switch org within a profile | `1ctl org switch <org-name>` |
| Check current org | `1ctl org current` |
| Deploy list for a specific profile | `1ctl --profile <name> -o json deploy list` |
| Destroy a misplaced deploy | `1ctl --profile <name> deploy destroy --config satusky.toml -y` |
