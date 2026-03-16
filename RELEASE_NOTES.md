# Release Notes

## Version 0.5.12 (16-03-2026)

### 🚀 Distribution

- **Homebrew tap**: `brew install SatuSkyCloud/tap/satuctl` (installs `1ctl` binary)
- **GitHub Action**: `SatuSkyCloud/setup-1ctl@v1` for CI/CD workflows
- **Install script**: `curl -sSL .../install.sh | bash` for quick installs
- **GoReleaser auto-publish**: Homebrew formula auto-updated on every release
- Simplified README installation and GitHub Actions sections
- Formula named `satuctl` because Homebrew can't handle names starting with a digit

---

## Version 0.5.11 (16-03-2026)

### 🔧 Fixes

- Fixed Homebrew formula class name (Ruby class can't start with a digit)

---

## Version 0.5.10 (16-03-2026)

### 🚀 Distribution (initial)

- Added Homebrew formula, GitHub Action, and install script
- GoReleaser brews config for auto-publish (class name fix pending)

---

## Version 0.5.9 (16-03-2026)

### 🔧 Improvements

- Internal maintenance release

---

## Version 0.5.8 (15-03-2026)

### 🐛 Bug Fixes

- **`org team list`**: Fix JSON field mapping — was showing blank Name/Email and zero UUID
- **`token list`**: Fix JSON field mapping — ID showed as zero UUID, status always "Disabled"
- **`user me`**: Fix endpoint path (`/auth/me` → `/users/profile`) and response struct
- **`notifications list`**: Handle paginated wrapper response, fix field names (`title` → `subject`)
- **`audit list`**: Handle paginated wrapper response, fix field names (`user_email` → `actor_email`)
- **`machine list`**: Fixed machine `owner_id` in prod DB (was nil UUID)

### 🔧 Requires

- Backend v0.44.4+ (fixes credit route parameter name mismatch)

---

## Version 0.5.7 (14-03-2026)

### 🐛 Bug Fixes

- **Fix CLI authentication**: Send `x-satusky-api-key` header in login requests. Backend's `InternalAccessMiddleware` (deployed in v0.44.0) requires this header for remote CLI access. Without it, `1ctl auth login` fails with 401.

---

## Version 0.5.6 (13-03-2026)

### 🔒 Security

- **Safe integer conversions**: All CLI flag value conversions (`int` → `int32`) now use `SafeInt32()` to prevent potential integer overflow (gosec G115)

### 🔧 Improvements

- Includes all features from v0.5.5 (PDB/HPA/VPA CLI flags)

---

## Version 0.5.5 (12-03-2026)

### ✨ New Features

- **PDB CLI Flags**: Configure PodDisruptionBudgets from the CLI
  - `--pdb` — Enable PDB (auto-enabled when replicas > 1)
  - `--pdb-type` — Strategy: `auto` (default), `fixed`, or `percent`
  - `--pdb-min-available` — Minimum available pods (for type=fixed)
  - `--pdb-percent` — Minimum available percentage (for type=percent)

- **HPA CLI Flags**: Configure HorizontalPodAutoscalers from the CLI
  - `--hpa` — Enable HPA
  - `--hpa-min-replicas` — Minimum replicas (default: 1)
  - `--hpa-max-replicas` — Maximum replicas (default: 10)
  - `--hpa-cpu-target` — Target CPU utilization % (default: 80)
  - `--hpa-memory-target` — Target memory utilization % (0 = disabled)

- **VPA CLI Flags**: Configure VerticalPodAutoscalers from the CLI
  - `--vpa` — Enable VPA
  - `--vpa-mode` — Update mode: `Off` (default), `Initial`, or `Auto`
  - `--vpa-min-cpu`, `--vpa-max-cpu` — CPU resource bounds
  - `--vpa-min-memory`, `--vpa-max-memory` — Memory resource bounds

- **Replica Count Override**: `--replicas` flag to set replica count manually instead of deriving from machine count

### 🔧 Improvements

- **Smart PDB Auto-Enable**: PDB automatically enabled when replicas > 1, even without `--pdb` flag
- **Validation**: HPA + VPA with mode `Auto` rejected (resource scaling conflict), HPA min/max bounds checked, PDB percent range validated (1-100)

### 📋 New Flags Summary

| Flag | Default | Description |
|------|---------|-------------|
| `--replicas` | auto | Manual replica count override |
| `--pdb` | false | Enable PodDisruptionBudget |
| `--pdb-type` | auto | PDB strategy (auto/fixed/percent) |
| `--hpa` | false | Enable HorizontalPodAutoscaler |
| `--hpa-min-replicas` | 1 | HPA minimum replicas |
| `--hpa-max-replicas` | 10 | HPA maximum replicas |
| `--hpa-cpu-target` | 80 | Target CPU % |
| `--vpa` | false | Enable VerticalPodAutoscaler |
| `--vpa-mode` | Off | VPA mode (Off/Initial/Auto) |

### 📚 New Command Usage

```bash
# Deploy with HPA auto-scaling
1ctl deploy --cpu 2 --memory 1Gi --hpa --hpa-max-replicas 5

# Deploy with VPA recommendations (read-only mode)
1ctl deploy --cpu 2 --memory 1Gi --vpa --vpa-mode Off

# Deploy with PDB for high availability
1ctl deploy --cpu 2 --memory 1Gi --replicas 3 --pdb --pdb-type percent --pdb-percent 60

# Combined: HPA + VPA in Off mode (recommendations only)
1ctl deploy --cpu 2 --memory 1Gi --hpa --vpa --vpa-mode Off
```

---

## Version 0.5.4 (12-03-2026)

### ✨ New Features

- **`domain purchase-status`**: New subcommand to poll the status of a pending domain purchase intent
- **`domain contact`**: New subcommand to view saved contact info from the last domain purchase

### 🔧 Improvements

- **Domain purchase flow**: `domain purchase` now initiates a Stripe Checkout session — returns a payment URL and intent ID instead of purchasing directly. Complete payment in browser, then use `domain purchase-status <intent-id>` to confirm.

---

## Version 0.5.3 (12-03-2026)

### 🔧 Improvements

- **GitHub integration removed**: `github` command removed — GitHub deployment feature is discontinued
- **`secret delete`**: Added missing `secret delete --secret-id` subcommand
- **`issuer delete`**: Added missing `issuer delete --issuer-id` subcommand
- **Ingress validation**: `--service-id` now gives a clear error when omitted instead of a confusing UUID parse error

### 🔒 Security

- Fixed gosec G204 false positive in Docker `SaveImage` (correct `// #nosec G204` directive)

---

## Version 0.5.2 (12-03-2026)

### ✨ New Features

- **Mac VM Agent Commands**: Full lifecycle control for Mac mini machines via the Mac agent
  - `machine vm status` — Show current VM state
  - `machine vm start/stop/reboot` — Power control commands
  - `machine vm stop --force` — Force stop with optional grace period
  - `machine vm resize --cpu N --memory N` — Resize VM resources
  - `machine vm apply-config --config-file PATH` — Apply Talos configuration (base64 encoded)
  - `machine vm console [--enable|--disable]` — Toggle console streaming

- **Domain Management**: Full lifecycle for external domain registration and DNS via OpenProvider
  - `domain list/get/create/delete/verify` — Domain lifecycle management
  - `domain check` — Check domain availability
  - `domain search` — Search available domains
  - `domain purchase` — Purchase a domain
  - `domain dns list/create/update/delete` — DNS record management

- **Machine Usage Tracking**: View and calculate usage costs for deployed machines
  - `machine usage list` — List all usage records for current user
  - `machine usage get --usage-id <id>` — Get usage record details
  - `machine usage cost --usage-id <id>` — Calculate cost for a usage record

- **Pricing Configuration**: Browse platform pricing information
  - `pricing list` — List all pricing configurations
  - `pricing get --id <id>` — Get specific pricing config
  - `pricing lookup --region <r> --type <t> --sla <s>` — Look up pricing by region/type/SLA
  - `pricing calculate --machine-ref-id <id> --machine-id <id>` — Calculate machine cost

- **Billing Settings**: Manage auto top-up and notification preferences
  - `credits auto-topup get` — View current auto top-up settings
  - `credits auto-topup set --enabled [--threshold N] [--amount N]` — Configure auto top-up
  - `credits notifications get` — View notification preferences
  - `credits notifications set --low-balance [--email] [--push] [--threshold N]` — Configure notifications

- **Live Log Streaming**: Real-time log streaming via WebSocket
  - `logs stream --deployment-id <id>` — Stream logs using deployment ID
  - `logs stream --namespace <ns> --app <label>` — Stream logs using namespace + app label
  - `logs stream --batch-size N` — Control log lines per batch

### 🔧 Technical Improvements

- **Backend CLI Route Expansion**: Added machine usage, pricing, and billing settings routes to CLI API
- **WebSocket Log Streaming**: Direct WebSocket connection replacing Loki log querying
- **Mac VM Agent Integration**: `POST /machines/:machineId/command` now exposed via CLI routes
- **Machine Model Enhancement**: Added `VMState` and `ConnectionMode` fields to Machine struct
- **New `gorilla/websocket` dependency**: WebSocket client for live log streaming

### 📋 New Commands Summary

| Category | Commands Added |
|----------|---------------|
| Machine VM | 7 subcommands |
| Domain | 9 subcommands + 4 DNS subcommands |
| Machine Usage | 3 subcommands |
| Pricing | 4 subcommands |
| Billing Settings | 4 subcommands |
| Logs Streaming | 1 subcommand |

---

## Version 0.5.1 (13-01-2026)

### ✨ New Features

- **Resource Exhausted Error Handling**: Enhanced error handling for resource quota exceeded scenarios
  - Beautiful formatted error display showing current tier limits
  - Clear guidance on which resources are exhausted (CPU, Memory, Pods, etc.)
  - Automatic tier upgrade suggestions when applicable
  - Support for displaying next tier limits and requirements

- **Tier Info Display**: Added tier information to credits commands
  - `credits balance` now shows current tier and limits
  - Display of highest achieved tier (peak tier)
  - Credits required to reach next tier
  - Current resource limits per tier

### 🔧 Technical Improvements

- **Resource Error Utilities**: New `internal/utils/resource_error.go` module
  - `ParseResourceExhaustedError()` - Parse API error responses for resource exhaustion
  - `FormatResourceExhaustedError()` - Beautiful CLI formatting for resource errors
  - Comprehensive test coverage with `resource_error_test.go`

- **Deploy Command Enhancement**: Improved error handling during deployment
  - Detects resource exhausted errors from API
  - Displays actionable upgrade guidance
  - Shows current vs required resources

- **Credits API Integration**: Enhanced credits balance endpoint integration
  - Added `TierInfo` struct with tier details
  - Support for tier limits display
  - Upgrade path information

---

## Version 0.5.0 (09-01-2026)

### ✨ New Features

- **Credits & Billing Management**: Complete billing and credits integration
  - `credits balance` - View organization credit balance
  - `credits transactions` - View transaction history
  - `credits usage` - View machine usage and costs
  - `credits topup` - Initiate credit top-up via Stripe
  - `credits invoices` - Manage invoices (list, get, download PDF, generate)

- **Storage Management (S3)**: Full S3-compatible object storage support
  - `storage list/get/create/delete` - Manage storage configurations
  - `storage buckets` - Bucket management operations
  - `storage files/upload/download` - File operations
  - `storage presign` - Generate presigned URLs
  - `storage usage` - View storage usage statistics

- **Logs Command**: Deployment log viewing and streaming
  - `logs --deployment-id` - View stored deployment logs
  - `logs --follow` - Stream logs in real-time (WebSocket)
  - `logs --stats` - View log statistics
  - `logs --tail` - Limit number of lines

- **GitHub Integration**: Complete GitHub OAuth and repository management
  - `github status` - Check GitHub connection status
  - `github connect/disconnect` - Manage GitHub account connection
  - `github repos` - List, sync, and get repository details
  - `github installation` - Manage GitHub App installation

- **Notifications**: In-app notification management
  - `notifications list` - List notifications with filtering
  - `notifications count` - Get unread notification count
  - `notifications read` - Mark notifications as read
  - `notifications delete` - Delete notifications

- **Marketplace**: Browse and deploy pre-configured applications
  - `marketplace list` - Browse available apps
  - `marketplace get` - Get app details
  - `marketplace deploy` - Deploy marketplace apps (WordPress, Immich, N8N, etc.)

- **User Profile Management**: Personal account management
  - `user me` - View current user profile
  - `user update` - Update profile (name, email)
  - `user password` - Change password (interactive)
  - `user permissions` - View role and permissions
  - `user sessions revoke` - Revoke all sessions

- **API Token Management**: Manage API tokens programmatically
  - `token list` - List all API tokens
  - `token create` - Create new tokens with expiry
  - `token get` - Get token details
  - `token enable/disable` - Toggle token state
  - `token delete` - Delete tokens

- **Audit Logs**: View organization audit trails
  - `audit list` - List audit logs with filtering
  - `audit get` - Get audit log details
  - `audit export` - Export logs to JSON/CSV

- **Talos Configuration**: Talos Linux machine configuration
  - `talos generate` - Generate Talos configuration
  - `talos apply` - Apply configuration to machines
  - `talos history` - View configuration history
  - `talos network` - View machine network info

- **Admin Operations**: Super-admin management tools
  - `admin usage` - Manage machine usage records
  - `admin credits` - Add/refund organization credits
  - `admin namespaces` - List all Kubernetes namespaces
  - `admin cluster-roles` - List cluster roles
  - `admin cleanup` - Cleanup resources by label

### 🔄 Enhanced Commands

- **Organization Command Enhancements**
  - `org list` - List all user organizations
  - `org create` - Create new organizations
  - `org delete` - Delete organizations
  - `org team list` - List team members
  - `org team add` - Add team members with role
  - `org team role` - Update member roles
  - `org team remove` - Remove team members
  - Enhanced `org switch` with `--org-name` support

### 🔧 Technical Improvements

- **Backend Route Expansion**: Added 70+ new CLI routes to backend
  - Full DI pattern implementation for CLI controllers
  - Split controllers for better separation of concerns
  - Enhanced authentication middleware

- **API Client Architecture**: Modular API client files
  - `api/credits.go` - Credits and billing operations
  - `api/storage.go` - S3 storage operations
  - `api/logs.go` - Log operations
  - `api/github.go` - GitHub integration
  - `api/notifications.go` - Notification operations
  - `api/marketplace.go` - Marketplace operations
  - `api/audit.go` - Audit log operations
  - `api/talos.go` - Talos configuration
  - `api/admin.go` - Admin operations
  - `api/user.go` - User profile operations
  - `api/token.go` - API token operations
  - `api/org.go` - Organization and team management

- **Command Structure**: 11 new command files with consistent patterns
  - Beautiful output formatting with status lines, headers, dividers
  - Comprehensive error handling
  - Flag validation and help text

### 📚 Documentation

- Updated README.md with all new commands and examples
- Added usage examples for:
  - Credits & billing operations
  - Storage management
  - Log streaming
  - GitHub integration
  - Marketplace deployment
  - Team management
  - API token management
  - Admin operations

### 📋 New Commands Summary

| Category | Commands Added |
|----------|---------------|
| Credits | 9 subcommands |
| Storage | 12 subcommands |
| Logs | 1 command with 4 flags |
| GitHub | 10 subcommands |
| Notifications | 5 subcommands |
| Marketplace | 3 subcommands |
| User | 5 subcommands |
| Token | 6 subcommands |
| Audit | 3 subcommands |
| Talos | 4 subcommands |
| Admin | 8 subcommands |
| Org (enhanced) | 7 new subcommands |

---

## Version 0.4.0 (18-12-2025)

### ✨ New Features

- **Multi-Cluster Deployment**: Deploy applications across multiple Kubernetes clusters for high availability
  - New `--multicluster` flag to enable multi-cluster replication
  - New `--multicluster-mode` flag to choose deployment strategy:
    - `active-active`: Both clusters serve traffic simultaneously with geo-routing (ideal for stateless apps)
    - `active-passive`: Primary serves traffic, secondary is standby with automatic failover (ideal for stateful apps)
  - New `--backup-schedule` flag to configure backup frequency: `hourly`, `daily`, `weekly`
  - New `--backup-retention` flag to configure backup retention: `24h`, `72h`, `168h`, `720h`

### 🔄 API Enhancements

- **MulticlusterConfig Model**: Added new configuration model for multi-cluster deployments

  - `Enabled`: Toggle multi-cluster replication
  - `Mode`: Active-Active or Active-Passive deployment strategy
  - `BackupEnabled`: Enable Velero backups for data protection
  - `BackupSchedule`: Cron-based backup scheduling
  - `BackupRetention`: Duration-based backup retention
  - `FailoverEnabled`: Automatic failover on primary cluster failure
  - `RestoreOnFailover`: Automatic restore from backup on failover

- **Deployment Model Update**: Extended `Deployment` struct with `MulticlusterConfig` field

### 🔧 Technical Improvements

- **Orchestrator Enhancement**: Updated deployment orchestration to build and send multi-cluster configuration

  - Automatic cron schedule conversion from friendly names (hourly/daily/weekly)
  - Smart defaults: Active-passive mode automatically enables backup, failover, and restore-on-failover
  - Seamless integration with existing deployment workflow

- **DeploymentOptions Update**: Added multi-cluster fields to deployment options struct
  - `MulticlusterEnabled`: Enable/disable multi-cluster
  - `MulticlusterMode`: Deployment strategy selection
  - `BackupSchedule`: User-friendly schedule selection
  - `BackupRetention`: Retention duration

### 📚 New Command Usage

```bash
# Deploy with multi-cluster enabled (active-passive with daily backups, 7-day retention)
1ctl deploy --cpu 100m --memory 256Mi --multicluster

# Deploy with active-active mode (geo-routing, no backups)
1ctl deploy --cpu 100m --memory 256Mi --multicluster --multicluster-mode active-active

# Deploy with custom backup configuration
1ctl deploy --cpu 100m --memory 256Mi --multicluster \
  --multicluster-mode active-passive \
  --backup-schedule hourly \
  --backup-retention 72h

# Full example with all options
1ctl deploy --cpu 500m --memory 1Gi \
  --machine my-machine-1 --machine my-machine-2 \
  --env DATABASE_URL=postgres://... \
  --multicluster \
  --multicluster-mode active-passive \
  --backup-schedule daily \
  --backup-retention 168h
```

### 📋 Multi-Cluster Configuration Reference

| Flag                  | Default          | Description                            |
| --------------------- | ---------------- | -------------------------------------- |
| `--multicluster`      | `false`          | Enable multi-cluster deployment        |
| `--multicluster-mode` | `active-passive` | Deployment strategy                    |
| `--backup-schedule`   | `daily`          | Backup frequency (hourly/daily/weekly) |
| `--backup-retention`  | `168h`           | Backup retention period                |

### Backup Schedule Mapping

| Schedule | Cron Expression | Description              |
| -------- | --------------- | ------------------------ |
| `hourly` | `0 * * * *`     | Every hour               |
| `daily`  | `0 0 * * *`     | Daily at midnight UTC    |
| `weekly` | `0 18 * * 6`    | Weekly (Sunday 2 AM MYT) |

### Retention Options

| Value  | Duration | Use Case                        |
| ------ | -------- | ------------------------------- |
| `24h`  | 1 Day    | Quick recovery, minimal storage |
| `72h`  | 3 Days   | Short-term retention            |
| `168h` | 7 Days   | Recommended for most apps       |
| `720h` | 30 Days  | Long-term retention             |

### 🔒 Security

- Multi-cluster configuration securely transmitted via existing API authentication
- Backup and failover operations handled by operator with proper RBAC
- Cloudflare Load Balancer hostname auto-generated by backend

---

## Version 0.3.0 (30-09-2025)

### ✨ New Features

- **Multi-Organization Support**: Full support for users belonging to multiple organizations
  - New `org current` command to view current organization context
  - New `org switch` command to switch between organizations
  - Enhanced `auth status` command to display organization ID and namespace
  - Organization context automatically saved and restored across sessions

### 🔄 API Enhancements

- **Multi-Tenant Authentication**: Updated authentication flow to support multi-tenant backend

  - Login now returns and stores complete organization information (ID, name, namespace)
  - Token validation includes organization context
  - All API calls properly scoped to user's current organization
  - Fixed `/api-tokens/list` endpoint to include organization ID parameter

- **Backend Compatibility**: Updated all endpoints to work with multi-tenant backend
  - Fixed `TokenValidate` model types (UUID → string) to match backend response
  - Added `GetUserProfile` API function for retrieving user organization details
  - Simplified login flow to use backend-provided organization data
  - Updated issuer endpoint to use upsert pattern (`/issuers/upsert`)

### 🔧 Technical Improvements

- **Enhanced Context Management**: Improved CLI context storage

  - Added `CurrentOrgID` and `CurrentOrgName` fields to context
  - New helper functions: `GetCurrentOrgID()`, `SetCurrentOrgID()`, `GetCurrentOrgName()`, `SetCurrentOrgName()`
  - New combined setter: `SetCurrentOrganization()` for atomic updates
  - Maintains backward compatibility with existing context files

- **Improved Organization Visibility**: Better user experience for multi-org environments

  - Auth status now shows: Email, Organization, Organization ID, Namespace, Token expiry
  - Deploy command properly uses current organization context by default
  - `--organization` flag allows deploying to specific organization/namespace

- **Comprehensive Test Coverage**: Added tests for all new features
  - 3 new test cases for organization context operations
  - 3 new test cases for org command structure
  - All tests passing with 100% coverage of new code
  - Updated existing tests to accommodate new fields

### 🛠️ Breaking Changes

- **TokenValidate Model Changes**: ID fields changed from `uuid.UUID` to `string` type
  - Affects: `UserID`, `TokenID`, `OrganizationID`
  - Added new fields: `Token`, `Namespace`, `Message`
  - Ensures compatibility with backend multi-tenant implementation

### 📚 New Commands

```bash
# View current organization
1ctl org current

# Switch to different organization
1ctl org switch --org-id <organization-id>

# Enhanced auth status
1ctl auth status
```

### 🔒 Security

- All operations properly scoped to user's organization
- Context file maintains secure 0600 permissions
- Organization switching validates user access

## Version 0.2.2 (08-08-2025)

### ✨ New Features

- **Enhanced Docker Image Upload**: Added support for large Docker image uploads (>500MB and <8GB)

### 🔄 API Enhancements

- **Docker Upload Infrastructure**: Migrated Docker image uploads to dedicated service
  - Previous: `/docker/images/upload` endpoint on main API
  - New: Direct upload to specialized Docker upload service
  - Optimized for handling large binary transfers
  - Supports file sizes from 500MB to 8GB

## Version 0.2.1 (20-06-2025)

### 🐛 Bug Fixes

- **Fixed ingress domain management**: Resolved issue where ingress upsert operations were generating new domains instead of reusing existing ones
  - Enhanced `upsertIngress` function to check for existing ingresses by deployment ID before creating new ones
  - Added proper existing domain name reuse logic to prevent unnecessary domain generation
  - Fixed Kubernetes error "ingress not found" by properly identifying and updating existing ingress resources
  - Improved ingress update flow to use existing domain names and preserve ingress IDs
  - Fixed ingress upsert response parsing to correctly handle backend's string response containing only the ingress ID

### 🔧 Technical Improvements

- **Enhanced Ingress API**: Added `GetIngressByDeploymentID` function to retrieve existing ingresses

  - Leverages existing backend route `/ingresses/deploymentId/:deploymentId`
  - Enables proper lookup of existing ingress resources during deployment updates
  - Simplifies client-side logic by leveraging backend's existing upsert capabilities

- **Improved Deployment Orchestration**: Enhanced ingress handling in deployment process
  - Better separation between new ingress creation and existing ingress updates
  - Consistent domain name handling across deployment scenarios
  - Improved logging and error messaging for ingress operations
  - Returns domain name from backend response to ensure consistency

### 🔄 API Enhancements

- **Ingress Management**: Streamlined ingress upsert operations to work seamlessly with backend logic
  - Client now properly leverages backend's existing ingress detection by namespace and app label
  - Reduced client-side complexity by relying on backend's robust upsert implementation
  - Enhanced error handling and response processing for ingress operations
  - Updated `UpsertIngress` to correctly unmarshal string ingress ID response from backend

## Version 0.2.0 (19-06-2025)

### ✨ Enhanced Command Structure

- **Simplified Deploy Command**: Removed `deploy create` subcommand and moved deployment flags directly to main `deploy` command

  - New usage: `1ctl deploy --cpu 100m --memory 20Mi --env HELLO=BYE --env SAY=WHAT`
  - Subcommands `list`, `get`, and `status` remain available
  - Enhanced validation with clear error messages for required flags

- **Simplified Service Command**: Removed `service create` subcommand and moved service flags directly to main `service` command

  - New usage: `1ctl service --deployment-id=123 --name=myservice --port=8080 --namespace=myorg`
  - Subcommands `list` and `delete` remain available
  - Required flags: `--deployment-id`, `--name`, `--port`

- **Simplified Ingress Command**: Removed `ingress create` subcommand and moved ingress flags directly to main `ingress` command
  - New usage: `1ctl ingress --deployment-id=123 --service-id=456 --app-label=myapp --namespace=myorg --domain=example.com`
  - Subcommands `list` and `delete` remain available
  - Required flags: `--deployment-id`, `--domain`, `--app-label`, `--namespace`

### 🔄 API Enhancements

- **Upsert Endpoints Migration**: Updated all resource creation to use upsert endpoints for idempotent operations

  - **Deployment**: `POST /deployments/upsert/:namespace/:appLabel` (namespace = organization, appLabel = app name)
  - **Service**: `POST /services/upsert/:namespace/:serviceName` (namespace = organization, serviceName = app name)
  - **Ingress**: `POST /ingresses/upsert/:namespace/:appLabel` (namespace = organization, appLabel = app name)

- **API Client Improvements**: Replaced create functions with upsert functions
  - `CreateDeployment` → `UpsertDeployment`
  - `CreateService` → `UpsertService`
  - `CreateIngress` → `UpsertIngress`
  - Maintained same payload structure for backward compatibility

### 🔧 Technical Improvements

- **Enhanced User Experience**: Streamlined command syntax eliminates unnecessary subcommand nesting

  - Direct flag access from main commands improves CLI ergonomics
  - Consistent command patterns across deploy, service, and ingress resources
  - Maintained backward compatibility for list/delete subcommands

- **Comprehensive Test Updates**: Updated all test suites to reflect new command structure

  - Mock API functions updated to use upsert patterns
  - Integration tests migrated to new endpoint structure
  - Command tests updated with new handler functions
  - Maintained test coverage across all functionality

- **Shell Completion Updates**: Updated all shell completion scripts for new command structure

  - **Bash**: Removed obsolete "create" subcommands, updated flag completions
  - **Zsh**: Enhanced completion with new command patterns
  - **Fish**: Updated subcommand and flag completion logic
  - **PowerShell**: Removed deprecated creation commands from completions

- **Deployment Orchestrator**: Updated orchestration logic to use upsert operations

  - Service creation now uses `UpsertService` for idempotent operations
  - Ingress creation now uses `UpsertIngress` for consistent updates
  - Dependency handling updated with new upsert patterns
  - Improved error messages with operation context

- **Enhanced Image Upload Reliability**: Added retry mechanism for Docker image uploads

  - Automatic retry up to 3 attempts on upload failures
  - Exponential backoff strategy (2, 4, 8 seconds between retries)
  - Clear progress indication and error reporting for each attempt
  - Improved resilience against temporary network issues or server errors

- **Enhanced Docker Validation**: Significantly improved Dockerfile validation to support modern Docker features
  - **Multistage Build Support**: Now properly validates `FROM image AS stage_name` syntax
  - **Line Continuation Handling**: Fixed parsing of multi-line commands using backslash (`\`) continuations
  - **Enhanced Image Name Validation**: Updated regex patterns to support registry URLs, namespaces, and case-insensitive matching
  - **COPY --from Syntax**: Added proper validation for `COPY --from=stage_name` commands in multistage builds
  - **Stage Name Validation**: Validates stage names allow alphanumeric characters, underscores, and hyphens
  - **Comprehensive Test Coverage**: Added extensive test cases for all multistage build scenarios

### 🛠️ Breaking Changes

- **Command Structure**: Removed `create` subcommands from `deploy`, `service`, and `ingress` commands
  - Old: `1ctl deploy create --cpu 100m --memory 20Mi`
  - New: `1ctl deploy --cpu 100m --memory 20Mi`
  - Old: `1ctl service create --deployment-id=123 --name=myservice --port=8080`
  - New: `1ctl service --deployment-id=123 --name=myservice --port=8080`
  - Old: `1ctl ingress create --deployment-id=123 --domain=example.com`
  - New: `1ctl ingress --deployment-id=123 --domain=example.com --app-label=myapp --namespace=myorg`

## Version 0.1.8 (17-06-2025)

### ✨ New Features

- **Machine Marketplace Discovery**: Added `machine available` command to browse and filter available machines for rent
  - Comprehensive filtering options: `--region`, `--zone`, `--min-cpu`, `--min-memory`, `--gpu`, `--recommended`, `--pricing-tier`
  - Enhanced display with pricing information, performance metrics, and recommendation indicators
  - Real-time availability status and resource specifications

### 🔧 Technical Improvements

- **Updated Machine Model**: Synchronized frontend Machine model with enhanced backend structure

  - Changed `MachineID` from `uuid.UUID` to `string` type
  - Added new fields: `ID`, `Status`, `LastHealthCheck`, `Recommended`, `ResourceScore`
  - Added performance metrics: `CPUUsagePercent`, `MemoryUsagePercent`, `StorageUsagePercent`, `NetworkUsageGbps`
  - Added hardware features: `HasGPU`, `HasHDD`, `HasNVME`, `NodeType`
  - Added pricing information: `PricingTier`, `HourlyCost`
  - Added reliability metrics: `UptimePercent`, `ResponseTimeMs`, `NetworkMetricsType`

- **Enhanced Hostname Mapping**: Updated deployment logic to use machine IDs instead of machine names

  - Both manual machine selection (`--machine` flag) and automatic selection now use machine IDs
  - Improved deduplication logic based on unique machine IDs
  - Ensures consistent backend integration with machine ID-based deployments

- **Improved Machine Information Display**: Enhanced machine listing with additional useful information
  - Added Status, Node Type, Pricing Tier, and Hourly Cost to machine details
  - Added conditional display of Resource Score and Uptime percentage
  - Separate optimized display format for available machines marketplace

### 🛠️ API Enhancements

- Added `GetAvailableMachines()` API function for fetching monetized machines
- Updated `MachineIDs` struct to use string array instead of UUID array

## Version 0.1.7 (12-06-2025)

### 🐛 Bug Fixes

- **Fixed hostname deduplication for monetized machines**: When multiple machines share the same hostname (e.g., "1"), the system now properly preserves the original hostname instead of incrementing it (e.g., "1" stays "1" instead of becoming "2")
  - Added hostname deduplication logic for owner's machines in automatic selection
  - Added hostname deduplication logic for manually specified machines via `--machine` flag
  - Ensures consistent hostname behavior across both user-owned and monetized machine deployments

### 🔧 Technical Improvements

- **Enhanced versioning system**: Fixed automatic version detection in build process
  - Updated `Taskfile.yml` to automatically detect version from Git tags instead of using hardcoded default
  - Added `task version` command to easily check current version information
  - Version now correctly reflects Git state with format like `v0.1.6-3-g94b0eb5` for commits ahead of tags
  - Improved build-time version injection with commit hash and build date

## Version 0.1.6 (22-03-2025)

### 🔧 Technical Improvements

- Version number fix only, no functional changes

## Version 0.1.5 (22-03-2025)

### 🔧 Technical Improvements

- Updated API endpoints for better resource management:
  - Fix secret and environment creation endpoints from `/create` to `upsert`
- Enforced minimum replica count for deployments (monetized).

## Version 0.1.4 (18-03-2025)

### 🔧 Technical Improvements

- Improved machine allocation system:
  - Automatic machine assignment if no specific hostnames provided
  - System now intelligently selects the most cost-effective machine based on resource requirements
- Enhanced hostname selection logic:
  - Prioritizes user-owned machines first
  - Falls back to monetized machines with automatic selection
  - Improved error handling for machine allocation

## Version 0.1.3 (17-01-2025)

### 🔧 Technical Improvements

- Introduced centralized error handling with `utils.NewError`
- Standardized error formatting across the codebase
- Improved error messages with better context and readability
- Added consistent error wrapping pattern
- Enhanced error handling in cleanup operations

## Version 0.1.2 (13-01-2025)

### 🔧 Technical Improvements

- Updated registry URL to use the new registry

## Version 0.1.1 (04-01-2025)

### 🔒 Security Improvements

- Added safe integer conversion handling to prevent overflows in port and replica configurations
- Enhanced path validation for file operations to prevent directory traversal attacks
- Improved Docker build input validation with tag format checking
- Implemented secure file permission handling (0750 for directories, 0600 for files)
- Added protection against command injection in Docker build operations
- Better error handling in cleanup operations for test utilities

### 🔧 Technical Improvements

- Introduced `SafeInt32` utility function for safe integer conversions
- Added path validation functions in Docker and context operations
- Enhanced error handling in file operations
- Improved input validation for Docker build options

## Version 0.1.0 (31-12-2024)

### 🎉 Genesis Release

First public release of the Satusky CLI (1ctl) with core functionality for managing containerized applications on Satusky Cloud Platform.

### ✨ Features

#### Authentication

- Login with API token support
- Automatic token management and validation
- Session status checking
- Secure logout functionality

#### Machine Management

- List and view machine details

#### Deployment Management

- Create new deployments with customizable resources
- List and view deployment details
- Support for multiple deployment environments
- Real-time deployment status tracking with progress indicators
- Automatic Dockerfile detection and validation

#### Container Management

- Automated Docker image building
- Secure registry authentication
- Support for custom Dockerfiles
- Automatic image versioning and tagging

#### Resource Configuration

- CPU and memory allocation management
- Environment variables management
- Volume management for persistent storage
- Custom domain configuration
- Service port configuration

#### Service Management

- Create and configure services
- Automatic service discovery
- Port mapping and exposure

#### Networking

- Ingress configuration and management
- Custom domain support
- SSL/TLS certificate management
- Domain validation and verification

### 🔧 Technical Improvements

- Color-coded CLI output for better user experience
- Progress spinners for long-running operations
- Structured error handling and user feedback
- Comprehensive input validation
- Secure credential management

### 📚 Documentation

- Basic usage documentation
- Command reference
- Installation instructions
- Resource limits documentation

### 🔒 Security

- Secure token storage
- Encrypted communication
- Input sanitization and validation

### 🐛 Known Issues

- Broken CI (both unit and integration tests are not completed yet)
- Darwin 386 builds are not supported

### 📋 Requirements

- Go 1.21 or higher
- Docker installed and running
- Verified Satusky Control Panel account

For detailed documentation, visit [https://docs.satusky.com/1ctl](https://docs.satusky.com/1ctl)
