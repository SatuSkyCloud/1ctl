# ML model API example

This complete CPU-only example loads and validates
`model/model-v1.json` before listening on port 8080. It provides:

- `GET /healthz` for process liveness
- `GET /readyz` for readiness and the loaded model version
- `POST /predict` for bounded two-feature inference
- a 1024-byte request limit
- four prediction slots per replica, with `429` back-pressure

Run the unit tests:

```sh
go test ./...
go vet ./...
```

Build and test the image locally:

```sh
podman build -t ml-model-api:local .
podman run --rm -d --name ml-model-api-local \
  -p 127.0.0.1:18082:8080 ml-model-api:local
curl --fail --silent http://127.0.0.1:18082/readyz
curl --fail --silent -X POST http://127.0.0.1:18082/predict \
  -H 'content-type: application/json' \
  --data '{"features":[1,0]}'
podman stop ml-model-api-local
```

Deploy from this directory with a unique application name:

```sh
APP="guide-ml-$(date +%s)"
1ctl deploy --name "$APP" --wait
```

The documentation guide contains the complete validation and cleanup workflow.
