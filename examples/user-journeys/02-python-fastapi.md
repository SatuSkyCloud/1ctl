# Deploying a Python FastAPI App to SatuSky

**Who this is for**: Backend Python developers who want to ship a FastAPI service without managing infrastructure.  
**What we're building**: A FastAPI app that caches responses in Upstash Redis and calls the OpenAI API. Credentials live in secrets; runtime config lives in env vars.  
**What you'll learn**: Cloud build, secret rotation, env var cleanup, and using `-o json` to script against the API.

---

## CLI Coverage

> ✅ **Fully covered** — every command in this guide works with the current CLI.
> No gaps.

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
Building image...  done (55s)
Pushing image...   done (11s)
Creating deployment my-fastapi...
Waiting for pods to be Running...
  my-fastapi-5b8c6d4f9-qr3m2   Running   ✓
Deploy complete. App is live.
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
# 2026-04-26T09:12:05Z [my-fastapi] Redis ping OK — connected to Upstash
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

Trigger a redeploy to apply the new vars:

```bash
1ctl deploy --config satusky.toml --wait
```

---

## 7. Check resource allocation with `-o json`

`-o json` (or `--output json`) returns machine-readable output — useful for CI scripts or just for double-checking.

```bash
1ctl deploy get --config satusky.toml -o json
```

```json
{
  "name": "my-fastapi",
  "status": "running",
  "version": 2,
  "cpu": "0.5",
  "memory": "512Mi",
  "replicas": 1,
  "machine": "compute-main-01",
  "url": "https://silentfox-b7r4n1.satusky.com",
  "deployed_at": "2026-04-26T09:14:22Z"
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
2026-04-26T09:15:01Z [my-fastapi] Application startup complete.
2026-04-26T09:15:14Z [my-fastapi] GET /health 200 2ms
2026-04-26T09:15:31Z [my-fastapi] POST /api/summarize 200 843ms cache=miss
2026-04-26T09:15:34Z [my-fastapi] POST /api/summarize 200 4ms  cache=hit
```

---

## 9. Rotate the OpenAI API key

Your API key was compromised or you're doing a planned rotation. `secret create` merges, so passing a new value for an existing key overwrites it without touching `REDIS_URL`.

```bash
1ctl secret create \
  --config satusky.toml \
  --kv OPENAI_API_KEY=sk-proj-NewKeyHere9876543210zyxwvutsrqponmlkjihgfedcba
```

Redeploy to inject the new secret into the running container:

```bash
1ctl deploy --config satusky.toml --wait
```

Watch the logs to confirm no auth errors on startup:

```bash
1ctl logs stream --config satusky.toml
# 2026-04-26T09:31:07Z [my-fastapi] OpenAI client initialized OK
```

---

## 10. Remove verbose logging

Your app stabilized. `LOG_LEVEL=info` is fine in production but you want to stop emitting info-level logs entirely to reduce noise. Remove the env var:

```bash
1ctl env unset --config satusky.toml --key LOG_LEVEL
```

`env unset` removes only the named key — `ENVIRONMENT` is untouched. Redeploy:

```bash
1ctl deploy --config satusky.toml --wait
```

---

## 11. Scripting with `-o json`

Extract the app URL in a shell script without parsing human-readable output:

```bash
APP_URL=$(1ctl deploy get --config satusky.toml -o json | jq -r '.url')
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
| JSON deploy info | `1ctl deploy get --config satusky.toml -o json` |
| Live logs | `1ctl logs stream --config satusky.toml` |
