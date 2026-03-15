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
- **Recovery Seed** — 12-word BIP39 seed phrase per user for password recovery
- **Multi-User Support** — Team vaults with Owner, Editor, and Viewer roles
- **SSH Key Management** — Securely store and sync SSH private keys
- **Cross-Platform** — Works on macOS, Linux, and Windows
- **Zero-Config** — Works out of the box with GitHub CLI authentication

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
git clone https://github.com/DeadBryam/vaulty.git
cd vaulty
make build        # Build for current platform
make build-all    # Build for all platforms
make install      # Install to $GOPATH/bin
```

Verify installation:

```bash
vty --version
```

---

## Quick Start

See **[USAGE.md](USAGE.md#configuration)** for full details.

### 1. Initialize Vaulty

```bash
vty init
```

Creates your vault and generates a recovery seed phrase. **Save it securely** — you'll need it if you forget your password.

### 2. Add Team Members (Optional)

```bash
vty add-user pablo
```

Owner can add collaborators and assign roles (Owner, Editor, Viewer).

### 3. Login (Multi-User)

```bash
vty login
```

Creates a session (valid 24h or until logout). After `vty init`, you're automatically logged in.

### 4. Push Secrets

Push environment files and SSH keys:

```bash
vty push env production .env.production    # Push environment file
vty push ssh laptop ~/.ssh/id_rsa          # Push SSH key
```

### 5. Pull Secrets

Pull and decrypt from GitHub:

```bash
vty pull env production                    # Download environment file
vty pull ssh laptop                        # Download SSH key
```

### 6. Vault Info & Management

```bash
vty info                                   # Show vault contents
vty delete env production                  # Delete environment
vty delete ssh laptop                      # Delete SSH key
vty logout                                 # Clear session
```

---

## Commands

Quick reference. See **[USAGE.md](USAGE.md)** for complete details on all flags and subcommands.

| Command | Purpose |
|---------|---------|
| `vty init` | Initialize vault with GitHub repository |
| `vty login` / `vty logout` | Manage sessions |
| `vty link` | Link to existing vault repository |
| `vty unlink` | Unlink current repository |
| `vty push env <name> <path>` | Upload environment file |
| `vty push ssh <name> <path>` | Upload SSH key |
| `vty push resource <name> <path>` | Upload file/directory to resources |
| `vty push config <name> <path>` | Upload file/directory to config |
| `vty pull env <name>` | Download environment file |
| `vty pull ssh <name>` | Download SSH key |
| `vty pull resource <name>` | Download file/directory from resources |
| `vty pull config <name>` | Download file/directory from config |
| `vty info` | Show vault contents |
| `vty delete env <name>` | Delete environment |
| `vty delete ssh <name>` | Delete SSH key |
| `vty delete resource <name>` | Delete resource |
| `vty delete config <name>` | Delete config |
| `vty add-user <user>` | Add team member (owner only) |
| `vty remove-user <user>` | Remove user and rotate keys (owner only) |
| `vty transfer-owner <user>` | Transfer ownership (owner only) |
| `vty recover --user <user> --seed "..."` | Recover vault using seed phrase |
| `vty config cache-duration [time]` | Configure password cache |

---

## Configuration

Vaulty stores config at `~/.vty/config.json`. Key settings:

- **repo** — GitHub repository (owner/name)
- **storage_type** — `auto` (keyring + fallback), `keyring`, or `memory`
- **cache_duration** — Password cache lifetime

---

## Roadmap

### High Priority
- [x] **Environments** — Native support for develop, staging, and production with isolation
- [x] **Team Resources** — Share encrypted docs, agents.md, utilities, .config
- [ ] **Local mode** — Store secrets locally without GitHub sync

### Medium Priority
- [ ] **Web Interface** — GUI for managing secrets
- [ ] **Modular Downloads** — Fetch specific secrets instead of entire vault
- [ ] **Security Mode** — Server mode that always requires password input
- [ ] **Multiple Sources** — Support multiple vault backends at same time (GitHub, local, cloud)

### Low Priority
- [x] **Reset/Clean** — **DESTRUCTIVE** Vault reset (requires seed phrase confirmation)
- [ ] **CI/CD Integration** — Inject secrets into pipelines without .env files
- [ ] **Audit Logging** — Compliance tracking
- [ ] **External Integration** — AWS Secrets Manager, Azure Key Vault

---

## Contributing

We welcome contributions! See [Contributing Guide](CONTRIBUTING.md) for details.

### Report Issues

- [Bug Report](https://github.com/DeadBryam/vaulty/issues/new?template=bug_report.md)
- [Feature Request](https://github.com/DeadBryam/vaulty/issues/new?template=feature_request.md)

---

## License

MIT License — see [LICENSE](LICENSE) for details.

---

<div align="center">

Made with care by [DeadBryam](https://github.com/DeadBryam)

</div>
