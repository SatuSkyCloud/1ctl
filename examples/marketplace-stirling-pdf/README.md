# Stirling PDF Helm marketplace package

This self-contained chart deploys one immutable, arm64-only Stirling PDF
instance. It uses a single 10 GiB `ceph-block-noreplica` ReadWriteOnce PVC and
the `Recreate` strategy so the persistent volume is not mounted by two Pods.
The init container creates every subPath directory before the application
starts.

The Satusky backend owns the reserved `satusky` runtime object. Do not pass it
as package input or commit credentials: this package declares no required
secrets and renders no Kubernetes Secret. At deployment time the backend sets
`appName`, `namespace`, `domainName`, `gatewayMode`, `gatewayName`,
`gatewayNamespace`, `cloudflareTunnelTarget`, and an empty `credentials` map.

## Validate locally

Run from this directory:

```sh
helm lint . \
  --set-string satusky.appName=stirling-pdf-lint \
  --set-string satusky.namespace=lint \
  --set-string satusky.domainName=stirling-pdf-lint.example.test \
  --set satusky.gatewayMode=true \
  --set-string satusky.gatewayName=satusky-gateway \
  --set-string satusky.gatewayNamespace=gateway-system \
  --set-string satusky.cloudflareTunnelTarget=http://tunnel-proxy:8080

helm template stirling-pdf-lint . --namespace lint \
  --set-string satusky.appName=stirling-pdf-lint \
  --set-string satusky.namespace=lint \
  --set-string satusky.domainName=stirling-pdf-lint.example.test \
  --set satusky.gatewayMode=true \
  --set-string satusky.gatewayName=satusky-gateway \
  --set-string satusky.gatewayNamespace=gateway-system \
  --set-string satusky.cloudflareTunnelTarget=http://tunnel-proxy:8080
```

## Create and publish a private package

Build the development CLI, then create the deterministic archive directly from
this chart. Publication is private by default; do not add `--public` for a live
robustness run.

```sh
cd /path/to/1ctl
go build -o bin/1ctl ./cmd/...

bin/1ctl package create \
  --chart examples/marketplace-stirling-pdf \
  --output /tmp/stirling-pdf-helm.tar.gz
bin/1ctl -o json package publish /tmp/stirling-pdf-helm.tar.gz
bin/1ctl -o json package list
bin/1ctl -o json package status <release-id>

bin/1ctl marketplace deploy <marketplace-id> stirling-pdf-live
```

The deploy command is intentionally not run by this example. It requires an
authenticated active organization and a backend that projects the reserved
runtime values.
