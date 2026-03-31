package github

import (
	"context"
	"fmt"
	"strings"
)

func (c *Client) EnsureSSHUserDir(ctx context.Context, owner, repo, username string) error {
	path := fmt.Sprintf("ssh/%s/.gitkeep", username)
	_, err := c.GetContent(ctx, owner, repo, path)
	if err == nil {
		return nil
	}
	if !strings.Contains(err.Error(), "404") {
		return fmt.Errorf("failed to check directory: %w", err)
	}

	req := ContentRequest{
		Message: fmt.Sprintf("Create SSH directory for %s", username),
		Content: c.EncodeContent([]byte("")),
	}

	return c.PutContent(ctx, owner, repo, path, req)
}

func (c *Client) ListSSHKeys(ctx context.Context, owner, repo, username string) ([]SSHKeyInfo, error) {
	path := fmt.Sprintf("ssh/%s", username)
	items, err := c.ListDirectory(ctx, owner, repo, path)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return []SSHKeyInfo{}, nil
		}
		return nil, fmt.Errorf("failed to list SSH keys: %w", err)
	}

	var keys []SSHKeyInfo
	for _, item := range items {
		if item.Type == "file" && strings.HasSuffix(item.Name, ".vty") {
			keys = append(keys, SSHKeyInfo{
				Username: username,
				KeyName:  strings.TrimSuffix(item.Name, ".vty"),
				Size:     item.Size,
			})
		}
	}

	return keys, nil
}

func (c *Client) ListAllSSHKeys(ctx context.Context, owner, repo string) ([]SSHKeyInfo, error) {
	items, err := c.ListDirectory(ctx, owner, repo, "ssh")
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return []SSHKeyInfo{}, nil
		}
		return nil, fmt.Errorf("failed to list SSH directory: %w", err)
	}

	var result []SSHKeyInfo
	for _, item := range items {
		if item.Type == "dir" {
			keys, err := c.ListSSHKeys(ctx, owner, repo, item.Name)
			if err != nil {
				continue
			}
			result = append(result, keys...)
		}
	}

	return result, nil
}
