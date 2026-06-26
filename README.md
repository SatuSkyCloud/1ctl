# Satusky CLI (1ctl)

A command-line tool for managing containerized applications with Satusky Cloud Platform.

## Installation

### Homebrew (macOS/Linux)

```bash
brew install SatuSkyCloud/tap/satuctl
```

### Shell script (Linux/macOS)

```bash
curl -sSL https://raw.githubusercontent.com/SatuSkyCloud/1ctl/main/install.sh | bash
```

### Windows

Download from [Releases](https://github.com/SatuSkyCloud/1ctl/releases/latest), extract, and add to PATH.

### Build from source

#### Local development build

A standard build connects to `api.satusky.com` by default:

```bash
git clone https://github.com/satuskycloud/1ctl.git
cd 1ctl
go build -o 1ctl ./cmd/...
```

To point the binary at a local API server, override the default URLs at compile time via ldflags:

```bash
go build \
  -ldflags "-X '1ctl/internal/config.defaultAPIURL=http://localhost:8080/v1/cli' \
            -X '1ctl/internal/config.defaultDockerUploadURL=http://localhost:3000'" \
  -o 1ctl ./cmd/...
```

Alternatively, override at runtime with environment variables (no rebuild needed):

```bash
export SATUSKY_API_URL=http://localhost:8080/v1/cli
export SATUSKY_DOCKER_API_URL=http://localhost:3000
./1ctl deploy ...
```

#### Production build (CI/CD)

Production releases are built by GoReleaser. Merges to `main` automatically read the top `RELEASE_NOTES.md` version, create the matching semver tag, and publish the release. Pull request CI runs Linux tests, linting, security scanning, and one Linux amd64 sanity build; the tag-triggered release pipeline owns the full cross-platform binary matrix.

Manual/emergency releases can still be triggered by pushing a semver tag:

```bash
git tag v0.X.Y && git push origin v0.X.Y
```

Do **not** run GoReleaser locally for production releases — the pipeline requires `HOMEBREW_TAP_TOKEN` and other CI secrets.

## Usage on GitHub Actions

```yaml
steps:
  - uses: actions/checkout@v4

  - uses: SatuSkyCloud/setup-1ctl@v1

  - name: Deploy app
    env:
      SATUSKY_API_KEY: ${{ secrets.SATUSKY_API_KEY }}
    run: |
      1ctl auth login
      1ctl deploy --cpu 100m --memory 256Mi \
        --env DATABASE_URL=${{ secrets.DATABASE_URL }}
```

## Quick Start

1. Get your API token from the SatuSky Control Panel at `https://cloud.satusky.com/<org-id>/token`

2. Authenticate:

```bash
1ctl auth login --token=your_api_token

# or using environment variable
export SATUSKY_API_KEY=your_api_token
1ctl auth login

# check authentication status (includes org info)
1ctl auth status

# logout
1ctl auth logout
```

3. Run the interactive wizard (new in v0.8.0):

```bash
# Detects your project runtime, suggests defaults, writes satusky.toml
cd your-project
1ctl launch

# Or skip the wizard and write satusky.toml yourself, then:
1ctl deploy
```

> **Memory unit suffix is required** as of v0.8.0: use `--memory 512Mi`, never `--memory 512`. Bare numbers are parsed by Kubernetes as bytes and silently OOMKill the pod.

## Usage Examples

### Deployments

```bash
# Basic deployment
1ctl deploy --cpu-request 250m --cpu-limit 1 --memory 512Mi

# Deploy to managed cloud, targeting a specific zone
1ctl deploy --cpu-request 250m --cpu-limit 1 --memory 1Gi --zone my-kul-1b

# Deploy with a custom domain (Let's Encrypt picked automatically for non-*.satusky.com hosts)
1ctl deploy --cpu-request 250m --cpu-limit 1 --memory 1Gi --domain example.com

# BYOA: deploy to one of YOUR machines, by name
1ctl deploy --cpu-request 250m --cpu-limit 1 --memory 1Gi --machine my-machine-1

# BYOA: deploy to all your machines labelled satusky.com/production
1ctl deploy --cpu-request 250m --cpu-limit 1 --memory 1Gi --machine-tag production

# Deploy a pre-built image (skips local Docker build and push)
1ctl deploy --cpu-request 250m --cpu-limit 1 --memory 512Mi --image registry.satusky.com/satusky-container-registry/myapp:abc1234

# Deploy with rolling update strategy (default: 25% max surge, 25% max unavailable)
1ctl deploy --cpu-request 250m --cpu-limit 1 --memory 1Gi --strategy rolling --rolling-max-surge 1 --rolling-max-unavailable 0

# Deploy with recreate strategy (stops all pods before starting new ones)
1ctl deploy --cpu-request 250m --cpu-limit 1 --memory 1Gi --strategy recreate

# Wait for TCP dependencies to be ready before the app starts
1ctl deploy --cpu-request 250m --cpu-limit 1 --memory 1Gi --wait-for postgres:5432 --wait-for redis:6379

# Block until pods are Running (5min default timeout)
1ctl deploy --cpu-request 250m --cpu-limit 1 --memory 1Gi --wait

# JSON output (global flag — works on deploy, env, secret, machine, token, ingress, service,
# credits, audit, notifications, pricing, cluster, domain, postgres, issuer, volumes)
1ctl --output json deploy list | jq '.[] | select(.status == "Running")'

# List deployments (NAME column shows the app label as of v0.8.0)
1ctl deploy list

# Get deployment info
1ctl deploy get --deployment-id=123

# Check deployment status
1ctl deploy status --deployment-id=123 --watch

# Open the deployed app URL in your browser
1ctl deploy open --deployment-id=123

# Scale an existing deployment without rebuilding
1ctl deploy scale --deployment-id=123 --replicas 4

# Roll back to the previous release
1ctl deploy rollback --deployment-id=123

# Rolling restart (pick up new env / config without redeploying the image)
1ctl deploy restart --deployment-id=123

# Tear down a deployment (prompts for confirmation; use --yes to skip)
1ctl deploy delete --deployment-id=123 --yes
```

> **Default targeting is managed cloud.** Even if you have registered machines, `1ctl deploy` (with no `--machine*` flag) goes to the marketplace. To use your own hardware pass `--machine` or `--machine-tag` explicitly. This changed in v0.8.0.

### High Availability (PDB, HPA, VPA)

```bash
# Deploy with HPA auto-scaling
1ctl deploy --cpu 2 --memory 1Gi --hpa --hpa-max-replicas 5

# Deploy with VPA recommendations (read-only mode)
1ctl deploy --cpu 2 --memory 1Gi --vpa --vpa-mode Off

# Deploy with VPA resource bounds
1ctl deploy --cpu 2 --memory 1Gi --vpa --vpa-mode Initial \
  --vpa-min-cpu 100m --vpa-max-cpu 4 \
  --vpa-min-memory 128Mi --vpa-max-memory 8Gi

# Deploy with PDB for disruption protection
1ctl deploy --cpu 2 --memory 1Gi --replicas 3 --pdb --pdb-type percent --pdb-percent 60

# Deploy with fixed PDB minimum
1ctl deploy --cpu 2 --memory 1Gi --replicas 3 --pdb --pdb-type fixed --pdb-min-available 2

# Combined: HPA + VPA in Off mode (scaling + recommendations)
1ctl deploy --cpu 2 --memory 1Gi --hpa --hpa-cpu-target 70 --vpa --vpa-mode Off

# Manual replica count
1ctl deploy --cpu 2 --memory 1Gi --replicas 3
```

**Note:** PDB auto-enables when replicas > 1. HPA and VPA with mode `Auto` cannot be used together.

<!-- The `1ctl service` command was hidden from `--help` in v0.8.0.
     Kubernetes Services are now a `deploy` implementation detail; the
     `deploy` orchestrator creates and updates the Service automatically.
     The command is still callable for scripts that depend on it. -->


### Secrets

```bash
# Create a secret
1ctl secret create --deployment-id=123 --name=mysecret --env="KEY1=value1" --env="KEY2=value2" --project=test-genesis-org

# List secrets
1ctl secret list
```

### Environment Variables

```bash
# Create or merge env vars (upsert — keys you don't pass are preserved)
1ctl env create --deployment-id=123 --env="DB_HOST=localhost" --env="DB_PORT=5432"

# List environments
1ctl env list
1ctl env list --deployment-id=123

# Remove a single key
1ctl env unset --deployment-id=123 --key=DB_HOST
```

> The wholesale `env delete` was removed in development; use `env unset --key=<name>` for per-key removal. Same change applies to `secret`.

### Custom Domains

```bash
# Attach a custom domain to an app (resolves deployment/service/namespace internally)
1ctl domains add app.example.com --app myapp

# *.satusky.com hostnames use the platform-managed wildcard cert
1ctl domains add foo.satusky.com --app myapp

# List domains in the current organization
1ctl domains list

# Show DNS / TLS status for a domain
1ctl domains check app.example.com

# Detach a domain (refuses cross-app removal even when the domain matches)
1ctl domains delete app.example.com --app myapp --yes
```

> **`1ctl ingress` and `1ctl issuer` were hidden from `--help` in v0.8.0.** Custom-domain workflows go through `1ctl domains`, which resolves IDs internally from `--app <name>` — no more passing deployment / service UUIDs by hand. The hidden commands still work for scripts that depend on them.

### Organizations (Multi-Tenant)

```bash
# List all organizations
1ctl org list

# View current organization
1ctl org current

# Switch to a different organization (for multi-org users)
1ctl org switch --org-id=<organization-uuid>
1ctl org switch --org-name=my-org

# Create a new organization
1ctl org create --name "My Organization" --description "Description"

# Delete an organization
1ctl org delete <org-id>

# Team management
1ctl org team list
1ctl org team add --email user@example.com --role member
1ctl org team role <org-user-id> --role admin
1ctl org team delete <org-user-id>
```

### Credits & Billing

```bash
# View credit balance
1ctl credits balance

# View transaction history
1ctl credits transactions --limit 10

# View machine usage
1ctl credits usage --days 7

```

> Top-up, invoices, auto-topup, and billing notifications were moved from the CLI to the SatuSky Control Panel (web UI). The CLI keeps read-only visibility (`balance` / `transactions` / `usage`).

### Managed Postgres

`1ctl postgres` manages CNPG-backed Postgres clusters exposed by the SatuSky backend storage API.

```bash
# See available Kubernetes storage classes
1ctl postgres storage-classes

# Create a managed Postgres cluster
1ctl postgres create app-db \
  --database app \
  --user app \
  --storage-class local-path \
  --storage-size 10Gi

# List, inspect, and watch readiness
1ctl postgres list
1ctl postgres get <storage-id>
1ctl postgres status <storage-id>

# Retrieve credentials or connect with psql
1ctl postgres credentials <storage-id>
1ctl postgres connect <storage-id>

# Forward a local port to the cluster service
1ctl postgres proxy <storage-id> --local-port 15432

# Manage database users
1ctl postgres users list <storage-id>
1ctl postgres users create <storage-id> reporting --createdb
1ctl postgres users delete <storage-id> reporting --yes

# Manage external access rules
1ctl postgres firewall list <storage-id>
1ctl postgres firewall add <storage-id> --cidr 203.0.113.10/32 --description "office"
1ctl postgres firewall disable <storage-id> <rule-id>

# Re-apply resources or destroy the cluster
1ctl postgres redeploy <storage-id>
1ctl postgres delete <storage-id> --yes
```

### Logs

```bash
# View deployment logs
1ctl logs --deployment-id <deployment-id>

# Stream logs in real-time
1ctl logs --deployment-id <deployment-id> --follow

# View log statistics
1ctl logs --deployment-id <deployment-id> --stats

# Limit number of lines
1ctl logs --deployment-id <deployment-id> --tail 50
```

### Notifications

```bash
# List notifications
1ctl notifications list
1ctl notifications list --unread --limit 10

# Get unread count
1ctl notifications count

# Mark notifications as read
1ctl notifications read --id <notification-id>
1ctl notifications read --all

# Delete a notification
1ctl notifications delete --id <notification-id>
```

### Marketplace

```bash
# Browse marketplace apps
1ctl marketplace list
1ctl marketplace list --limit 10 --sort popularity

# Get app details
1ctl marketplace get <marketplace-id>

# Deploy a marketplace app
1ctl marketplace deploy <marketplace-id> --name my-wordpress \
  --hostname my-machine --cpu 2 --memory 4Gi \
  --storage-size 20Gi
```

### User Profile

```bash
# View current user profile
1ctl user me

# Update profile
1ctl user update --name "New Name" --email "new@email.com"

# Change password (interactive)
1ctl user password

# View permissions
1ctl user permissions

# Revoke all sessions
1ctl user sessions revoke
```

### API Tokens

```bash
# List API tokens
1ctl token list

# Create a new token
1ctl token create --name "CI/CD Token" --expires 90

# Get token details
1ctl token get <token-id>

# Enable/disable a token
1ctl token enable <token-id>
1ctl token disable <token-id>

# Delete a token
1ctl token delete <token-id>
```

### Audit Logs

```bash
# List audit logs
1ctl audit list
1ctl audit list --limit 20 --action create --user <user-id>

# Get audit log details
1ctl audit get <log-id>
```

> Audit export was moved to the SatuSky Control Panel (web UI). The CLI keeps list + get.

<!-- `1ctl talos` and `1ctl admin` were removed in the v0.7.x cleanup
     commit (f07d3f8). They were operator-facing surfaces moved to
     dedicated internal tooling. The README sections are kept out of
     this user-facing doc so the public command list stays accurate. -->

### Machines

```bash
# List machines
1ctl machine list

# Get machine info
1ctl machine info --machine-name=my-machine
```

### Shell Completion

Auto-complete commands, flags, and app names with TAB — works across zsh, bash, fish, and PowerShell:

```bash
# One command to install — auto-detects your shell
1ctl completion install
```

This installs a script to the right location for your shell and prints the one config line to add to your `~/.zshrc` / `~/.bashrc`. Completions auto-update when `1ctl` changes — no re-install needed.

Manual generation (if you prefer):
```bash
# Zsh (recommended: fpath, lazy-loaded)
1ctl completion zsh > ~/.zsh/completions/_1ctl
echo 'fpath=(~/.zsh/completions $fpath)' >> ~/.zshrc
rm -f ~/.zcompdump && compinit

# Bash
1ctl completion bash > ~/.bash_completion.d/1ctl
echo 'source ~/.bash_completion.d/1ctl' >> ~/.bashrc

# Fish (auto-loaded from ~/.config/fish/completions/)
1ctl completion fish > ~/.config/fish/completions/1ctl.fish

# PowerShell
1ctl completion powershell >> $PROFILE
```

### Help & Version

```bash
# Get help
1ctl --help
1ctl deploy --help

# View version
1ctl --version
```

## Contributing guide

See [CONTRIBUTING.md](CONTRIBUTING.md)
