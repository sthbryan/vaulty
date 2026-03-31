package github

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const GitHubAPIURL = "https://api.github.com"

type Client struct {
	HTTPClient *http.Client
	Token      string
	BaseURL    string
}

func GetGitHubToken() (string, error) {
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return strings.TrimSpace(string(output)), nil
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return "", fmt.Errorf("no GitHub token found: install gh CLI or set GITHUB_TOKEN")
	}

	return token, nil
}

func NewClient(token string) *Client {
	return &Client{
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
		Token:      token,
		BaseURL:    GitHubAPIURL,
	}
}

func ParseRepo(repo string) (owner, name string, err error) {
	if repo == "local://" {
		return "local", "", nil
	}
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repo format: %s (expected owner/repo)", repo)
	}
	if parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repo format: owner and name cannot be empty")
	}
	return parts[0], parts[1], nil
}

func (c *Client) DecodeContent(content *ContentResponse) ([]byte, error) {
	if content.Encoding == "" || content.Encoding == "base64" {
		cleanContent := strings.ReplaceAll(content.Content, "\n", "")
		return base64.StdEncoding.DecodeString(cleanContent)
	}
	if content.Encoding == "none" {
		if content.DownloadURL == "" {
			return nil, fmt.Errorf("no download URL available for large file")
		}
		resp, err := c.HTTPClient.Get(content.DownloadURL)
		if err != nil {
			return nil, fmt.Errorf("failed to download large file: %w", err)
		}
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	}
	return nil, fmt.Errorf("unsupported encoding: %s", content.Encoding)
}

func (c *Client) EncodeContent(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func (c *Client) RepoExists(ctx context.Context, owner, repo string) (bool, error) {
	url := fmt.Sprintf("%s/repos/%s/%s", c.BaseURL, owner, repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func CreateRepoWithCLI(repo string) error {
	cmd := exec.Command("gh", "repo", "create", repo, "--private")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create repo: %s", string(output))
	}

	return nil
}
