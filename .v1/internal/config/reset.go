package config

import (
	"fmt"
	"os"
	"path/filepath"
)

func ResetVaultConfig() error {
	configPath := DefaultPath()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil
	}

	if err := os.Remove(configPath); err != nil {
		return fmt.Errorf("removing config: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		vaultyDir := filepath.Join(homeDir, ".vty")

		_ = vaultyDir
	}

	return nil
}
