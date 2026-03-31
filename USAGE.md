# Vaulty Usage Guide

## Commands Overview

| Category | Commands |
|----------|----------|
| **Init** | `init`, `link`, `unlink` |
| **Auth** | `login`, `logout` |
| **Push** | `push env`, `push ssh`, `push resource`, `push config` |
| **Pull** | `pull env`, `pull ssh`, `pull resource`, `pull config` |
| **Show** | `show env`, `show ssh`, `show resource`, `show config` |
| **Run** | `run env` |
| **Backup** | `export`, `import` |
| **Delete** | `delete env`, `delete ssh`, `delete resource`, `delete config` |
| **Info** | `info` |
| **Team** | `add-user`, `remove-user`, `transfer-owner`, `recover` |
| **Config** | `config` |

---

## Push Commands

```bash
vty push env <name> <path> [-e env] [-f]
vty push ssh <name> <path> [-f]
vty push resource <name> <path> [--tag tag] [-f]
vty push config <name> <path> [--tag tag] [-f]
```

## Pull Commands

```bash
vty pull env <name> [-e env] [-o file]
vty pull ssh <name> [-o file]
vty pull resource <name> [-o path] [--tag tag]
vty pull config <name> [-o path] [--tag tag]
```

## Show Commands

Display secrets without downloading files. Uses `bat` if available.

```bash
vty show env <name> [-e env]
vty show ssh <name>
vty show resource <name>
vty show config <name>
```

## Run Command (CI/CD)

Inject secrets into commands without .env files.

```bash
vty run env <name> [-e env] -- <command> [args...]

# Examples:
vty run env api -- npm run build
vty run env api -e production -- npm run deploy
vty run env api -e staging -- sh -c 'npm run migrate && npm run start'
```

Rules:
- `--` is mandatory to separate Vaulty flags from the command
- `-e` (or `--env`) is optional; if omitted, uses shared secrets
- Vault values overwrite existing environment variables

## Export / Import

```bash
# Export entire vault to backup file
vty export [-o backup.vtyb]

# Import backup (wipes current vault first)
vty import -i backup.vtyb
```

Import requires typing `yes` to confirm the wipe.

## Delete Commands

```bash
vty delete env <name> [-e env]
vty delete ssh <name>
vty delete resource <name>
vty delete config <name>
vty delete envs        # Delete all environments
vty delete vault       # Delete entire vault
```

## Info

```bash
vty info              # Show vault contents
vty info -e prod      # Filter by environment
```

## Team Management

```bash
vty add-user <username>          # Add team member
vty remove-user <username>       # Remove and rotate keys
vty transfer-owner <username>    # Transfer ownership
vty recover --user <u> --seed "word1 word2 ..."
```

## Configuration

```bash
vty config cache-duration [time]   # Password cache duration
```

---

## Quick Start

```bash
# Initialize
vty init

# Push secrets
vty push env production .env.production
vty push ssh laptop ~/.ssh/id_rsa

# Run with secrets (no .env file needed)
vty run env production -- npm run build

# Backup vault
vty export -o vault-backup.vtyb
```
