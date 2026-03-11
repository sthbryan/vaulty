# Vaulty Usage Guide

Complete reference for all Vaulty commands, flags, and options.

## Table of Contents

1. [Global Flags](#global-flags)
2. [Vault Management](#vault-management)
3. [Secrets Management](#secrets-management)
4. [Multi-User Management](#multi-user-management)
5. [Configuration](#configuration)
6. [Recovery](#recovery)
7. [Storage & Encryption](#storage--encryption)
8. [Examples](#examples)

---

## Global Flags

Available with all commands:

| Flag | Description |
|------|-------------|
| `-h, --help` | Show help for a command |
| `-v, --version` | Display Vaulty version |

---

## Vault Management

### vty init

Initialize Vaulty with a new or existing GitHub repository.

**What it does:**
- Prompts for username and master password
- Creates or links to a private GitHub repository
- Generates a masterKey (never stored locally)
- Generates a 12-word BIP39 recovery seed phrase
- Sets up your vault with owner role

**Flags:**
- `-h, --help` — Show help

**Example:**
```bash
vty init
```

**Output:** Recovery seed phrase (save securely!)

---

### vty link

Link to an existing Vaulty vault on GitHub without reinitializing.

**What it does:**
- Fetches vault metadata from GitHub repository
- Stores configuration locally at `~/.vty/config.json`
- Prepares you to login with `vty login`

**Flags:**
- `-h, --help` — Show help

**Example:**
```bash
vty link
```

**Note:** Use this when you've already run `vty init` on another machine and want to access the same vault.

---

### vty login

Authenticate to Vaulty and create an active session.

**What it does:**
- Prompts for username (suggests current_user from config if available)
- Prompts for master password
- Decrypts user's copy of masterKey
- Creates a session (valid 24 hours or until logout)

**Flags:**
- `-h, --help` — Show help

**Example:**
```bash
vty login
```

**Note:** After `vty init`, you're automatically logged in — no need to run `vty login` immediately.

---

### vty logout

Clear stored master password and end session.

**What it does:**
- Removes session from memory
- Clears any cached password
- Revokes access to vault secrets

**Flags:**
- `-f, --force` — Skip confirmation prompt
- `-h, --help` — Show help

**Example:**
```bash
vty logout
vty logout --force    # Skip confirmation
```

---

### vty unlink

Unlink Vaulty and remove local configuration.

**  WARNING:** This deletes `~/.vty/config.json` but does NOT delete secrets from GitHub.

**What it does:**
- Removes local configuration file
- Keeps all encrypted secrets safe in GitHub repository
- You can re-link anytime with `vty init` or `vty link`

**Flags:**
- `-f, --force` — Skip confirmation
- `-h, --help` — Show help

**Example:**
```bash
vty unlink
vty unlink --force    # Skip confirmation
```

---

## Secrets Management

### vty push env

Upload and encrypt an environment file to the vault.

**What it does:**
1. Reads the environment file from disk
2. Compresses using gzip
3. Encrypts with AES-256-GCM
4. Uploads to GitHub in `envs/{name}.vty` or `envs/{env}/{name}.vty`

**Syntax:**
```bash
vty push env <name> <path> [flags]
```

**Parameters:**
- `<name>` — Secret name (e.g., `production`, `api`, `db`)
- `<path>` — Local file path (e.g., `.env.production`, `./config/.env`)

**Flags:**
- `-e, --env string` — Target environment (optional: `production`, `staging`, `development`)
  - If specified, stores in `envs/{env}/{name}.vty`
  - If not specified, stores in `envs/{name}.vty` (shared)
- `-f, --force` — Overwrite without confirmation
- `-h, --help` — Show help

**Examples:**
```bash
vty push env production .env.production
vty push env staging .env.staging --force
vty push env db .env --env production      # Stores in envs/production/db.vty
vty push env secrets .env.secrets --env staging --force
```

**Requirements:**
- Must be logged in
- Editor or owner role required

---

### vty push ssh

Upload and encrypt an SSH private key to the vault.

**What it does:**
1. Reads SSH private key from disk
2. Compresses using gzip
3. Encrypts with AES-256-GCM
4. Uploads to `ssh/{username}/{name}.vty`

**Syntax:**
```bash
vty push ssh <name> <path> [flags]
```

**Parameters:**
- `<name>` — Key name (e.g., `laptop`, `work`, `server`)
- `<path>` — Local file path (e.g., `~/.ssh/id_rsa`, `~/.ssh/github_key`)

**Flags:**
- `-e, --env string` — Target environment (optional, for organization purposes)
- `-f, --force` — Overwrite without confirmation
- `-h, --help` — Show help

**Examples:**
```bash
vty push ssh laptop ~/.ssh/id_rsa
vty push ssh work ~/.ssh/work_key --force
vty push ssh server ~/.ssh/server_key --env production
```

**Requirements:**
- Must be logged in
- Editor or owner role required
- SSH keys are stored per-user in `ssh/{username}/`
- Users can only push to their own directory (unless owner)

---

### vty pull env

Download and decrypt environment file from the vault.

**What it does:**
1. Fetches encrypted file from GitHub (`envs/{name}.vty`)
2. Decrypts using masterKey
3. Decompresses (ungzip)
4. Saves to disk

**Syntax:**
```bash
vty pull env <name> [flags]
```

**Parameters:**
- `<name>` — Secret name to pull (e.g., `production`, `api`)

**Flags:**
- `-e, --env string` — Source environment (optional: `production`, `staging`, `development`)
  - If specified, pulls from `envs/{env}/{name}.vty`
  - If not specified, pulls from `envs/{name}.vty` (shared)
- `-o, --output string` — Output filename (default: `.env`, use `-` for stdout)
- `-i, --interactive` — Prompt for filename instead of using defaults
- `-h, --help` — Show help

**Examples:**
```bash
vty pull env production                              # Saves to .env
vty pull env production -o .env.production           # Custom filename
vty pull env db --env staging -o .env.staging        # Pull from envs/staging/db.vty
vty pull env secrets -o -                            # Output to stdout
vty pull env api --interactive                       # Prompt for filename
```

**Requirements:**
- Must be logged in
- Any role (viewer, editor, owner) can pull

---

### vty pull ssh

Download and decrypt SSH key from the vault.

**What it does:**
1. Fetches encrypted key from GitHub (`ssh/{user}/{name}.vty`)
2. Decrypts using masterKey
3. Decompresses (ungzip)
4. Saves to disk with proper permissions (mode 600)

**Syntax:**
```bash
vty pull ssh <name> [flags]
```

**Parameters:**
- `<name>` — Key name to pull (e.g., `laptop`, `work`)

**Flags:**
- `-o, --output string` — Output filename (default: key name, use `-` for stdout)
- `-u, --user string` — Pull another user's key (owner only)
- `-i, --interactive` — Prompt for filename instead of using defaults
- `-h, --help` — Show help

**Examples:**
```bash
vty pull ssh laptop                          # Saves to ./laptop
vty pull ssh work -o ~/.ssh/work_key         # Custom output path
vty pull ssh team-key -u alice               # Owner: pull alice's key
vty pull ssh server -o - | ssh-add -         # Pipe to ssh-add
vty pull ssh personal --interactive          # Prompt for filename
```

**Requirements:**
- Must be logged in
- Users see only their own keys
- Owner can pull any user's key
- Viewer, editor, owner roles can pull

---

### vty delete env

Delete an environment file from the vault.

**What it does:**
- Removes encrypted environment file from GitHub
- Deletes from `envs/{name}.vty` or `envs/{env}/{name}.vty`

**Syntax:**
```bash
vty delete env <name> [flags]
```

**Parameters:**
- `<name>` — Environment name to delete

**Flags:**
- `--env string` — Environment namespace (optional)
  - If specified, deletes from `envs/{env}/{name}.vty`
  - If not specified, deletes from `envs/{name}.vty`
- `-f, --force` — Delete without confirmation
- `-h, --help` — Show help

**Examples:**
```bash
vty delete env production
vty delete env db --env staging --force
vty delete env old-secrets -f
```

**Requirements:**
- Must be logged in
- Editor or owner role required

---

### vty delete ssh

Delete an SSH key from the vault.

**What it does:**
- Removes encrypted SSH key from GitHub
- Deletes from `ssh/{user}/{name}.vty`

**Syntax:**
```bash
vty delete ssh <name> [flags]
```

**Parameters:**
- `<name>` — Key name to delete

**Flags:**
- `-u, --user string` — User directory (optional)
  - If specified, deletes from `ssh/{user}/{name}.vty`
  - If not specified, deletes from current user's directory
  - Owner can delete any user's key
- `-f, --force` — Delete without confirmation
- `-h, --help` — Show help

**Examples:**
```bash
vty delete ssh laptop
vty delete ssh work -u alice                 # Owner: delete alice's key
vty delete ssh old-key --force
```

**Requirements:**
- Must be logged in
- Users can only delete their own keys
- Owner can delete any key
- Editor or owner role required

---

### vty info

Display all secrets stored in your vault.

**What it does:**
- Lists all environment files in `envs/` directory
- Lists all SSH keys in `ssh/` directory
- Shows name, type, size, and last updated timestamp
- Optionally filters by environment

**Syntax:**
```bash
vty info [flags]
```

**Flags:**
- `-e, --env string` — Filter by environment (optional)
- `-h, --help` — Show help

**Examples:**
```bash
vty info                            # Show all secrets
vty info --env production           # Show only production environment
vty info -e staging                 # Show only staging
```

**Output Example:**
```
Environments:
  production      1.2 KB   2024-03-10 14:30:00
  staging         0.8 KB   2024-03-09 10:15:00

SSH Keys:
  alice/laptop    2.1 KB   2024-03-10 09:45:00
  alice/work      2.0 KB   2024-03-08 16:20:00
  bob/personal    1.9 KB   2024-03-07 13:00:00
```

**Requirements:**
- Must be logged in
- Any role can view info

---

## Multi-User Management

### vty add-user

Add a new user to the vault.

**What it does:**
- Creates a new user entry in vault
- Generates their recovery seed phrase
- Encrypts masterKey for their account
- Assigns user role (editor or viewer)

**Syntax:**
```bash
vty add-user <username> [flags]
```

**Parameters:**
- `<username>` — Username for the new team member

**Flags:**
- `-r, --role string` — User role to assign
  - `editor` (default) — Can push/pull/delete secrets
  - `viewer` — Read-only access
- `-h, --help` — Show help

**Examples:**
```bash
vty add-user pablo                           # Add as editor (default)
vty add-user maria --role viewer             # Add as viewer (read-only)
vty add-user juan --role editor
```

**Process:**
1. Requests your (owner's) master password for verification
2. Prompts for new user's password
3. Generates their recovery seed
4. Shows seed to new user (they must save it)
5. Encrypts masterKey for their account
6. Uploads to GitHub

**Requirements:**
- Must be logged in as owner
- Owner role required

---

### vty remove-user

Remove a user from the vault and rotate the master key.

**  WARNING:** This action is irreversible. The user will lose access to all vault secrets.

**What it does:**
1. Verifies you are vault owner (requests password)
2. Decrypts current vault with old masterKey
3. Generates a new masterKey
4. Re-encrypts all secrets with new masterKey
5. Re-encrypts new masterKey for all remaining users
6. Uploads all changes to GitHub
7. Deletes removed user's key file

**Syntax:**
```bash
vty remove-user <username> [flags]
```

**Parameters:**
- `<username>` — Username to remove

**Flags:**
- `-h, --help` — Show help

**Examples:**
```bash
vty remove-user pablo
vty remove-user temporary-contractor
```

**Requirements:**
- Must be logged in as owner
- Owner role required
- Removed user loses all access immediately

---

## Configuration

### vty config cache-duration

Get or set password cache duration.

**What it does:**
- Stores password in memory to avoid repeated prompts
- Configurable lifetime from 1 minute to 24 hours
- Default: 15 minutes

**Syntax:**
```bash
vty config cache-duration [duration]
```

**Parameters:**
- `[duration]` — Cache lifetime (optional)
  - Format: `1m`, `15m`, `1h`, `24h`, etc.
  - Omit to view current setting
  - Use `0` to disable caching

**Examples:**
```bash
vty config cache-duration                    # Show current setting
vty config cache-duration 30m                # Set to 30 minutes
vty config cache-duration 1h                 # Set to 1 hour
vty config cache-duration 0                  # Disable (always prompt)
vty config cache-duration 24h                # Maximum (24 hours)
```

**Valid Durations:**
- `1m`, `5m`, `15m`, `30m` — Minutes
- `1h`, `2h`, `6h`, `12h`, `24h` — Hours
- `0` — Disable caching

**Storage:**
- Saved in `~/.vty/config.json` as `cache_duration`

**Requirements:**
- None (applies to next session)

---

## Recovery

### vty recover

Recover vault access using a 12-word BIP39 seed phrase.

**  WARNING:** This command requires a valid seed phrase. If lost, vault data cannot be recovered.

**What it does:**
1. Prompts for your 12-word recovery seed phrase
2. Validates seed against vault metadata
3. Generates new master password from seed
4. Allows you to set a new password
5. Grants access to vault

**Syntax:**
```bash
vty recover --seed "<word1 word2 word3 ... word12>"
```

**Flags:**
- `--seed string` — Your 12-word seed phrase (required)
- `-h, --help` — Show help

**Example:**
```bash
vty recover --seed "abandon ability able about above absent absorb abstract abuse access accident account"
```

**Process:**
1. Enter seed phrase when prompted (or via `--seed` flag)
2. Validate seed (verification against vault data)
3. Prompt for new master password
4. Vault access restored

**Requirements:**
- Valid BIP39 seed phrase (12 words)
- Original vault must exist on GitHub
- No active session required

**Recovery Phrases:**
- Generated during `vty init` or `vty add-user`
- Unique per user
- 12-word BIP39 format
- **Store securely** (password manager, physical backup, etc.)

---

### GitHub Vault Structure

```
.vaulty/
├── metadata.vty              ← User list, vault info (gzip + hex)
├── vault.vty                 ← Vault data (gzip + hex)
├── keys/
│   ├── alice.vty             ← Alice's encrypted masterKey
│   └── bob.vty               ← Bob's encrypted masterKey
└── recovery/
    ├── alice.recovery.vty    ← Alice's seed phrase (encrypted)
    └── bob.recovery.vty      ← Bob's seed phrase (encrypted)

envs/
├── production.vty            ← Encrypted environment file
├── staging.vty
└── {env}/
    ├── db.vty                ← Environment-scoped file
    └── api.vty

ssh/
├── alice/
│   ├── laptop.vty            ← Alice's laptop key
│   └── work.vty              ← Alice's work key
└── bob/
    └── personal.vty          ← Bob's personal key
```

---

## Examples

### Scenario 1: Solo Developer Setup

```bash
# Initialize vault
vty init
# → Creates vault, save recovery seed!

# Push environment file
vty push env production .env.production

# Push SSH key
vty push ssh laptop ~/.ssh/id_rsa

# Later: Pull secrets on new machine
vty login
vty pull env production -o .env.production
vty pull ssh laptop -o ~/.ssh/id_rsa
chmod 600 ~/.ssh/id_rsa
```

### Scenario 2: Team with Three Members

```bash
# Alice (owner) initializes
vty init
# → Vault created, saves seed

# Alice adds Bob (editor)
vty add-user bob --role editor
# → Bob gets recovery seed

# Alice adds Maria (viewer)
vty add-user maria --role viewer
# → Maria gets recovery seed

# Alice pushes team secrets
vty push env production .env.production

# Bob can pull and push
vty login
vty pull env production
vty push env staging .env.staging

# Maria can only pull
vty login
vty pull env production
# (Cannot push or delete)

# Alice removes contractor Bob
vty remove-user bob
# → MasterKey rotates, all secrets re-encrypted for remaining users
```

### Scenario 3: Managing Per-Environment Secrets

```bash
# Push environment-scoped secrets
vty push env db .env --env production      # envs/production/db.vty
vty push env db .env --env staging         # envs/staging/db.vty
vty push env api .env --env production     # envs/production/api.vty

# Pull by environment
vty pull env db --env production -o .env
vty pull env api --env staging -o .env

# View all secrets in production
vty info --env production

# Delete environment-scoped secret
vty delete env db --env staging --force
```

### Scenario 4: SSH Key Management

```bash
# Push multiple keys
vty push ssh laptop ~/.ssh/id_rsa
vty push ssh work ~/.ssh/work_key
vty push ssh github ~/.ssh/github_key

# View all keys
vty info

# Pull specific key for new machine
vty pull ssh laptop -o ~/.ssh/laptop_key

# Pull and add to agent
vty pull ssh work -o /tmp/work_key
chmod 600 /tmp/work_key
ssh-add /tmp/work_key

# Owner pulls another user's key
vty pull ssh server -u deployment

# Clean up old key
vty delete ssh github --force
```

### Scenario 5: Password Recovery

```bash
# Forgot password? Use recovery seed
vty recover --seed "abandon ability able about above absent absorb abstract abuse access accident account"
# → Prompts for new password
# → Vault access restored

# Forgot recovery seed too?
# → Cannot recover. Create new vault with `vty init` and migrate secrets.
```

### Scenario 6: Cache Configuration

```bash
# Check current cache duration
vty config cache-duration
# → Output: "15m"

# Change to 1 hour for less frequent prompts
vty config cache-duration 1h

# Disable cache (always prompt for password)
vty config cache-duration 0

# Maximum security: only 5 minutes
vty config cache-duration 5m
```

### Scenario 7: Multi-Machine Sync

```bash
# Machine 1: Initialize
vty init
# → Vault created, save recovery seed

# Machine 2: Link to existing vault
vty link
# → Asks for repo owner and name
# → Downloads metadata from GitHub

# Machine 2: Login
vty login
# → Username: alice, Password: ***

# Machine 2: Pull secrets
vty pull env production
vty pull ssh laptop

# Both machines now synced!
```

---

## Troubleshooting

### "Unknown command" errors
- Ensure you're using subcommands correctly:
  -  `vty push env ...` (not `vty push-env`)
  -  `vty pull ssh ...` (not `vty pull-ssh`)
  -  `vty add-user ...` (hyphenated)

### "Not logged in" errors
- Run `vty login` before accessing secrets
- Session expires after 24 hours; login again
- Check active session: `vty info`

### "Permission denied" errors
- Viewers can only pull, not push/delete
- Ask owner to promote role or push secrets
- Non-owners cannot delete other users' SSH keys

### "Could not fetch from GitHub"
- Verify GitHub CLI is installed: `gh auth status`
- Ensure repository is private and accessible
- Check internet connectivity

### "Invalid seed phrase"
- Verify all 12 words match (case-sensitive in some contexts)
- Seeds are space-separated: `word1 word2 word3 ...`
- Contact vault owner if seed is lost

---

## Related

- [README.md](README.md) — Quick start and overview
- [CONTRIBUTING.md](CONTRIBUTING.md) — Contribution guidelines
- [LICENSE](LICENSE) — MIT License
