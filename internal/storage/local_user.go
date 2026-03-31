package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type LocalUserStorage struct {
	baseDir string
}

func NewLocalUserStorage(basePath string) (*LocalUserStorage, error) {
	if basePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		basePath = filepath.Join(homeDir, ".vty", "vault")
	}
	return &LocalUserStorage{baseDir: basePath}, nil
}

func (l *LocalUserStorage) GetUserKeys(ctx context.Context, username string) ([]byte, error) {
	path := filepath.Join(l.baseDir, "keys", fmt.Sprintf("%s.vty", username))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("user keys not found for %s", username)
		}
		return nil, fmt.Errorf("failed to read user keys: %w", err)
	}
	return data, nil
}

func (l *LocalUserStorage) PutUserKeys(ctx context.Context, username string, data []byte) error {
	keysDir := filepath.Join(l.baseDir, "keys")
	if err := os.MkdirAll(keysDir, 0700); err != nil {
		return fmt.Errorf("failed to create keys directory: %w", err)
	}
	path := filepath.Join(keysDir, fmt.Sprintf("%s.vty", username))
	return os.WriteFile(path, data, 0600)
}

func (l *LocalUserStorage) GetRecoverySeed(ctx context.Context, username string) ([]byte, error) {
	path := filepath.Join(l.baseDir, "recovery", fmt.Sprintf("%s.recovery.vty", username))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("recovery seed not found for %s", username)
		}
		return nil, fmt.Errorf("failed to read recovery seed: %w", err)
	}
	return data, nil
}

func (l *LocalUserStorage) PutRecoverySeed(ctx context.Context, username string, data []byte) error {
	recoveryDir := filepath.Join(l.baseDir, "recovery")
	if err := os.MkdirAll(recoveryDir, 0700); err != nil {
		return fmt.Errorf("failed to create recovery directory: %w", err)
	}
	path := filepath.Join(recoveryDir, fmt.Sprintf("%s.recovery.vty", username))
	return os.WriteFile(path, data, 0600)
}

func (l *LocalUserStorage) GetUserList(ctx context.Context) ([]byte, error) {
	return nil, fmt.Errorf("GetUserList not implemented")
}
