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
# View current organization
1ctl org current

# Switch to a different organization (for multi-org users)
1ctl org switch --org-id=<organization-uuid>

# Check authentication status (shows current org)
1ctl auth status
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
