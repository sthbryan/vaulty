package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/sthbryan/vaulty/v2/pkg/models"
)

func TestNewLocalProvider(t *testing.T) {
	t.Run("creates correct path", func(t *testing.T) {
		provider := NewLocalProvider("testowner", "testrepo")
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("Failed to get home dir: %v", err)
		}
		expected := filepath.Join(home, ".vaulty", "testowner", "testrepo")
		if provider.baseURL != expected {
			t.Errorf("Expected baseURL %q, got %q", expected, provider.baseURL)
		}
		if provider.owner != "testowner" {
			t.Errorf("Expected owner %q, got %q", "testowner", provider.owner)
		}
		if provider.repo != "testrepo" {
			t.Errorf("Expected repo %q, got %q", "testrepo", provider.repo)
		}
	})
}

func TestLocalProvider_Roundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	provider := NewLocalProvider("testowner", "testrepo")
	provider.baseURL = tmpDir

	ctx := context.Background()
	testData := []byte("hello world")
	testPath := "secrets/test.txt"

	t.Run("Upload", func(t *testing.T) {
		err := provider.Upload(ctx, testPath, testData)
		if err != nil {
			t.Fatalf("Upload failed: %v", err)
		}
	})

	t.Run("Download", func(t *testing.T) {
		data, err := provider.Download(ctx, testPath)
		if err != nil {
			t.Fatalf("Download failed: %v", err)
		}
		if string(data) != string(testData) {
			t.Errorf("Expected %q, got %q", testData, data)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := provider.Delete(ctx, testPath)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		exists, _ := provider.Exists(ctx, testPath)
		if exists {
			t.Error("File should not exist after delete")
		}
	})
}

func TestLocalProvider_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	provider := NewLocalProvider("testowner", "testrepo")
	provider.baseURL = tmpDir

	ctx := context.Background()
	testPath := "secrets/exists.txt"

	exists, err := provider.Exists(ctx, testPath)
	if err != nil {
		t.Fatalf("Exists check failed: %v", err)
	}
	if exists {
		t.Error("File should not exist initially")
	}

	err = provider.Upload(ctx, testPath, []byte("test"))
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	exists, err = provider.Exists(ctx, testPath)
	if err != nil {
		t.Fatalf("Exists check failed: %v", err)
	}
	if !exists {
		t.Error("File should exist after upload")
	}
}

func TestLocalProvider_List(t *testing.T) {
	tmpDir := t.TempDir()
	provider := NewLocalProvider("testowner", "testrepo")
	provider.baseURL = tmpDir

	ctx := context.Background()

	err := os.MkdirAll(filepath.Join(tmpDir, "secrets"), 0755)
	if err != nil {
		t.Fatalf("Failed to create dir: %v", err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, "secrets", "a.txt"), []byte("a"), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, "secrets", "b.txt"), []byte("b"), 0644)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	files, err := provider.List(ctx, "secrets")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}
}

func TestLocalProvider_Ping(t *testing.T) {
	tmpDir := t.TempDir()
	provider := NewLocalProvider("testowner", "testrepo")
	provider.baseURL = tmpDir

	ctx := context.Background()

	err := provider.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	provider.baseURL = filepath.Join(tmpDir, "nonexistent")
	err = provider.Ping(ctx)
	if err != nil {
		t.Logf("Ping created directory as expected: %v", err)
	}
}

func TestLocalProvider_CreateRepo(t *testing.T) {
	tmpDir := t.TempDir()
	provider := NewLocalProvider("testowner", "testrepo")
	provider.baseURL = filepath.Join(tmpDir, "vault")

	ctx := context.Background()

	err := provider.CreateRepo(ctx)
	if err != nil {
		t.Fatalf("CreateRepo failed: %v", err)
	}

	info, err := os.Stat(provider.baseURL)
	if err != nil {
		t.Fatalf("Failed to stat directory: %v", err)
	}
	if !info.IsDir() {
		t.Error("Path should be a directory")
	}
}

func TestLocalProvider_CheckVault(t *testing.T) {
	tmpDir := t.TempDir()
	provider := NewLocalProvider("testowner", "testrepo")
	provider.baseURL = tmpDir

	if !provider.CheckVault() {
		t.Error("CheckVault should return true for existing directory")
	}

	provider.baseURL = filepath.Join(tmpDir, "nonexistent")
	if provider.CheckVault() {
		t.Error("CheckVault should return false for nonexistent directory")
	}
}

