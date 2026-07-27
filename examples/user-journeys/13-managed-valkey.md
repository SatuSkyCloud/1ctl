# Deploying and Operating Managed Valkey

Use `1ctl valkey` to deploy Valkey `9.1.1` with the platform-pinned official
Valkey chart `0.11.0`. The service and its credentials belong to the active
organization namespace.

## Prerequisites and context

Create or select a profile, authenticate, and confirm the organization whose
namespace should own the service:

```bash
1ctl profile create --url https://api.satusky.com/v1/cli prod
1ctl profile use prod
1ctl auth login --token <api-token>
1ctl org current

# If necessary:
1ctl org switch <organization-name-or-id>
```

All examples accept a storage UUID instead of the service name. Add
`--output json` before `valkey` for machine-readable output.

## Create a service

The standalone defaults are one instance, an `8Gi` persistent volume, AOF with
`everysec` fsync, the `allkeys-lru` eviction policy, 75% of the memory limit for
Valkey data, and Prometheus metrics:

```bash
1ctl valkey create sessions \
  --memory 1Gi \
  --cpu 500m
```

By default, the platform intersects live Kubernetes Nodes with commercially
eligible machine records, selects a machine, and persists its ID with the
service. To choose a specific eligible machine:

```bash
1ctl valkey create sessions-local \
  --machine-id 89905f5f769867452a7bd6c7505ab34d \
  --memory 1Gi \
  --cpu 500m
```

The machine must be Ready, schedulable, monetized, verified, reconciled as
rentable capacity, and open for marketplace admission. Reconciliation fails
closed if the persisted machine stops satisfying those requirements; it does
not silently move stateful data to another machine. Persistent standalone
services use Kubernetes `Recreate` rollout strategy so a ReadWriteOnce volume
is detached before the replacement pod starts.

For a read-heavy workload, deploy one primary plus two read replicas:

```bash
1ctl valkey create shared-cache \
  --topology replicated \
  --instances 3 \
  --storage-size 16Gi \
  --memory 2Gi \
  --maxmemory-policy allkeys-lru
```

Replicated topology provides a primary endpoint and a read-only endpoint, but
does **not** provide automatic failover or primary promotion. All instances
currently share the service's one persisted machine placement, so replicated
topology is not machine-level high availability.

## Status and private connectivity

```bash
1ctl valkey list
1ctl valkey status sessions
1ctl valkey credentials sessions
```

Credentials include `valkey://` URIs for private ClusterIP DNS names on port
`6379`. They are intended for application workloads in the same Kubernetes
namespace; there is no public endpoint. Store the URI in the application's
secret configuration rather than source control or `satusky.toml`.

## Safe updates

The CLI only exposes settings the platform can reconcile safely:

```bash
# Scale a replicated service and increase its resources
1ctl valkey update shared-cache \
  --instances 5 \
  --cpu-request 500m \
  --cpu 1 \
  --memory-request 2Gi \
  --memory 4Gi

# Change durability, eviction, memory headroom, or metrics
1ctl valkey update sessions \
  --append-only \
  --append-fsync everysec \
  --maxmemory-policy allkeys-lfu \
  --maxmemory-percent 75 \
  --metrics
```

Replicated services support 2–10 total instances. Topology, persistence,
storage size/class, service identity, and platform chart/image pins cannot be
changed in place.

## ACL users and password rotation

Custom users use platform-owned `admin`, `read_write`, or `read_only` presets.
Key and channel patterns are optional globs; do not include raw `~` or `&` ACL
prefixes:

```bash
1ctl valkey users create sessions cache-api \
  --preset read_write \
  --key-pattern 'cache:*' \
  --channel-pattern 'events:*'

1ctl valkey users list sessions

1ctl valkey users update sessions cache-api \
  --preset read_only \
  --key-pattern 'cache:public:*'
```

The generated password from `users create` is returned once. Save it
immediately. Password rotation invalidates clients using the previous password
and prompts for confirmation unless `--yes` is supplied:

```bash
1ctl valkey users rotate-password sessions cache-api --yes
1ctl valkey rotate-credentials sessions --yes
```

Each rotated password is also returned once. If the API reports that the
rolling restart is pending, keep the password, check `valkey status`, and wait
before reconnecting. The `default` and `replication` users are protected system
users; manage the default password with `rotate-credentials`.

Delete a custom user when it is no longer needed:

```bash
1ctl valkey users delete sessions cache-api --yes
```

## Metrics, logs, and lifecycle operations

```bash
1ctl valkey metrics sessions
1ctl valkey logs sessions --tail 200

# Rolling workload restart
1ctl valkey restart sessions

# Re-apply the durable managed configuration
1ctl valkey reconcile sessions

# Permanently remove the service and its data
1ctl valkey delete sessions --yes
```

Metrics are best effort; an empty value means the exporter has not produced
that series yet. Log tails are bounded to 2,000 lines and 1 MiB by the platform.

## Current limitations

- No automatic failover, Sentinel, or Cluster topology.
- No automatic cross-machine evacuation; placement remains pinned until a
  managed migration workflow is implemented.
- No public endpoint or external network exposure.
- No managed backup or restore workflow.
- No raw `CONFIG` or ACL command execution.
- No TLS endpoint yet.

TLS is intentionally not exposed because official chart `0.11.0` requires a
pre-provisioned Kubernetes Secret containing X.509 certificates and keys. The
platform does not yet own certificate issuance, renewal, rotation, or trust
distribution, so enabling the chart option would not provide a complete managed
TLS lifecycle.
