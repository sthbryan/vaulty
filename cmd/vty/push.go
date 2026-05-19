package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/v2/internal/compress"
	"github.com/sthbryan/vaulty/v2/internal/crypto"
	"github.com/sthbryan/vaulty/v2/internal/ui"
	"github.com/sthbryan/vaulty/v2/internal/vault"
	"github.com/sthbryan/vaulty/v2/pkg/models"
)

// <--- Main --->

var pushForce bool
var pushEnv string
var pushEncrypt bool

func runPush(cmd *cobra.Command, args []string) error {
	secretType := models.SecretType(args[0])
	name := args[1]
	path := args[2]
	env := pushEnv
	if env == "" {
		env = "default"
	}

	session, err := vault.RequireSession()
	if err != nil {
		return err
	}

	masterKey, err := vault.GetMasterKey(session)
	if err != nil {
		return err
	}

	if err := validatePushName(name); err != nil {
		return err
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &CommandError{
				Message: fmt.Sprintf("file not found: %s", path),
				Hint:    "Check that the file path exists",
			}
		}
		return fmt.Errorf("cannot access path: %w", err)
	}

	isDir := info.IsDir()

	if secretType == models.SecretTypeEnv {
		if err := validateEnvFile(path, isDir); err != nil {
			return err
		}
	}

	if secretType == models.SecretTypeSSH {
		if err := validateSSHFile(path); err != nil {
			return err
		}
	}

	ui.PrintBold("-- PUSH --")
	fmt.Println()
	ui.PrintInfo(fmt.Sprintf("Type:    %s", secretType))
	ui.PrintInfo(fmt.Sprintf("Name:    %s", name))
	ui.PrintInfo(fmt.Sprintf("Env:     %s", env))
	ui.PrintInfo(fmt.Sprintf("Path:    %s", path))
	if !pushEncrypt {
		ui.PrintWarning("WARNING: File will NOT be encrypted")
	}
	fmt.Println()

	ok, err := ui.Confirm("Push to vault?")
	if err != nil || !ok {
		return fmt.Errorf("cancelled")
	}

	fmt.Println()

	storagePath, encrypted, err := pushPrepareAndEncrypt(path, name, env, isDir, pushEncrypt, masterKey, secretType)
	if err != nil {
		return err
	}

	provider := vault.NewProviderFromConfig()
	if provider == nil {
		return fmt.Errorf("failed to create storage provider")
	}

	ui.PrintInfo("Uploading to storage...")

	if err := provider.Upload(context.Background(), storagePath, encrypted); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to upload: %v", err))
		return fmt.Errorf("failed to upload")
	}

	ui.PrintSuccess("Pushed successfully!")
	ui.PrintInfo(fmt.Sprintf("Storage path: %s", storagePath))

	return nil
}

func pushPrepareAndEncrypt(path, name, env string, isDir, encrypt bool, masterKey []byte, secretType models.SecretType) (string, []byte, error) {
	ui.PrintInfo("Reading file...")

	content, err := os.ReadFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read file: %w", err)
	}

	originalSize := int64(len(content))
	ui.PrintStats(fmt.Sprintf("Original size: %s", ui.FormatBytes(originalSize)))

	var finalContent []byte

	if isDir {
		ui.PrintInfo("Compressing directory...")
		compressed, err := compress.CompressDirectory(path)
		if err != nil {
			return "", nil, fmt.Errorf("failed to compress: %w", err)
		}
		finalContent = compressed
		name = name + ".tar.gz"
		ui.PrintStats(fmt.Sprintf("Compressed: %s", ui.FormatBytes(int64(len(finalContent)))))
	} else {
		ui.PrintInfo("Compressing...")
		compressed, err := compress.Compress(content)
		if err != nil {
			return "", nil, fmt.Errorf("failed to compress: %w", err)
		}
		finalContent = compressed
		ui.PrintStats(fmt.Sprintf("Compressed: %s (%.1f%% reduction)",
			ui.FormatBytes(int64(len(finalContent))),
			float64(originalSize-int64(len(finalContent)))/float64(originalSize)*100))
	}

	var encrypted []byte

	if encrypt {
		ui.PrintLock("Encrypting...")
		hash := sha256.Sum256(finalContent)
		checksum := fmt.Sprintf("%x", hash)

		secretFile := &models.SecretFile{
			Metadata: models.SecretMetadata{
				Name:      name,
				Type:      secretType,
				Env:       env,
				IsDir:     isDir,
				Encrypted: true,
				Size:      originalSize,
				Checksum:  checksum,
			},
			Data: finalContent,
		}

		jsonData, err := json.Marshal(secretFile)
		if err != nil {
			return "", nil, fmt.Errorf("failed to marshal secret file: %w", err)
		}

		encryptedData, err := crypto.EncryptWithKey(jsonData, masterKey)
		if err != nil {
			return "", nil, fmt.Errorf("failed to encrypt: %w", err)
		}
		encrypted = crypto.SerializeEncryptedData(encryptedData)
	} else {
		encrypted = finalContent
	}

	storagePath := filepath.Join(secretType.FolderName(), env, name)
	if encrypt {
		storagePath = storagePath + ".vty"
	}

	ui.PrintLock(fmt.Sprintf("Uploading: %s", storagePath))

	return storagePath, encrypted, nil
}

func validatePushName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("name cannot contain path separators")
	}
	return nil
}

func validateEnvFile(path string, isDir bool) error {
	if isDir {
		return nil
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".env" {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	validLines := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, "=") {
			validLines++
		}
	}

	if validLines > 0 {
		return nil
	}

	return fmt.Errorf("file does not appear to be .env format (no KEY=value patterns found)")
}

func validateSSHFile(path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if strings.Contains(string(content), "-----BEGIN") {
		return nil
	}

	return fmt.Errorf("file does not appear to be an SSH key (no -----BEGIN marker)")
}

// <--- Cobra --->

var pushCmd = &cobra.Command{
	Use:   "push <type> <name> <path>",
	Short: "Push a secret to vault",
	Long: `Push a file or directory to the vault.

Examples:
  vty push env api .env
  vty push env app .env -e production
  vty push ssh deploy id_rsa -f
  vty push config settings config.json
  vty push resources assets ./public --encrypt=false`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 3 {
			return &CommandError{
				Message: "requires 3 arguments: <type> <name> <path>",
				Hint:    "Run 'vty push --help' for usage",
			}
		}
		secretType := models.SecretType(args[0])
		if !secretType.IsValid() {
			return &CommandError{
				Message: "invalid secret type: " + args[0],
				Hint:    "Valid types: env, config, ssh, resources",
				Examples: []string{
					"vty push env api .env",
					"vty push ssh deploy id_rsa",
				},
			}
		}
		return nil
	},
	RunE: runPush,
}

func init() {
	pushCmd.Flags().BoolVarP(&pushForce, "force", "f", false, "Overwrite existing file")
	pushCmd.Flags().StringVarP(&pushEnv, "env", "e", "", "Environment (default: default)")
	pushCmd.Flags().BoolVar(&pushEncrypt, "encrypt", true, "Encrypt the file (default: true)")
	rootCmd.AddCommand(pushCmd)
}
