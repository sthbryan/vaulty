package start

import (
	"context"
	"fmt"
	"os"

	"github.com/sthbryan/vaulty/v2/internal/storage"
	"github.com/sthbryan/vaulty/v2/internal/ui"
	"github.com/sthbryan/vaulty/v2/pkg/models"
	"gopkg.in/yaml.v3"
)

func GetGitHubToken() (string, error) {
	authMethod, err := ui.Select("GitHub authentication method", []ui.SelectOption{
		{ID: "cli", Label: "GitHub CLI (gh) - recommended"},
		{ID: "env", Label: "GITHUB_TOKEN environment variable"},
		{ID: "manual", Label: "Enter token manually"},
	})
	if err != nil {
		return "", fmt.Errorf("wizard cancelled")
	}

	switch authMethod {
	case "cli":
		return ui.GetGitHubTokenCLI()

	case "env":
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" {
			ui.PrintError("GITHUB_TOKEN not set")
			return "", fmt.Errorf("GITHUB_TOKEN not set")
		}
		return token, nil

	case "manual":
		return ui.Password("GitHub personal access token", "••••••••")
	}

	return "", fmt.Errorf("invalid auth method")
}

func CheckGitHubVault(token, username, vaultID string) bool {
	ghStorage := storage.NewGitHubStorage(token, username, vaultID)
	return ghStorage.Ping(context.Background()) == nil
}

func CreateGitHubRepo(token, username, vaultID string) error {
	ghStorage := storage.NewGitHubStorage(token, username, vaultID)
	return ghStorage.CreateRepo(context.Background())
}

func LoadMetaFromGitHub(token, username, vaultID string) (*models.VaultMeta, error) {
	ghStorage := storage.NewGitHubStorage(token, username, vaultID)
	data, err := ghStorage.Download(context.Background(), "vault.meta")
	if err != nil {
		return nil, fmt.Errorf("downloading vault.meta: %w", err)
	}

	var meta models.VaultMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parsing vault.meta: %w", err)
	}

	return &meta, nil
}

func UploadMetaToGitHub(token, username, vaultID string, meta *models.VaultMeta) error {
	ghStorage := storage.NewGitHubStorage(token, username, vaultID)
	metaData, err := yaml.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshaling meta: %w", err)
	}
	return ghStorage.Upload(context.Background(), "vault.meta", metaData)
}
