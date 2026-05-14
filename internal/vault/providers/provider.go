package providers

import (
	"context"

	"github.com/sthbryan/vaulty/v2/pkg/models"
)

type StorageProvider interface {
	Ping(ctx context.Context) error
	Upload(ctx context.Context, path string, data []byte) error
	Download(ctx context.Context, path string) ([]byte, error)
	Delete(ctx context.Context, path string) error
	List(ctx context.Context, prefix string) ([]string, error)
	Exists(ctx context.Context, path string) (bool, error)
	CreateRepo(ctx context.Context) error
	CheckVault() bool
	SetupStorage() error
	LoadMeta() (*models.VaultMeta, error)
	SaveMeta(meta *models.VaultMeta) error
}

type ProviderConfig struct {
	token string
	owner string
	repo  string
}

type Provider struct {
	ProviderConfig
	baseURL string
}

type ProviderType string

const (
	ProviderGitHub ProviderType = "github"
	ProviderLocal  ProviderType = "local"
	// Future provider -> maybe gitlab.
)

func NewProvider(providerType ProviderType, params ...string) StorageProvider {
	switch providerType {
	case ProviderGitHub:
		if len(params) >= 3 {
			return NewGitHubProvider(params[0], params[1], params[2])
		}
	case ProviderLocal:
		if len(params) >= 2 {
			return NewLocalProvider(params[0], params[1]) // ~/.vaulty/username/vaultid
		}
	}
	return nil
}
