package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
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
	"github.com/sthbryan/vaulty/internal/password"
	"github.com/sthbryan/vaulty/internal/ui"
	"github.com/sthbryan/vaulty/pkg/models"
)

var (
	syncForce bool
)

var syncCmd = &cobra.Command{
	Use:   "sync <name> <path>",
	Short: "📦 Sync an environment file to Vaulty",
	Long: `Compress, encrypt, and upload an environment file to your Vaulty repository.

The file will be:
  1. Compressed using gzip for efficiency
  2. Encrypted using AES-256-GCM with your password
  3. Uploaded to your GitHub repository in the envs/ directory

Examples:
  vty sync production .env.production
  vty sync staging .env.staging --force
  vty sync api .env --password-stdin < password.txt`,
	Args: cobra.ExactArgs(2),
	RunE: runSync,
}

func runSync(cmd *cobra.Command, args []string) error {
	name := args[0]
	path := args[1]

	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("name cannot contain path separators")
	}
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("name cannot start with a dot")
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", path)
		}
		return fmt.Errorf("cannot access file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("path must be a file, not a directory: %s", path)
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("failed to get GitHub token: %w", err)
	}
	client := github.NewClient(token)

	storage, err := password.NewStorage()
	if err != nil {
		return fmt.Errorf("failed to create password storage: %w", err)
	}
	pwd, err := storage.Get()
	if err != nil {
		return fmt.Errorf("Password not found. Run 'vty init' or 'vty recover'")
	}

	ui.PrintInfo("📦 Reading file: %s", path)

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	originalSize := int64(len(content))
	ui.PrintInfo("📊 Original size: %s", ui.FormatBytes(originalSize))

	hash := sha256.Sum256(content)
	checksum := fmt.Sprintf("%x", hash)

	ui.PrintInfo("🗜️  Compressing...")
	compressed, err := compress.Compress(content)
	if err != nil {
		return fmt.Errorf("failed to compress: %w", err)
	}

	compressedSize := int64(len(compressed))
	ui.PrintInfo("📊 Compressed size: %s (%.1f%% reduction)",
		ui.FormatBytes(compressedSize),
		float64(originalSize-compressedSize)/float64(originalSize)*100)

	ui.PrintInfo("🔒 Encrypting...")
	encrypted, err := crypto.Encrypt(compressed, pwd)
	if err != nil {
		return fmt.Errorf("failed to encrypt: %w", err)
	}

	vaultFile := models.VaultFile{
		Metadata: models.SecretMetadata{
			Name:      name,
			Type:      models.SecretTypeEnv,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Size:      originalSize,
			Checksum:  checksum,
		},
		Data: *encrypted,
	}

	vaultData, err := json.MarshalIndent(vaultFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal vault file: %w", err)
	}

	ctx := context.Background()
	owner, repoName, err := github.ParseRepo(cfg.Repo)
	if err != nil {
		return fmt.Errorf("invalid repo format: %w", err)
	}

	remotePath := fmt.Sprintf("envs/%s.vty", name)
	ui.PrintInfo("☁️  Checking remote: %s/%s/%s", owner, repoName, remotePath)

	var existingSha string
	existingContent, err := client.GetContent(ctx, owner, repoName, remotePath)
	if err == nil && existingContent != nil {

		if !syncForce {
			ui.PrintWarning("⚠️  File already exists on remote")
			confirmed, confirmErr := ui.AskConfirm("Overwrite existing file?", false)
			if confirmErr != nil {
				return fmt.Errorf("confirmation failed: %w", confirmErr)
			}
			if !confirmed {
				ui.PrintInfo("❌ Sync cancelled")
				return nil
			}
		}
		existingSha = existingContent.Sha
		ui.PrintInfo("📝 Will overwrite existing file")
	}

	ui.PrintInfo("☁️  Uploading to GitHub...")

	encodedContent := base64.StdEncoding.EncodeToString(vaultData)
	commitMsg := fmt.Sprintf("Update %s via Vaulty sync", name)
	if existingSha == "" {
		commitMsg = fmt.Sprintf("Add %s via Vaulty sync", name)
	}

	req := github.ContentRequest{
		Message: commitMsg,
		Content: encodedContent,
		Sha:     existingSha,
	}

	if err := client.PutContent(ctx, owner, repoName, remotePath, req); err != nil {
		return fmt.Errorf("failed to upload: %w", err)
	}

	ui.PrintSuccess("✅ Synced successfully!")
	fmt.Println()
	fmt.Printf("  Name:    %s\n", name)
	fmt.Printf("  Path:    %s\n", remotePath)
	fmt.Printf("  Size:    %s → %s\n",
		ui.FormatBytes(originalSize),
		ui.FormatBytes(int64(len(vaultData))))
	fmt.Printf("  Repo:    %s\n", cfg.Repo)

	return nil
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().BoolVarP(&syncForce, "force", "f", false, "Overwrite without prompting")
}
