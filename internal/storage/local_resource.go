package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type LocalResourceStorage struct {
	baseDir string
}

func NewLocalResourceStorage(basePath string) (*LocalResourceStorage, error) {
	if basePath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		basePath = filepath.Join(homeDir, ".vty", "vault")
	}
	return &LocalResourceStorage{baseDir: basePath}, nil
}

func (l *LocalResourceStorage) ListResources(ctx context.Context) ([]string, error) {
	var resources []string

	resourcesDir := filepath.Join(l.baseDir, "resources")
	err := filepath.Walk(resourcesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Ext(path) == ".vty" {
			rel, _ := filepath.Rel(resourcesDir, path)
			resources = append(resources, filepath.Join("resources", rel))
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	configDir := filepath.Join(l.baseDir, "config")
	err = filepath.Walk(configDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Ext(path) == ".vty" {
			rel, _ := filepath.Rel(configDir, path)
			resources = append(resources, filepath.Join("config", rel))
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to list config: %w", err)
	}

	return resources, nil
}

func (l *LocalResourceStorage) resolveSafePath(path string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(path))
	if clean == "." || clean == "" {
		return "", fmt.Errorf("invalid empty path")
	}
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid path: %s", path)
	}

	fullPath := filepath.Join(l.baseDir, clean)
	rel, err := filepath.Rel(l.baseDir, fullPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes vault directory")
	}
	return fullPath, nil
}

func (l *LocalResourceStorage) PutResource(ctx context.Context, path string, data []byte) error {
	fullPath, err := l.resolveSafePath(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(fullPath)

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return os.WriteFile(fullPath, data, 0600)
}

func (l *LocalResourceStorage) GetResource(ctx context.Context, path string) ([]byte, error) {
	fullPath, err := l.resolveSafePath(path)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("resource not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read resource: %w", err)
	}
	return data, nil
}

func (l *LocalResourceStorage) DeleteResource(ctx context.Context, path string) error {
	fullPath, err := l.resolveSafePath(path)
	if err != nil {
		return err
	}

	err = os.Remove(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("resource not found: %s", path)
		}
		return fmt.Errorf("failed to delete resource: %w", err)
	}

	dir := filepath.Dir(fullPath)
	_ = l.cleanEmptyDir(dir)
	return nil
}

func (l *LocalResourceStorage) cleanEmptyDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	if len(entries) == 0 {
		return os.Remove(dir)
	}
	return nil
}
