package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/DeadBryam/vaulty/internal/compress"
	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/DeadBryam/vaulty/pkg/models"
	"github.com/spf13/cobra"
)

var (
	pushResourceTag string
)

var pushResourceCmd = &cobra.Command{
	Use:   "resource <name> <path>",
	Short: "Push a file or directory to resources",
	Long: `Compress, encrypt, and upload a file or directory to the resources/ directory.

The file/directory will be:
  1. Compressed using tar+gzip for efficiency
  2. Encrypted using AES-256-GCM
  3. Uploaded to your GitHub repository as .vty file

Examples:
  vty push resource agents ./AGENTS.md
  vty push resource zellij ~/.config/zellij --tag dev
  vty push resource config.yml ./config.yml --tag team`,
	Args: cobra.ExactArgs(2),
	RunE: runPushResource,
}

var pushConfigCmd = &cobra.Command{
	Use:   "config <name> <path>",
	Short: "Push a file or directory to config",
	Long: `Compress, encrypt, and upload a file or directory to the config/ directory.

The file/directory will be:
  1. Compressed using tar+gzip for efficiency
  2. Encrypted using AES-256-GCM
  3. Uploaded to your GitHub repository as .vty file

Examples:
  vty push config opencode ~/.config/opencode
  vty push config zellij ~/.config/zellij --tag team`,
	Args: cobra.ExactArgs(2),
	RunE: runPushConfig,
}

type ResourceVaultFile struct {
	Metadata models.ResourceMetadata `json:"metadata"`
	Data     []byte                  `json:"data"`
}

func runPushResource(cmd *cobra.Command, args []string) error {
	return runPushResourceOrConfig(args[0], args[1], models.SecretTypeResource, "resources")
}

func runPushConfig(cmd *cobra.Command, args []string) error {
	return runPushResourceOrConfig(args[0], args[1], models.SecretTypeConfig, "config")
}

func runPushResourceOrConfig(name, path string, secretType models.SecretType, baseDir string) error {
	if err := validateName(name); err != nil {
		return err
	}

	cfg, client, err := loadConfigAndClient()
	if err != nil {
		return err
	}

	sess, err := ensureAuthenticated(cfg)
	if err != nil {
		return err
	}

	if err := checkPushPermissions(sess.Role); err != nil {
		return err
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path not found: %s", path)
		}
		return fmt.Errorf("cannot access path: %w", err)
	}

	isDirectory := info.IsDir()

	vaultFile, originalSize, err := prepareResourceFile(path, name, secretType, isDirectory)
	if err != nil {
		return err
	}

	var remotePath string
	if pushResourceTag != "" {
		remotePath = fmt.Sprintf("%s/%s/%s.vty", baseDir, pushResourceTag, name)
	} else {
		remotePath = fmt.Sprintf("%s/%s.vty", baseDir, name)
	}

	encryptedSize, err := encryptAndUploadResource(client, cfg, remotePath, vaultFile, sess.MasterKey, name)
	if err != nil {
		return err
	}

	ui.PrintSuccess("Pushed successfully!")
	fmt.Println()
	fmt.Printf("  Name:      %s\n", name)
	fmt.Printf("  Type:      %s\n", secretType)
	fmt.Printf("  Path:      %s\n", remotePath)
	fmt.Printf("  Encrypted: true\n")
	fmt.Printf("  Directory: %v\n", isDirectory)
	if pushResourceTag != "" {
		fmt.Printf("  Tag:       %s\n", pushResourceTag)
	}
	fmt.Printf("  Size:      %s → %s\n",
		ui.FormatBytes(originalSize),
		ui.FormatBytes(int64(encryptedSize)))
	fmt.Printf("  Repo:      %s\n", cfg.Repo)

	return nil
}

