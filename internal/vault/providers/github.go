package providers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sthbryan/vaulty/v2/pkg/models"
	"gopkg.in/yaml.v3"
)

type GitHubProvider struct {
	*Provider
	client *http.Client
}

type authTransport struct {
	token string
}

func NewGitHubProvider(token, owner, repo string) *GitHubProvider {
	return &GitHubProvider{
		Provider: &Provider{
			ProviderConfig: ProviderConfig{token: token, owner: owner, repo: repo},
			baseURL:        "https://api.github.com",
		},
		client: newGitHubClient(token),
	}
}

func newGitHubClient(token string) *http.Client {
	return &http.Client{
		Transport: &authTransport{token: token},
		Timeout:   30 * time.Second,
	}
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	return http.DefaultTransport.RoundTrip(req)
}

func (p *GitHubProvider) doRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	return p.client.Do(req)
}

func (p *GitHubProvider) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/repos/%s/%s", p.baseURL, p.owner, p.repo)
	resp, err := p.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("verifying repository: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("repository not found or inaccessible")
	}
	return nil
}

func (p *GitHubProvider) Upload(ctx context.Context, path string, data []byte) error {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", p.baseURL, p.owner, p.repo, path)

	sha, _ := p.getFileSHA(ctx, path)

	content := base64.StdEncoding.EncodeToString(data)

	body := map[string]interface{}{
		"message": fmt.Sprintf("Update %s", path),
		"content": content,
	}
	if sha != "" {
		body["sha"] = sha
	}

	jsonBody, _ := json.Marshal(body)

	resp, err := p.doRequest(ctx, http.MethodPut, url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("uploading file: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed: %s", string(respBody))
	}

	return nil
}

func (p *GitHubProvider) Download(ctx context.Context, path string) ([]byte, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", p.baseURL, p.owner, p.repo, path)

	resp, err := p.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("getting file: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	if content, ok := result["content"].(string); ok {
		encoding, _ := result["encoding"].(string)
		if encoding == "base64" {
			cleanContent := strings.ReplaceAll(content, "\n", "")
			return base64.StdEncoding.DecodeString(cleanContent)
		}
	}

	if downloadURL, ok := result["download_url"].(string); ok && downloadURL != "" {
		resp, err := p.client.Get(downloadURL)
		if err != nil {
			return nil, fmt.Errorf("downloading file: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		return io.ReadAll(resp.Body)
	}

	return nil, fmt.Errorf("unsupported file format")
}

func (p *GitHubProvider) Delete(ctx context.Context, path string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", p.baseURL, p.owner, p.repo, path)

	sha, err := p.getFileSHA(ctx, path)
	if err != nil {
		return fmt.Errorf("file not found: %s", path)
	}

	body := map[string]interface{}{
		"message": fmt.Sprintf("Delete %s", path),
		"sha":     sha,
	}

	jsonBody, _ := json.Marshal(body)

	resp, err := p.doRequest(ctx, http.MethodDelete, url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("deleting file: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed: %s", string(respBody))
	}

	return nil
}

func (p *GitHubProvider) List(ctx context.Context, prefix string) ([]string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", p.baseURL, p.owner, p.repo, prefix)

	resp, err := p.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("listing contents: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	var items []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	var files []string
	for _, item := range items {
		if itemType, ok := item["type"].(string); ok && itemType == "file" {
			if path, ok := item["path"].(string); ok {
				files = append(files, path)
			}
		}
	}

	return files, nil
}

func (p *GitHubProvider) Exists(ctx context.Context, path string) (bool, error) {
	_, err := p.getFileSHA(ctx, path)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (p *GitHubProvider) CreateRepo(ctx context.Context) error {
	url := fmt.Sprintf("%s/user/repos", p.baseURL)

	body := map[string]interface{}{
		"name":    p.repo,
		"private": true,
	}

	jsonBody, _ := json.Marshal(body)

	resp, err := p.doRequest(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("creating repository: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create repo failed: %s", string(respBody))
	}

	return nil
}

func (p *GitHubProvider) getFileSHA(ctx context.Context, path string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", p.baseURL, p.owner, p.repo, path)

	resp, err := p.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("file not found")
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if sha, ok := result["sha"].(string); ok {
		return sha, nil
	}

	return "", fmt.Errorf("sha not found")
}

func (p *GitHubProvider) CheckVault() bool {
	return p.Ping(context.Background()) == nil
}

func (p *GitHubProvider) SetupStorage() error {
	if p.CheckVault() {
		return fmt.Errorf("repository already exists")
	}
	return p.CreateRepo(context.Background())
}

func (p *GitHubProvider) LoadMeta() (*models.VaultMeta, error) {
	data, err := p.Download(context.Background(), "vault.meta")
	if err != nil {
		return nil, fmt.Errorf("downloading vault.meta: %w", err)
	}

	var meta models.VaultMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parsing vault.meta: %w", err)
	}

	return &meta, nil
}

func (p *GitHubProvider) SaveMeta(meta *models.VaultMeta) error {
	data, err := yaml.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshaling meta: %w", err)
	}

	return p.Upload(context.Background(), "vault.meta", data)
}

func IsFileNotFound(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "file not found") || strings.Contains(errStr, "not found")
}
