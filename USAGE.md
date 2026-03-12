# Vaulty Usage Guide

Comprehensive guide to all Vaulty commands and features.

---

## Table of Contents

- [Configuration](#configuration)
- [Initialization](#initialization)
- [Authentication](#authentication)
- [Push Commands](#push-commands)
- [Pull Commands](#pull-commands)
- [Delete Commands](#delete-commands)
- [Team Management](#team-management)
- [Settings](#settings)

---

## Configuration

Vaulty stores configuration at `~/.vty/config.json`. On first run, it will guide you through setup.

### Quick Setup

```bash
vty init
```

This creates:
- A GitHub repository (via GitHub CLI)
- Encrypted vault in the repository
- Recovery seed phrase (save this securely!)

### Linking Existing Vault

```bash
vty link owner/repo-name
```

---

## Initialization

### `vty init`

Initialize a new vault in a GitHub repository.

```bash
vty init
```

Prompts for:
1. GitHub repository (or creates new one)
2. Master password
3. Recovery seed phrase (12 BIP39 words)

The recovery seed allows password recovery if forgotten.

---

## Authentication

### `vty login`

Create a session (valid 24 hours by default).

```bash
vty login
```

Prompts for master password. Session is cached based on `cache-duration` setting.

### `vty logout`

Clear current session.

```bash
vty logout
```

### `vty link`

Link to an existing vault repository.

```bash
vty link owner/repo-name
```

### `vty unlink`

Unlink current repository.

```bash
vty unlink
```

---

## Push Commands

Push secrets from local machine to encrypted vault.

### Environment Files

```bash
vty push env <name> <path> [flags]
```

**Example:**
```bash
vty push env production .env.production
vty push env staging .env.staging --force
```

**Flags:**
- `-f, --force` - Overwrite without prompting
- `-e, --env` - Target environment (production, staging, development)

### SSH Keys

```bash
vty push ssh <name> <path> [flags]
```

**Example:**
```bash
vty push ssh laptop ~/.ssh/id_rsa
vty push ssh server ~/.ssh/server_key --force
```

**Flags:**
- `-f, --force` - Overwrite without prompting

### Resources

```bash
vty push resource <name> <path> [flags]
```

Upload encrypted files or directories to resources.

**Example:**
```bash
vty push resource agents ./AGENTS.md
vty push resource zellij ~/.config/zellij --tag dev
vty push resource config.yml ./config.yml --tag team
```

**Flags:**
- `-t, --tag` - Tag for organizing resources (e.g., dev, team)
- `-f, --force` - Overwrite without prompting

### Config

```bash
vty push config <name> <path> [flags]
```

Upload encrypted config files or directories.

**Example:**
```bash
vty push config opencode ~/.config/opencode
vty push config zellij ~/.config/zellij --tag team
```

**Flags:**
- `-t, --tag` - Tag for organizing configs
- `-f, --force` - Overwrite without prompting

---

## Pull Commands

Pull secrets from encrypted vault to local machine.

### Environment Files

```bash
vty pull env <name> [flags]
```

**Example:**
```bash
vty pull env production
vty pull env production -o .env.local
```

**Flags:**
- `-o, --output` - Output file path (default: original name in current directory)

### SSH Keys

```bash
vty pull ssh <name> [flags]
```

**Example:**
```bash
vty pull ssh laptop
vty pull ssh server -o ~/.ssh/server_key
```

**Flags:**
- `-o, --output` - Output file path

### Resources

```bash
vty pull resource <name> [flags]
```

Download encrypted files or directories from resources.

**Example:**
```bash
vty pull resource agents
vty pull resource zellij -o ~/.config/zellij
vty pull resource config.yml --tag team
```

**Flags:**
- `-o, --output` - Output file/directory path
- `-t, --tag` - Tag to pull from (if organized by tags)

### Config

```bash
vty pull config <name> [flags]
```

Download encrypted config files or directories.

**Example:**
```bash
vty pull config opencode
vty pull config zellij -o ~/.config/zellij
```

**Flags:**
- `-o, --output` - Output file/directory path

---

## Delete Commands

### Environment Files

```bash
vty delete env <name>
```

### SSH Keys

```bash
vty delete ssh <name>
```

### Resources

```bash
vty delete resource <name>
```

### Config

```bash
vty delete config <name>
```

### Delete All Environments

```bash
vty delete envs
```

### Delete Entire Vault

```bash
vty delete vault
```

**Warning:** This deletes all secrets, SSH keys, resources, and team data. Requires confirmation.

---

## Vault Information

### `vty info`

Show vault contents including environments, SSH keys, resources, and team members.

```bash
vty info
```

---

## Team Management

### Add User

```bash
vty add-user <username>
```

Add a team member. Only vault owners can add users.

Prompts for:
1. User's GitHub username
2. Role (Editor or Viewer)
3. User's password (for key generation)

### Remove User

```bash
vty remove-user <username>
```

Remove a team member and rotate encryption keys.

Only vault owners can remove users. This:
1. Removes user's access to vault
2. Rotates master key
3. Re-encrypts all secrets
4. Generates new recovery seed for remaining users

### Transfer Ownership

```bash
vty transfer-owner <username>
```

Transfer vault ownership to another user.

Only the current owner can transfer ownership. After transfer:
- Current owner becomes an editor
- New owner gains full control

Example:
```bash
vty transfer-owner alice
```

### Recover Vault Access

```bash
vty recover --user <username> --seed "word1 word2 ... word12"
vty recover --user <username> --file /path/to/seed.txt
```

Recover access to your vault if you've forgotten your password.

You need:
1. Your username in the vault
2. Your 12-word recovery seed (provided when you were added to the vault)

Example:
```bash
vty recover --user john --seed "abandon ability able about above absent..."
vty recover --user john --file ~/vaulty-recovery-john.txt
```

---

## Settings

### Cache Duration

Configure how long the password is cached.

```bash
vty config cache-duration [duration]
```

**Example:**
```bash
vty config cache-duration          # Show current duration
vty config cache-duration 30m     # Cache for 30 minutes
vty config cache-duration 2h      # Cache for 2 hours
vty config cache-duration 24h     # Cache for 24 hours
```

**Valid range:** 1 minute to 24 hours

---

## Security

### Encryption

All secrets are encrypted using:
- **AES-256-GCM** for authenticated encryption
- **PBKDF2** with 100,000 iterations for key derivation
- Unique salt per user
- Unique nonce per encryption

### Recovery

If you forget your password:

```bash
vty recover --user <your-username> --seed "word1 word2 ... word12"
```

You need your 12-word recovery seed (saved when you were added to the vault).

---

## Examples

### Complete Workflow

```bash
# Initialize vault
vty init

# Push environment files
vty push env production .env.production
vty push env staging .env.staging

# Push SSH keys
vty push ssh laptop ~/.ssh/id_rsa
vty push ssh server ~/.ssh/id_ed25519

# Push resources
vty push resource agents ./AGENTS.md
vty push resource docker-compose ~/projects/app/docker-compose.yml

# Push configs
vty push config opencode ~/.config/opencode
vty push config zellij ~/.config/zellij --tag team

# On another machine, pull secrets
vty pull env production
vty pull ssh laptop

# Check what's in vault
vty info

# Delete old secrets
vty delete env staging
```

### Team Setup

```bash
# Owner initializes vault
vty init

# Add team members
vty add-user alice  # Editor
vty add-user bob    # Viewer

# Team members login
vty login

# Push shared resources
vty push config team-config ./config.yml --tag team
```
