package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/DeadBryam/vaulty/internal/application/ports"
	"github.com/DeadBryam/vaulty/internal/github"
)

type GitHubSSHStorage struct {
	client *github.Client
	owner  string
	repo   string
}

func NewGitHubSSHStorage(client *github.Client, repo string) *GitHubSSHStorage {
	owner, repoName, _ := github.ParseRepo(repo)
	return &GitHubSSHStorage{
		client: client,
		owner:  owner,
		repo:   repoName,
	}
}

func (g *GitHubSSHStorage) ListSSHKeys(ctx context.Context, username string) ([]ports.SSHKeyInfo, error) {
	keys, err := g.client.ListSSHKeys(ctx, g.owner, g.repo, username)
	if err != nil {
		return nil, fmt.Errorf("failed to list SSH keys: %w", err)
	}

	result := make([]ports.SSHKeyInfo, len(keys))
	for i, key := range keys {
		result[i] = ports.SSHKeyInfo{
			Name: key.KeyName,
			Data: nil,
		}
	}
	return result, nil
}

func (g *GitHubSSHStorage) PutSSHKey(ctx context.Context, username, keyName string, data []byte) error {
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

func (g *GitHubSSHStorage) GetSSHKey(ctx context.Context, username, keyName string) ([]byte, error) {
	path := fmt.Sprintf("ssh/%s/%s.vty", username, keyName)
	content, err := g.client.GetContent(ctx, g.owner, g.repo, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSH key: %w", err)
	}
	return g.client.DecodeContent(content)
}

func (g *GitHubSSHStorage) DeleteSSHKey(ctx context.Context, username, keyName, sha string) error {
	path := fmt.Sprintf("ssh/%s/%s.vty", username, keyName)
	return g.client.DeleteContent(ctx, g.owner, g.repo, path, sha)
}
