package start

import (
	"context"
	"fmt"
	"time"

	"github.com/sthbryan/vaulty/v2/internal/auth"
	"github.com/sthbryan/vaulty/v2/internal/storage"
	"github.com/sthbryan/vaulty/v2/internal/ui"
	"github.com/sthbryan/vaulty/v2/internal/vault"
	"github.com/sthbryan/vaulty/v2/pkg/models"
	"gopkg.in/yaml.v3"
)

func LoadMeta() (*models.VaultMeta, error) {
	return vault.LoadMeta()
}

func SaveMeta(meta *models.VaultMeta) error {
	return vault.SaveMeta(meta)
}

func LoadConfig() (*models.VaultConfig, error) {
	return vault.LoadConfig()
}

func SaveConfig(config *models.VaultConfig) error {
	return vault.SaveConfig(config)
}

func ConfigExists() bool {
	return vault.ConfigExists()
}

func GetCurrentUser() string {
	return vault.CurrentUser()
}

func GetVaultPath(vaultID string) string {
	return vault.VaultPath(vaultID)
}

func GetStoragePath(username, vaultID string) string {
	return fmt.Sprintf("%s/%s", username, vaultID)
}

func CheckInStorage(detect *ui.DetectState, storageType string) (bool, string) {
	if storageType == "local" {
		config, err := LoadConfig()
		if err == nil && config.Username == detect.Username && config.VaultID == detect.VaultID && config.StorageType == "local" {
			return true, ""
		}
		return false, ""
	}

	token, err := GetGitHubToken()
	if err != nil {
		return false, ""
	}

	if CheckGitHubVault(token, detect.Username, detect.VaultID) {
		return true, token
	}

	return false, token
}

func SetupGitHubStorage(username, vaultID string, token string) (string, error) {
	ui.PrintInfo("Setting up GitHub storage...")

	stopSpinner := ui.PrintSpinner("Checking repository...")
	if CheckGitHubVault(token, username, vaultID) {
		stopSpinner()
		return "", fmt.Errorf("repository already exists")
	}
	stopSpinner()

	createRepo, err := ui.Confirm("Repository not found. Create it?")
	if err != nil {
		return "", fmt.Errorf("cancelled")
	}

	if !createRepo {
		return "", fmt.Errorf("repository not accessible")
	}

	ui.PrintInfo("Creating repository...")
	if err := CreateGitHubRepo(token, username, vaultID); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to create repository: %v", err))
		return "", fmt.Errorf("failed to create repository")
	}

	return fmt.Sprintf("%s/%s", username, vaultID), nil
}

func CreateVault(username, vaultID, storageType, storagePath, password, token string) error {
	authSvc := auth.New()

	salt, err := authSvc.GenerateSalt()
	if err != nil {
		return err
	}

	vaultKey, err := authSvc.GenerateVaultKey()
	if err != nil {
		return err
	}

	encryptedKey, _, err := authSvc.EncryptVaultKey(vaultKey, password)
	if err != nil {
		return err
	}

	now := time.Now()
	config := &models.VaultConfig{
		StorageType: storageType,
		StoragePath: storagePath,
		Username:    username,
		VaultID:     vaultID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := SaveConfig(config); err != nil {
		return err
	}

	meta := &models.VaultMeta{
		Salt:         salt,
		EncryptedKey: encryptedKey,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if storageType == "github" && token != "" {
		if err := UploadMetaToGitHub(token, username, vaultID, meta); err != nil {
			return fmt.Errorf("uploading vault meta to GitHub")
		}
	} else {
		if err := SaveMeta(meta); err != nil {
			return err
		}
	}

	return nil
}

func ValidateCreate(state *ui.CreateState) error {
	if state.Password != state.ConfirmPassword {
		return fmt.Errorf("passwords do not match")
	}
	return ui.ValidatePassword(state.Password)
}

func IsFileNotFound(err error) bool {
	return err != nil && containsString(err.Error(), "file not found")
}

func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func CreateSessionFromRunner(username, vaultID, storageType, duration string) error {
	authSvc := auth.New()
	hours, err := authSvc.GenerateSessionDuration(duration)
	if err != nil {
		return err
	}
	return CreateSession(username, vaultID, storageType, hours)
}

func LoadMetaFromGitHubForRunner(token, username, vaultID string) (*models.VaultMeta, error) {
	ghStorage := storage.NewGitHubStorage(token, username, vaultID)
	data, err := ghStorage.Download(context.Background(), "vault.meta")
	if err != nil {
		return nil, fmt.Errorf("downloading vault.meta: %w", err)
	}

	var meta models.VaultMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parsing vault.meta: %w", err)
	}

	return &meta, nil
}
