package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/DeadBryam/vaulty/internal/crypto"
)

type LocalVaultStorage struct {
	baseDir string
}

func NewLocalVaultStorage(basePath string) (*LocalVaultStorage, error) {
	if basePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		basePath = filepath.Join(homeDir, ".vty", "vault")
	}

	if err := os.MkdirAll(basePath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create vault directory: %w", err)
	}

	return &LocalVaultStorage{baseDir: basePath}, nil
}

func (l *LocalVaultStorage) GetVault(ctx context.Context) ([]byte, error) {
	path := filepath.Join(l.baseDir, "vault.vty")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("vault not found")
		}
		return nil, fmt.Errorf("failed to read vault: %w", err)
	}
	return data, nil
}

func (l *LocalVaultStorage) PutVault(ctx context.Context, data []byte) error {
	path := filepath.Join(l.baseDir, "vault.vty")
	return os.WriteFile(path, data, 0600)
}

func (l *LocalVaultStorage) GetMetadata(ctx context.Context) ([]byte, error) {
	path := filepath.Join(l.baseDir, "metadata.vty")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("metadata not found")
		}
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}
	return crypto.DecompressHex(string(data))
}

func (l *LocalVaultStorage) PutMetadata(ctx context.Context, data []byte) error {
	path := filepath.Join(l.baseDir, "metadata.vty")
	compressed, err := crypto.CompressHex(data)
	if err != nil {
		return fmt.Errorf("failed to compress metadata: %w", err)
	}
	return os.WriteFile(path, []byte(compressed), 0600)
}
