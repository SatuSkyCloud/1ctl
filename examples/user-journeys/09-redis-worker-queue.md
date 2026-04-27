# User Journey 9: Redis-Backed Worker Queue

**Who this is for**: Backend developers building a task processing system with a web API that enqueues jobs and a separate worker process that consumes them.

**Goal**: Deploy two independent SatuSky services — `job-api` and `job-worker` — that share a Redis connection secret backed by Upstash.

---

## Overview

The system has two parts:

- `job-api` — an HTTP service that accepts requests and pushes job IDs onto a Redis list.
- `job-worker` — a long-running Python process that `BLPOP`s from the same list and processes each job.

Both services are deployed independently. They share a secret (`REDIS_URL`) that points at the same Upstash instance.

---

## Project Structure

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

## Step 1: Dockerfile — Job API

```dockerfile
# api/Dockerfile
FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY app.py .
EXPOSE 8080
CMD ["python", "app.py"]
```

`api/requirements.txt`:

```
flask==3.0.3
redis==5.0.4
```

---

## Step 2: Dockerfile — Job Worker

The worker runs `worker.py` continuously — it never binds an HTTP port. The platform still requires a `port` in `satusky.toml`; use 8080 as a health-check endpoint that the worker exposes on a background thread.

```dockerfile
# worker/Dockerfile
FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY worker.py .
CMD ["python", "worker.py"]
```

`worker/requirements.txt`:

```
redis==5.0.4
flask==3.0.3
```

`worker/worker.py` (abbreviated):

```python
import os
import threading
import redis
from flask import Flask

REDIS_URL = os.environ["REDIS_URL"]
r = redis.from_url(REDIS_URL)

# Minimal health-check server so the platform can probe port 8080
app = Flask(__name__)

@app.route("/healthz")
def health():
    return {"status": "ok"}, 200

def process_loop():
    print("Worker started, waiting for jobs...", flush=True)
    while True:
        _, job_id = r.blpop("jobs")
        print(f"Processing job {job_id.decode()}", flush=True)
        # ... do the work ...

threading.Thread(target=process_loop, daemon=True).start()

if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
```

---

## Step 3: satusky.toml Files

`api/satusky.toml`:

```toml
name   = "job-api"
port   = 8080
cpu    = "0.5"
memory = "256Mi"
```

`worker/satusky.toml`:

```toml
name   = "job-worker"
port   = 8080
cpu    = "0.5"
memory = "256Mi"
```

---

## Step 4: Deploy the API

```bash
cd task-system/api
1ctl-dev deploy --config satusky.toml --wait
```

```
Building image...  done (28s)
Pushing image...   done (5s)
Creating deployment job-api...
Waiting for pods to be Running...
  job-api-5f9c7d4b8-r2lp6   Running   ✓
Deploy complete. App is live.
```

---

## Step 5: Deploy the Worker

```bash
cd ../worker
1ctl-dev deploy --config satusky.toml --wait
```

```
Building image...  done (24s)
Pushing image...   done (4s)
Creating deployment job-worker...
Waiting for pods to be Running...
  job-worker-8a2e1c6f4-k9mn3   Running   ✓
Deploy complete. App is live.
```

---

## Step 6: Add the Shared Redis Secret

Both deployments need the same `REDIS_URL`. Add it to each independently — secrets are scoped per deployment.

```bash
# Add to the API
1ctl-dev secret create \
  --config api/satusky.toml \
  --kv REDIS_URL=rediss://default:your-upstash-token@global-fly-12345.upstash.io:6379

# Add to the worker
1ctl-dev secret create \
  --config worker/satusky.toml \
  --kv REDIS_URL=rediss://default:your-upstash-token@global-fly-12345.upstash.io:6379
```

`secret create` merges — you can run this multiple times and it will only update the keys you specify.

Secrets are not live until the pods restart. Trigger a fresh deploy for each:

```bash
1ctl-dev deploy --config api/satusky.toml --wait
1ctl-dev deploy --config worker/satusky.toml --wait
```

---

## Step 7: Verify Both Deployments

```bash
1ctl-dev -o json deploy list
```

```json
[
  {
    "name": "job-api",
    "status": "running",
    "image": "registry.satusky.com/job-api:c3d7f21",
    "created_at": "2026-04-26T11:00:00Z"
  },
  {
    "name": "job-worker",
    "status": "running",
    "image": "registry.satusky.com/job-worker:a9b4e88",
    "created_at": "2026-04-26T11:05:00Z"
  }
]
```

---

## Step 8: Monitor Both with Live Logs

Open two terminals — one per service:

**Terminal 1 — API logs:**

```bash
1ctl-dev logs stream --config api/satusky.toml
```

```
2026-04-26T11:12:04Z [job-api] POST /enqueue 200 4ms  job_id=a3f9c811
2026-04-26T11:12:09Z [job-api] POST /enqueue 200 3ms  job_id=b7e02f44
```

**Terminal 2 — Worker logs:**

```bash
1ctl-dev logs stream --config worker/satusky.toml
```

```
2026-04-26T11:12:05Z [job-worker] Processing job a3f9c811
2026-04-26T11:12:05Z [job-worker] Job a3f9c811 complete (took 210ms)
2026-04-26T11:12:10Z [job-worker] Processing job b7e02f44
2026-04-26T11:12:10Z [job-worker] Job b7e02f44 complete (took 195ms)
```

---

## Step 9: Scale the Worker

If the job queue is growing faster than the worker can drain it, give the worker more CPU and memory.

Edit `worker/satusky.toml`:

```toml
name   = "job-worker"
port   = 8080
cpu    = "1"
memory = "512Mi"
```

Then redeploy:

```bash
1ctl-dev deploy --config worker/satusky.toml --wait
```

The API keeps running without interruption — you only redeployed the worker.

---

## Step 10: Take the Worker Down for Maintenance

To pause job processing without touching the API:

```bash
1ctl-dev deploy destroy --config worker/satusky.toml -y
```

```
Destroying deployment job-worker...  done
```

The API continues accepting and enqueuing jobs. Redis holds them until the worker comes back. When you're ready to resume:

```bash
1ctl-dev deploy --config worker/satusky.toml --wait
```

---

## Summary

| Task | Command |
|---|---|
| Deploy API | `1ctl-dev deploy --config api/satusky.toml --wait` |
| Deploy worker | `1ctl-dev deploy --config worker/satusky.toml --wait` |
| Add Redis secret | `1ctl-dev secret create --config <dir>/satusky.toml --kv REDIS_URL=...` |
| Remove a secret | `1ctl-dev secret unset --config <dir>/satusky.toml --key REDIS_URL` |
| Watch worker process jobs | `1ctl-dev logs stream --config worker/satusky.toml` |
| Scale worker resources | Edit satusky.toml, then `1ctl-dev deploy --config worker/satusky.toml --wait` |
| Pause worker (maintenance) | `1ctl-dev deploy destroy --config worker/satusky.toml -y` |
| List all deployments | `1ctl-dev -o json deploy list` |
