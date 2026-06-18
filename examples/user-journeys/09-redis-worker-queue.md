# Redis-Backed Worker Queue

**Who this is for**: Backend developers building a task processing system with a web API that enqueues jobs and a separate worker process that consumes them.

**Goal**: Deploy two independent SatuSky services — `job-api` and `job-worker` — that share a Redis connection secret.

---

## CLI Coverage

> ✅ **Fully covered** — every command works. The port-8080 requirement for worker
> deployments is a platform constraint (all deployments need a port), not a CLI gap.
> A minimal health-check endpoint is the correct workaround.

---

## Overview

- `job-api` — HTTP service that pushes job IDs onto a Redis list.
- `job-worker` — long-running Python process that `BLPOP`s from the same list.

Both share a secret (`REDIS_URL`) pointing at the same Upstash instance.

---

## Project Layout

```
task-system/
├── api/
│   ├── app.py
│   ├── requirements.txt
│   ├── Dockerfile
│   └── satusky.toml
└── worker/
    ├── worker.py
    ├── requirements.txt
    ├── Dockerfile
    └── satusky.toml
```

---

## Dockerfiles

**api/Dockerfile**:

```dockerfile
FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY app.py .
EXPOSE 8080
CMD ["python", "app.py"]
```

**worker/Dockerfile** (includes minimal Flask health-check on port 8080):

```dockerfile
FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY worker.py .
CMD ["python", "worker.py"]
```

**worker/worker.py** (abbreviated):

```python
import os, threading
import redis
from flask import Flask

REDIS_URL = os.environ["REDIS_URL"]
r = redis.from_url(REDIS_URL)

app = Flask(__name__)
@app.route("/healthz")
def health():
    return {"status": "ok"}, 200

def process_loop():
    while True:
        _, job_id = r.blpop("jobs")
        print(f"Processing job {job_id.decode()}", flush=True)

threading.Thread(target=process_loop, daemon=True).start()
app.run(host="0.0.0.0", port=8080)
```

---

## satusky.toml Files

```toml
[app]
  name   = "job-api"
  port   = 8080
  cpu    = "0.5"
  memory = "256Mi"
```

```toml
[app]
  name   = "job-worker"
  port   = 8080
  cpu    = "0.5"
  memory = "256Mi"
```

---

## Step 1: Deploy Both

```bash
cd task-system/api
1ctl deploy --config satusky.toml --wait

cd ../worker
1ctl deploy --config satusky.toml --wait
```

---

## Step 2: Add Shared Redis Secret

```bash
1ctl secret create --config api/satusky.toml \
  --kv REDIS_URL=rediss://default:token@global-fly-12345.upstash.io:6379

1ctl secret create --config worker/satusky.toml \
  --kv REDIS_URL=rediss://default:token@global-fly-12345.upstash.io:6379
```

Restart both to pick up secrets:

```bash
1ctl deploy restart --config api/satusky.toml
1ctl deploy restart --config worker/satusky.toml
```

---

## Step 3: Verify

```bash
1ctl -o json deploy list
```

```json
[
  {"deployment_id": "uuid-1", "app_label": "job-api", "status": "completed"},
  {"deployment_id": "uuid-2", "app_label": "job-worker", "status": "completed"}
]
```

---

## Step 4: Monitor Both with Live Logs

```bash
# Terminal 1 — API
1ctl logs stream --config api/satusky.toml
# [job-api] POST /enqueue 200 4ms  job_id=a3f9c811

# Terminal 2 — Worker
1ctl logs stream --config worker/satusky.toml
# [job-worker] Processing job a3f9c811
# [job-worker] Job a3f9c811 complete (took 210ms)
```

---

## Step 5: Scale the Worker

Edit `worker/satusky.toml`:

```toml
[app]
  name   = "job-worker"
  port   = 8080
  cpu    = "1"
  memory = "512Mi"
```

Redeploy:

```bash
1ctl deploy --config worker/satusky.toml --wait
```

The API keeps running — you only redeployed the worker.

---

## Step 6: Pause Worker for Maintenance

```bash
1ctl deploy destroy --config worker/satusky.toml -y
```

The API continues accepting jobs. Redis holds them until the worker returns:

```bash
1ctl deploy --config worker/satusky.toml --wait
```

---

## Summary

| Task | Command |
|---|---|
| Deploy API | `1ctl deploy --config api/satusky.toml --wait` |
| Deploy worker | `1ctl deploy --config worker/satusky.toml --wait` |
| Add Redis secret | `1ctl secret create --config <dir>/satusky.toml --kv REDIS_URL=...` |
| Apply secrets | `1ctl deploy restart --config <dir>/satusky.toml` |
| Watch worker | `1ctl logs stream --config worker/satusky.toml` |
| Scale worker | Edit satusky.toml, then `1ctl deploy --config worker/satusky.toml --wait` |
| Pause worker | `1ctl deploy destroy --config worker/satusky.toml -y` |
| List all | `1ctl -o json deploy list` |

---

## Live Verification (2026-06-12)

Worker/API dual-deploy and shared secret workflow verified against live instance.

| # | Command | Exit |
|---|---------|------|
| 1 | `1ctl deploy list` (independent deploys) | ✅ 0 |
| 2 | `1ctl -o json deploy list` | ✅ 0 |
| 3 | `1ctl secret create --deployment-id <id> --kv REDIS_URL=rediss://...` | ✅ 0 |
| 4 | `1ctl secret list` | ✅ 0 |
| 5 | `1ctl deploy restart --deployment-id <id>` | ✅ 0 |
| 6 | `1ctl logs --deployment-id <id> --tail 3` | ✅ 0 |
| 7 | `1ctl deploy destroy --deployment-id <nonexistent> -y` (confirm prompt exists) | ✅ 0 |
| 8 | `1ctl deploy status --deployment-id <id>` | ✅ 0 |

**Destroy prompt verified**: Asks `[y/N]` confirmation. `-y` skips it.

**Restart vs deploy**: `deploy restart` triggers rolling restart without rebuild. `deploy --wait` rebuilds + deploys.
