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
- **Recovery Seed** — 12-word BIP39 seed phrase per user for password recovery
- **Multi-User Support** — Team vaults with Owner, Editor, and Viewer roles
- **Session Management** — Lock/unlock vault, membership validation on operations
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
- Prompt for your username and master password
- Create or link to an existing private repository
- Generate a masterKey (never stored locally)
- Generate a 12-word recovery seed phrase
- Set up your vault with owner role

**Important:** Save your recovery seed phrase securely. You will need it if you forget your master password.

### 2. Add Team Members (Multi-User)

As the vault owner, add collaborators:

```bash
vty add-user pablo
```

This will:
- Ask for your password (to verify ownership)
- Prompt for pablo's password
- Generate pablo's recovery seed phrase
- Set pablo's role (viewer by default, can be changed)

### 3. Login to the Vault

Team members login to access secrets:

```bash
vty login
```

This will:
- Ask for username and password
- Decrypt their copy of the masterKey
- Create a session (valid for 24h or until logout)

**Note:** After `vty init`, you're automatically logged in — no need to run `vty login`.

### 4. Push Environment Files

```bash
vty push env production .env.production
```

This compresses, encrypts, and uploads your environment file to GitHub in the `envs/` directory. Only works when logged in.

### 5. Pull Environment Files

```bash
vty pull env production
```

Download and decrypt the environment file to your current directory.

### 6. Push/Pull SSH Keys

Store SSH keys securely per-user:

```bash
# Push an SSH key
vty push ssh laptop ~/.ssh/id_rsa

# Pull an SSH key
vty pull ssh laptop
```

SSH keys are stored per-user in `ssh/{username}/` — you only see your own SSH keys (owner sees all).

---

## Commands

### Vault Management

| Command | Description | Example |
|---------|-------------|---------|
| `vty init` | Initialize or link to a GitHub repository | `vty init` |
| `vty login` | Login and create session (multi-user) | `vty login` |
| `vty lock` | Lock vault without logging out | `vty lock` |
| `vty logout` | Clear stored master password | `vty logout` |
| `vty unlink` | Unlink Vaulty (keeps GitHub data) | `vty unlink` |

### Secret Management

| Command | Description | Example |
|---------|-------------|---------|
| `vty push env <name> <path>` | Push environment file to vault | `vty push env api .env` |
| `vty push ssh <name> <path>` | Push SSH key to vault | `vty push ssh work ~/.ssh/id_rsa` |
| `vty pull env <name>` | Pull and decrypt environment file | `vty pull env api` |
| `vty pull ssh <name>` | Pull and decrypt SSH key | `vty pull ssh work` |
| `vty info` | Show vault info (envs, SSH keys, users) | `vty info` |
| `vty delete env <name>` | Delete environment file from vault | `vty delete env api` |
| `vty delete ssh <name>` | Delete SSH key from vault | `vty delete ssh work` |

### Multi-User Management

| Command | Description | Example |
|---------|-------------|---------|
| `vty add-user <username>` | Owner adds a collaborator (requires your password) | `vty add-user pablo` |
| `vty remove-user <username>` | Owner removes a user and rotates masterKey | `vty remove-user pablo` |
| `vty transfer-owner <username>` | Transfer ownership to another user | `vty transfer-owner pablo` |

### Recovery & Configuration

| Command | Description | Example |
|---------|-------------|---------|
| `vty recover --seed "..."` | Recover vault using seed phrase | `vty recover --seed "word1 word2 ..."` |
| `vty config cache-duration [duration]` | Get/set password cache duration | `vty config cache-duration 30m` |

---

## Security

Vaulty takes security seriously:

- **Encryption** — AES-256-GCM with randomly generated salts and IVs
- **Key Derivation** — PBKDF2 with 100,000 iterations
- **Device Salt** — Unique per-machine salt for additional security
- **MasterKey** — Single encryption key for all vault data, never stored on disk (memory-only during session)
- **Per-User Keys** — Each user's password encrypts their copy of the masterKey
- **Password Storage** — OS keyring (macOS Keychain, Linux Secret Service, Windows Credential Manager) with memory cache fallback
- **No Plaintext** — Secrets are never stored in plaintext locally or on GitHub
- **Recovery** — BIP39 seed phrase per user for password recovery without exposing secrets
- **Membership Validation** — User access validated on every pull/push/sync operation
- **Automatic Key Rotation** — MasterKey rotated when users are removed, all remaining users re-encrypted

### Multi-User Security

- **Role-Based Access** — Owner (admin), Editor (write), Viewer (read-only)
- **Auto-Unlink** — User automatically unlinked if removed from vault during operations
- **Cached Encryption** — Vault cached locally with 24h TTL, encrypted with user password
- **Session Management** — MasterKey loaded only during session, locked/cleared on logout
- **Per-User SSH Keys** — SSH keys stored per-user in `ssh/{username}/` — users only see their own keys (owner sees all)

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
  "current_user": "ana",
  "current_role": "owner",
  "cache_duration": "15m",
  "storage_type": "auto"
}
```

### Storage Types

- `auto` — Use OS keyring if available, fallback to memory
- `keyring` — Force OS keyring (fails if unavailable)
- `memory` — Use memory-only storage (password cleared on logout)

### GitHub Vault Structure

```
envs/
├── production.vty          ← Encrypted environment file
└── staging.vty
ssh/
├── alice/
│   ├── laptop.vty         ← Encrypted SSH private key
│   └── work.vty
└── bob/
    └── personal.vty
.vaulty/
├── metadata.json          ← Repo owner, user list, version
└── recovery/
    ├── ana.recovery       ← Ana's recovery seed phrase
    └── pablo.recovery     ← Pablo's recovery seed phrase
```

---

## Roadmap

Features planned for future releases:

### High Priority

- [ ] **Environments** — Native support for develop, staging, and production environments with isolation
- [ ] **Team Resources** — Share skills, agents.md, documentation, and utilities (encrypted or plaintext)
- [ ] **CI/CD Integration** — Seamless injection of environment variables in pipelines without .env files on servers

### Medium Priority

- [ ] **Status Command** — Check current status: linked/unlinked, last sync, storage type, cache status
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
