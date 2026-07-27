# WordPress + MariaDB Helm marketplace package

This self-contained chart is a production-shaped ARM64 marketplace package:
WordPress and MariaDB use immutable images, generated runtime credentials, two
Services, two persistent volumes, readiness/startup/liveness probes, and a
platform-selected HTTPRoute or Ingress. It deliberately omits phpMyAdmin.

The `satusky` values object is reserved for the backend. Do not pass it to
`1ctl` or store credential values in this repository. The backend must inject
the declared `databasePassword` and `databaseRootPassword` credentials at
render time.

## Validate locally

Use throwaway non-production values only to exercise Helm's required runtime
fields; they are not chart defaults and must never be published as secrets.

```bash
cd examples/marketplace-wordpress

helm lint chart \
  --set-string satusky.appName=wordpress-lint \
  --set-string satusky.namespace=lint \
  --set-string satusky.domainName=wordpress-lint.example.test \
  --set satusky.gatewayMode=true \
  --set-string satusky.gatewayName=satusky-gateway \
  --set-string satusky.gatewayNamespace=gateway-system \
  --set-string satusky.cloudflareTunnelTarget=lint.example.test \
  --set-string satusky.credentials.databasePassword=lint-placeholder \
  --set-string satusky.credentials.databaseRootPassword=lint-placeholder

helm template wordpress-lint chart --namespace lint \
  --set-string satusky.appName=wordpress-lint \
  --set-string satusky.namespace=lint \
  --set-string satusky.domainName=wordpress-lint.example.test \
  --set satusky.gatewayMode=true \
  --set-string satusky.gatewayName=satusky-gateway \
  --set-string satusky.gatewayNamespace=gateway-system \
  --set-string satusky.cloudflareTunnelTarget=lint.example.test \
  --set-string satusky.credentials.databasePassword=lint-placeholder \
  --set-string satusky.credentials.databaseRootPassword=lint-placeholder
```

## Create and publish a private package

Build the development CLI first, then create the deterministic archive directly
from the self-contained chart. Publication is private by default; do not add
`--public` for a live robustness run.

```bash
cd /path/to/1ctl
go build -o bin/1ctl ./cmd/...

bin/1ctl package create \
  --chart examples/marketplace-wordpress/chart \
  --output /tmp/wordpress-mariadb-helm.tar.gz
bin/1ctl -o json package publish /tmp/wordpress-mariadb-helm.tar.gz
bin/1ctl -o json package list
bin/1ctl -o json package status <release-id>

bin/1ctl marketplace deploy <marketplace-id> wordpress-mariadb-live
```

The final deploy command is intentionally not run by this example. It requires
an authenticated active organization and backend support for securely projecting
the chart's declared runtime credentials.
