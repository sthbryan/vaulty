package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/sthbryan/vaulty/internal/github"
)

type GitHubResourceStorage struct {
	client *github.Client
	owner  string
	repo   string
}

func NewGitHubResourceStorage(client *github.Client, repo string) *GitHubResourceStorage {
	owner, repoName, _ := github.ParseRepo(repo)
	return &GitHubResourceStorage{
		client: client,
		owner:  owner,
		repo:   repoName,
	}
}

func (g *GitHubResourceStorage) ListResources(ctx context.Context) ([]string, error) {
	entries, err := g.client.ListDirectory(ctx, g.owner, g.repo, ".vaulty/resources")
	if err != nil {
		return []string{}, nil
	}

	var resources []string
	for _, entry := range entries {
		if entry.Type != "dir" && strings.HasSuffix(entry.Name, ".vty") {
			resources = append(resources, entry.Name)
		}
	}
	return resources, nil
}

func (g *GitHubResourceStorage) PutResource(ctx context.Context, path string, data []byte) error {
	content, err := g.client.GetContent(ctx, g.owner, g.repo, path)
	sha := ""
	if err == nil && content != nil {
		sha = content.Sha
	}

	req := github.ContentRequest{
		Message: fmt.Sprintf("Update resource %s via Vaulty", path),
		Content: g.client.EncodeContent(data),
		Sha:     sha,
	}

	return g.client.PutContent(ctx, g.owner, g.repo, path, req)
}

func (g *GitHubResourceStorage) GetResource(ctx context.Context, path string) ([]byte, error) {
	content, err := g.client.GetContent(ctx, g.owner, g.repo, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource: %w", err)
	}
	return g.client.DecodeContent(content)
}

func (g *GitHubResourceStorage) DeleteResource(ctx context.Context, path string) error {
	content, err := g.client.GetContent(ctx, g.owner, g.repo, path)
	if err != nil {
		return fmt.Errorf("resource not found: %w", err)
	}
	return g.client.DeleteContent(ctx, g.owner, g.repo, path, content.Sha)
}
