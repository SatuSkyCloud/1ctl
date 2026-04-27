# User Journey 12: Troubleshooting Common Deployment Problems

**Who this is for**: Any developer hitting a wall after deploying. This is a reference guide for the six most common problems and exactly how to diagnose and fix each one.

**Goal**: Get from "something is wrong" to "fixed and running" as fast as possible using the CLI alone.

---

## Quick Reference

| Symptom | Problem | Jump to |
|---|---|---|
| Pods restarting, never healthy | CrashLoopBackOff | Problem 1 |
| Deploy succeeded but app returns 502/504 | Wrong bind address | Problem 2 |
| Build fails before deploy | Dockerfile error | Problem 3 |
| App can't read env vars or secrets | Config not reloaded | Problem 4 |
| Wrong app got updated | Wrong `--config` path | Problem 5 |
| `app "x" not found` error | App missing or wrong context | Problem 6 |

---

## Problem 1: Pod Keeps Restarting (CrashLoopBackOff)

### How to spot it

```bash
1ctl-dev deploy status --config satusky.toml
```

```
NAME     STATUS     READY   RESTARTS   AGE
my-api   Pending    0/1     4          3m
```

`READY` is `0/1` and `RESTARTS` keeps climbing. The pod starts, crashes, and gets restarted by the platform in a loop.

### Diagnose

Stream the logs to see what the process prints before it dies:

```bash
1ctl-dev logs stream --config satusky.toml
```

```
2026-04-26T14:01:12Z [my-api] Starting server...
2026-04-26T14:01:12Z [my-api] panic: DATABASE_URL environment variable not set
2026-04-26T14:01:12Z [my-api] goroutine 1 [running]:
2026-04-26T14:01:12Z [my-api] main.main()
2026-04-26T14:01:13Z [my-api] Starting server...   ← pod restarted
```

The crash message tells you exactly what's wrong. Common causes:

- A required env var or secret is missing.
- The app tries to bind a port that is already in use or is `0` (misconfigured).
- The app panics during initialization (database connection fails, config file missing, etc.).

### Fix

Add the missing variable. If it's a credential, use a secret:

```bash
1ctl-dev secret create --config satusky.toml --kv DATABASE_URL=postgres://user:pass@host/db
```

If it's non-sensitive config, use an env var:

```bash
1ctl-dev env create --config satusky.toml --env DATABASE_URL=postgres://user:pass@host/db
```

Redeploy to pick up the new value:

```bash
1ctl-dev deploy --config satusky.toml --wait
```

Stream logs again to confirm a clean startup:

```bash
1ctl-dev logs stream --config satusky.toml
```

```
2026-04-26T14:06:01Z [my-api] Starting server...
2026-04-26T14:06:01Z [my-api] Connected to database
2026-04-26T14:06:01Z [my-api] Listening on 0.0.0.0:8080
```

---

## Problem 2: Deployed but Can't Reach the App (502/504)

### How to spot it

The deploy succeeded and status shows `Running`, but every HTTP request returns a gateway error:

```bash
curl https://my-api.satusky.com/health
# curl: (22) The requested URL returned error: 502
```

### Diagnose

The app is running but the platform's load balancer can't reach it. Stream the logs to see what address the app bound to:

```bash
1ctl-dev logs stream --config satusky.toml
```

```
2026-04-26T14:10:05Z [my-api] Listening on 127.0.0.1:8080
```

There it is. The app is listening on loopback (`127.0.0.1`) which is only reachable from inside the same container. The load balancer hits the pod's network interface, not loopback, so every request times out.

### Fix

Update your app to bind on `0.0.0.0` (all interfaces):

```
# Before
app.listen(8080, '127.0.0.1')

# After
app.listen(8080, '0.0.0.0')
```

For frameworks that default to loopback in production mode, the fix is usually an environment variable:

```bash
1ctl-dev env create --config satusky.toml --env HOST=0.0.0.0
```

Redeploy:

```bash
1ctl-dev deploy --config satusky.toml --wait
```

Confirm the bind address in the logs:

```bash
1ctl-dev logs stream --config satusky.toml
```

```
2026-04-26T14:15:00Z [my-api] Listening on 0.0.0.0:8080
```

```bash
curl https://my-api.satusky.com/health
# {"status":"ok"}
```

---

## Problem 3: Cloud Build Fails

### How to spot it

The deploy command streams build output and then stops with a non-zero exit:

```bash
1ctl-dev deploy --config satusky.toml --wait
```

```
Building image...
  Step 1/6 : FROM python:3.12-slim
  Step 2/6 : COPY requirements.txt .
  Step 3/6 : RUN pip install -r requirements.txt
  ERROR: Could not find a version that satisfies the requirement torch==99.0.0
  The command '/bin/sh -c pip install -r requirements.txt' returned a non-zero code: 1
Build failed.
```

### Diagnose

Read the streamed output. The error appears inline — there is no separate log to fetch. Common causes:

