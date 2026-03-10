# Vaulty

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/DeadBryam/vaulty)](https://github.com/DeadBryam/vaulty/releases)

**Secure environment and SSH key vault synced with GitHub**

Vaulty is a secure CLI tool for managing environment variables, SSH keys, and team resources, seamlessly synchronized with your GitHub repositories. Keep your secrets safe, organized, and accessible across all your development environments.

---

## Features

- **Secure Storage** — AES-256-GCM encryption with PBKDF2 key derivation
- **GitHub Backend** — Store encrypted secrets in your private GitHub repository
- **Master Password** — Single password for all operations with secure OS keyring storage
- **Recovery Seed** — 12-word BIP39 seed phrase for password recovery
- **SSH Key Management** — Securely store and sync SSH private keys
- **Cross-Platform** — Works on macOS, Linux, and Windows
- **Zero-Config** — Works out of the box with GitHub CLI authentication
- **Compressed** — Automatic gzip compression before encryption

---

## Requirements

- **Go 1.21+** — [Download Go](https://golang.org/dl/) (only for building from source)
- **GitHub CLI** — [Install gh](https://cli.github.com/manual/installation)
- **GitHub Account** — With a private repository for storage

---

## Installation

### From Releases (Recommended)

Download the latest binary for your platform from the [releases page](https://github.com/DeadBryam/vaulty/releases):

```bash
# macOS (Apple Silicon)
curl -L https://github.com/DeadBryam/vaulty/releases/latest/download/vty-darwin-arm64 -o vty
chmod +x vty
sudo mv vty /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/DeadBryam/vaulty/releases/latest/download/vty-darwin-amd64 -o vty
chmod +x vty
sudo mv vty /usr/local/bin/

# Linux (AMD64)
curl -L https://github.com/DeadBryam/vaulty/releases/latest/download/vty-linux-amd64 -o vty
chmod +x vty
sudo mv vty /usr/local/bin/

# Linux (ARM64)
curl -L https://github.com/DeadBryam/vaulty/releases/latest/download/vty-linux-arm64 -o vty
chmod +x vty
sudo mv vty /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/DeadBryam/vaulty/cmd/vty@latest
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/DeadBryam/vaulty.git
cd vaulty

# Build for current platform
make build

# Or build for all platforms
make build-all

# Install to $GOPATH/bin
make install
```

Verify installation:

```bash
vty --version
```

---

## Quick Start

### 1. Initialize Vaulty

```bash
vty init
```

This will:
- Prompt for a GitHub repository (owner/repo format)
- Create or link to an existing private repository
- Set up your master password
- Generate a 12-word recovery seed phrase

**Important:** Save your recovery seed phrase securely. You will need it if you forget your master password.

### 2. Sync an Environment File

```bash
vty sync production .env.production
```

This compresses, encrypts, and uploads your environment file to GitHub.

### 3. Pull Secrets

```bash
vty pull production
```

Download and decrypt the environment file to your current directory.

### 4. List Your Secrets

```bash
vty list
```

View all stored environment files and SSH keys.

---

## Commands

| Command | Description | Example |
|---------|-------------|---------|
| `vty init` | Initialize or link to a GitHub repository | `vty init` |
| `vty sync <name> <path>` | Sync an environment file to vault | `vty sync api .env` |
| `vty sync-ssh <name> <key_path>` | Sync an SSH private key | `vty sync-ssh work ~/.ssh/id_rsa` |
| `vty pull <name>` | Pull and decrypt secrets | `vty pull api` |
| `vty list` | List all secrets in the vault | `vty list` |
| `vty list --type=env` | List only environment files | `vty list --type=env` |
| `vty list --type=ssh` | List only SSH keys | `vty list --type=ssh` |
| `vty delete <name>` | Delete a secret from the vault | `vty delete api` |
| `vty delete <name> --type=ssh` | Delete an SSH key | `vty delete work --type=ssh` |
| `vty recover --seed "..."` | Recover vault using seed phrase | `vty recover --seed "word1 word2 ..."` |
| `vty logout` | Clear stored master password | `vty logout` |
| `vty unlink` | Unlink Vaulty (keeps GitHub data) | `vty unlink` |
| `vty config cache-duration [duration]` | Get/set password cache duration | `vty config cache-duration 30m` |

---

## Security

Vaulty takes security seriously:

- **Encryption** — AES-256-GCM with randomly generated salts and IVs
- **Key Derivation** — PBKDF2 with 100,000 iterations
- **Device Salt** — Unique per-machine salt for additional security
- **Password Storage** — OS keyring (macOS Keychain, Linux Secret Service, Windows Credential Manager) with memory cache fallback
- **No Plaintext** — Secrets are never stored in plaintext locally or on GitHub
- **Recovery** — BIP39 seed phrase for password recovery without exposing secrets

### Password Cache

By default, Vaulty caches your password in memory for 15 minutes to avoid repeated prompts. You can configure this:

```bash
# Set cache duration (1m to 24h)
vty config cache-duration 30m

# Disable cache (always prompt)
vty config cache-duration 0

# Maximum cache (24 hours)
vty config cache-duration 24h
```

### Recovery

If you forget your master password, use your 12-word recovery seed phrase:

```bash
vty recover --seed "your twelve word seed phrase here"
```

You will be prompted to set a new master password. The recovery process validates your seed against the vault without exposing any secrets.

---

## Configuration

Configuration is stored at `~/.vty/config.json`:

```json
{
  "repo": "owner/my-vault",
  "device_salt": "base64-encoded-salt",
  "cache_duration": "15m",
  "storage_type": "auto"
}
```

### Storage Types

- `auto` — Use OS keyring if available, fallback to memory
- `keyring` — Force OS keyring (fails if unavailable)
- `memory` — Use memory-only storage (password cleared on logout)

---

## Roadmap

Features planned for future releases:

### High Priority

- [ ] **Multi-user Support** — Multiple users/colaborators per vault with access control
- [ ] **Environments** — Native support for develop, staging, and production environments with isolation
- [ ] **Team Resources** — Share skills, agents.md, documentation, and utilities (encrypted or plaintext)
- [ ] **CI/CD Integration** — Seamless injection of environment variables in pipelines without .env files on servers

### Medium Priority

- [ ] **Stats Command** — View vault statistics (secret count, size, last sync, storage usage)
- [ ] **Status Command** — Check current status: linked/unlinked, last sync, storage type, cache status
- [ ] **Lock/Unlock** — Lock vault to read-only mode (pull allowed, sync/push blocked). Requires unlock for writes
- [ ] **Modular Downloads** — Option to download only specific secrets instead of entire vault metadata
- [ ] **Security Mode** — Server mode that always requires password input (no caching)
- [ ] **Config View** — Display current configuration with `vty config` (no subcommand)

### Low Priority

- [ ] **Reset/Clean** — **DESTRUCTIVE** Complete vault reset. Removes: local config, canary, recovery data, and all secrets from GitHub. Requires seed phrase for confirmation

### Ideas Under Consideration

- Web interface for managing secrets
- Audit logging for compliance
- Secret versioning and rollback
- Integration with external secret managers (AWS Secrets Manager, Azure Key Vault)

---

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Reporting Issues

- [Bug Report](https://github.com/DeadBryam/vaulty/issues/new?template=bug_report.md)
- [Feature Request](https://github.com/DeadBryam/vaulty/issues/new?template=feature_request.md)

---

## License

MIT License — see [LICENSE](LICENSE) for details.

---

<div align="center">

Made with care by [DeadBryam](https://github.com/DeadBryam)

</div>