func TestLocalProvider_SetupStorage(t *testing.T) {
	tmpDir := t.TempDir()
	provider := NewLocalProvider("testowner", "testrepo")
	provider.baseURL = tmpDir

	err := provider.SetupStorage()
	if err == nil {
		t.Error("SetupStorage should fail when vault already exists")
	}

	provider.baseURL = filepath.Join(tmpDir, "newvault")
	err = provider.SetupStorage()
	if err != nil {
		t.Fatalf("SetupStorage failed: %v", err)
	}
}

func TestLocalProvider_Meta(t *testing.T) {
	tmpDir := t.TempDir()
	provider := NewLocalProvider("testowner", "testrepo")
	provider.baseURL = tmpDir

	meta := &models.VaultMeta{
		Salt:         "testsalt",
		EncryptedKey: "testkey",
	}

	err := provider.SaveMeta(meta)
	if err != nil {
		t.Fatalf("SaveMeta failed: %v", err)
	}

	loaded, err := provider.LoadMeta()
	if err != nil {
		t.Fatalf("LoadMeta failed: %v", err)
	}
	if loaded.Salt != meta.Salt {
		t.Errorf("Expected salt %q, got %q", meta.Salt, loaded.Salt)
	}
	if loaded.EncryptedKey != meta.EncryptedKey {
		t.Errorf("Expected key %q, got %q", meta.EncryptedKey, loaded.EncryptedKey)
	}
}

func TestNewGitHubProvider(t *testing.T) {
	t.Run("sets correct baseURL", func(t *testing.T) {
		provider := NewGitHubProvider("testtoken", "testowner", "testrepo")
		if provider.baseURL != "https://api.github.com" {
			t.Errorf("Expected baseURL %q, got %q", "https://api.github.com", provider.baseURL)
		}
		if provider.owner != "testowner" {
			t.Errorf("Expected owner %q, got %q", "testowner", provider.owner)
		}
		if provider.repo != "testrepo" {
			t.Errorf("Expected repo %q, got %q", "testrepo", provider.repo)
		}
		if provider.client == nil {
			t.Error("client should not be nil")
		}
	})
}

func TestGitHubProvider_Ping(t *testing.T) {
	t.Run("successful ping", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"name": "testrepo"})
		}))
		defer server.Close()

		client := &http.Client{Transport: &authTransport{token: "testtoken"}}
		provider := &GitHubProvider{
			Provider: &Provider{
				ProviderConfig: ProviderConfig{token: "testtoken", owner: "testowner", repo: "testrepo"},
				baseURL:        server.URL,
			},
			client: client,
		}

		err := provider.Ping(context.Background())
		if err != nil {
			t.Errorf("Ping should succeed, got: %v", err)
		}
	})

	t.Run("failed ping", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := &http.Client{Transport: &authTransport{token: "testtoken"}}
		provider := &GitHubProvider{
			Provider: &Provider{
				ProviderConfig: ProviderConfig{token: "testtoken", owner: "testowner", repo: "testrepo"},
				baseURL:        server.URL,
			},
			client: client,
		}

		err := provider.Ping(context.Background())
		if err == nil {
			t.Error("Ping should fail for nonexistent repo")
		}
	})
}

func TestGitHubProvider_Upload(t *testing.T) {
	var capturedMethod string
	var capturedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		json.NewDecoder(r.Body).Decode(&capturedBody)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"content": map[string]interface{}{}})
	}))
	defer server.Close()

	client := &http.Client{Transport: &authTransport{token: "testtoken"}}
	provider := &GitHubProvider{
		Provider: &Provider{
			ProviderConfig: ProviderConfig{token: "testtoken", owner: "testowner", repo: "testrepo"},
			baseURL:        server.URL,
		},
		client: client,
	}

	err := provider.Upload(context.Background(), "secrets/test.txt", []byte("hello"))
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	if capturedMethod != http.MethodPut {
		t.Errorf("Expected PUT method, got %s", capturedMethod)
	}
	if capturedBody["message"] == nil {
		t.Error("Request body should contain message")
	}
}

