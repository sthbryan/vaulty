package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/sthbryan/vaulty/internal/crypto"
)

func (c *Client) GetMetadata(ctx context.Context, owner, repo string) ([]byte, error) {
	path := ".vaulty/metadata.vty"
	content, err := c.GetContent(ctx, owner, repo, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}
	encodedData, err := c.DecodeContent(content)
	if err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}
	return crypto.DecompressHex(string(encodedData))
}

func (c *Client) PutMetadata(ctx context.Context, owner, repo string, metadata []byte) error {
	path := ".vaulty/metadata.vty"
	sha, err := c.getContentSha(ctx, owner, repo, path)
	if err != nil && !strings.Contains(err.Error(), "404") {
		return fmt.Errorf("failed to get current sha: %w", err)
	}

	metadataHex, err := crypto.CompressHex(metadata)
	if err != nil {
		return fmt.Errorf("failed to compress metadata: %w", err)
	}

	req := ContentRequest{
		Message: "Update metadata.vty via Vaulty",
		Content: c.EncodeContent([]byte(metadataHex)),
		Sha:     sha,
	}

	return c.PutContent(ctx, owner, repo, path, req)
}

func (c *Client) GetVault(ctx context.Context, owner, repo string) (*ContentResponse, error) {
	path := ".vaulty/vault.vty"
	return c.GetContent(ctx, owner, repo, path)
}

func (c *Client) PutVault(ctx context.Context, owner, repo string, data []byte) error {
	path := ".vaulty/vault.vty"
	sha, err := c.getContentSha(ctx, owner, repo, path)
	action := "Update"
	if err != nil {
		if !strings.Contains(err.Error(), "404") {
			return fmt.Errorf("failed to get current sha: %w", err)
		}
		action = "Create"
		sha = ""
	}

	encoded := c.EncodeContent(data)

	req := ContentRequest{
		Message: action + " vault via Vaulty",
		Content: encoded,
		Sha:     sha,
	}

	return c.PutContent(ctx, owner, repo, path, req)
}
