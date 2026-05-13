package storage

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
	"time"
)

type GitHubStorage struct {
	token   string
	owner   string
	repo    string
	baseURL string
}

func NewGitHubStorage(token, owner, repo string) *GitHubStorage {
	return &GitHubStorage{
		token:   token,
		owner:   owner,
		repo:    repo,
		baseURL: "https://api.github.com",
	}
}

func GetGitHubToken() (string, error) {
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		return strings.TrimSpace(string(output)), nil
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token != "" {
		return token, nil
	}

	return "", fmt.Errorf("no GitHub token found: install gh CLI or set GITHUB_TOKEN")
}

func (s *GitHubStorage) Ping(ctx context.Context) error {
	url := fmt.Sprintf("%s/repos/%s/%s", s.baseURL, s.owner, s.repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("verifying repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("repository not found or inaccessible")
	}

	return nil
}

func (s *GitHubStorage) Upload(ctx context.Context, path string, data []byte) error {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", s.baseURL, s.owner, s.repo, path)

	sha, _ := s.getFileSHA(ctx, path)

	content := base64.StdEncoding.EncodeToString(data)

	body := map[string]interface{}{
		"message": fmt.Sprintf("Update %s", path),
		"content": content,
	}
	if sha != "" {
		body["sha"] = sha
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("uploading file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed: %s", string(respBody))
	}

	return nil
}

func (s *GitHubStorage) Download(ctx context.Context, path string) ([]byte, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", s.baseURL, s.owner, s.repo, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting file: %w", err)
	}
	defer resp.Body.Close()

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
		resp, err := client.Get(downloadURL)
		if err != nil {
			return nil, fmt.Errorf("downloading file: %w", err)
		}
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	}

	return nil, fmt.Errorf("unsupported file format")
}

func (s *GitHubStorage) Delete(ctx context.Context, path string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", s.baseURL, s.owner, s.repo, path)

	sha, err := s.getFileSHA(ctx, path)
	if err != nil {
		return fmt.Errorf("file not found: %s", path)
	}

	body := map[string]interface{}{
		"message": fmt.Sprintf("Delete %s", path),
		"sha":      sha,
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("deleting file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed: %s", string(respBody))
	}

	return nil
}

func (s *GitHubStorage) List(ctx context.Context, prefix string) ([]string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", s.baseURL, s.owner, s.repo, prefix)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("listing contents: %w", err)
	}
	defer resp.Body.Close()

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

func (s *GitHubStorage) Exists(ctx context.Context, path string) (bool, error) {
	_, err := s.getFileSHA(ctx, path)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (s *GitHubStorage) getFileSHA(ctx context.Context, path string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", s.baseURL, s.owner, s.repo, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

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

func (s *GitHubStorage) CreateRepo(ctx context.Context) error {
	url := fmt.Sprintf("%s/user/repos", s.baseURL)

	body := map[string]interface{}{
		"name":    s.repo,
		"private": true,
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("creating repository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create repo failed: %s", string(respBody))
	}

	return nil
}