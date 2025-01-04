# Satusky CLI (1ctl)

A command-line tool for managing containerized applications with Satusky Cloud Platform.

## Installation

### Option 1: Download Binary (Recommended)

Download the latest release for your platform:

#### Linux (64-bit)
```bash
curl -LO https://github.com/satuskycloud/1ctl/releases/latest/download/1ctl-linux-amd64
chmod +x 1ctl-linux-amd64
sudo mv 1ctl-linux-amd64 /usr/local/bin/1ctl
```

#### macOS
```bash
# Intel Mac
curl -LO https://github.com/satuskycloud/1ctl/releases/latest/download/1ctl-darwin-amd64
chmod +x 1ctl-darwin-amd64
sudo mv 1ctl-darwin-amd64 /usr/local/bin/1ctl

# Apple Silicon (M1/M2)
curl -LO https://github.com/satuskycloud/1ctl/releases/latest/download/1ctl-darwin-arm64
chmod +x 1ctl-darwin-arm64
sudo mv 1ctl-darwin-arm64 /usr/local/bin/1ctl
```

#### Windows
1. Download [1ctl-windows-amd64.exe](https://github.com/satuskycloud/1ctl/releases/latest/download/1ctl-windows-amd64.exe)
2. Rename to `1ctl.exe`
3. Add to your PATH

### Option 2: Build from Source
Requires Go 1.21 or higher:
```bash
git clone https://github.com/satuskycloud/1ctl.git
cd 1ctl
task build
```

## Usage on GitHub Actions

```yaml
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Download and install 1ctl
        run: |
          curl -L -o 1ctl https://github.com/satuskycloud/1ctl/releases/latest/download/1ctl-linux-amd64
          chmod +x 1ctl
          sudo mv 1ctl /usr/local/bin/
      
      - name: Deploy app to Satusky
        run: |
          1ctl auth login
          1ctl deploy --cpu 100m --memory 6Mi
        env:
          SATUSKY_API_KEY: ${{ secrets.SATUSKY_API_KEY }}
```

## Quick Start

1. Get your API token from [Satusky Control Panel](https://cloud.satusky.com/token)

2. Authenticate:
```bash
1ctl auth login --token=your_api_token

# or using environment variable
export SATUSKY_API_KEY=your_api_token
1ctl auth login

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