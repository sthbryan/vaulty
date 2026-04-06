package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/DeadBryam/vaulty/internal/compress"
	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/models"
	"github.com/DeadBryam/vaulty/internal/storage"
	"github.com/DeadBryam/vaulty/internal/ui"
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
  vty push resource zellij ~/.config/zellij --tag dev
  vty push resource zellij ~/.config/zellij --tag team
  vty push resource opencode ~/.config/opencode
  vty push resource vscode-settings ~/Library/Application\ Support/Code/User/settings.json`,
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
  vty push config zellij ~/.config/zellij --tag team
  vty push config vscode-settings ~/Library/Application\ Support/Code/User/settings.json`,
	RunE: runPushConfig,
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
	if pushResourceTag != "" {
		if err := validateName(pushResourceTag); err != nil {
			return fmt.Errorf("invalid tag: %w", err)
		}
	}

	cfg, s, err := loadConfigAndStorage()
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

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path not found: %s", path)
		}
		return fmt.Errorf("cannot access path: %w", err)
	}

	isDirectory := info.IsDir()

	vaultFile, originalSize, err := prepareResourceFile(absPath, name, secretType, isDirectory)
	if err != nil {
		return err
	}

	var remotePath string
	if pushResourceTag != "" {
		remotePath = fmt.Sprintf("%s/%s/%s.vty", baseDir, pushResourceTag, name)
	} else {
		remotePath = fmt.Sprintf("%s/%s.vty", baseDir, name)
	}

	encryptedSize, err := encryptAndUploadResource(s, remotePath, vaultFile, sess.MasterKey, cfg)
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
	fmt.Printf("  Size:      %s → %s\n",
		ui.FormatBytes(originalSize),
		ui.FormatBytes(int64(encryptedSize)))

	if cfg.IsLocalMode() {
		fmt.Println()
		fmt.Println(ui.MutedStyle.Render("  Local storage: ~/.vaulty"))
	}

	return nil
}

func prepareResourceFile(path, name string, secretType models.SecretType, isDirectory bool) (*ResourceVaultFile, int64, error) {
	ui.PrintInfo("Reading path: %s", path)

	var originalData []byte
	var originalSize int64

	if isDirectory {
		ui.PrintInfo("Compressing directory...")
		tarData, err := compress.CompressDirectory(path)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to compress directory: %w", err)
		}
		originalData = tarData
		originalSize = int64(len(tarData))
	} else {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read file: %w", err)
		}
		originalData = data
		originalSize = int64(len(data))
	}

	ui.PrintStats("Original size: %s", ui.FormatBytes(originalSize))

	ui.PrintInfo("Compressing...")
	compressed, err := compress.Compress(originalData)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to compress: %w", err)
	}

	compressedSize := int64(len(compressed))
	ui.PrintStats("Compressed size: %s (%.1f%% reduction)",
		ui.FormatBytes(compressedSize),
		ui.FormatBytes(originalSize),
		float64(originalSize-compressedSize)/float64(originalSize)*100)

	hash := sha256.Sum256(originalData)
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

func encryptAndUploadResource(s storage.Storage, remotePath string, vaultFile *ResourceVaultFile, masterKey []byte, cfg *config.Config) (int, error) {
	ui.PrintLock("Encrypting...")

	vaultData, err := json.Marshal(vaultFile)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal vault file: %w", err)
	}

	hexEncrypted, err := crypto.EncryptBinary(vaultData, masterKey)
	if err != nil {
		return 0, fmt.Errorf("failed to encrypt: %w", err)
	}

	if err := uploadResourceToStorage(s, remotePath, []byte(hexEncrypted), cfg); err != nil {
		return 0, err
	}

	return len(hexEncrypted), nil
}

func uploadResourceToStorage(s storage.Storage, remotePath string, vaultData []byte, cfg *config.Config) error {
	ctx := context.Background()

	if cfg.IsLocalMode() {
		ui.PrintCloud("Saving to local storage: %s", remotePath)
	} else {
		ui.PrintCloud("Checking remote: %s/%s", s.GetRepo(), remotePath)
	}

	err := s.PutResource(ctx, remotePath, vaultData)
	if err != nil {
		return fmt.Errorf("failed to upload: %w", err)
	}

	if !cfg.IsLocalMode() {
		ui.PrintSuccess("Uploaded to GitHub")
	}

	return nil
}

func validateFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", path)
		}
		return fmt.Errorf("cannot access file: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("expected a file, got directory: %s", path)
	}

	return nil
}

func getPlatform() string {
	if runtime.GOOS == "darwin" {
		return "macOS"
	}
	return runtime.GOOS
}

const maxConcurrentPull = 5
