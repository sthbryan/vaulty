# Contributing to Vaulty

First off, thank you for considering contributing to Vaulty! It's people like you that make Vaulty such a great tool.

## Code of Conduct

This project and everyone participating in it is governed by our [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## How Can I Contribute?

### Reporting Bugs

Before creating bug reports, please check the existing issues to see if the problem has already been reported. When you are creating a bug report, please include as many details as possible:

- **Use a clear and descriptive title**
- **Describe the exact steps to reproduce the problem**
- **Provide specific examples to demonstrate the steps**
- **Describe the behavior you observed and what behavior you expected**
- **Include screenshots or GIFs** if applicable
- **Include your environment details** (OS, Go version, Vaulty version)

### Suggesting Enhancements

Enhancement suggestions are tracked as GitHub issues. When creating an enhancement suggestion, please include:

- **Use a clear and descriptive title**
- **Provide a step-by-step description of the suggested enhancement**
- **Provide specific examples to demonstrate the enhancement**
- **Explain why this enhancement would be useful**
- **List some other tools where this enhancement exists**, if applicable

### Pull Requests

1. Fork the repository and create your branch from `main`
2. If you've added code that should be tested, add tests
3. If you've changed APIs, update the documentation
4. Ensure the test suite passes (`go test ./...`)
5. Make sure your code follows the existing code style
6. Issue the pull request!

## Development Setup

### Prerequisites

- Go 1.21 or higher
- GitHub CLI (`gh`) installed and authenticated
- Make (optional, for using Makefile commands)

### Setup

```bash
# Fork the repository on GitHub, then clone your fork
git clone https://github.com/YOUR_USERNAME/vaulty.git
cd vaulty

# Install dependencies
go mod download

# Run tests
go test ./...

# Build the binary
make build

# Or build and run
./bin/vty --help
```

### Project Structure

```
vaulty/
├── cmd/vty/          # CLI commands
├── internal/         # Internal packages
│   ├── compress/     # Compression utilities
│   ├── config/       # Configuration management
│   ├── crypto/       # Encryption/decryption
│   ├── github/       # GitHub API client
│   ├── password/     # Password storage
│   └── ui/           # UI components
├── pkg/models/       # Shared models
└── bin/              # Build output
```

### Coding Standards

- Follow standard Go conventions ([Effective Go](https://golang.org/doc/effective_go.html))
- Use `gofmt` to format your code
- Write clear, concise commit messages
- No comments in code (code should be self-explanatory)
- No emojis in command short descriptions
- Keep functions focused and small
- Add tests for new functionality

### Commit Messages

We follow [Conventional Commits](https://www.conventionalcommits.org/) with gitmoji:

- `✨ feat:` — New feature
- `🐛 fix:` — Bug fix
- `📚 docs:` — Documentation changes
- `💄 style:` — Code style changes (formatting, etc.)
- `♻️ refactor:` — Code refactoring
- `⚡ perf:` — Performance improvements
- `✅ test:` — Adding or updating tests
- `🔧 chore:` — Build process or auxiliary tool changes

Example:
```
✨ feat: add support for custom file extensions

Adds the ability to specify custom file extensions when syncing
files to the vault. This allows for better organization of
non-.env files like .json, .yaml, etc.
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/crypto/...
```

### Documentation

- Update the README.md if you change functionality
- Update command help text if you modify commands
- Add examples for new features

## Release Process

Releases are handled by maintainers. The process includes:

1. Update version in relevant files
2. Update CHANGELOG.md
3. Create a new release on GitHub with binaries for all platforms
4. Update the documentation

## Questions?

Feel free to open an issue with your question or reach out to the maintainers.

Thank you for contributing! 
