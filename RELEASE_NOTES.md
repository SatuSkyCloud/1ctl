# Release Notes

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