func TestGitHubProvider_Download(t *testing.T) {
	t.Run("download with base64 content", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"content":  "aGVsbG8=",
				"encoding": "base64",
			})
		}))
		defer server.Close()

		client := &http.Client{Transport: &authTransport{token: "testtoken"}}
		provider := &GitHubProvider{
			Provider: &Provider{
				ProviderConfig: ProviderConfig{token: "testtoken", owner: "testowner", repo: "testrepo"},
				baseURL:        server.URL,
			},
			client: client,
		}

		data, err := provider.Download(context.Background(), "secrets/test.txt")
		if err != nil {
			t.Fatalf("Download failed: %v", err)
		}
		if string(data) != "hello" {
			t.Errorf("Expected 'hello', got %q", string(data))
		}
	})

	t.Run("download 404", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := &http.Client{Transport: &authTransport{token: "testtoken"}}
		provider := &GitHubProvider{
			Provider: &Provider{
				ProviderConfig: ProviderConfig{token: "testtoken", owner: "testowner", repo: "testrepo"},
				baseURL:        server.URL,
			},
			client: client,
		}

		_, err := provider.Download(context.Background(), "nonexistent.txt")
		if err == nil {
			t.Error("Download should fail for nonexistent file")
		}
	})
}

func TestGitHubProvider_Delete(t *testing.T) {
	var capturedMethod string
	var capturedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"sha": "abc123",
			})
		} else if r.Method == http.MethodDelete {
			capturedMethod = r.Method
			json.NewDecoder(r.Body).Decode(&capturedBody)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	client := &http.Client{Transport: &authTransport{token: "testtoken"}}
	provider := &GitHubProvider{
		Provider: &Provider{
			ProviderConfig: ProviderConfig{token: "testtoken", owner: "testowner", repo: "testrepo"},
			baseURL:        server.URL,
		},
		client: client,
	}

	err := provider.Delete(context.Background(), "secrets/test.txt")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if capturedMethod != http.MethodDelete {
		t.Errorf("Expected DELETE method, got %s", capturedMethod)
	}
	if capturedBody["sha"] != "abc123" {
		t.Errorf("Expected sha 'abc123', got %v", capturedBody["sha"])
	}
}

func TestGitHubProvider_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"type": "file", "path": "secrets/a.txt"},
			{"type": "file", "path": "secrets/b.txt"},
			{"type": "dir", "path": "secrets/nested"},
		})
	}))
	defer server.Close()

	client := &http.Client{Transport: &authTransport{token: "testtoken"}}
	provider := &GitHubProvider{
		Provider: &Provider{
			ProviderConfig: ProviderConfig{token: "testtoken", owner: "testowner", repo: "testrepo"},
			baseURL:        server.URL,
		},
		client: client,
	}

	files, err := provider.List(context.Background(), "secrets")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}
}

func TestGitHubProvider_Exists(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"sha": "abc123"})
		}))
		defer server.Close()

		client := &http.Client{Transport: &authTransport{token: "testtoken"}}
		provider := &GitHubProvider{
			Provider: &Provider{
				ProviderConfig: ProviderConfig{token: "testtoken", owner: "testowner", repo: "testrepo"},
				baseURL:        server.URL,
			},
			client: client,
		}

		exists, err := provider.Exists(context.Background(), "secrets/test.txt")
		if err != nil {
			t.Fatalf("Exists check failed: %v", err)
		}
		if !exists {
			t.Error("File should exist")
		}
	})

	t.Run("not exists", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := &http.Client{Transport: &authTransport{token: "testtoken"}}
		provider := &GitHubProvider{
			Provider: &Provider{
				ProviderConfig: ProviderConfig{token: "testtoken", owner: "testowner", repo: "testrepo"},
				baseURL:        server.URL,
			},
			client: client,
		}

		exists, err := provider.Exists(context.Background(), "secrets/test.txt")
		if err != nil {
			t.Fatalf("Exists check failed: %v", err)
		}
		if exists {
			t.Error("File should not exist")
		}
	})
}

