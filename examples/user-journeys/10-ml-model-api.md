# User Journey 10: Deploying an ML Inference API

**Who this is for**: ML engineers deploying a fine-tuned model wrapped in a FastAPI endpoint. The model is around 500 MB and is loaded into memory at startup.

**Goal**: Deploy an ML inference API, handle the larger memory footprint, protect it with an API key secret, and recover from an OOM crash.

---

## CLI Coverage

> ✅ **Fully covered** — every command in this guide works with the current CLI.
> `cpu = "2"` and `memory = "2Gi"` in `satusky.toml` are respected, cloud build
> handles the large Dockerfile, and `.dockerignore` is honoured by the build
> context packager. No gaps.

---

## Overview

Large model weights mean the default resource limits won't cut it. This journey shows how to set `cpu = "2"` and `memory = "2Gi"` from the start, stream the slow cloud build, verify resource allocation via JSON output, and bump memory if the pod OOMKills.

---

## Project Structure

```
ml-api/
├── app/
│   └── main.py
├── model/
│   └── weights.bin          ← large local file, excluded via .dockerignore
├── notebooks/               ← excluded via .dockerignore
├── data/                    ← excluded via .dockerignore
├── requirements.txt
├── Dockerfile
├── .dockerignore
└── satusky.toml
```

---

## Step 1: .dockerignore

Large files must be excluded before the build context is sent to the cloud builder. Without this, your 500 MB weights directory gets uploaded every time even if nothing changed.

`.dockerignore`:

```
__pycache__
*.pyc
*.pyo
*.pyd
data/
notebooks/
*.ipynb
model/
.git
.env
```

The model weights are baked into the image via a `RUN` step that downloads them at build time, not via `COPY` — this way the `.dockerignore` keeps the build context small while the image still ships the weights.

---

## Step 2: Dockerfile

```dockerfile
# syntax=docker/dockerfile:1
FROM python:3.12-slim AS base

# System deps for torch
RUN apt-get update && apt-get install -y --no-install-recommends \
    libgomp1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Install heavy deps first — this layer is cached as long as requirements.txt doesn't change
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Download model weights at build time
RUN python -c "from transformers import AutoModelForCausalLM, AutoTokenizer; \
    AutoTokenizer.from_pretrained('my-org/my-fine-tuned-model'); \
    AutoModelForCausalLM.from_pretrained('my-org/my-fine-tuned-model')"

# Copy application code last so code changes don't bust the model cache layer
COPY app/ ./app/

EXPOSE 8080
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8080"]
```

`requirements.txt`:

```
fastapi==0.111.0
uvicorn[standard]==0.30.1
torch==2.3.0
transformers==4.41.2
```

---

## Step 3: satusky.toml

```toml
name   = "ml-api"
port   = 8080
cpu    = "2"
memory = "2Gi"
```

The `cpu` and `memory` fields are the only resource controls in the TOML. Start at `2Gi` — you can bump it without rebuilding the image.

---

## Step 4: First Deploy (Cloud Build)

The cloud build downloads torch and transformers, which takes longer than a typical build. The CLI streams build output in real time so you can see what's happening.

```bash
cd ml-api
1ctl deploy --config satusky.toml --wait
```

You'll see the build stream directly to your terminal:

```
Building image...
  Step 1/8 : FROM python:3.12-slim
  Step 2/8 : RUN apt-get update ...
  Step 3/8 : COPY requirements.txt .
  Step 4/8 : RUN pip install ...  (this takes ~3 minutes for torch)
  Step 5/8 : RUN python -c "from transformers import ..." (downloading ~500MB)
  Step 6/8 : COPY app/ ./app/
  Step 7/8 : EXPOSE 8080
  Step 8/8 : CMD ["uvicorn", ...]
Building image...  done (4m 12s)
Pushing image...   done (18s)
Creating deployment ml-api...
Waiting for pods to be Running...
  ml-api-9b4c7f2d1-v8kx5   Running   ✓
Deploy complete. App is live.
```

