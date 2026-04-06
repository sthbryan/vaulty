package storage

import (
	"context"
	"fmt"

	"github.com/DeadBryam/vaulty/internal/github"
)

type GitHubUserStorage struct {
	client *github.Client
	owner  string
	repo   string
}

func NewGitHubUserStorage(client *github.Client, repo string) *GitHubUserStorage {
	owner, repoName, _ := github.ParseRepo(repo)
	return &GitHubUserStorage{
		client: client,
		owner:  owner,
		repo:   repoName,
	}
}

func (g *GitHubUserStorage) GetUserKeys(ctx context.Context, username string) ([]byte, error) {
	content, err := g.client.GetUserKeys(ctx, g.owner, g.repo, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user keys: %w", err)
	}
	return g.client.DecodeContent(content)
}

func (g *GitHubUserStorage) PutUserKeys(ctx context.Context, username string, data []byte) error {
	return g.client.PutUserKeys(ctx, g.owner, g.repo, username, data)
}

func (g *GitHubUserStorage) GetUserList(ctx context.Context) ([]byte, error) {
	return nil, fmt.Errorf("GetUserList not implemented")
}
