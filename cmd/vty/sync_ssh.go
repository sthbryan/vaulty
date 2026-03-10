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

	"github.com/DeadBryam/vaulty/internal/compress"
	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/password"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/DeadBryam/vaulty/pkg/models"
	"github.com/spf13/cobra"
)

var syncSSHCmd = &cobra.Command{
	Use:   "sync-ssh <name> <key_path>",
	Short: "Sync an SSH private key to your vault",
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

	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if strings.HasSuffix(keyPath, ".pub") {
		logger.Error("This appears to be a public key file. Please provide a private key (without .pub extension)")
		return fmt.Errorf("public key file provided")
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		logger.Error("Key file not found", "path", keyPath)
		return fmt.Errorf("key file not found: %s", keyPath)
	}

	fmt.Println()
	logger.Warn("🔐 WARNING: You are about to upload a PRIVATE SSH key to GitHub")
	logger.Warn("🔑 Private keys are sensitive credentials that grant access to your systems")
	logger.Warn("⚠️ Ensure this repository is private and you understand the security implications")
	fmt.Println()

	confirmed, err := ui.AskConfirm("☁️  Do you want to continue uploading this private key?", false)
	if err != nil {
		return fmt.Errorf("failed to get confirmation: %w", err)
	}
	if !confirmed {
		logger.Info("Upload cancelled")
		return nil
	}

	cfg, err := config.Load("")
	if err != nil {
		logger.Error("Failed to load config", "error", err)
		return err
	}

	if err := cfg.Validate(); err != nil {
		logger.Error("Repository not configured. Run: vty init <owner/repo>")
		return err
	}

	token, err := github.GetGitHubToken()
	if err != nil {
		logger.Error("Failed to get GitHub token", "error", err)
		return err
	}

	logger.Info("🔑 Reading private key...", "path", keyPath)
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		logger.Error("Failed to read key file", "error", err)
		return err
	}

	content := string(keyData)
	if !strings.Contains(content, "BEGIN") || !strings.Contains(content, "PRIVATE KEY") {
		logger.Warn("This file does not appear to be a valid SSH private key")
		confirmed, err := ui.AskConfirm("Continue anyway?", false)
		if err != nil || !confirmed {
			logger.Info("Upload cancelled")
			return nil
		}
	}

	hash := sha256.Sum256(keyData)
	checksum := hex.EncodeToString(hash[:])

	logger.Info("🗜️  Compressing...")
	compressed, err := compress.Compress(keyData)
	if err != nil {
		logger.Error("Failed to compress", "error", err)
		return err
	}

	storage, err := password.NewStorage()
	if err != nil {
		logger.Error("Failed to create password storage", "error", err)
		return err
	}
	password, err := storage.Get()
	if err != nil {
		logger.Error("Failed to get password from storage", "error", err)
		return err
	}
	if password == "" {
		return fmt.Errorf("Password not found. Run 'vty init' or 'vty recover'")
	}

	logger.Info("🔒 Encrypting...")
	encrypted, err := crypto.Encrypt(compressed, password)
	if err != nil {
		logger.Error("Failed to encrypt", "error", err)
		return err
	}

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

	vaultJSON, err := json.MarshalIndent(vaultFile, "", "  ")
	if err != nil {
		logger.Error("Failed to marshal vault file", "error", err)
		return err
	}

	logger.Info("☁️  Uploading to GitHub...", "repo", cfg.Repo)
	client := github.NewClient(token)
	owner, repoName, err := github.ParseRepo(cfg.Repo)
	if err != nil {
		logger.Error("Invalid repo format", "error", err)
		return err
	}

	remotePath := fmt.Sprintf("ssh/%s.vty", name)
	ctx := context.Background()

	var sha string
	existing, _ := client.GetContent(ctx, owner, repoName, remotePath)
	if existing != nil {
		sha = existing.Sha
	}

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

	err = client.PutContent(ctx, owner, repoName, remotePath, req)
	if err != nil {
		logger.Error("Failed to upload", "error", err)
		return err
	}

	fmt.Println()
	logger.Info("✅ SSH key synced successfully!")
	logger.Info(fmt.Sprintf("   Name: %s", name))
	logger.Info(fmt.Sprintf("   Path: %s", remotePath))
	logger.Info(fmt.Sprintf("   Original size: %s", ui.FormatBytes(int64(len(keyData)))))
	logger.Info(fmt.Sprintf("   Compressed: %s", ui.FormatBytes(int64(len(compressed)))))
	logger.Info(fmt.Sprintf("   Encrypted: %s", ui.FormatBytes(int64(len(vaultJSON)))))

	return nil
}

func init() {
	rootCmd.AddCommand(syncSSHCmd)
}
