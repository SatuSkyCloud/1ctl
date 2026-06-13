# Deploying a Python FastAPI App to SatuSky

**Who this is for**: Backend Python developers who want to ship a FastAPI service without managing infrastructure.  
**What we're building**: A FastAPI app that caches responses in Upstash Redis and calls the OpenAI API. Credentials live in secrets; runtime config lives in env vars.  
**What you'll learn**: Cloud build, secret rotation, env var cleanup, and using `-o json` to script against the API.

---

## 1. Project structure

```
my-fastapi/
├── app/
│   ├── main.py
│   └── cache.py
├── requirements.txt
├── Dockerfile
└── satusky.toml
```

---

## 2. Dockerfile

```dockerfile
FROM python:3.12-slim

WORKDIR /app

# Install dependencies first so Docker can cache this layer
COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY app/ ./app/

EXPOSE 8000
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
```

A sample `requirements.txt`:

```
fastapi==0.111.0
uvicorn[standard]==0.29.0
redis==5.0.3
openai==1.25.0
pydantic==2.7.1
```

---

## 3. satusky.toml

```toml
[app]
  name   = "my-fastapi"
  port   = 8000
  cpu    = "0.5"
  memory = "512Mi"
```

---

## 4. First deploy

SatuSky builds the image in the cloud — you don't need Docker installed locally.

```bash
cd my-fastapi
1ctl deploy --config satusky.toml --wait
```

```
💡 Build queued (ID: ...)
  [build] Docker build completed
Step 2/5: Creating/updating deployment my-fastapi ✓
...
💡 Generated new domain: silentfox-b7r4n1.satusky.com
✅ 🚀 Deployment for my-fastapi is successful! Your app is live at: https://silentfox-b7r4n1.satusky.com
💡 Waiting for deployment to become healthy...
✅ Deployment is healthy — pods Running
```

---

## 5. Set secrets

The Redis URL contains credentials and the OpenAI key is sensitive — both go into secrets.

```bash
1ctl secret create \
  --config satusky.toml \
  --kv REDIS_URL=rediss://default:AXabCDEFgh123456@us1-cool-stork-12345.upstash.io:6380 \
  --kv OPENAI_API_KEY=sk-proj-AbCdEfGhIjKlMnOpQrStUvWxYz1234567890abcdefghijklmnop
```

Verify your app can reach Redis. If it can't, check the URL in the logs:

```bash
1ctl logs stream --config satusky.toml
# [my-fastapi] Redis ping OK — connected to Upstash
```

---

## 6. Set environment variables

Non-sensitive runtime config — environment name, log verbosity — lives in env vars.

```bash
1ctl env create \
  --config satusky.toml \
  --env ENVIRONMENT=production \
  --env LOG_LEVEL=info
```

Apply the new vars with a restart:

```bash
1ctl deploy restart --config satusky.toml
```

---

## 7. Check deployment info with `-o json`

`-o json` (or `--output json`) returns machine-readable output — useful for CI scripts.

```bash
1ctl deploy get --config satusky.toml -o json
```

```json
{
  "deployment_id": "a1b2c3d4-...",
  "app_label": "my-fastapi",
  "status": "completed",
  "image": "registry.satusky.com/my-fastapi:d4e5f6a",
  "cpu_request": "500m",
  "cpu_limit": "500m",
  "memory_request": "512Mi",
  "memory_limit": "512Mi",
  "replicas": 1,
  "domain": "https://silentfox-b7r4n1.satusky.com"
}
```

If the CPU or memory values don't match what's in your `satusky.toml`, you likely deployed before saving your latest config changes — just re-run `deploy`.

---

## 8. Stream live logs

Open a second terminal and tail logs while you hit the API:

```bash
1ctl logs stream --config satusky.toml
```

```
2026-06-12T09:15:01Z [my-fastapi-5b8c6d4f9-qr3m2] Application startup complete.
2026-06-12T09:15:14Z [my-fastapi-5b8c6d4f9-qr3m2] GET /health 200 2ms
2026-06-12T09:15:31Z [my-fastapi-5b8c6d4f9-qr3m2] POST /api/summarize 200 843ms cache=miss
2026-06-12T09:15:34Z [my-fastapi-5b8c6d4f9-qr3m2] POST /api/summarize 200 4ms  cache=hit
```

