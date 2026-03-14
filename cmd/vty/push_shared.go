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

	"github.com/DeadBryam/vaulty/internal/compress"
	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/github"
	"github.com/DeadBryam/vaulty/internal/session"
	"github.com/DeadBryam/vaulty/internal/storage"
	"github.com/DeadBryam/vaulty/internal/ui"
	"github.com/DeadBryam/vaulty/pkg/models"
)

type BinaryVaultFile struct {
	Metadata models.SecretMetadata `json:"metadata"`
	Data     []byte                `json:"data"`
}

func checkPushPermissions(role string) error {
	if role == "" {
		return fmt.Errorf("no active session. Run 'vty login' first")
	}
	if role == "viewer" {
		return fmt.Errorf("viewers cannot push secrets. Contact the repository owner for access")
	}
	return nil
}

func validateName(name string) error {
	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("name cannot contain path separators")
	}
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("name cannot start with a dot")
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
		return fmt.Errorf("path must be a file, not a directory: %s", path)
	}
	return nil
}

func loadConfigAndClient() (*config.Config, *github.Client, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, nil, fmt.Errorf("configuration error: %w", err)
	}

	token, err := github.GetGitHubToken()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get GitHub token: %w", err)
	}
	client := github.NewClient(token)

	return cfg, client, nil
}

func getStorage(cfg *config.Config) (storage.Storage, error) {
	if cfg.IsLocalMode() {
		return storage.NewLocalStorage()
	}

	if cfg.Repo == "" {
		return nil, fmt.Errorf("no repository configured for cloud mode")
	}

	token, err := github.GetGitHubToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub token: %w", err)
	}

	return storage.NewGitHubStorage(token, cfg.Repo)
}

func loadConfigAndStorage() (*config.Config, storage.Storage, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	s, err := getStorage(cfg)
	if err != nil {
		return nil, nil, err
	}

	return cfg, s, nil
}

func encryptAndPrepareFileWithSession(path, name string, secretType models.SecretType, sess *session.Session) (*BinaryVaultFile, int64, error) {
	ui.PrintInfo("Reading file: %s", path)

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read file: %w", err)
	}

	originalSize := int64(len(content))
	ui.PrintStats("Original size: %s", ui.FormatBytes(originalSize))

	hash := sha256.Sum256(content)
	checksum := fmt.Sprintf("%x", hash)

	ui.PrintInfo("Compressing...")
	compressed, err := compress.Compress(content)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to compress: %w", err)
	}

	compressedSize := int64(len(compressed))
	ui.PrintStats("Compressed size: %s (%.1f%% reduction)",
		ui.FormatBytes(compressedSize),
		float64(originalSize-compressedSize)/float64(originalSize)*100)

	vaultFile := &BinaryVaultFile{
		Metadata: models.SecretMetadata{
			Name:      name,
			Type:      secretType,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Size:      originalSize,
			Checksum:  checksum,
		},
		Data: compressed,
	}

	return vaultFile, originalSize, nil
}

func encryptAndUploadBinary(client *github.Client, cfg *config.Config, remotePath string, vaultFile *BinaryVaultFile, masterKey []byte, name string) (int, error) {
	ui.PrintLock("Encrypting as binary...")

	vaultData, err := json.Marshal(vaultFile)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal vault file: %w", err)
	}

	hexEncrypted, err := crypto.EncryptBinary(vaultData, masterKey)
	if err != nil {
		return 0, fmt.Errorf("failed to encrypt binary: %w", err)
	}

	if err := uploadToGitHub(client, cfg, remotePath, []byte(hexEncrypted), name); err != nil {
		return 0, err
	}

	return len(hexEncrypted), nil
}

func encryptAndUploadWithStorage(s storage.Storage, cfg *config.Config, remotePath string, vaultFile *BinaryVaultFile, masterKey []byte, name string) (int, error) {
	ui.PrintLock("Encrypting as binary...")

	vaultData, err := json.Marshal(vaultFile)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal vault file: %w", err)
	}

	hexEncrypted, err := crypto.EncryptBinary(vaultData, masterKey)
	if err != nil {
		return 0, fmt.Errorf("failed to encrypt binary: %w", err)
	}

	if err := uploadToStorage(s, cfg, remotePath, []byte(hexEncrypted), name); err != nil {
		return 0, err
	}

	return len(hexEncrypted), nil
}

func uploadToGitHub(client *github.Client, cfg *config.Config, remotePath string, vaultData []byte, name string) error {
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

func uploadToStorage(s storage.Storage, cfg *config.Config, remotePath string, vaultData []byte, name string) error {
	ctx := context.Background()

	var env, envName string

	if strings.HasPrefix(remotePath, "envs/") {
		parts := strings.Split(strings.TrimPrefix(remotePath, "envs/"), "/")
		if len(parts) == 1 {

			env = ""
			envName = strings.TrimSuffix(parts[0], ".vty")
		} else if len(parts) == 2 {

			env = parts[0]
			envName = strings.TrimSuffix(parts[1], ".vty")
		}

		ui.PrintCloud("Checking remote: %s/%s", s.GetRepo(), remotePath)

		_, err := s.GetEnv(ctx, env, envName)
		if err == nil {
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
			ui.PrintInfo("Will overwrite existing file")
		}

		ui.PrintCloud("Uploading to storage...")
		if err := s.PutEnv(ctx, env, envName, vaultData); err != nil {
			return fmt.Errorf("failed to upload: %w", err)
		}
		return nil
	}

	if strings.HasPrefix(remotePath, "ssh/") {
		parts := strings.Split(strings.TrimPrefix(remotePath, "ssh/"), "/")
		if len(parts) == 2 {
			username := parts[0]
			keyName := strings.TrimSuffix(parts[1], ".vty")

			ui.PrintCloud("Checking remote: %s/ssh/%s", s.GetRepo(), username)

			_, err := s.GetSSHKey(ctx, username, keyName)
			if err == nil {
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
				ui.PrintInfo("Will overwrite existing file")
			}

			ui.PrintCloud("Uploading SSH key to storage...")
			if err := s.PutSSHKey(ctx, username, keyName, vaultData); err != nil {
				return fmt.Errorf("failed to upload SSH key: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("unsupported path format: %s", remotePath)
}
