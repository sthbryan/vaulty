package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const GitHubAPIURL = "https://api.github.com"

type ContentResponse struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Sha         string `json:"sha"`
	Size        int    `json:"size"`
	Type        string `json:"type"`
	Content     string `json:"content"`
	Encoding    string `json:"encoding"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	DownloadURL string `json:"download_url"`
}

type ContentRequest struct {
	Message string `json:"message"`
	Content string `json:"content"`
	Sha     string `json:"sha,omitempty"`
	Branch  string `json:"branch,omitempty"`
}

type Client struct {
	HTTPClient *http.Client
	Token      string
	BaseURL    string
}

type DirectoryItem struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Sha         string `json:"sha"`
	Size        int    `json:"size"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	DownloadURL string `json:"download_url,omitempty"`
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
		HTTPClient: &http.Client{},
		Token:      token,
		BaseURL:    GitHubAPIURL,
	}
}

func ParseRepo(repo string) (owner, name string, err error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repo format: %s (expected owner/repo)", repo)
	}
	if parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repo format: owner and name cannot be empty")
	}
	return parts[0], parts[1], nil
}

func (c *Client) GetContent(ctx context.Context, owner, repo, path string) (*ContentResponse, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", c.BaseURL, owner, repo, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var content ContentResponse
	if err := json.Unmarshal(body, &content); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &content, nil
}

func (c *Client) PutContent(ctx context.Context, owner, repo, path string, req ContentRequest) error {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", c.BaseURL, owner, repo, path)

	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.Token)
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) ListDirectory(ctx context.Context, owner, repo, path string) ([]DirectoryItem, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", c.BaseURL, owner, repo, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var items []DirectoryItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return items, nil
}

func (c *Client) DecodeContent(content *ContentResponse) ([]byte, error) {
	if content.Encoding != "base64" {
		return nil, fmt.Errorf("unsupported encoding: %s", content.Encoding)
	}
	cleanContent := strings.ReplaceAll(content.Content, "\n", "")
	return base64.StdEncoding.DecodeString(cleanContent)
}

func (c *Client) EncodeContent(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

type DeleteRequest struct {
	Message string `json:"message"`
	Sha     string `json:"sha"`
	Branch  string `json:"branch,omitempty"`
}

type UserEntry struct {
	Username  string `json:"username"`
	PublicKey string `json:"public_key"`
	AddedAt   string `json:"added_at"`
}

type SSHKeyInfo struct {
	Username string
	KeyName  string
	Size     int
}

func (c *Client) DeleteContent(ctx context.Context, owner, repo, path, sha string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", c.BaseURL, owner, repo, path)

	req := DeleteRequest{
		Message: fmt.Sprintf("Delete %s via Vaulty", path),
		Sha:     sha,
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.Token)
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *Client) GetUserKeys(ctx context.Context, owner, repo, username string) (*ContentResponse, error) {
	path := fmt.Sprintf(".vaulty/keys/%s.enc", username)
	return c.GetContent(ctx, owner, repo, path)
}

func (c *Client) PutUserKeys(ctx context.Context, owner, repo, username string, data []byte) error {
	path := fmt.Sprintf(".vaulty/keys/%s.enc", username)
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

func (c *Client) GetMetadata(ctx context.Context, owner, repo string) ([]byte, error) {
	path := ".vaulty/metadata.json"
	content, err := c.GetContent(ctx, owner, repo, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}
	return c.DecodeContent(content)
}

func (c *Client) PutMetadata(ctx context.Context, owner, repo string, metadata []byte) error {
	path := ".vaulty/metadata.json"
	sha, err := c.getContentSha(ctx, owner, repo, path)
	if err != nil && !strings.Contains(err.Error(), "404") {
		return fmt.Errorf("failed to get current sha: %w", err)
	}

	req := ContentRequest{
		Message: "Update metadata.json via Vaulty",
		Content: c.EncodeContent(metadata),
		Sha:     sha,
	}

	return c.PutContent(ctx, owner, repo, path, req)
}

func (c *Client) GetRecoverySeed(ctx context.Context, owner, repo, username string) (*ContentResponse, error) {
	path := fmt.Sprintf(".vaulty/recovery/%s.recovery.enc", username)
	return c.GetContent(ctx, owner, repo, path)
}

func (c *Client) PutRecoverySeed(ctx context.Context, owner, repo, username string, data []byte) error {
	path := fmt.Sprintf(".vaulty/recovery/%s.recovery.enc", username)
	sha, err := c.getContentSha(ctx, owner, repo, path)
	if err != nil && !strings.Contains(err.Error(), "404") {
		return fmt.Errorf("failed to get current sha: %w", err)
	}

	req := ContentRequest{
		Message: fmt.Sprintf("Update recovery seed for %s", username),
		Content: c.EncodeContent(data),
		Sha:     sha,
	}

	return c.PutContent(ctx, owner, repo, path, req)
}

func (c *Client) GetVault(ctx context.Context, owner, repo string) (*ContentResponse, error) {
	path := ".vaulty/vault.enc"
	return c.GetContent(ctx, owner, repo, path)
}

func (c *Client) PutVault(ctx context.Context, owner, repo string, data []byte) error {
	path := ".vaulty/vault.enc"
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

func (c *Client) getContentSha(ctx context.Context, owner, repo, path string) (string, error) {
	content, err := c.GetContent(ctx, owner, repo, path)
	if err != nil {
		return "", err
	}
	return content.Sha, nil
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

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
