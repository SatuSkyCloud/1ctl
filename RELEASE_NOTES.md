# Release Notes

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