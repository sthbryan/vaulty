package storage

import (
	"context"
	"fmt"

	"github.com/DeadBryam/vaulty/internal/github"
)

type GitHubVaultStorage struct {
	client *github.Client
	owner  string
	repo   string
}

func NewGitHubVaultStorage(client *github.Client, repo string) *GitHubVaultStorage {
	owner, repoName, _ := github.ParseRepo(repo)
	return &GitHubVaultStorage{
		client: client,
		owner:  owner,
		repo:   repoName,
	}
}

func (g *GitHubVaultStorage) GetVault(ctx context.Context) ([]byte, error) {
	content, err := g.client.GetVault(ctx, g.owner, g.repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault: %w", err)
	}
	return g.client.DecodeContent(content)
}

func (g *GitHubVaultStorage) PutVault(ctx context.Context, data []byte) error {
	return g.client.PutVault(ctx, g.owner, g.repo, data)
}

func (g *GitHubVaultStorage) GetMetadata(ctx context.Context) ([]byte, error) {
	return g.client.GetMetadata(ctx, g.owner, g.repo)
}

func (g *GitHubVaultStorage) PutMetadata(ctx context.Context, data []byte) error {
	return g.client.PutMetadata(ctx, g.owner, g.repo, data)
}
