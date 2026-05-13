package github

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func (c *Client) GetUserKeys(ctx context.Context, owner, repo, username string) (*ContentResponse, error) {
	path := fmt.Sprintf(".vaulty/keys/%s.vty", username)
	return c.GetContent(ctx, owner, repo, path)
}

func (c *Client) PutUserKeys(ctx context.Context, owner, repo, username string, data []byte) error {
	path := fmt.Sprintf(".vaulty/keys/%s.vty", username)
	sha, err := c.getContentSha(ctx, owner, repo, path)
	if err != nil {
		if !strings.Contains(err.Error(), "404") {
			return fmt.Errorf("failed to get current sha: %w", err)
		}
		sha = ""
	}

	encoded := c.EncodeContent(data)

	req := ContentRequest{
		Message: fmt.Sprintf("Update keys for %s", username),
		Content: encoded,
		Sha:     sha,
	}

	return c.PutContent(ctx, owner, repo, path, req)
}

func (c *Client) GetUserList(ctx context.Context, owner, repo string) ([]UserEntry, error) {
	metadataBytes, err := c.GetMetadata(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	users, ok := metadata["users"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("users field not found or not an array in metadata")
	}

	var userList []UserEntry
	for _, u := range users {
		userMap, ok := u.(map[string]interface{})
		if !ok {
			continue
		}

		entry := UserEntry{
			Username:  toString(userMap["username"]),
			PublicKey: toString(userMap["public_key"]),
			AddedAt:   toString(userMap["added_at"]),
		}
		userList = append(userList, entry)
	}

	return userList, nil
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
