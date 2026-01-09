# Satusky CLI (1ctl)

A command-line tool for managing containerized applications with Satusky Cloud Platform.

## Installation

### Option 1: Download Binary (Recommended)

Download the latest release for your platform:

#### Linux (64-bit)

```bash
# Get latest version
VERSION=$(curl -s https://api.github.com/repos/SatuSkyCloud/1ctl/releases/latest | jq -r .tag_name)
CLEAN_VERSION=${VERSION#v}

# Download and install
curl -L -o 1ctl.tar.gz "https://github.com/SatuSkyCloud/1ctl/releases/download/$VERSION/1ctl-$CLEAN_VERSION-linux-amd64.tar.gz"
tar -xzvf 1ctl.tar.gz
chmod +x 1ctl
sudo mv 1ctl /usr/local/bin/
rm 1ctl.tar.gz
```

#### macOS

```bash
# Get latest version
VERSION=$(curl -s https://api.github.com/repos/SatuSkyCloud/1ctl/releases/latest | jq -r .tag_name)
CLEAN_VERSION=${VERSION#v}

# Intel Mac
curl -L -o 1ctl.tar.gz "https://github.com/SatuSkyCloud/1ctl/releases/download/$VERSION/1ctl-$CLEAN_VERSION-darwin-amd64.tar.gz"
tar -xzvf 1ctl.tar.gz
chmod +x 1ctl
sudo mv 1ctl /usr/local/bin/
rm 1ctl.tar.gz

# Apple Silicon (M1/M2)
curl -L -o 1ctl.tar.gz "https://github.com/SatuSkyCloud/1ctl/releases/download/$VERSION/1ctl-$CLEAN_VERSION-darwin-arm64.tar.gz"
tar -xzvf 1ctl.tar.gz
chmod +x 1ctl
sudo mv 1ctl /usr/local/bin/
rm 1ctl.tar.gz
```

#### Windows

