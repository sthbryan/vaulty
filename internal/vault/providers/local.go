package providers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sthbryan/vaulty/v2/pkg/models"
	"gopkg.in/yaml.v3"
)

type LocalProvider struct {
	*Provider
}

func NewLocalProvider(owner, repo string) *LocalProvider {
	home, _ := os.UserHomeDir()
	baseURL := filepath.Join(home, ".vaulty", owner, repo)

	return &LocalProvider{
		Provider: &Provider{
			ProviderConfig: ProviderConfig{owner: owner, repo: repo},
			baseURL:        baseURL,
		},
	}
}

func (p *LocalProvider) Ping(ctx context.Context) error {
	info, err := os.Stat(p.baseURL)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(p.baseURL, 0700)
		}
		return fmt.Errorf("Accessing path: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("Path is not a directory")
	}
	return nil
}

func (p *LocalProvider) Upload(ctx context.Context, path string, data []byte) error {
	fullPath := filepath.Join(p.baseURL, path)

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("Creating directory: %w", err)
	}

	if err := os.WriteFile(fullPath, data, 0600); err != nil {
		return fmt.Errorf("Writing file: %w", err)
	}

	return nil
}

func (p *LocalProvider) Download(ctx context.Context, path string) ([]byte, error) {
	fullPath := filepath.Join(p.baseURL, path)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("File not found: %s", path)
		}
		return nil, fmt.Errorf("Reading file: %w", err)
	}

	return data, nil
}

func (p *LocalProvider) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(p.baseURL, path)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("File not found: %s", path)
		}
		return fmt.Errorf("Removing file: %w", err)
	}

	return nil
}

func (p *LocalProvider) List(ctx context.Context, prefix string) ([]string, error) {
	fullPath := filepath.Join(p.baseURL, prefix)

	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("Accessing path: %w", err)
	}

	if !info.IsDir() {
		return []string{prefix}, nil
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("Reading directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, filepath.Join(prefix, entry.Name()))
		}
	}

	return files, nil
}

func (p *LocalProvider) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(p.baseURL, path)
	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("Checking path: %w", err)
	}
	return true, nil
}

func (p *LocalProvider) CreateRepo(ctx context.Context) error {
	return os.MkdirAll(p.baseURL, 0700)
}

func (p *LocalProvider) CheckVault() bool {
	info, err := os.Stat(p.baseURL)
	return err == nil && info.IsDir()
}

func (p *LocalProvider) SetupStorage() error {
	if p.CheckVault() {
		return fmt.Errorf("Vault directory already exists")
	}
	return p.CreateRepo(context.Background())
}

func (p *LocalProvider) LoadMeta() (*models.VaultMeta, error) {
	data, err := p.Download(context.Background(), "vault.meta")
	if err != nil {
		return nil, fmt.Errorf("Downloading vault.meta: %w", err)
	}

	var meta models.VaultMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("Parsing vault.meta: %w", err)
	}

	return &meta, nil
}

func (p *LocalProvider) SaveMeta(meta *models.VaultMeta) error {
	data, err := yaml.Marshal(meta)
	if err != nil {
		return fmt.Errorf("Marshaling meta: %w", err)
	}

	return p.Upload(context.Background(), "vault.meta", data)
}
