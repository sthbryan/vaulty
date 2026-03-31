package storage

import (
	"fmt"

	"github.com/DeadBryam/vaulty/internal/application/ports"
	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/github"
)

type Factory struct {
	cfg *config.Config
}

func NewFactory(cfg *config.Config) *Factory {
	return &Factory{cfg: cfg}
}

func (f *Factory) CreateVaultStorage() (ports.VaultStorage, error) {
	switch f.cfg.StorageType {
	case "github":
		client, err := github.GetAuthenticatedClient()
		if err != nil {
			return nil, err
		}
		return NewGitHubVaultStorage(client, f.cfg.Repo), nil
	case "local":
		return NewLocalVaultStorage(f.cfg.LocalVaultPath)
	default:
		return nil, fmt.Errorf("unknown storage type: %s", f.cfg.StorageType)
	}
}

func (f *Factory) CreateUserStorage() (ports.UserStorage, error) {
	switch f.cfg.StorageType {
	case "github":
		client, err := github.GetAuthenticatedClient()
		if err != nil {
			return nil, err
		}
		return NewGitHubUserStorage(client, f.cfg.Repo), nil
	case "local":
		return NewLocalUserStorage(f.cfg.LocalVaultPath)
	default:
		return nil, fmt.Errorf("unknown storage type: %s", f.cfg.StorageType)
	}
}

func (f *Factory) CreateSSHStorage() (ports.SSHStorage, error) {
	switch f.cfg.StorageType {
	case "github":
		client, err := github.GetAuthenticatedClient()
		if err != nil {
			return nil, err
		}
		return NewGitHubSSHStorage(client, f.cfg.Repo), nil
	case "local":
		return NewLocalSSHStorage(f.cfg.LocalVaultPath)
	default:
		return nil, fmt.Errorf("unknown storage type: %s", f.cfg.StorageType)
	}
}

func (f *Factory) CreateEnvStorage() (ports.EnvStorage, error) {
	switch f.cfg.StorageType {
	case "github":
		client, err := github.GetAuthenticatedClient()
		if err != nil {
			return nil, err
		}
		return NewGitHubEnvStorage(client, f.cfg.Repo), nil
	case "local":
		return NewLocalEnvStorage(f.cfg.LocalVaultPath)
	default:
		return nil, fmt.Errorf("unknown storage type: %s", f.cfg.StorageType)
	}
}

func (f *Factory) CreateResourceStorage() (ports.ResourceStorage, error) {
	switch f.cfg.StorageType {
	case "github":
		client, err := github.GetAuthenticatedClient()
		if err != nil {
			return nil, err
		}
		return NewGitHubResourceStorage(client, f.cfg.Repo), nil
	case "local":
		return NewLocalResourceStorage(f.cfg.LocalVaultPath)
	default:
		return nil, fmt.Errorf("unknown storage type: %s", f.cfg.StorageType)
	}
}

func (f *Factory) CreateStorage() (Storage, error) {
	switch f.cfg.StorageType {
	case "github":
		token, err := github.GetGitHubToken()
		if err != nil {
			return nil, err
		}
		return NewGitHubStorage(token, f.cfg.Repo)
	case "local":
		return NewLocalStorage()
	default:
		return nil, fmt.Errorf("unknown storage type: %s", f.cfg.StorageType)
	}
}