1. Download the latest release from [SatuSky 1ctl Releases](https://github.com/SatuSkyCloud/1ctl/releases/latest)
2. Extract the zip file
3. Rename the executable to `1ctl.exe`
4. Add to your PATH

### Option 2: Build from Source

Requires Go 1.21 or higher:

```bash
git clone https://github.com/satuskycloud/1ctl.git
cd 1ctl
task build
```

## Usage on GitHub Actions

```yaml
name: Deploy App to SatuSky
on:
  push:
    branches:
      - main
  workflow_dispatch:

concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

env:
  SATUSKY_API_KEY: ${{ secrets.SATUSKY_API_KEY }}
  CPU_REQUEST: 100m
  MEMORY_REQUEST: 6Mi

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Fetch latest 1ctl version
        run: |
          VERSION=$(curl -s https://api.github.com/repos/SatuSkyCloud/1ctl/releases/latest | jq -r .tag_name)
          [[ -z "$VERSION" || "$VERSION" == "null" ]] && exit 1
          echo "SATUSKY_CLI_VERSION=$VERSION" >> $GITHUB_ENV
          echo "CLEAN_VERSION=${VERSION#v}" >> $GITHUB_ENV

      - name: Setup 1ctl cache
        uses: actions/cache@v4
        with:
          path: /usr/local/bin/1ctl
          key: ${{ runner.os }}-1ctl-${{ env.SATUSKY_CLI_VERSION }}

      - name: Install 1ctl if not cached
        if: steps.cache.outputs.cache-hit != 'true'
        run: |
          curl -L -o 1ctl.tar.gz "https://github.com/SatuSkyCloud/1ctl/releases/download/${{ env.SATUSKY_CLI_VERSION }}/1ctl-${{ env.CLEAN_VERSION }}-linux-amd64.tar.gz"
          tar -xzvf 1ctl.tar.gz
          chmod +x 1ctl
          sudo mv 1ctl /usr/local/bin/
          rm 1ctl.tar.gz

      - name: Deploy app to Satusky
        run: |
          1ctl auth login
          1ctl deploy create --cpu ${{ env.CPU_REQUEST }} --memory ${{ env.MEMORY_REQUEST }} \
           --env DATABASE_URL=postgresql://postgres:postgres@localhost:5432/postgres \
           --env SECRET_KEY=secret-key-hahahaha \
           --env HELLO_WORLD=hello-world-hahahaha \
           --env GOODBYE_WORLD=goodbye-world-hahahaha
```

## Quick Start

1. Get your API token from [Satusky Control Panel](https://cloud.satusky.com/token)

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

3. Deploy your first application:

```bash
# Navigate to your project directory with a Dockerfile
cd your-project

# Deploy the application
1ctl deploy create --cpu=1 --memory=512Mi --project=myproject
```

## Usage Examples

### Deployments

```bash
# Create a deployment
1ctl deploy create --cpu=2 --memory=512Mi --domain=example.com --project=myproject

# List deployments
1ctl deploy list

# Get deployment info
1ctl deploy info --deployment-id=123
```

### Services

```bash
# Create a service
1ctl service create --deployment-id=123 --name=myapp --port=8080 --project=test-genesis-org

# List services
1ctl service list

# Delete a service
1ctl service delete --service-id=456
```

### Secrets

```bash
# Create a secret
1ctl secret create --deployment-id=123 --name=mysecret --env="KEY1=value1" --env="KEY2=value2" --project=test-genesis-org

# List secrets
1ctl secret list
```

### Environment Variables

```bash
# Create environment variables
1ctl env create --deployment-id=123 --name=myenv --env="DB_HOST=localhost" --env="DB_PORT=5432" --project=test-genesis-org

# List environments
1ctl env list
1ctl env list --deployment-id=123
1ctl env list --project=test-genesis-org

# Delete environment
1ctl env delete --env-id=789
```

### Ingress/DNS

```bash
# Create ingress
1ctl ingress create --deployment-id=123 --domain=myapp.example.com --custom-dns=true

# List ingress rules
1ctl ingress list

# Delete ingress
1ctl ingress delete --ingress-id=789
```

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
1ctl org team remove <org-user-id>
```

### Credits & Billing

```bash
# View credit balance
1ctl credits balance

# View transaction history
1ctl credits transactions --limit 10

# View machine usage
1ctl credits usage --days 7

# Initiate a top-up
1ctl credits topup --amount 100

# Manage invoices
1ctl credits invoices
1ctl credits invoices get <invoice-id>
1ctl credits invoices download <invoice-id> --output invoice.pdf
1ctl credits invoices generate --start-date 2025-01-01 --end-date 2025-01-31
```

### Storage (S3)

```bash
# List storage configurations
1ctl storage list

# Get storage details
1ctl storage get <storage-id>

# Create storage
1ctl storage create --name my-storage --type s3 --size 10Gi

# Delete storage
1ctl storage delete <storage-id>

# Bucket operations
1ctl storage buckets
1ctl storage buckets create --name my-bucket
1ctl storage buckets delete <bucket-name>

# File operations
1ctl storage files <storage-id>
1ctl storage upload <storage-id> ./myfile.txt
1ctl storage download <object-id> --output ./downloaded.txt
1ctl storage presign <storage-id> --file myfile.txt --expires 3600

# Usage info
1ctl storage usage <storage-id>
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

### GitHub Integration

```bash
# Check GitHub connection status
1ctl github status

# Connect GitHub account
1ctl github connect

# Disconnect GitHub account
1ctl github disconnect

# List synced repositories
1ctl github repos
1ctl github repos --page 1 --limit 20

# Sync repositories from GitHub
1ctl github repos sync

# Get repository details
1ctl github repos get --repo-id <repo-id>

# GitHub App installation
1ctl github installation info
1ctl github installation revoke
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

# Export audit logs
1ctl audit export --format json --output audit.json
```

### Talos Configuration

```bash
# Generate Talos configuration
1ctl talos generate --machine-id <id> --cluster-name my-cluster --role worker

# Apply configuration to a machine
1ctl talos apply --machine-id <id> --config-file talos.yaml

# View configuration history
1ctl talos history <machine-id>

# View network info
1ctl talos network <machine-id>
```

### Admin Operations (Super-admin only)

```bash
# Machine usage management
1ctl admin usage unbilled
1ctl admin usage machine <machine-id>
1ctl admin usage bill <usage-id>

# Credits management
1ctl admin credits add <org-id> --amount 100 --description "Bonus"
1ctl admin credits refund <org-id> --amount 50 --description "Refund"

# View all namespaces
1ctl admin namespaces

# View cluster roles
1ctl admin cluster-roles

# Cleanup resources
1ctl admin cleanup --label app=test --namespace my-ns
```

### Machines

```bash
# List machines
1ctl machine list

# Get machine info
1ctl machine info --machine-name=my-machine
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
