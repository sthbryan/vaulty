package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/DeadBryam/vaulty/internal/github"
)

type GitHubStorage struct {
	client *github.Client
	owner  string
	repo   string
}

func NewGitHubStorage(token, repo string) (*GitHubStorage, error) {
	owner, repoName, err := github.ParseRepo(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repo: %w", err)
	}

	client := github.NewClient(token)

	return &GitHubStorage{
		client: client,
		owner:  owner,
		repo:   repoName,
	}, nil
}

func (g *GitHubStorage) GetVault(ctx context.Context) ([]byte, error) {
	content, err := g.client.GetVault(ctx, g.owner, g.repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get vault: %w", err)
	}

	decoded, err := g.client.DecodeContent(content)
	if err != nil {
		return nil, fmt.Errorf("failed to decode vault: %w", err)
	}

	return decoded, nil
}

func (g *GitHubStorage) PutVault(ctx context.Context, data []byte) error {
	return g.client.PutVault(ctx, g.owner, g.repo, data)
}

func (g *GitHubStorage) GetMetadata(ctx context.Context) ([]byte, error) {

	return g.client.GetMetadata(ctx, g.owner, g.repo)
}

func (g *GitHubStorage) PutMetadata(ctx context.Context, data []byte) error {
	return g.client.PutMetadata(ctx, g.owner, g.repo, data)
}

func (g *GitHubStorage) GetUserKeys(ctx context.Context, username string) ([]byte, error) {
	content, err := g.client.GetUserKeys(ctx, g.owner, g.repo, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user keys: %w", err)
	}

	return g.client.DecodeContent(content)
}

func (g *GitHubStorage) PutUserKeys(ctx context.Context, username string, data []byte) error {
	return g.client.PutUserKeys(ctx, g.owner, g.repo, username, data)
}

func (g *GitHubStorage) GetRecoverySeed(ctx context.Context, username string) ([]byte, error) {
	content, err := g.client.GetRecoverySeed(ctx, g.owner, g.repo, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get recovery seed: %w", err)
	}

	return g.client.DecodeContent(content)
}

func (g *GitHubStorage) PutRecoverySeed(ctx context.Context, username string, data []byte) error {
	return g.client.PutRecoverySeed(ctx, g.owner, g.repo, username, data)
}

func (g *GitHubStorage) ListSSHKeys(ctx context.Context, username string) ([]SSHKeyInfo, error) {
	keys, err := g.client.ListSSHKeys(ctx, g.owner, g.repo, username)
	if err != nil {
		return nil, fmt.Errorf("failed to list SSH keys: %w", err)
	}

	result := make([]SSHKeyInfo, len(keys))
	for i, key := range keys {
		result[i] = SSHKeyInfo{
			Username: key.Username,
			KeyName:  key.KeyName,
			Size:     key.Size,
		}
	}

	return result, nil
}

func (g *GitHubStorage) PutSSHKey(ctx context.Context, username, keyName string, data []byte) error {

	if err := g.client.EnsureSSHUserDir(ctx, g.owner, g.repo, username); err != nil {
		return fmt.Errorf("failed to ensure SSH directory: %w", err)
	}

	path := fmt.Sprintf("ssh/%s/%s.vty", username, keyName)
	content, err := g.client.GetContent(ctx, g.owner, g.repo, path)
	sha := ""
	if err != nil {
		if !strings.Contains(err.Error(), "404") {
			return fmt.Errorf("failed to get current sha: %w", err)
		}
	} else {
		sha = content.Sha
	}

	req := github.ContentRequest{
		Message: fmt.Sprintf("Update SSH key %s for %s", keyName, username),
		Content: g.client.EncodeContent(data),
		Sha:     sha,
	}

	return g.client.PutContent(ctx, g.owner, g.repo, path, req)
}

func (g *GitHubStorage) GetSSHKey(ctx context.Context, username, keyName string) ([]byte, error) {
	path := fmt.Sprintf("ssh/%s/%s.vty", username, keyName)
	content, err := g.client.GetContent(ctx, g.owner, g.repo, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH key: %w", err)
	}

	return g.client.DecodeContent(content)
}

func (g *GitHubStorage) DeleteSSHKey(ctx context.Context, username, keyName, sha string) error {
	path := fmt.Sprintf("ssh/%s/%s.vty", username, keyName)
	return g.client.DeleteContent(ctx, g.owner, g.repo, path, sha)
}

func (g *GitHubStorage) IsLocal() bool {
	return false
}

func (g *GitHubStorage) GetRepo() string {
	return g.owner + "/" + g.repo
}

func (g *GitHubStorage) ListEnvs(ctx context.Context) ([]string, error) {

	_, err := g.client.GetContent(ctx, g.owner, g.repo, "envs")
	if err != nil {
		return []string{}, nil
	}

	return []string{}, nil
}

func (g *GitHubStorage) PutEnv(ctx context.Context, env, name string, data []byte) error {
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

func (g *GitHubStorage) GetEnv(ctx context.Context, env, name string) ([]byte, error) {
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

func (g *GitHubStorage) DeleteEnv(ctx context.Context, env, name string) error {
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