func prepareResourceFile(path, name string, secretType models.SecretType, isDirectory bool) (*ResourceVaultFile, int64, error) {
	ui.PrintInfo("Reading path: %s", path)

	var content []byte
	var originalSize int64

	if isDirectory {
		ui.PrintInfo("Compressing directory...")
		compressed, err := compress.CompressDirectory(path)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to compress directory: %w", err)
		}
		content = compressed
		originalSize = int64(len(compressed))
	} else {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read file: %w", err)
		}
		content = data
		originalSize = int64(len(data))
	}

	ui.PrintStats("Original size: %s", ui.FormatBytes(originalSize))

	ui.PrintInfo("Compressing...")
	compressed, err := compress.Compress(content)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to compress: %w", err)
	}

	compressedSize := int64(len(compressed))
	ui.PrintStats("Compressed size: %s (%.1f%% reduction)",
		ui.FormatBytes(compressedSize),
		float64(originalSize-compressedSize)/float64(originalSize)*100)

	hash := sha256.Sum256(content)
	checksum := fmt.Sprintf("%x", hash)

	vaultFile := &ResourceVaultFile{
		Metadata: models.ResourceMetadata{
			Name:        name,
			Type:        secretType,
			Tag:         pushResourceTag,
			IsEncrypted: true,
			IsDirectory: isDirectory,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Size:        originalSize,
			Checksum:    checksum,
		},
		Data: compressed,
	}

	return vaultFile, originalSize, nil
}

func encryptAndUploadResource(client *github.Client, cfg *config.Config, remotePath string, vaultFile *ResourceVaultFile, masterKey []byte, name string) (int, error) {
	ui.PrintLock("Encrypting...")

	vaultData, err := json.Marshal(vaultFile)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal vault file: %w", err)
	}

	hexEncrypted, err := crypto.EncryptBinary(vaultData, masterKey)
	if err != nil {
		return 0, fmt.Errorf("failed to encrypt: %w", err)
	}

	if err := uploadResourceToGitHub(client, cfg, remotePath, []byte(hexEncrypted), name); err != nil {
		return 0, err
	}

	return len(hexEncrypted), nil
}

func uploadResourceToGitHub(client *github.Client, cfg *config.Config, remotePath string, vaultData []byte, name string) error {
	ctx := context.Background()
	owner, repoName, err := github.ParseRepo(cfg.Repo)
	if err != nil {
		return fmt.Errorf("invalid repo format: %w", err)
	}

	ui.PrintCloud("Checking remote: %s/%s/%s", owner, repoName, remotePath)

	var existingSha string
	existingContent, err := client.GetContent(ctx, owner, repoName, remotePath)
	if err == nil && existingContent != nil {
		if !pushForce {
			ui.PrintWarning("File already exists on remote")
			confirmed, confirmErr := ui.AskConfirm("Overwrite existing file?", false)
			if confirmErr != nil {
				return fmt.Errorf("confirmation failed: %w", confirmErr)
			}
			if !confirmed {
				ui.PrintInfo("Push cancelled")
				return nil
			}
		}
		existingSha = existingContent.Sha
		ui.PrintInfo("Will overwrite existing file")
	}

	ui.PrintCloud("Uploading to GitHub...")

	encodedContent := base64.StdEncoding.EncodeToString(vaultData)
	commitMsg := fmt.Sprintf("Update %s via Vaulty push", name)
	if existingSha == "" {
		commitMsg = fmt.Sprintf("Add %s via Vaulty push", name)
	}

	req := github.ContentRequest{
		Message: commitMsg,
		Content: encodedContent,
		Sha:     existingSha,
	}

	if err := client.PutContent(ctx, owner, repoName, remotePath, req); err != nil {
		return fmt.Errorf("failed to upload: %w", err)
	}

	return nil
}

func init() {
	pushResourceCmd.Flags().StringVarP(&pushResourceTag, "tag", "t", "", "Tag for organizing resources (e.g., dev, team)")
	pushResourceCmd.Flags().BoolVarP(&pushForce, "force", "f", false, "Overwrite without prompting")

	pushConfigCmd.Flags().StringVarP(&pushResourceTag, "tag", "t", "", "Tag for organizing configs (e.g., dev, team)")
	pushConfigCmd.Flags().BoolVarP(&pushForce, "force", "f", false, "Overwrite without prompting")

	pushCmd.AddCommand(pushResourceCmd)
	pushCmd.AddCommand(pushConfigCmd)
}
