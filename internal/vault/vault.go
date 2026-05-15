package vault

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/sthbryan/vaulty/v2/internal/auth"
	"github.com/sthbryan/vaulty/v2/internal/ui"
	"github.com/sthbryan/vaulty/v2/internal/vault/providers"
	"github.com/sthbryan/vaulty/v2/pkg/models"
	"gopkg.in/yaml.v3"
)

type VaultInfo struct {
	Username    string
	VaultID     string
	StorageType string
}

type VaultInfoWithPassword struct {
	VaultInfo
	Password string
}

const vaultDir = ".vaulty"

func VaultDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, vaultDir)
}

func ConfigPath() string {
	return filepath.Join(VaultDir(), "config.yaml")
}

func MetaPath() string {
	return filepath.Join(VaultDir(), "vault.meta")
}

func SessionPath() string {
	return filepath.Join(VaultDir(), "session.yaml")
}

func ConfigExists() bool {
	_, err := os.Stat(ConfigPath())
	return err == nil
}

func LoadConfig() (*models.VaultConfig, error) {
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return nil, fmt.Errorf("Reading config: %w", err)
	}

	var config models.VaultConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("Parsing config: %w", err)
	}

	return &config, nil
}

func SaveConfig(config *models.VaultConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("Marshaling config: %w", err)
	}

	if err := os.MkdirAll(VaultDir(), 0700); err != nil {
		return fmt.Errorf("Creating vault dir: %w", err)
	}

	if err := os.WriteFile(ConfigPath(), data, 0600); err != nil {
		return fmt.Errorf("Writing config: %w", err)
	}

	return nil
}

func SessionExists() bool {
	_, err := os.Stat(SessionPath())
	return err == nil
}

func LoadSession() (*models.Session, error) {
	data, err := os.ReadFile(SessionPath())
	if err != nil {
		return nil, fmt.Errorf("Reading session: %w", err)
	}

	var session models.Session
	if err := yaml.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("Parsing session: %w", err)
	}

	return &session, nil
}

func SaveSession(session *models.Session) error {
	data, err := yaml.Marshal(session)
	if err != nil {
		return fmt.Errorf("Marshaling session: %w", err)
	}

	if err := os.MkdirAll(VaultDir(), 0700); err != nil {
		return fmt.Errorf("Creating vault dir: %w", err)
	}

	if err := os.WriteFile(SessionPath(), data, 0600); err != nil {
		return fmt.Errorf("Writing session: %w", err)
	}

	return nil
}

func DeleteSession() error {
	return os.Remove(SessionPath())
}

func CreateSession(username, vaultID, storageType string, hours int) error {
	session := &models.Session{
		Username:    username,
		VaultID:     vaultID,
		StorageType: storageType,
		ExpiresAt:   time.Now().Add(time.Duration(hours) * time.Hour),
	}
	return SaveSession(session)
}

func LoadMeta() (*models.VaultMeta, error) {
	data, err := os.ReadFile(MetaPath())
	if err != nil {
		return nil, fmt.Errorf("Reading meta: %w", err)
	}

	var meta models.VaultMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("Parsing meta: %w", err)
	}

	return &meta, nil
}

func SaveMeta(meta *models.VaultMeta) error {
	data, err := yaml.Marshal(meta)
	if err != nil {
		return fmt.Errorf("Marshaling meta: %w", err)
	}

	if err := os.MkdirAll(VaultDir(), 0700); err != nil {
		return fmt.Errorf("Creating vault dir: %w", err)
	}

	if err := os.WriteFile(MetaPath(), data, 0600); err != nil {
		return fmt.Errorf("Writing meta: %w", err)
	}

	return nil
}

func MetaExists() bool {
	_, err := os.Stat(MetaPath())
	return err == nil
}

