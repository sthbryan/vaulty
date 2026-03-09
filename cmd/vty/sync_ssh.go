package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/internal/compress"
	"github.com/sthbryan/vaulty/internal/config"
	"github.com/sthbryan/vaulty/internal/crypto"
	"github.com/sthbryan/vaulty/internal/github"
	"github.com/sthbryan/vaulty/internal/ui"
	"github.com/sthbryan/vaulty/pkg/models"
)

// syncSSHCmd represents the sync-ssh command
var syncSSHCmd = &cobra.Command{
	Use:   "sync-ssh <name> <key_path>",
	Short: "🔐 Sync an SSH private key to your vault",
	Long: `Sync an SSH private key to your vault.

This command will:
  🔑 Read your SSH private key
  🗜️  Compress it for efficient storage
  🔒 Encrypt it with your password
  ☁️  Upload it to your GitHub repository

WARNING: Only upload private keys to private repositories you control.`,
	Args: cobra.ExactArgs(2),
	RunE: runSyncSSH,
}

func runSyncSSH(cmd *cobra.Command, args []string) error {
	name := args[0]
	keyPath := args[1]

	// Validate name
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	// Validate it's not a public key
	if strings.HasSuffix(keyPath, ".pub") {
		logger.Error("This appears to be a public key file. Please provide a private key (without .pub extension)")
		return fmt.Errorf("public key file provided")
	}

	// Check if file exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		logger.Error("Key file not found", "path", keyPath)
		return fmt.Errorf("key file not found: %s", keyPath)
	}

	// Show warning about private key upload
	fmt.Println()
	logger.Warn("🔐 WARNING: You are about to upload a PRIVATE SSH key to GitHub")
	logger.Warn("🔑 Private keys are sensitive credentials that grant access to your systems")
	logger.Warn("⚠️  Ensure this repository is private and you understand the security implications")
	fmt.Println()

	// Ask for confirmation
	confirmed, err := ui.AskConfirm("☁️  Do you want to continue uploading this private key?", false)
	if err != nil {
		return fmt.Errorf("failed to get confirmation: %w", err)
	}
	if !confirmed {
		logger.Info("Upload cancelled")
		return nil
	}

	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		logger.Error("Failed to load config", "error", err)
		return err
	}

	if err := cfg.Validate(); err != nil {
		logger.Error("Repository not configured. Run: vty init <owner/repo>")
		return err
	}

	// Get GitHub token
	token, err := github.GetGitHubToken()
	if err != nil {
		logger.Error("Failed to get GitHub token", "error", err)
		return err
	}

	// Read the private key file
	logger.Info("🔑 Reading private key...", "path", keyPath)
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		logger.Error("Failed to read key file", "error", err)
		return err
	}

	// Validate it looks like an SSH private key
	content := string(keyData)
	if !strings.Contains(content, "BEGIN") || !strings.Contains(content, "PRIVATE KEY") {
		logger.Warn("This file does not appear to be a valid SSH private key")
		confirmed, err := ui.AskConfirm("Continue anyway?", false)
		if err != nil || !confirmed {
			logger.Info("Upload cancelled")
			return nil
		}
	}

	// Calculate checksum
	hash := sha256.Sum256(keyData)
	checksum := hex.EncodeToString(hash[:])

	// Compress the data
	logger.Info("🗜️  Compressing...")
	compressed, err := compress.Compress(keyData)
	if err != nil {
		logger.Error("Failed to compress", "error", err)
		return err
	}

	// Get password for encryption
	password, err := ui.AskPassword("🔒 Enter password to encrypt the SSH key")
	if err != nil {
		return err
	}

	// Encrypt the data
	logger.Info("🔒 Encrypting...")
	encrypted, err := crypto.Encrypt(compressed, password)
	if err != nil {
		logger.Error("Failed to encrypt", "error", err)
		return err
	}

	// Create vault file
	now := time.Now()
	vaultFile := models.VaultFile{
		Metadata: models.SecretMetadata{
			Name:      name,
			Type:      models.SecretTypeSSH,
			CreatedAt: now,
			UpdatedAt: now,
			Size:      int64(len(keyData)),
			Checksum:  checksum,
		},
		Data: *encrypted,
	}

	// Serialize to JSON
	vaultJSON, err := json.MarshalIndent(vaultFile, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal vault file", "error", err)
		return err
	}

	// Upload to GitHub
	logger.Info("☁️  Uploading to GitHub...", "repo", cfg.Repo)
	client := github.NewClient(token)
	owner, repoName, err := github.ParseRepo(cfg.Repo)
	if err != nil {
		logger.Error("Invalid repo format", "error", err)
		return err
	}

	remotePath := fmt.Sprintf("ssh/%s.json", name)
	ctx := context.Background()

	// Check if file already exists
	var sha string
	existing, _ := client.GetContent(ctx, owner, repoName, remotePath)
	if existing != nil {
		sha = existing.Sha
	}

	// Prepare request
	req := github.ContentRequest{
		Message: fmt.Sprintf("Update SSH key: %s", name),
		Content: base64.StdEncoding.EncodeToString(vaultJSON),
	}
	if sha != "" {
		req.Sha = sha
		req.Message = fmt.Sprintf("Update SSH key: %s", name)
	} else {
		req.Message = fmt.Sprintf("Add SSH key: %s", name)
	}

	// Upload
	err = client.PutContent(ctx, owner, repoName, remotePath, req)
	if err != nil {
		logger.Error("Failed to upload", "error", err)
		return err
	}

	// Success!
	fmt.Println()
	logger.Info("✅ SSH key synced successfully!")
	logger.Info("   Name: %s", name)
	logger.Info("   Path: %s", remotePath)
	logger.Info("   Original size: %s", ui.FormatBytes(int64(len(keyData))))
	logger.Info("   Compressed: %s", ui.FormatBytes(int64(len(compressed))))
	logger.Info("   Encrypted: %s", ui.FormatBytes(int64(len(vaultJSON))))

	return nil
}

func init() {
	rootCmd.AddCommand(syncSSHCmd)
}
