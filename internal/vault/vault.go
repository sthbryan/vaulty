package vault

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/sthbryan/vaulty/v2/pkg/models"
	"gopkg.in/yaml.v3"
)

var vaultDir = func() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".vaulty")
}()

func configPath() string {
	return filepath.Join(vaultDir, "config.yml")
}

func metaPath() string {
	return filepath.Join(vaultDir, "vault.meta")
}

func sessionPath() string {
	return filepath.Join(vaultDir, "session.yml")
}

func LoadConfig() (*models.VaultConfig, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var config models.VaultConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &config, nil
}

func SaveConfig(config *models.VaultConfig) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.MkdirAll(vaultDir, 0755); err != nil {
		return fmt.Errorf("creating vault dir: %w", err)
	}

	if err := os.WriteFile(configPath(), data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

func ConfigExists() bool {
	_, err := os.Stat(configPath())
	return err == nil
}

func LoadMeta() (*models.VaultMeta, error) {
	data, err := os.ReadFile(metaPath())
	if err != nil {
		return nil, fmt.Errorf("reading meta: %w", err)
	}

	var meta models.VaultMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parsing meta: %w", err)
	}

	return &meta, nil
}

func SaveMeta(meta *models.VaultMeta) error {
	data, err := yaml.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshaling meta: %w", err)
	}

	if err := os.MkdirAll(vaultDir, 0755); err != nil {
		return fmt.Errorf("creating vault dir: %w", err)
	}

	if err := os.WriteFile(metaPath(), data, 0600); err != nil {
		return fmt.Errorf("writing meta: %w", err)
	}

	return nil
}

func LoadSession() (*models.Session, error) {
	data, err := os.ReadFile(sessionPath())
	if err != nil {
		return nil, fmt.Errorf("reading session: %w", err)
	}

	var session models.Session
	if err := yaml.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("parsing session: %w", err)
	}

	return &session, nil
}

func SaveSession(session *models.Session) error {
	data, err := yaml.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}

	if err := os.MkdirAll(vaultDir, 0755); err != nil {
		return fmt.Errorf("creating vault dir: %w", err)
	}

	if err := os.WriteFile(sessionPath(), data, 0600); err != nil {
		return fmt.Errorf("writing session: %w", err)
	}

	return nil
}

func SessionExists() bool {
	_, err := os.Stat(sessionPath())
	return err == nil
}

func DeleteSession() error {
	if err := os.Remove(sessionPath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing session: %w", err)
	}
	return nil
}

func CreateSession(username, vaultID, storageType string, durationSeconds int64) error {
	session := &models.Session{
		Username:    username,
		VaultID:     vaultID,
		StorageType: storageType,
		ExpiresAt:   time.Now().Add(time.Duration(durationSeconds) * time.Second),
	}
	return SaveSession(session)
}

func CurrentUser() string {
	if u, err := user.Current(); err == nil {
		return u.Username
	}
	return ""
}

func VaultPath(vaultID string) string {
	return filepath.Join(vaultDir, vaultID)
}

func StoragePath(username, vaultID string) string {
	return fmt.Sprintf("%s/%s", username, vaultID)
}
