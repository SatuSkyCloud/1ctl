# Contributing to 1ctl

Thank you for your interest in contributing to 1ctl! This document provides guidelines and instructions for contributing.

## Development Prerequisites

- Go 1.21 or higher
- Task runner ([Installation Guide](https://taskfile.dev/#/installation))
- Docker (for container builds and testing)
- Git

## Getting Started

1. Fork the repository and clone your fork:
```bash
git clone https://github.com/YOUR_USERNAME/1ctl.git
cd 1ctl
```

2. Add the original repository as upstream:
```bash
git remote add upstream https://github.com/satuskycloud/1ctl.git
```

3. Install dependencies:
```bash
task init
```

## Development Workflow

1. Create a new branch for your changes:
```bash
git checkout -b feature/your-feature-name
```

2. Make your changes and test them:
```bash
# Run all tests
task test

# Run unit tests only
task test:unit

# Run integration tests
task test:integration

# Run linters
task lint
```

3. Build the CLI:
```bash
task build
```

4. Run the CLI:
```bash
task run -- <command>
```

## Code Style

- Follow standard Go code style and conventions
- Run `task format` to format your code
- Run `task lint` to check your code for linting errors
- Write meaningful commit messages following [conventional commits](https://www.conventionalcommits.org/)

## Testing

- Write tests for new features
- Ensure all tests pass before submitting PRs
- Include both unit and integration tests where appropriate
- Use testdata files in `internal/testing` for test fixtures

## Documentation

- Update README.md for new features or changes
- Add godoc comments to exported functions
- Update command help text when adding or modifying commands
- Keep RELEASE_NOTES.md up to date

## Pull Request Process

1. Update your fork with the latest changes:
```bash
git fetch upstream
git rebase upstream/main
```

2. Ensure your PR:
- Passes all tests and linting
- Includes relevant tests
- Updates documentation
- Has a clear description of changes

3. Submit your PR with:
- Clear description
- Type of changes
- Testing performed
- Obey the checklist
- Screenshots for UI changes (if applicable)
- Additional notes (if applicable)''
- Reference to related issues

## Release Process

1. Update version in `internal/version/version.go`

2. Update RELEASE_NOTES.md with changes

3. Create and push a new tag:
```bash
git tag v0.1.0
git push origin v0.1.0
```

## Getting Help

- Open an issue for bugs or feature requests
- Join our community discussions
- Read our [documentation](https://docs.satusky.com/1ctl)

## License

By contributing to 1ctl, you agree that your contributions will be licensed under the Apache License 2.0.