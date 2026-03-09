package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/sthbryan/vaulty/internal/compress"
	"github.com/sthbryan/vaulty/internal/config"
	"github.com/sthbryan/vaulty/internal/crypto"
	"github.com/sthbryan/vaulty/internal/github"
	"github.com/sthbryan/vaulty/internal/ui"
	"github.com/sthbryan/vaulty/pkg/models"
)

// SyncCommand represents the sync command
type SyncCommand struct {
	name          string
	path          string
	password      string
	passwordStdin bool
	force         bool
	config        *config.Config
	githubClient  *github.Client
}

// NewSyncCommand creates a new sync command instance
func NewSyncCommand() *SyncCommand {
	return &SyncCommand{}
}

// Name returns the command name
func (c *SyncCommand) Name() string {
	return "sync"
}

// Usage returns the command usage
func (c *SyncCommand) Usage() string {
	return "sync <name> <path>"
}

// Description returns the command description
func (c *SyncCommand) Description() string {
	return "📦 Sync an environment file to Vaulty"
}

// Execute runs the sync command
func (c *SyncCommand) Execute(args []string) error {
	if err := c.parseArgs(args); err != nil {
		return err
	}

	if err := c.validate(); err != nil {
		return err
	}

	if err := c.loadConfig(); err != nil {
		return err
	}

	if err := c.initGitHubClient(); err != nil {
		return err
	}

	if err := c.getPassword(); err != nil {
		return err
	}

	return c.sync()
}

// parseArgs parses command arguments and flags
func (c *SyncCommand) parseArgs(args []string) error {
	// Parse flags first
	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch arg {
		case "--password":
			if i+1 >= len(args) {
				return fmt.Errorf("--password requires a value")
			}
			c.password = args[i+1]
			i++
		case "--password-stdin":
			c.passwordStdin = true
		case "-f", "--force":
			c.force = true
		default:
			// Skip flags that start with -
			if strings.HasPrefix(arg, "-") {
				continue
			}
			// First non-flag arg is name
			if c.name == "" {
				c.name = arg
			} else if c.path == "" {
				c.path = arg
			}
		}
	}

	return nil
}

// validate validates the command inputs
func (c *SyncCommand) validate() error {
	// Validate name
	if c.name == "" {
		return fmt.Errorf("name is required")
	}

	if strings.Contains(c.name, "/") || strings.Contains(c.name, "\\") {
		return fmt.Errorf("name cannot contain path separators")
	}

	if strings.HasPrefix(c.name, ".") {
		return fmt.Errorf("name cannot start with a dot")
	}

	// Validate path
	if c.path == "" {
		return fmt.Errorf("path is required")
	}

	// Check if file exists
	info, err := os.Stat(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", c.path)
		}
		return fmt.Errorf("cannot access file: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("path must be a file, not a directory: %s", c.path)
	}

	// Validate password flags
	if c.password != "" && c.passwordStdin {
		return fmt.Errorf("cannot use both --password and --password-stdin")
	}

	return nil
}

// loadConfig loads the Vaulty configuration
func (c *SyncCommand) loadConfig() error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	c.config = cfg
	return nil
}

// initGitHubClient initializes the GitHub client
func (c *SyncCommand) initGitHubClient() error {
	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("failed to get GitHub token: %w", err)
	}

	c.githubClient = github.NewClient(token)
	return nil
}

// getPassword retrieves the password from flags, stdin, or prompts
func (c *SyncCommand) getPassword() error {
	if c.password != "" {
		return nil
	}

	if c.passwordStdin {
		reader := bufio.NewReader(os.Stdin)
		password, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read password from stdin: %w", err)
		}
		c.password = strings.TrimSpace(password)
		if c.password == "" {
			return fmt.Errorf("password from stdin cannot be empty")
		}
		return nil
	}

	// Prompt for password
	password, err := ui.AskPassword("🔐 Enter encryption password")
	if err != nil {
		return fmt.Errorf("failed to get password: %w", err)
	}

	c.password = password
	return nil
}