The next deploy will be faster because Docker layer caching skips the `pip install` step if `requirements.txt` hasn't changed.

---

## Step 5: Add API Key Secret

Protect the endpoint with an API key injected as a secret:

```bash
1ctl secret create \
  --config satusky.toml \
  --kv MODEL_API_KEY=sk-ml-prod-f83a91bc2e4d5071
```

Secrets take effect on the next pod start. Trigger a fresh deploy:

```bash
1ctl deploy --config satusky.toml --wait
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

Confirm the platform actually scheduled the deployment with the requested CPU and memory:

```bash
1ctl -o json deploy get --config satusky.toml
```

```json
{
  "name": "ml-api",
  "status": "running",
  "cpu": "2",
  "memory": "2Gi",
  "image": "registry.satusky.com/ml-api:9f3c1a4",
  "replicas": 1,
  "created_at": "2026-04-26T13:00:00Z"
}
```

---

## Step 8: Watch the Model Load at Startup

Stream the logs right after deploy to confirm the model loads correctly:

```bash
1ctl logs stream --config satusky.toml
```

```
2026-04-26T13:00:08Z [ml-api] Loading tokenizer...
2026-04-26T13:00:09Z [ml-api] Loading model weights...
2026-04-26T13:00:22Z [ml-api] Model ready (loaded in 14.3s)
2026-04-26T13:00:22Z [ml-api] Uvicorn running on http://0.0.0.0:8080
2026-04-26T13:00:35Z [ml-api] POST /predict 200 312ms
```

A slow startup here (14 seconds) is normal. The platform waits for the health check to pass before routing traffic, so `--wait` won't return until the model has finished loading.

---

## Step 9: Handle OOMKill — Bump Memory

If the model is larger than expected, the pod gets killed by the OS. You'll see it in the logs:

```bash
1ctl logs stream --config satusky.toml
```

```
2026-04-26T13:01:10Z [ml-api] Loading model weights...
2026-04-26T13:01:18Z [ml-api] Killed
2026-04-26T13:01:19Z [ml-api] OOMKilled: container exceeded memory limit (2Gi)
2026-04-26T13:01:20Z [ml-api] Loading model weights...   ← pod restarted
2026-04-26T13:01:28Z [ml-api] Killed
```

The pod keeps restarting and crashing. Fix it by increasing memory in `satusky.toml`:

```toml
name   = "ml-api"
port   = 8080
cpu    = "2"
memory = "4Gi"
```

Then redeploy — no image rebuild needed because the code and weights haven't changed, but the platform still runs a fresh deploy with the new resource spec:

```bash
1ctl deploy --config satusky.toml --wait
```

```
Building image...  done (11s)
Pushing image...   done (4s)
Waiting for pods to be Running...
  ml-api-3c6d9a1f7-w4pz2   Running   ✓
Deploy complete. App is live.
```

Stream the logs again to confirm a clean startup:

```bash
1ctl logs stream --config satusky.toml
```

```
2026-04-26T13:08:04Z [ml-api] Loading model weights...
2026-04-26T13:08:21Z [ml-api] Model ready (loaded in 17.1s)
2026-04-26T13:08:21Z [ml-api] Uvicorn running on http://0.0.0.0:8080
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
| First deploy (slow build, streaming logs) | `1ctl deploy --config satusky.toml --wait` |
| Add API key secret | `1ctl secret create --config satusky.toml --kv MODEL_API_KEY=...` |
| Verify CPU/memory allocation | `1ctl -o json deploy get --config satusky.toml` |
| Watch model load at startup | `1ctl logs stream --config satusky.toml` |
| Bump memory after OOMKill | Edit `memory = "4Gi"` in satusky.toml, then redeploy |
| Destroy | `1ctl deploy destroy --config satusky.toml -y` |
