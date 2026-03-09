#  Vaulty

> Secure environment and SSH key vault synced with GitHub

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**Vaulty** is a secure CLI tool for managing environment variables and SSH keys, seamlessly synchronized with your GitHub repositories. Keep your secrets safe, organized, and accessible across all your development environments.

---

##  Features

-  **Secure Storage** — AES-256 encrypted vault for all your secrets
-  **GitHub Sync** — Automatically sync with GitHub Secrets and SSH keys
-  **Environment-Aware** — Per-branch and per-repository secret management
-  **Quick Access** — Fast CLI commands for everyday workflows
-  **Audit Trail** — Track changes and access to your secrets
-  **Cross-Platform** — Works on macOS, Linux, and Windows
-  **SSH Key Management** — Generate, rotate, and sync SSH keys effortlessly
-  **Team Ready** — Share secrets securely with your team

---

##  Requirements

- **Go 1.21+** — [Download Go](https://golang.org/dl/)
- **GitHub CLI** — [Install gh](https://cli.github.com/manual/installation)

---

##  Installation

```bash
go install github.com/sthbryan/vaulty/cmd/vty@latest
```

Verify installation:

```bash
vty --version
```

---

##  Quick Start

### 1. Initialize Vaulty

```bash
vty init
```

This creates your local vault and authenticates with GitHub.

### 2. Sync with GitHub

```bash
vty sync
```

Synchronize your existing GitHub Secrets and SSH keys.

### 3. List Your Secrets

```bash
vty list
```

View all stored environment variables and keys.

### 4. Pull Secrets for a Project

```bash
cd my-project
vty pull
```

Automatically inject secrets into your project environment.

---

##  Commands

| Command | Description | Example |
|---------|-------------|---------|
| `vty init` | Initialize Vaulty and authenticate | `vty init` |
| `vty sync` | Sync vault with GitHub | `vty sync` |
| `vty list` | List all stored secrets | `vty list --format table` |
| `vty pull` | Pull secrets to current directory | `vty pull --env .env` |
| `vty push` | Push local secrets to vault | `vty push .env` |
| `vty set <key> <value>` | Set a secret value | `vty set API_KEY abc123` |
| `vty get <key>` | Get a secret value | `vty get API_KEY` |
| `vty delete <key>` | Delete a secret | `vty delete API_KEY` |
| `vty ssh list` | List SSH keys | `vty ssh list` |
| `vty ssh generate` | Generate new SSH key | `vty ssh generate --name work` |
| `vty ssh import` | Import existing SSH key | `vty ssh import ~/.ssh/id_rsa` |
| `vty status` | Check vault status | `vty status` |
| `vty config` | Configure settings | `vty config set editor vim` |

---

##  Security

Vaulty takes security seriously:

- **Encryption at Rest** — All secrets are encrypted using AES-256-GCM
- **No Plaintext Storage** — Keys are never stored in plaintext
- **Secure Memory** — Secrets are cleared from memory after use
- **GitHub Integration** — Leverages GitHub's battle-tested security infrastructure
- **Local-First** — Your vault is stored locally; you're in control
- **Audit Logging** — Optional access logging for compliance

### Best Practices

-  Use a strong master password
-  Regularly rotate your SSH keys
-  Enable audit logging for sensitive environments
-  Never commit the `.vaulty` directory
-  Lock your vault when not in use: `vty lock`

---

##  License

MIT License — see [LICENSE](LICENSE) for details.

---

<div align="center">

Made with  by [sthbryan](https://github.com/sthbryan)

</div>
