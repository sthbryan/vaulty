package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/DeadBryam/vaulty/internal/application/ports"
	"github.com/DeadBryam/vaulty/internal/github"
)

type GitHubEnvStorage struct {
	client *github.Client
	owner  string
	repo   string
}

func NewGitHubEnvStorage(client *github.Client, repo string) *GitHubEnvStorage {
	owner, repoName, _ := github.ParseRepo(repo)
	return &GitHubEnvStorage{
		client: client,
		owner:  owner,
		repo:   repoName,
	}
}

func (g *GitHubEnvStorage) ListEnvs(ctx context.Context) ([]string, error) {
	_, err := g.client.GetContent(ctx, g.owner, g.repo, "envs")
	if err != nil {
		return []string{}, nil
	}
	return []string{}, nil
}

func (g *GitHubEnvStorage) ListEnvSecrets(ctx context.Context, env string) ([]ports.SecretEntry, error) {
	envPath := fmt.Sprintf("envs/%s", env)
	items, err := g.client.ListDirectory(ctx, g.owner, g.repo, envPath)
	if err != nil {
		return nil, nil
	}

	var secrets []ports.SecretEntry
	for _, item := range items {
		if strings.HasSuffix(item.Name, ".vty") {
			secrets = append(secrets, ports.SecretEntry{
				Name: strings.TrimSuffix(item.Name, ".vty"),
				Size: int64(item.Size),
			})
		}
	}
	return secrets, nil
}

func (g *GitHubEnvStorage) PutEnv(ctx context.Context, env, name string, data []byte) error {
	var path string
	if env == "" {
		path = fmt.Sprintf("envs/%s.vty", name)
	} else {
		path = fmt.Sprintf("envs/%s/%s.vty", env, name)
	}

	content, err := g.client.GetContent(ctx, g.owner, g.repo, path)
	sha := ""
	if err == nil && content != nil {
		sha = content.Sha
	}

	req := github.ContentRequest{
		Message: fmt.Sprintf("Update env %s", name),
		Content: g.client.EncodeContent(data),
		Sha:     sha,
	}

	return g.client.PutContent(ctx, g.owner, g.repo, path, req)
}

func (g *GitHubEnvStorage) GetEnv(ctx context.Context, env, name string) ([]byte, error) {
	var path string
	if env == "" {
		path = fmt.Sprintf("envs/%s.vty", name)
	} else {
		path = fmt.Sprintf("envs/%s/%s.vty", env, name)
	}

	content, err := g.client.GetContent(ctx, g.owner, g.repo, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get env: %w", err)
	}
	return g.client.DecodeContent(content)
}

func (g *GitHubEnvStorage) DeleteEnv(ctx context.Context, env, name string) error {
	var path string
	if env == "" {
		path = fmt.Sprintf("envs/%s.vty", name)
	} else {
		path = fmt.Sprintf("envs/%s/%s.vty", env, name)
	}

	content, err := g.client.GetContent(ctx, g.owner, g.repo, path)
	if err != nil {
		return fmt.Errorf("env not found: %w", err)
	}
	return g.client.DeleteContent(ctx, g.owner, g.repo, path, content.Sha)
}
