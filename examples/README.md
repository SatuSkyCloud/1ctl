# Tested 1ctl examples

These projects are the checked-in source for the copy-paste workflows in the
[SatuSky guides](https://docs.satusky.com/guides/deploy-backend/). Each guide
states the exact directory, commands, generated resource names, verification,
and cleanup sequence.

## Prerequisites

Use a current `1ctl` build whose generated command contract matches the docs:

```sh
git clone https://github.com/SatuSkyCloud/1ctl.git satusky-1ctl
cd satusky-1ctl
go build -o ./bin/1ctl ./cmd/1ctl
export PATH="$PWD/bin:$PATH"
1ctl --version
1ctl auth status
```

For local backend development:

```sh
1ctl profile use local
1ctl auth status
```

The local profile must point to the running API. Start the API and worker from
the backend repository with `task dev.api` and `task dev.jobs`.

## Guide examples

| Directory | Guide |
| --- | --- |
| `api-with-database` | API with a database |
| `backend` | Deploy a backend |
| `cicd-github-actions` | CI/CD |
| `environment-config` | Environment configuration |
| `frontend` | Deploy a frontend |
| `guide-autoscaling` | Autoscaling |
| `guide-domain-api` | Custom domains |
| `managed-nats` | Managed NATS |
| `microservices` | Microservices |
| `ml-model-api` | ML model API |
| `nodejs-api` | Deploy a Node.js API |
| `python-fastapi` | Deploy a Python API |

The other marketplace and application directories are standalone examples.
Their local README files describe any additional requirements.

The `test` directory is a minimal configuration-parser fixture. It has no
Dockerfile or image and is intentionally not deployable.

## Running an example

Do not guess a generic deploy command. Open the matching guide and run its
complete workflow. Guides deliberately use either a stable example name or a
generated unique name, and some require resources to be created in a specific
order.

For example, the backend guide starts here:

```sh
cd satusky-1ctl/examples/backend
1ctl deploy --config ./satusky.toml --wait
```

Managed-service credentials must stay outside source control. The repository
root ignores `.secrets/`; never print, commit, or place credentials in
`satusky.toml`.

## Verification standard

The guide examples are checked with the applicable combination of:

- language tests and static checks;
- local Podman builds and HTTP assertions;
- cloud multi-architecture builds;
- `1ctl` status, logs, config, secret, release, and deletion commands;
- Kubernetes workload, service, route, autoscaling, and storage inspection;
- docs contract synchronization, tests, Astro type checks, and production
  builds.

Run the repository checks after changing an example:

```sh
cd satusky-1ctl
go test ./...
go run ./cmd/contractgen --check
```
