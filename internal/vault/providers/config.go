package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sthbryan/vaulty/v2/pkg/models"
	"gopkg.in/yaml.v3"
)

const vaultDir = ".vaulty"

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, vaultDir, "config.yaml")
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

	home, _ := os.UserHomeDir()
	vaultDir := filepath.Join(home, ".vaulty")
	if err := os.MkdirAll(vaultDir, 0700); err != nil {
		return fmt.Errorf("creating vault dir: %w", err)
	}

	if err := os.WriteFile(configPath(), data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

func UpdateAuthSettings(provider, method, encryptedToken string) error {
	config, err := LoadConfig()
	if err != nil {

		config = &models.VaultConfig{}
	}

	config.Auth = models.AuthSettings{
		Provider:       provider,
		Method:         method,
		EncryptedToken: encryptedToken,
	}
	config.UpdatedAt = time.Now()

	return SaveConfig(config)
}