// sync performs the sync operation
func (c *SyncCommand) sync() error {
	ui.PrintInfo("Reading file: %s", c.path)

	// Read file
	content, err := os.ReadFile(c.path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	originalSize := int64(len(content))
	ui.PrintInfo("Original size: %s", ui.FormatBytes(originalSize))

	// Calculate checksum
	hash := sha256.Sum256(content)
	checksum := fmt.Sprintf("%x", hash)

	// Compress
	ui.PrintInfo("🗜️  Compressing...")
	compressed, err := compress.Compress(content)
	if err != nil {
		return fmt.Errorf("failed to compress: %w", err)
	}

	compressedSize := int64(len(compressed))
	ui.PrintInfo("Compressed size: %s (%.1f%% reduction)",
		ui.FormatBytes(compressedSize),
		float64(originalSize-compressedSize)/float64(originalSize)*100)

	// Encrypt
	ui.PrintInfo("🔒 Encrypting...")
	encrypted, err := crypto.Encrypt(compressed, c.password)
	if err != nil {
		return fmt.Errorf("failed to encrypt: %w", err)
	}

	// Create vault file
	vaultFile := models.VaultFile{
		Metadata: models.SecretMetadata{
			Name:      c.name,
			Type:      models.SecretTypeEnv,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Size:      originalSize,
			Checksum:  checksum,
		},
		Data: *encrypted,
	}

	// Serialize to JSON
	vaultData, err := json.MarshalIndent(vaultFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal vault file: %w", err)
	}

	// Check if file exists on GitHub
	ctx := context.Background()
	owner, repo, err := github.ParseRepo(c.config.Repo)
	if err != nil {
		return fmt.Errorf("invalid repo format: %w", err)
	}

	remotePath := fmt.Sprintf("envs/%s.json", c.name)
	ui.PrintInfo("☁️  Checking remote: %s/%s/%s", owner, repo, remotePath)

	var existingSha string
	existingContent, err := c.githubClient.GetContent(ctx, owner, repo, remotePath)
	if err == nil && existingContent != nil {
		// File exists
		if !c.force {
			ui.PrintWarning("File already exists on remote")
			confirmed, confirmErr := ui.AskConfirm("Overwrite existing file?", false)
			if confirmErr != nil {
				return fmt.Errorf("confirmation failed: %w", confirmErr)
			}
			if !confirmed {
				ui.PrintInfo("Sync cancelled")
				return nil
			}
		}
		existingSha = existingContent.Sha
		ui.PrintInfo("Will overwrite existing file")
	}

	// Upload to GitHub
	ui.PrintInfo("☁️  Uploading to GitHub...")

	encodedContent := base64.StdEncoding.EncodeToString(vaultData)
	commitMsg := fmt.Sprintf("Update %s via Vaulty sync", c.name)
	if existingSha == "" {
		commitMsg = fmt.Sprintf("Add %s via Vaulty sync", c.name)
	}

	req := github.ContentRequest{
		Message: commitMsg,
		Content: encodedContent,
		Sha:     existingSha,
	}

	if err := c.githubClient.PutContent(ctx, owner, repo, remotePath, req); err != nil {
		return fmt.Errorf("failed to upload: %w", err)
	}

	// Success
	ui.PrintSuccess("✅ Synced successfully!")
	fmt.Println()
	fmt.Printf("  Name:    %s\n", c.name)
	fmt.Printf("  Path:    %s\n", remotePath)
	fmt.Printf("  Size:    %s → %s\n",
		ui.FormatBytes(originalSize),
		ui.FormatBytes(int64(len(vaultData))))
	fmt.Printf("  Repo:    %s\n", c.config.Repo)

	return nil
}

// RunSync is the entry point for the sync command
func RunSync(args []string) error {
	cmd := NewSyncCommand()
	return cmd.Execute(args)
}