- A package version in `requirements.txt` or `package.json` does not exist.
- A `COPY` path is wrong (file doesn't exist in the build context, or `.dockerignore` excluded it).
- A `RUN` command references a binary that isn't installed in the base image.
- A multi-stage `COPY --from` references a stage name that was renamed.

### Fix

Fix the Dockerfile or dependency file. In this case, use the correct torch version:

```
# requirements.txt before
torch==99.0.0

# requirements.txt after
torch==2.3.0
```

Re-deploy once the fix is in place:

```bash
1ctl-dev deploy --config satusky.toml --wait
```

The build streams again from the top. Docker layer caching means unchanged layers are skipped — if only `requirements.txt` changed, only the `pip install` layer and everything after it is rebuilt.

---

## Problem 4: Secrets and Env Vars Not Available in the App

### How to spot it

You added a secret or env var and the app still logs "env var not set" or uses an empty string.

```
2026-04-26T15:02:14Z [my-api] WARN: API_KEY is empty, auth disabled
```

### Diagnose

First confirm the key actually exists:

```bash
1ctl-dev env list --config satusky.toml
```

```
KEY          VALUE
LOG_LEVEL    info
```

```bash
1ctl-dev secret list --config satusky.toml
```

```
KEY
API_KEY
```

The key is there. The problem is that env vars and secrets are injected at pod startup. A running pod does not pick up changes made after it started. The pod is still running with the old environment from before you called `env create` or `secret create`.

### Fix

Restart the deployment so a fresh pod starts with the updated environment:

```bash
1ctl-dev deploy restart --config satusky.toml
```

Stream the logs to confirm the app now sees the value:

```bash
1ctl-dev logs stream --config satusky.toml
```

```
2026-04-26T15:05:00Z [my-api] API_KEY loaded from environment
2026-04-26T15:05:00Z [my-api] Auth enabled
```

Note: `deploy restart` replaces pods without rebuilding the image. Use it whenever you've only changed env vars or secrets and don't have new code to ship. If you also have new code, just run `deploy --config satusky.toml --wait` instead — that redeploys and picks up env changes in the same step.

---

## Problem 5: Wrong Deployment Updated

### How to spot it

You deployed but the wrong app changed. You're in `client-b-shop/` and accidentally ran with the `client-a-dashboard/satusky.toml` path.

```bash
1ctl-dev -o json deploy list | jq '.[] | {name, status}'
```

```json
{"name": "dashboard", "status": "running"}
{"name": "shop-api", "status": "running"}
```

Both look running, but you notice `dashboard` just got a fresh deploy timestamp even though you didn't intend to touch it.

### Diagnose

Check the release history of the deployment you suspect was touched:

```bash
1ctl-dev deploy releases --config ~/projects/client-a-dashboard/satusky.toml
```

```
VERSION  STATUS    DEPLOYED AT           MESSAGE
5        active    2026-04-26 15:20:08   (no message)
4        inactive  2026-04-25 10:14:33   feat: add dark mode
```

Version 5 was just deployed moments ago. That was the accidental deploy.

### Fix

Roll back to the last known-good version:

```bash
1ctl-dev deploy rollback \
  --config ~/projects/client-a-dashboard/satusky.toml \
  --version 4 -y
```

```
Rolling back dashboard to version 4...
Waiting for pods to be Running...
  dashboard-7a3f2c1b9-q6wr8   Running   ✓
Rollback complete. Now running version 4.
```

No image rebuild — the platform reruns the version 4 image immediately.

---

## Problem 6: "App Not Found" Error

### How to spot it

```bash
1ctl-dev deploy status --config satusky.toml
```

```
Error: app "my-api" not found
```

### Diagnose

Work through three checks in order.

**Check 1 — Does the app exist at all?**

```bash
1ctl-dev deploy list
```

```
NAME         STATUS    AGE
shop-api     running   3d
```

`my-api` is not listed. It either was never deployed, was destroyed, or lives in a different org.

**Check 2 — Are you in the right org?**

```bash
1ctl-dev org current
```

```
client-b-prod
```

Wrong org. You're looking in `client-b-prod` but `my-api` belongs to `client-a-prod`.

**Check 3 — Are you using the right profile?**

```bash
1ctl-dev profile list
```

```
NAME       URL                                    ACTIVE
client-a   https://api.satusky.com/v1/cli         no
client-b   https://api.satusky.com/v1/cli         yes
local      http://localhost:8081/v1/cli            no
```

`client-b` is active. Switch to `client-a`:

```bash
1ctl-dev profile use client-a
1ctl-dev org switch client-a-prod
1ctl-dev deploy list
```

```
NAME     STATUS    AGE
my-api   running   5d
```

There it is.

### Fix (app was actually destroyed and needs to be recreated)

If the app genuinely does not exist — never deployed, or previously destroyed — create it by deploying:

```bash
1ctl-dev deploy --config satusky.toml --wait
```

This creates the deployment from scratch and blocks until it is Running.

---

## General Debugging Checklist

When something is wrong and you're not sure where to start:

```bash
# 1. Confirm you're in the right profile and org
1ctl-dev profile list
1ctl-dev org current

# 2. Confirm the app exists
1ctl-dev deploy list

# 3. Check pod status
1ctl-dev deploy status --config satusky.toml

# 4. Stream live logs
1ctl-dev logs stream --config satusky.toml

# 5. Verify env vars and secrets are set
1ctl-dev env list --config satusky.toml
1ctl-dev secret list --config satusky.toml

# 6. Check release history
1ctl-dev deploy releases --config satusky.toml
```

Most problems are visible in step 4. If the logs look fine but the app is still unreachable, the bind address (Problem 2) is the most common culprit.