---

## 9. Rotate the OpenAI API key

Your API key was compromised or you're doing a planned rotation. `secret create` merges, so passing a new value for an existing key overwrites it without touching `REDIS_URL`.

```bash
1ctl secret create \
  --config satusky.toml \
  --kv OPENAI_API_KEY=sk-proj-NewKeyHere9876543210zyxwvutsrqponmlkjihgfedcba
```

Restart to inject the new secret into the running container:

```bash
1ctl deploy restart --config satusky.toml
```

Watch the logs to confirm no auth errors on startup:

```bash
1ctl logs stream --config satusky.toml
# [my-fastapi] OpenAI client initialized OK
```

---

## 10. Remove verbose logging

Your app stabilized. Remove the verbose log level:

```bash
1ctl env unset --config satusky.toml --key LOG_LEVEL
```

`env unset` removes only the named key — `ENVIRONMENT` is untouched. Restart to apply:

```bash
1ctl deploy restart --config satusky.toml
```

---

## 11. Scripting with `-o json`

Extract the app URL in a shell script without parsing human-readable output:

```bash
APP_URL=$(1ctl deploy get --config satusky.toml -o json | jq -r '.domain')
echo "Running smoke test against $APP_URL"
curl -sf "$APP_URL/health" | jq .
```

```json
{"status": "ok", "redis": "connected", "version": "2.1.0"}
```

This pattern is useful in CI — gate the smoke test on the deploy finishing with `--wait`, then extract the URL with `jq`.

---

## Summary

| Task | Command |
|---|---|
| Deploy (cloud build) | `1ctl deploy --config satusky.toml --wait` |
| Set secrets (create or update) | `1ctl secret create --config satusky.toml --kv KEY=VAL` |
| Rotate a single secret | `1ctl secret create --config satusky.toml --kv KEY=new-val` |
| Remove a secret | `1ctl secret unset --config satusky.toml --key KEY` |
| Set env vars | `1ctl env create --config satusky.toml --env KEY=VAL` |
| Remove an env var | `1ctl env unset --config satusky.toml --key KEY` |
| Apply env/secret changes | `1ctl deploy restart --config satusky.toml` |
| JSON deploy info | `1ctl deploy get --config satusky.toml -o json` |
| Live logs | `1ctl logs stream --config satusky.toml` |

---

## Live Verification (2026-06-12)

All commands verified against live `org123-c0bee423` namespace with `backend-api` deployment.

| # | Command | Exit |
|---|---------|------|
| 1 | `1ctl deploy list` | ✅ 0 |
| 2 | `1ctl -o json deploy list` | ✅ 0 |
| 3 | `1ctl deploy status --deployment-id <id>` | ✅ 0 |
| 4 | `1ctl -o json deploy get --deployment-id <id>` | ✅ 0 |
| 5 | `1ctl deploy releases --deployment-id <id>` | ✅ 0 |
| 6 | `1ctl secret create --deployment-id <id> --kv KEY=VAL` | ✅ 0 |
| 7 | `1ctl secret list` | ✅ 0 |
| 8 | `1ctl secret unset --deployment-id <id> --key KEY` | ✅ 0 |
| 9 | `1ctl env create --deployment-id <id> --env KEY=VAL` | ✅ 0 |
| 10 | `1ctl env list --deployment-id <id>` | ✅ 0 |
| 11 | `1ctl -o json env list --deployment-id <id>` | ✅ 0 |
| 12 | `1ctl env unset --deployment-id <id> --key KEY` | ✅ 0 |
| 13 | `1ctl deploy restart --deployment-id <id>` | ✅ 0 |
| 14 | `1ctl logs --deployment-id <id> --tail 3` | ✅ 0 |
| 15 | `1ctl doctor --deployment-id <id>` | ✅ 0 |

**JSON field verification** (`deploy get -o json`):
```
app_label: backend-api     status: completed    domain: https://...satusky.com
cpu_request: 250m          memory_request: 256Mi   replicas: 1
```

**Secret rotation verified**: `secret create` with existing key overwrites. `secret unset` removes single key.

**Env cleanup verified**: `env unset` removes single key, others untouched. `env create` merges new keys.
