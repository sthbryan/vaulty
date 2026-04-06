package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestGetGitHubToken(t *testing.T) {
	t.Run("returns GITHUB_TOKEN when gh CLI fails", func(t *testing.T) {
	_ = os.Setenv("GITHUB_TOKEN", "test-token-from-env")
	defer func() { _ = os.Unsetenv("GITHUB_TOKEN") }()

		token, err := GetGitHubToken()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if token != "test-token-from-env" {
			t.Errorf("expected token to be 'test-token-from-env', got %s", token)
		}
	})

}

func TestNewClient(t *testing.T) {
	client := NewClient("test-token")
	if client == nil {
		t.Fatal("expected client to not be nil")
	}
	if client.Token != "test-token" {
		t.Errorf("expected token to be 'test-token', got %s", client.Token)
	}
	if client.BaseURL != GitHubAPIURL {
		t.Errorf("expected base URL to be %s, got %s", GitHubAPIURL, client.BaseURL)
	}
	if client.HTTPClient == nil {
		t.Error("expected HTTPClient to not be nil")
	}
}

func TestParseRepo(t *testing.T) {
	tests := []struct {
		name        string
		repo        string
		wantOwner   string
		wantName    string
		wantErr     bool
		errContains string
	}{
		{
			name:      "valid repo",
			repo:      "owner/repo",
			wantOwner: "owner",
			wantName:  "repo",
			wantErr:   false,
		},
		{
			name:      "local mode",
			repo:      "local://",
			wantOwner: "local",
			wantName:  "",
			wantErr:   false,
		},
		{
			name:        "missing slash",
			repo:        "ownerrepo",
			wantErr:     true,
			errContains: "invalid repo format",
		},
		{
			name:        "empty owner",
			repo:        "/repo",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:        "empty name",
			repo:        "owner/",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:        "too many slashes",
			repo:        "owner/repo/extra",
			wantErr:     true,
			errContains: "invalid repo format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, name, err := ParseRepo(tt.repo)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("expected error to contain '%s', got %v", tt.errContains, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if owner != tt.wantOwner {
				t.Errorf("expected owner to be '%s', got '%s'", tt.wantOwner, owner)
			}
			if name != tt.wantName {
				t.Errorf("expected name to be '%s', got '%s'", tt.wantName, name)
			}
		})
	}
}

func TestClientGetContent(t *testing.T) {
	t.Run("successfully gets file content", func(t *testing.T) {
		content := ContentResponse{
			Name:    "test.txt",
			Path:    "path/to/test.txt",
			Sha:     "abc123",
			Size:    12,
			Type:    "file",
			Content: base64.StdEncoding.EncodeToString([]byte("hello world")),
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test-token" {
				t.Errorf("expected Authorization header to be 'Bearer test-token', got '%s'", auth)
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(content)
		}))
		defer server.Close()

		client := NewClient("test-token")
		client.BaseURL = server.URL

		ctx := context.Background()
		result, err := client.GetContent(ctx, "owner", "repo", "path/to/test.txt")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if result.Name != content.Name {
			t.Errorf("expected name to be '%s', got '%s'", content.Name, result.Name)
		}
		if result.Path != content.Path {
			t.Errorf("expected path to be '%s', got '%s'", content.Path, result.Path)
		}
	})

	t.Run("returns error on non-200 status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message": "Not Found"}`))
		}))
		defer server.Close()

		client := NewClient("test-token")
		client.BaseURL = server.URL

		ctx := context.Background()
		_, err := client.GetContent(ctx, "owner", "repo", "nonexistent.txt")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "unexpected status code 404") {
			t.Errorf("expected error to contain 'unexpected status code 404', got %v", err)
		}
	})
}

func TestClientPutContent(t *testing.T) {
	t.Run("successfully creates file", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPut {
				t.Errorf("expected PUT, got %s", r.Method)
			}
			auth := r.Header.Get("Authorization")
			if auth != "Bearer test-token" {
				t.Errorf("expected Authorization header to be 'Bearer test-token', got '%s'", auth)
			}

			var req ContentRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("failed to decode request: %v", err)
			}

			if req.Message != "Create test file" {
				t.Errorf("expected message to be 'Create test file', got '%s'", req.Message)
			}

			w.WriteHeader(http.StatusCreated)
		}))
		defer server.Close()

		client := NewClient("test-token")
		client.BaseURL = server.URL

		ctx := context.Background()
		req := ContentRequest{
			Message: "Create test file",
			Content: base64.StdEncoding.EncodeToString([]byte("content")),
		}
		err := client.PutContent(ctx, "owner", "repo", "test.txt", req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("returns error on non-2xx status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message": "Bad credentials"}`))
		}))
		defer server.Close()

		client := NewClient("test-token")
		client.BaseURL = server.URL

		ctx := context.Background()
		req := ContentRequest{
			Message: "Create test file",
			Content: base64.StdEncoding.EncodeToString([]byte("content")),
		}
		err := client.PutContent(ctx, "owner", "repo", "test.txt", req)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "unexpected status code 401") {
			t.Errorf("expected error to contain 'unexpected status code 401', got %v", err)
		}
	})
}

func TestClientListDirectory(t *testing.T) {
	t.Run("successfully lists directory", func(t *testing.T) {
		items := []DirectoryItem{
			{
				Name: "file1.txt",
				Path: "dir/file1.txt",
				Type: "file",
				Size: 100,
			},
			{
				Name: "subdir",
				Path: "dir/subdir",
				Type: "dir",
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(items)
		}))
		defer server.Close()

		client := NewClient("test-token")
		client.BaseURL = server.URL

		ctx := context.Background()
		result, err := client.ListDirectory(ctx, "owner", "repo", "dir")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(result) != 2 {
			t.Fatalf("expected 2 items, got %d", len(result))
		}
		if result[0].Name != "file1.txt" {
			t.Errorf("expected first item name to be 'file1.txt', got '%s'", result[0].Name)
		}
		if result[1].Type != "dir" {
			t.Errorf("expected second item type to be 'dir', got '%s'", result[1].Type)
		}
	})

	t.Run("returns error on non-200 status", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := NewClient("test-token")
		client.BaseURL = server.URL

		ctx := context.Background()
		_, err := client.ListDirectory(ctx, "owner", "repo", "nonexistent")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestClientDecodeContent(t *testing.T) {
	client := NewClient("test-token")

	t.Run("decodes base64 content", func(t *testing.T) {
		original := []byte("hello world")
		content := ContentResponse{
			Content: base64.StdEncoding.EncodeToString(original),
		}

		decoded, err := client.DecodeContent(&content)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if string(decoded) != string(original) {
			t.Errorf("expected decoded content to be '%s', got '%s'", original, decoded)
		}
	})

	t.Run("returns error for unsupported encoding", func(t *testing.T) {
		content := ContentResponse{
			Content:  "some content",
			Encoding: "utf-8",
		}

		_, err := client.DecodeContent(&content)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "unsupported encoding") {
			t.Errorf("expected error to contain 'unsupported encoding', got %v", err)
		}
	})
}