func CreateVault(info VaultInfoWithPassword) error {
	authSvc := auth.New()

	salt, err := authSvc.GenerateSalt()
	if err != nil {
		return fmt.Errorf("Generating salt: %w", err)
	}

	vaultKey, err := authSvc.GenerateVaultKey()
	if err != nil {
		return fmt.Errorf("Generating vault key: %w", err)
	}

	encryptedKey, _, err := authSvc.EncryptVaultKey(vaultKey, info.Password)
	if err != nil {
		return fmt.Errorf("Encrypting vault key: %w", err)
	}

	now := time.Now()

	config := &models.VaultConfig{
		StorageType: info.StorageType,
		StoragePath: StoragePath(info.Username, info.VaultID),
		Username:    info.Username,
		VaultID:     info.VaultID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := SaveConfig(config); err != nil {
		return fmt.Errorf("Saving config: %w", err)
	}

	meta := &models.VaultMeta{
		Salt:         salt,
		EncryptedKey: encryptedKey,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	switch info.StorageType {
	case "github":
		token, err := providers.GetTokenForProvider("github")
		if err != nil {
			return fmt.Errorf("wizard cancelled")
		}
		p := providers.NewProvider(providers.ProviderGitHub, token, info.Username, info.VaultID)
		if err := p.SaveMeta(meta); err != nil {
			return fmt.Errorf("saving meta to GitHub: %w", err)
		}
	case "local":
		if err := SaveMeta(meta); err != nil {
			return fmt.Errorf("saving meta: %w", err)
		}
	default:
		return fmt.Errorf("Unsupported storage type: %s", info.StorageType)
	}

	return nil
}

func NewProvider(providerType providers.ProviderType, params ...string) providers.StorageProvider {
	switch providerType {
	case providers.ProviderGitHub:
		if len(params) >= 3 {
			return providers.NewProvider(providers.ProviderGitHub, params[0], params[1], params[2])
		}
	case providers.ProviderLocal:
		if len(params) >= 2 {
			return providers.NewProvider(providers.ProviderLocal, "", params[0], params[1])
		}
	}
	return nil
}

func SetupStorage(info VaultInfo) error {
	switch info.StorageType {
	case "github":
		token, err := providers.GetTokenForProvider("github")
		if err != nil {
			return fmt.Errorf("wizard cancelled")
		}

		ui.PrintInfo("Setting up GitHub storage...")
		stopSpinner := ui.PrintSpinner("Checking repository...")
		p := providers.NewProvider(providers.ProviderGitHub, token, info.Username, info.VaultID)
		if p.CheckVault() {
			stopSpinner()
			return fmt.Errorf("Repository already exists")
		}
		stopSpinner()

		if ok, _ := ui.Confirm("Repository not found. Create it?"); !ok {
			return fmt.Errorf("Cancelled")
		}

		ui.PrintInfo("Creating repository...")
		if err := p.SetupStorage(); err != nil {
			return fmt.Errorf("Failed to create repository: %w", err)
		}

	case "local":
		return nil

	default:
		return fmt.Errorf("Unsupported storage type: %s", info.StorageType)
	}

	return nil
}

func LoadMetaFromStorage(info VaultInfo) (*models.VaultMeta, error) {
	switch info.StorageType {
	case "github":
		token, err := providers.NewTokenManager("github").GetToken()
		if err != nil {
			return nil, fmt.Errorf("Wizard cancelled")
		}
		provider := providers.NewProvider(providers.ProviderGitHub, token, info.Username, info.VaultID)
		return provider.LoadMeta()
	case "local":
		return LoadMeta()
	default:
		return nil, fmt.Errorf("Unsupported storage type: %s", info.StorageType)
	}
}

func SaveMetaToRemote(providerType providers.ProviderType, meta *models.VaultMeta, params ...string) error {
	provider := NewProvider(providerType, params...)
	if provider == nil {
		return fmt.Errorf("Unsupported provider: %s", providerType)
	}

	data, err := yaml.Marshal(meta)
	if err != nil {
		return fmt.Errorf("Marshaling meta: %w", err)
	}

	return provider.Upload(context.Background(), "vault.meta", data)
}

func CheckVault(providerType providers.ProviderType, params ...string) bool {
	provider := NewProvider(providerType, params...)
	if provider == nil {
		return false
	}
	return provider.CheckVault()
}

func SaveAuthMethod(providerType providers.ProviderType, method string) error {
	return providers.UpdateAuthSettings(providerType, method, "")
}

func CurrentUser() string {
	user, err := user.Current()
	if err != nil {
		return ""
	}
	return user.Username
}

func VaultPath(vaultID string) string {
	return filepath.Join(VaultDir(), vaultID)
}

func StoragePath(username, vaultID string) string {
	return fmt.Sprintf("%s/%s", username, vaultID)
}

func CheckVaultAvailability(info VaultInfo) (bool, error) {
	var provider providers.StorageProvider

	switch info.StorageType {
	case "github":
		token, err := providers.GetTokenForProvider("github")
		if err != nil {
			return false, fmt.Errorf("wizard cancelled")
		}
		provider = providers.NewProvider(providers.ProviderGitHub, token, info.Username, info.VaultID)
	case "local":
		provider = providers.NewProvider(providers.ProviderLocal, "", info.Username, info.VaultID)
	default:
		return false, fmt.Errorf("Unsupported storage type: %s", info.StorageType)
	}

	if provider == nil {
		return false, fmt.Errorf("Failed to create provider")
	}

	return provider.CheckVault(), nil
}
