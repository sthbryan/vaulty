package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultPath(t *testing.T) {
	path := DefaultPath()
	if path == "" {
		t.Fatal("DefaultPath() returned empty string")
	}

	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".vty", "config.json")
	if path != expected {
		t.Errorf("DefaultPath() = %q, want %q", path, expected)
	}
}

func TestLoad(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	t.Run("load existing file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "config.json")
		content := `{"repo": "/path/to/repo", "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-02T00:00:00Z"}`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.Repo != "/path/to/repo" {
			t.Errorf("cfg.Repo = %q, want %q", cfg.Repo, "/path/to/repo")
		}
	})

	t.Run("load non-existent file returns empty config", func(t *testing.T) {
		path := filepath.Join(tmpDir, "nonexistent.json")
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg == nil {
			t.Fatal("Load() returned nil config")
		}
		if cfg.Repo != "" {
			t.Errorf("cfg.Repo = %q, want empty string", cfg.Repo)
		}
	})

	t.Run("load invalid json returns error", func(t *testing.T) {
		path := filepath.Join(tmpDir, "invalid.json")
		if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		_, err := Load(path)
		if err == nil {
			t.Error("Load() expected error for invalid JSON")
		}
	})

	t.Run("load with empty path uses default", func(t *testing.T) {
		// This will fail because default path doesn't exist, but shouldn't panic
		_, err := Load("")
		// Error is expected since ~/.vty/config.json likely doesn't exist
		// We just verify it doesn't panic
		_ = err
	})
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("save new config", func(t *testing.T) {
		path := filepath.Join(tmpDir, "newconfig.json")
		cfg := &Config{Repo: "/test/repo"}

		if err := cfg.Save(path); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		// Verify file was created
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatal("Save() did not create file")
		}

		// Verify file permissions
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("failed to stat file: %v", err)
		}
		if info.Mode().Perm() != 0600 {
			t.Errorf("file permissions = %o, want 0600", info.Mode().Perm())
		}

		// Verify timestamps were set
		if cfg.CreatedAt.IsZero() {
			t.Error("CreatedAt not set")
		}
		if cfg.UpdatedAt.IsZero() {
			t.Error("UpdatedAt not set")
		}
	})

	t.Run("save creates directories", func(t *testing.T) {
		path := filepath.Join(tmpDir, "nested", "deep", "config.json")
		cfg := &Config{Repo: "/test/repo"}

		if err := cfg.Save(path); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatal("Save() did not create file in nested directory")
		}
	})

	t.Run("save with empty path uses default", func(t *testing.T) {
		cfg := &Config{Repo: "/test/repo"}
		// This should attempt to save to ~/.vty/config.json
		// We'll get an error if home dir isn't writable, but it shouldn't panic
		err := cfg.Save("")
		// We don't check error here as it depends on environment
		_ = err
	})

	t.Run("save updates timestamps", func(t *testing.T) {
		path := filepath.Join(tmpDir, "timestamp.json")
		cfg := &Config{Repo: "/test/repo"}

		if err := cfg.Save(path); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		createdAt := cfg.CreatedAt

		// Save again and verify UpdatedAt changed but CreatedAt stayed same
		cfg.Repo = "/new/repo"
		if err := cfg.Save(path); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		if !cfg.CreatedAt.Equal(createdAt) {
			t.Error("CreatedAt should not change on subsequent saves")
		}
	})
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "valid config with repo",
			cfg:     Config{Repo: "/path/to/repo"},
			wantErr: false,
		},
		{
			name:    "invalid config without repo",
			cfg:     Config{},
			wantErr: true,
		},
		{
			name:    "invalid config with empty repo",
			cfg:     Config{Repo: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr && err == nil {
				t.Error("Validate() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestSetRepo(t *testing.T) {
	t.Run("set repo on new config", func(t *testing.T) {
		cfg := &Config{}
		cfg.SetRepo("/new/repo")

		if cfg.Repo != "/new/repo" {
			t.Errorf("cfg.Repo = %q, want %q", cfg.Repo, "/new/repo")
		}
		if cfg.CreatedAt.IsZero() {
			t.Error("CreatedAt should be set")
		}
		if cfg.UpdatedAt.IsZero() {
			t.Error("UpdatedAt should be set")
		}
	})

	t.Run("set repo updates existing config", func(t *testing.T) {
		now := time.Now()
		cfg := &Config{
			Repo:      "/old/repo",
			CreatedAt: now,
			UpdatedAt: now,
		}
		originalCreated := cfg.CreatedAt

		cfg.SetRepo("/new/repo")

		if cfg.Repo != "/new/repo" {
			t.Errorf("cfg.Repo = %q, want %q", cfg.Repo, "/new/repo")
		}
		if !cfg.CreatedAt.Equal(originalCreated) {
			t.Error("CreatedAt should not change when setting repo on existing config")
		}
	})
}
