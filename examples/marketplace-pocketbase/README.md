# PocketBase marketplace Helm example

This chart packages a single PocketBase instance for Satusky. PocketBase stores
its SQLite database on a 5 GiB `ceph-block` `ReadWriteOnce` volume and uses a
`Recreate` deployment so two PocketBase processes never access the database at
the same time.

The workload is intentionally pinned to the supplied immutable image digest and
to `arm64` nodes. `/pb_public` and `/pb_hooks` are empty for each Pod lifetime;
only `/pb_data` is persistent. The Satusky control plane supplies the reserved
`satusky` values at deployment time. The chart does not create or render
credentials or Kubernetes Secrets.

## Package and deploy

Run these commands from this directory:

```sh
1ctl package create --chart . --output pocketbase-0.1.0.tar.gz
1ctl package publish pocketbase-0.1.0.tar.gz
1ctl marketplace deploy pocketbase pocketbase
```

`package publish` is private by default. Use the marketplace ID returned by the
publish command in place of `pocketbase` if the published package is assigned a
different ID.

## Routing

When the runtime sets `satusky.gatewayMode` to `true`, the chart emits an
`HTTPRoute` attached to `satusky.gatewayName` in
`satusky.gatewayNamespace`. Otherwise it emits an nginx `Ingress`. Both modes
route `satusky.domainName` to PocketBase on port 8090. The external-dns target
annotation is emitted only when `satusky.cloudflareTunnelTarget` is non-empty.
