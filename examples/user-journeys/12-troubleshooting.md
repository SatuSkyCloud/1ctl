# Troubleshooting Common Deployment Problems

**Who this is for**: Any developer hitting a wall after deploying. A reference guide for the six most common problems and exactly how to diagnose and fix each one.

---

## Quick Reference

| Symptom | Problem |
|---|---|
| Pods restarting, never healthy | CrashLoopBackOff |
| Deploy succeeded but app returns 502/504 | Wrong bind address |
| Build fails before deploy | Dockerfile error |
| App can't read env vars or secrets | Config not reloaded |
| Wrong app got updated | Wrong `--config` path |
| `app "x" not found` error | App missing or wrong context |

---

## Problem 1: Pod Keeps Restarting (CrashLoopBackOff)

### Spot it

```bash
1ctl deploy status --config satusky.toml
```

```
Workload: Pending   (or show restart count)
```

### Diagnose

```bash
1ctl logs stream --config satusky.toml
```

```
[my-api] Starting server...
[my-api] panic: DATABASE_URL environment variable not set
```

### Fix

```bash
1ctl secret create --config satusky.toml --kv DATABASE_URL=postgres://user:pass@host/db
1ctl deploy restart --config satusky.toml
```

---

## Problem 2: Deployed but Returns 502/504

### Spot it

```bash
curl https://my-api.satusky.com/health
# curl: (22) The requested URL returned error: 502
```

### Diagnose

```bash
1ctl logs stream --config satusky.toml
# [my-api] Listening on 127.0.0.1:8080   ← loopback!
```

### Fix

Bind on `0.0.0.0`:

```bash
1ctl env create --config satusky.toml --env HOST=0.0.0.0
1ctl deploy --config satusky.toml --wait
```

---

## Problem 3: Cloud Build Fails

### Spot it

Build output shows the error inline:

```
Step 3/6 : RUN pip install -r requirements.txt
ERROR: Could not find a version that satisfies the requirement torch==99.0.0
Build failed.
```

### Fix

Fix the dependency version in `requirements.txt` or `package.json`, then redeploy:

```bash
1ctl deploy --config satusky.toml --wait
```

---

## Problem 4: Env Vars or Secrets Not Available

### Spot it

```bash
1ctl env list --config satusky.toml
# KEY shows up, but app logs say "API_KEY not set"
```

### Cause

Env vars and secrets are injected at pod startup. A running pod doesn't pick up changes made after it started.

### Fix

```bash
1ctl deploy restart --config satusky.toml
```

Use `deploy restart` when you've only changed env vars/secrets. Use `deploy --wait` if you also have new code.

---

## Problem 5: Wrong Deployment Updated

### Spot it

You deployed but the wrong app changed.

### Diagnose

```bash
1ctl deploy releases --config ~/projects/wrong-app/satusky.toml
```

A fresh version just appeared.

### Fix

```bash
1ctl deploy rollback --config ~/projects/wrong-app/satusky.toml --version 3 -y
```

---

## Problem 6: "App Not Found" Error

### Diagnose — work through these checks in order:

```bash
# Check 1 — Does the app exist at all?
1ctl deploy list

# Check 2 — Are you in the right org?
1ctl org current

# Check 3 — Are you using the right profile?
1ctl profile list
```

### Fix

Switch to the correct profile/org, or deploy from scratch:

```bash
1ctl profile use client-a
1ctl org switch client-a-prod
1ctl deploy --config satusky.toml --wait
```

---

## General Debugging Checklist

```bash
# 1. Confirm profile and org
1ctl profile list
1ctl org current

# 2. Confirm the app exists
1ctl deploy list

# 3. Check pod status
1ctl deploy status --config satusky.toml

# 4. Stream live logs (most problems visible here)
1ctl logs stream --config satusky.toml

# 5. Verify env vars and secrets
1ctl env list --config satusky.toml
1ctl secret list

# 6. Check release history
1ctl deploy releases --config satusky.toml
```

Most problems are visible in step 4. If logs look fine but the app is unreachable, Problem 2 (bind address) is the most common culprit.

---

## Live Verification (2026-06-12)

All diagnostic commands verified against live `org123-c0bee423` namespace.

| # | Command | Exit | Notes |
|---|---------|------|-------|
| 1 | `1ctl profile list` | ✅ 0 | Shows active profile, URL, auth, org |
| 2 | `1ctl org current` | ✅ 0 | Returns org name + ID |
| 3 | `1ctl deploy list` | ✅ 0 | Lists all deployments |
| 4 | `1ctl deploy status --deployment-id <id>` | ✅ 0 | Multi-line: Workload, Message, Progress |
| 5 | `1ctl logs --deployment-id <id> --tail 3` | ✅ 0 | Stored logs; degrades gracefully when Loki down |
| 6 | `1ctl env list --deployment-id <id>` | ✅ 0 | Shows env vars + created time |
| 7 | `1ctl secret list` | ✅ 0 | Global list; no `--config` flag |
| 8 | `1ctl deploy releases --deployment-id <id>` | ✅ 0 | VERSION IMAGE STATUS DEPLOYED |
| 9 | `1ctl deploy rollback --help` | ✅ 0 | `--version`, `-y` exist; no `--wait` |
| 10 | `1ctl deploy destroy --deployment-id <id>` (no `-y`) | ✅ 0 | Prompts `[y/N]` |
| 11 | `1ctl doctor` | ✅ 0 | No smoke by default |
| 12 | `1ctl doctor --deployment-id <id>` | ✅ 0 | Smokes automatically in targeted mode |
| 13 | `1ctl doctor --smoke` | ✅ 0 | Smokes all deployments |
| 14 | `1ctl doctor --health-path /health` | ✅ 0 | Strict 2xx/3xx enforcement |
| 15 | `1ctl doctor --health-path invalid` | ❌ 1 | Rejected: "must start with /" |

**Key findings for troubleshooting guide:**
- `deploy status` does NOT show `READY RESTARTS AGE` columns — shows `Workload:` + `Message:` + `Progress:`
- `secret list` has NO `--config`/`--deployment-id` flags — global list only
- `rollback` has NO `--wait` flag — only `--version`, `--yes/-y`
- `logs` gracefully degrades when Loki is unavailable (shows fallback notice)
- `doctor --health-path` validates input (rejects paths without leading `/`)
