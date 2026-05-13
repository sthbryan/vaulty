package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sthbryan/vaulty/internal/application/ports"
)

type LocalSSHStorage struct {
	baseDir string
}

func NewLocalSSHStorage(basePath string) (*LocalSSHStorage, error) {
	if basePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		basePath = filepath.Join(homeDir, ".vty", "vault")
	}
	return &LocalSSHStorage{baseDir: basePath}, nil
}

func (l *LocalSSHStorage) ListSSHKeys(ctx context.Context, username string) ([]ports.SSHKeyInfo, error) {
	sshDir := filepath.Join(l.baseDir, "ssh", username)
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ports.SSHKeyInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read SSH directory: %w", err)
	}

	var keys []ports.SSHKeyInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) == ".vty" {
			keys = append(keys, ports.SSHKeyInfo{
				Name: name[:len(name)-len(".vty")],
				Data: nil,
			})
		}
	}
	return keys, nil
}

func (l *LocalSSHStorage) PutSSHKey(ctx context.Context, username, keyName string, data []byte) error {
	sshDir := filepath.Join(l.baseDir, "ssh", username)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create SSH directory: %w", err)
	}
	path := filepath.Join(sshDir, fmt.Sprintf("%s.vty", keyName))
	return os.WriteFile(path, data, 0600)
}

func (l *LocalSSHStorage) GetSSHKey(ctx context.Context, username, keyName string) ([]byte, error) {
	path := filepath.Join(l.baseDir, "ssh", username, fmt.Sprintf("%s.vty", keyName))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("SSH key not found: %s/%s", username, keyName)
		}
		return nil, fmt.Errorf("failed to read SSH key: %w", err)
	}
	return data, nil
}

func (l *LocalSSHStorage) DeleteSSHKey(ctx context.Context, username, keyName, sha string) error {
	path := filepath.Join(l.baseDir, "ssh", username, fmt.Sprintf("%s.vty", keyName))
	err := os.Remove(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("SSH key not found: %s/%s", username, keyName)
		}
		return fmt.Errorf("failed to delete SSH key: %w", err)
	}
	userDir := filepath.Join(l.baseDir, "ssh", username)
	_ = l.cleanEmptyDir(userDir)
	return nil
}

func (l *LocalSSHStorage) cleanEmptyDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	if len(entries) == 0 {
		return os.Remove(dir)
	}
	return nil
}
