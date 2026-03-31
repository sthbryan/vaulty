package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/DeadBryam/vaulty/internal/application/ports"
)

type LocalEnvStorage struct {
	baseDir string
}

func NewLocalEnvStorage(basePath string) (*LocalEnvStorage, error) {
	if basePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		basePath = filepath.Join(homeDir, ".vty", "vault")
	}
	return &LocalEnvStorage{baseDir: basePath}, nil
}

func (l *LocalEnvStorage) ensureDir(dir string) error {
	return os.MkdirAll(dir, 0700)
}

func (l *LocalEnvStorage) ListEnvs(ctx context.Context) ([]string, error) {
	envsDir := filepath.Join(l.baseDir, "envs")
	entries, err := os.ReadDir(envsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read envs directory: %w", err)
	}

	var envs []string
	for _, entry := range entries {
		if entry.IsDir() {
			envs = append(envs, entry.Name())
		} else if strings.HasSuffix(entry.Name(), ".vty") {
			envs = append(envs, strings.TrimSuffix(entry.Name(), ".vty"))
		}
	}
	return envs, nil
}

func (l *LocalEnvStorage) ListEnvSecrets(ctx context.Context, env string) ([]ports.SecretEntry, error) {
	if env == "" {
		return []ports.SecretEntry{}, nil
	}

	envPath := filepath.Join(l.baseDir, "envs", env)
	entries, err := os.ReadDir(envPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []ports.SecretEntry{}, nil
		}
		return nil, fmt.Errorf("failed to read env directory: %w", err)
	}

	var secrets []ports.SecretEntry
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".vty") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("failed to read env secret info: %w", err)
		}
		secrets = append(secrets, ports.SecretEntry{
			Name: strings.TrimSuffix(name, ".vty"),
			Size: info.Size(),
		})
	}
	return secrets, nil
}

func (l *LocalEnvStorage) PutEnv(ctx context.Context, env, name string, data []byte) error {
	var envDir string
	if env == "" {
		envDir = filepath.Join(l.baseDir, "envs")
	} else {
		envDir = filepath.Join(l.baseDir, "envs", env)
	}

	if err := l.ensureDir(envDir); err != nil {
		return fmt.Errorf("failed to create env directory: %w", err)
	}

	path := filepath.Join(envDir, fmt.Sprintf("%s.vty", name))
	return os.WriteFile(path, data, 0600)
}

func (l *LocalEnvStorage) GetEnv(ctx context.Context, env, name string) ([]byte, error) {
	var path string
	if env == "" {
		path = filepath.Join(l.baseDir, "envs", fmt.Sprintf("%s.vty", name))
	} else {
		path = filepath.Join(l.baseDir, "envs", env, fmt.Sprintf("%s.vty", name))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("env not found: %s/%s", env, name)
		}
		return nil, fmt.Errorf("failed to read env: %w", err)
	}
	return data, nil
}

func (l *LocalEnvStorage) DeleteEnv(ctx context.Context, env, name string) error {
	var path string
	if env == "" {
		path = filepath.Join(l.baseDir, "envs", fmt.Sprintf("%s.vty", name))
	} else {
		path = filepath.Join(l.baseDir, "envs", env, fmt.Sprintf("%s.vty", name))
	}

	err := os.Remove(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("env not found: %s/%s", env, name)
		}
		return fmt.Errorf("failed to delete env: %w", err)
	}

	if env != "" {
		envDir := filepath.Join(l.baseDir, "envs", env)
		l.cleanEmptyDir(envDir)
	}
	return nil
}

func (l *LocalEnvStorage) cleanEmptyDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	if len(entries) == 0 {
		return os.Remove(dir)
	}
	return nil
}
