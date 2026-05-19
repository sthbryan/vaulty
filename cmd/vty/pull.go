package main

import (
	"context"
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
	"github.com/sthbryan/vaulty/v2/internal/vault/providers"
	"github.com/sthbryan/vaulty/v2/pkg/models"
)

// <--- Main --->

var pullEnv string
var pullOutput string

func runPull(cmd *cobra.Command, args []string) error {
	if len(args) < 2 {
		return &CommandError{
			Message: "requires 2 arguments: <type> <name>",
			Hint:    "Usage: vty pull <type> <name> [-e env] [-o path]",
			Examples: []string{
				"vty pull env api -e production -o .env",
				"vty pull ssh deploy id_rsa",
				"vty pull config settings -o config.json",
			},
		}
	}

	secretType := models.SecretType(args[0])
	if !secretType.IsValid() {
		return &CommandError{
			Message: "invalid secret type: " + args[0],
			Hint:    "Valid types: env, config, ssh, resources",
		}
	}

	name := args[1]
	env := pullEnv
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

	provider := vault.NewProviderFromConfig()
	if provider == nil {
		return fmt.Errorf("failed to create storage provider")
	}

	ui.PrintBold("-- PULL --")
	fmt.Println()
	ui.PrintInfo(fmt.Sprintf("Type:    %s", secretType))
	ui.PrintInfo(fmt.Sprintf("Name:    %s", name))
	ui.PrintInfo(fmt.Sprintf("Env:     %s", env))
	fmt.Println()

	storagePath := filepath.Join(secretType.FolderName(), env, name)
	ui.PrintInfo(fmt.Sprintf("Looking: %s", storagePath))

	data, err := downloadFile(provider, storagePath+".vty")
	if err != nil {
		data, err = downloadFile(provider, storagePath)
		if err != nil {
			ui.PrintError(fmt.Sprintf("File not found: %s", name))
			ui.PrintInfo("Use 'vty list' to see available secrets")
			return &CommandError{
				Message: fmt.Sprintf("secret '%s' not found in %s/%s", name, secretType, env),
				Hint:    "Use 'vty show' to check available secrets",
			}
		}
	}

	var secretFile *models.SecretFile
	isEncrypted := false

	secretFile, err = tryDecrypt(data, masterKey)
	if err == nil && secretFile != nil {
		isEncrypted = true
		ui.PrintLock("Decrypted vault file")
	} else {
		secretFile = &models.SecretFile{
			Metadata: models.SecretMetadata{
				Name:  name,
				Type:  secretType,
				Env:   env,
				IsDir: strings.HasSuffix(name, ".tar.gz"),
			},
			Data: data,
		}
		ui.PrintInfo("Downloaded (unencrypted)")
	}

	outputPath := pullOutput
	if outputPath == "" {
		outputPath = secretFile.Metadata.Name
	}

	if secretFile.Metadata.IsDir {
		if !strings.HasSuffix(outputPath, ".tar.gz") && !strings.HasSuffix(outputPath, "/") {
			outputPath = outputPath + ".tar.gz"
		}
	}

	fmt.Println()

	if secretFile.Metadata.IsDir {
		ui.PrintInfo("Decompressing directory...")

		dirPath := strings.TrimSuffix(outputPath, ".tar.gz")
		if err := compress.DecompressDirectory(secretFile.Data, dirPath); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to decompress: %v", err))
			return fmt.Errorf("failed to decompress")
		}

		ui.PrintSuccess(fmt.Sprintf("Downloaded to: %s/", dirPath))
	} else {
		ui.PrintInfo(fmt.Sprintf("Writing: %s", outputPath))

		dir := filepath.Dir(outputPath)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				ui.PrintError(fmt.Sprintf("Failed to create directory: %v", err))
				return fmt.Errorf("failed to create directory")
			}
		}

		if err := os.WriteFile(outputPath, secretFile.Data, 0644); err != nil {
			ui.PrintError(fmt.Sprintf("Failed to write file: %v", err))
			return fmt.Errorf("failed to write file")
		}

		ui.PrintSuccess(fmt.Sprintf("Downloaded: %s", outputPath))
		if isEncrypted {
			ui.PrintInfo("File was encrypted and has been decrypted")
		}
	}

	return nil
}

func downloadFile(provider providers.StorageProvider, path string) ([]byte, error) {
	data, err := provider.Download(context.Background(), path)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func tryDecrypt(data []byte, masterKey []byte) (*models.SecretFile, error) {
	encryptedData, err := crypto.DeserializeEncryptedData(data)
	if err != nil {
		return nil, err
	}

	decrypted, err := crypto.DecryptWithKey(encryptedData, masterKey)
	if err != nil {
		return nil, err
	}

	var secretFile models.SecretFile
	if err := json.Unmarshal(decrypted, &secretFile); err != nil {
		return nil, err
	}

	return &secretFile, nil
}

// <--- Cobra --->

var pullCmd = &cobra.Command{
	Use:   "pull <type> <name>",
	Short: "Pull a secret from vault",
	Long: `Pull a file or directory from the vault.

Examples:
  vty pull env api -e production -o .env
  vty pull ssh deploy id_rsa
  vty pull config settings -o config.json
  vty pull resources assets -o ./public`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return &CommandError{
				Message: "requires 2 arguments: <type> <name>",
				Hint:    "Usage: vty pull <type> <name> [-e env] [-o path]",
			}
		}
		secretType := models.SecretType(args[0])
		if !secretType.IsValid() {
			return &CommandError{
				Message: "invalid secret type: " + args[0],
				Hint:    "Valid types: env, config, ssh, resources",
			}
		}
		return nil
	},
	RunE: runPull,
}

func init() {
	pullCmd.Flags().StringVarP(&pullEnv, "env", "e", "", "Environment (default: default)")
	pullCmd.Flags().StringVarP(&pullOutput, "output", "o", "", "Output path (default: current directory)")
	rootCmd.AddCommand(pullCmd)
}