func TestGitHubProvider_CheckVault(t *testing.T) {
	t.Run("vault exists", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"name": "testrepo"})
		}))
		defer server.Close()

		client := &http.Client{Transport: &authTransport{token: "testtoken"}}
		provider := &GitHubProvider{
			Provider: &Provider{
				ProviderConfig: ProviderConfig{token: "testtoken", owner: "testowner", repo: "testrepo"},
				baseURL:        server.URL,
			},
			client: client,
		}

		if !provider.CheckVault() {
			t.Error("CheckVault should return true")
		}
	})

	t.Run("vault does not exist", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := &http.Client{Transport: &authTransport{token: "testtoken"}}
		provider := &GitHubProvider{
			Provider: &Provider{
				ProviderConfig: ProviderConfig{token: "testtoken", owner: "testowner", repo: "testrepo"},
				baseURL:        server.URL,
			},
			client: client,
		}

		if provider.CheckVault() {
			t.Error("CheckVault should return false")
		}
	})
}

func TestGitHubProvider_SetupStorage(t *testing.T) {
	t.Run("vault exists fails", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"name": "testrepo"})
		}))
		defer server.Close()

		client := &http.Client{Transport: &authTransport{token: "testtoken"}}
		provider := &GitHubProvider{
			Provider: &Provider{
				ProviderConfig: ProviderConfig{token: "testtoken", owner: "testowner", repo: "testrepo"},
				baseURL:        server.URL,
			},
			client: client,
		}

		err := provider.SetupStorage()
		if err == nil {
			t.Error("SetupStorage should fail when vault exists")
		}
	})

	t.Run("create new vault", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusNotFound)
			} else if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusCreated)
			}
		}))
		defer server.Close()

		client := &http.Client{Transport: &authTransport{token: "testtoken"}}
		provider := &GitHubProvider{
			Provider: &Provider{
				ProviderConfig: ProviderConfig{token: "testtoken", owner: "testowner", repo: "newrepo"},
				baseURL:        server.URL,
			},
			client: client,
		}

		err := provider.SetupStorage()
		if err != nil {
			t.Fatalf("SetupStorage failed: %v", err)
		}
	})
}

func TestGitHubProvider_CreateRepo(t *testing.T) {
	var capturedMethod string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{"name": "newrepo"})
	}))
	defer server.Close()

	client := &http.Client{Transport: &authTransport{token: "testtoken"}}
	provider := &GitHubProvider{
		Provider: &Provider{
			ProviderConfig: ProviderConfig{token: "testtoken", owner: "testowner", repo: "newrepo"},
			baseURL:        server.URL,
		},
		client: client,
	}

	err := provider.CreateRepo(context.Background())
	if err != nil {
		t.Fatalf("CreateRepo failed: %v", err)
	}

	if capturedMethod != http.MethodPost {
		t.Errorf("Expected POST method, got %s", capturedMethod)
	}
}

func TestIsFileNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"file not found", fmt.Errorf("file not found: secrets/test.txt"), true},
		{"not found", fmt.Errorf("not found: secrets/test.txt"), true},
		{"other error", fmt.Errorf("connection refused"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsFileNotFound(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestNewProvider(t *testing.T) {
	t.Run("github provider", func(t *testing.T) {
		provider := NewProvider(ProviderGitHub, "token", "owner", "repo")
		if provider == nil {
			t.Error("Provider should not be nil")
		}
		ghProvider, ok := provider.(*GitHubProvider)
		if !ok {
			t.Fatal("Expected GitHubProvider")
		}
		if ghProvider.owner != "owner" {
			t.Errorf("Expected owner 'owner', got %q", ghProvider.owner)
		}
	})

	t.Run("local provider", func(t *testing.T) {
		provider := NewProvider(ProviderLocal, "owner", "repo")
		if provider == nil {
			t.Error("Provider should not be nil")
		}
		localProvider, ok := provider.(*LocalProvider)
		if !ok {
			t.Fatal("Expected LocalProvider")
		}
		if localProvider.owner != "owner" {
			t.Errorf("Expected owner 'owner', got %q", localProvider.owner)
		}
	})

	t.Run("insufficient params", func(t *testing.T) {
		provider := NewProvider(ProviderGitHub)
		if provider != nil {
			t.Error("Provider should be nil with insufficient params")
		}
	})
}
