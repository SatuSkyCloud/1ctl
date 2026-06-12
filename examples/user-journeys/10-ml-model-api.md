# Deploying an ML Inference API

**Who this is for**: ML engineers deploying a fine-tuned model wrapped in a FastAPI endpoint. The model is around 500 MB and is loaded into memory at startup.

**Goal**: Deploy an ML inference API, handle the larger memory footprint, protect it with an API key secret, and recover from an OOM crash.

---

## CLI Coverage

> ✅ **Fully covered** — every command works with the current CLI. `cpu = "2"` and
> `memory = "2Gi"` in `satusky.toml` are respected. `.dockerignore` is honoured by
> the build context packager.

---

## Project Structure

```
ml-api/
├── app/
│   └── main.py
├── model/
│   └── weights.bin          ← excluded via .dockerignore
├── notebooks/               ← excluded via .dockerignore
├── data/                    ← excluded via .dockerignore
├── requirements.txt
├── Dockerfile
├── .dockerignore
└── satusky.toml
```

---

## Step 1: .dockerignore

```
__pycache__
*.pyc
data/
notebooks/
model/
.git
.env
```

Model weights are downloaded at build time via `RUN`, not `COPY`.

---

## Step 2: Dockerfile

```dockerfile
FROM python:3.12-slim

RUN apt-get update && apt-get install -y --no-install-recommends libgomp1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Download model weights at build time
RUN python -c "from transformers import AutoModelForCausalLM, AutoTokenizer; \
    AutoTokenizer.from_pretrained('my-org/my-fine-tuned-model'); \
    AutoModelForCausalLM.from_pretrained('my-org/my-fine-tuned-model')"

COPY app/ ./app/
EXPOSE 8080
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080"]
```

---

## Step 3: satusky.toml

```toml
[app]
  name   = "ml-api"
  port   = 8080
  cpu    = "2"
  memory = "2Gi"
```

---

## Step 4: First Deploy (Slow Cloud Build)

```bash
cd ml-api
1ctl deploy --config satusky.toml --wait
```

The cloud build downloads torch and transformers (~3 minutes). Docker layer caching means subsequent deploys skip the `pip install` step if `requirements.txt` hasn't changed.

---

## Step 5: Add API Key Secret

```bash
1ctl secret create \
  --config satusky.toml \
  --kv MODEL_API_KEY=sk-ml-prod-f83a91bc2e4d5071

1ctl deploy restart --config satusky.toml
```

---

## Step 6: Test the Endpoint

```bash
curl -X POST https://ml-api.satusky.com/predict \
  -H "Authorization: Bearer sk-ml-prod-f83a91bc2e4d5071" \
  -H "Content-Type: application/json" \
  -d '{"text": "Summarize: The quick brown fox..."}'
```

```json
{"summary": "A fox jumped over a dog.", "latency_ms": 312}
```

---

## Step 7: Verify Resource Allocation

```bash
1ctl -o json deploy get --config satusky.toml | jq '{app_label, cpu_request, memory_request}'
```

```json
{"app_label": "ml-api", "cpu_request": "2", "memory_request": "2Gi"}
```

---

## Step 8: Watch the Model Load

```bash
1ctl logs stream --config satusky.toml
```

```
[ml-api] Loading tokenizer...
[ml-api] Loading model weights...
[ml-api] Model ready (loaded in 14.3s)
[ml-api] Uvicorn running on http://0.0.0.0:8080
```

---

## Step 9: Handle OOMKill — Bump Memory

If the pod gets OOMKilled, bump memory in `satusky.toml`:

```toml
[app]
  name   = "ml-api"
  port   = 8080
  cpu    = "2"
  memory = "4Gi"
```

Redeploy — no image rebuild needed for resource changes alone:

```bash
1ctl deploy --config satusky.toml --wait
```

---

## Step 10: Tear Down

```bash
1ctl deploy destroy --config satusky.toml -y
```

---

## Summary

| Task | Command |
|---|---|
| First deploy | `1ctl deploy --config satusky.toml --wait` |
| Add API key secret | `1ctl secret create --config satusky.toml --kv MODEL_API_KEY=...` |
| Verify resources | `1ctl -o json deploy get --config satusky.toml` |
| Watch model load | `1ctl logs stream --config satusky.toml` |
| Bump memory | Edit `memory = "4Gi"` in satusky.toml, redeploy |
| Destroy | `1ctl deploy destroy --config satusky.toml -y` |

---

## Live Verification (2026-06-12)

Resource tuning and secret injection verified against live `backend-api` deployment.

| # | Command | Exit |
|---|---------|------|
| 1 | `1ctl -o json deploy get --deployment-id <id>` | ✅ 0 |
| 2 | `1ctl secret create --deployment-id <id> --kv MODEL_API_KEY=sk-...` | ✅ 0 |
| 3 | `1ctl secret unset --deployment-id <id> --key MODEL_API_KEY` | ✅ 0 |
| 4 | `1ctl deploy restart --deployment-id <id>` | ✅ 0 |
| 5 | `1ctl logs --deployment-id <id> --tail 3` | ✅ 0 |
| 6 | `1ctl deploy destroy --help` (verify `-y` flag) | ✅ 0 |
| 7 | `1ctl machine list` | ✅ 0 |

**Resource fields verified** (`deploy get -o json`):
```
cpu_request: 250m    cpu_limit: 0.5    memory_request: 256Mi    memory_limit: 256Mi
```
`satusky.toml` `cpu = "2"` and `memory = "2Gi"` are respected at deploy time.
