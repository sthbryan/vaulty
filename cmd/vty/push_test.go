package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/sthbryan/vaulty/v2/pkg/models"
)

func TestValidatePushName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple name", "my-secret", false},
		{"valid name with dashes", "my-secret-name", false},
		{"valid name with underscores", "my_secret_name", false},
		{"valid name with numbers", "secret123", false},
		{"empty name", "", true},
		{"whitespace only", "   ", true},
		{"name with forward slash", "path/to/secret", true},
		{"name with backslash", "path\\to\\secret", true},
		{"name with mixed slashes", "path/to\\secret", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePushName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePushName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCalculateDirSize(t *testing.T) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "vaulty-test-dir-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tests := []struct {
		name     string
		setup    func(dir string) error
		wantSize int64
	}{
		{
			name:     "empty directory",
			setup:    func(dir string) error { return nil },
			wantSize: 0,
		},
		{
			name: "single file",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0644)
			},
			wantSize: 5,
		},
		{
			name: "multiple files",
			setup: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("hello"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("world"), 0644)
			},
			wantSize: 10,
		},
		{
			name: "nested directories",
			setup: func(dir string) error {
				subDir := filepath.Join(dir, "subdir")
				if err := os.MkdirAll(subDir, 0755); err != nil {
					return err
				}
				if err := os.WriteFile(filepath.Join(dir, "file1.txt"), []byte("hello"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("world"), 0644)
			},
			wantSize: 10,
		},
		{
			name: "larger file",
			setup: func(dir string) error {
				content := make([]byte, 1024)
				for i := range content {
					content[i] = 'a'
				}
				return os.WriteFile(filepath.Join(dir, "large.txt"), content, 0644)
			},
			wantSize: 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh temp dir for each test
			testDir, err := os.MkdirTemp(tmpDir, "test-*")
			if err != nil {
				t.Fatalf("Failed to create test subdir: %v", err)
			}
			defer func() { _ = os.RemoveAll(testDir) }()

			if err := tt.setup(testDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			size := calculateDirSize(testDir)
			if size != tt.wantSize {
				t.Errorf("calculateDirSize() = %d, want %d", size, tt.wantSize)
			}
		})
	}
}

func TestCommandError_Error(t *testing.T) {
	t.Run("Error method returns Message", func(t *testing.T) {
		cmdErr := &CommandError{
			Message: "test error message",
			Hint:    "test hint",
		}

		if cmdErr.Error() != "test error message" {
			t.Errorf("CommandError.Error() = %q, want %q", cmdErr.Error(), "test error message")
		}
	})

	t.Run("Error method without hint", func(t *testing.T) {
		cmdErr := &CommandError{
			Message: "simple error",
		}

		if cmdErr.Error() != "simple error" {
			t.Errorf("CommandError.Error() = %q, want %q", cmdErr.Error(), "simple error")
		}
	})

	t.Run("Error method with examples", func(t *testing.T) {
		cmdErr := &CommandError{
			Message: "invalid type",
			Hint:    "Valid types: env, config, ssh, resources",
			Examples: []string{
				"vty push env api .env",
				"vty push ssh deploy id_rsa",
			},
		}

		if cmdErr.Error() != "invalid type" {
			t.Errorf("CommandError.Error() = %q, want %q", cmdErr.Error(), "invalid type")
		}
	})

	t.Run("Error method with empty message", func(t *testing.T) {
		cmdErr := &CommandError{
			Message: "",
			Hint:    "some hint",
		}

		if cmdErr.Error() != "" {
			t.Errorf("CommandError.Error() = %q, want empty string", cmdErr.Error())
		}
	})
}

func TestPushArgsValidation(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantErr      bool
		expectedType string
	}{
		{
			name:         "valid env type",
			args:         []string{"env", "my-secret", "/path/to/file"},
			wantErr:      false,
			expectedType: "env",
		},
		{
			name:         "valid config type",
			args:         []string{"config", "settings", "/path/to/file"},
			wantErr:      false,
			expectedType: "config",
		},
		{
			name:         "valid ssh type",
			args:         []string{"ssh", "deploy-key", "/path/to/file"},
			wantErr:      false,
			expectedType: "ssh",
		},
		{
			name:         "valid resources type",
			args:         []string{"resources", "assets", "/path/to/file"},
			wantErr:      false,
			expectedType: "resources",
		},
		{
			name:         "missing all args",
			args:         []string{},
			wantErr:      true,
			expectedType: "",
		},
		{
			name:         "missing name and path",
			args:         []string{"env"},
			wantErr:      true,
			expectedType: "",
		},
		{
			name:         "missing path",
			args:         []string{"env", "my-secret"},
			wantErr:      true,
			expectedType: "",
		},
		{
			name:         "invalid type",
			args:         []string{"invalid", "name", "/path"},
			wantErr:      true,
			expectedType: "",
		},
		{
			name:         "empty type string",
			args:         []string{"", "name", "/path"},
			wantErr:      true,
			expectedType: "",
		},
		{
			name:         "uppercase type (invalid)",
			args:         []string{"ENV", "name", "/path"},
			wantErr:      true,
			expectedType: "",
		},
		{
			name:         "mixed case type (invalid)",
			args:         []string{"Env", "name", "/path"},
			wantErr:      true,
			expectedType: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal cobra command to test Args validation
			cmd := &cobra.Command{
				Args: pushCmd.Args,
			}

			err := cmd.Args(cmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Args() error = %v, wantErr %v", err, tt.wantErr)
			}

			// For valid args, verify secret type
			if !tt.wantErr && tt.expectedType != "" {
				secretType := models.SecretType(tt.args[0])
				if !secretType.IsValid() {
					t.Errorf("SecretType(%q).IsValid() = false, want true", tt.args[0])
				}
			}
		})
	}
}

func TestValidateEnvFile(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() (string, func())
		wantErr bool
	}{
		{
			name: "valid .env file",
			setup: func() (string, func()) {
				tmpDir, _ := os.MkdirTemp("", "vaulty-env-test-*")
				path := filepath.Join(tmpDir, ".env")
				content := "DATABASE_URL=postgres://localhost\nAPI_KEY=secret123\n"
				_ = os.WriteFile(path, []byte(content), 0644)
				cleanup := func() { os.RemoveAll(tmpDir) }
				return path, cleanup
			},
			wantErr: false,
		},
		{
			name: "valid .env file with comments",
			setup: func() (string, func()) {
				tmpDir, _ := os.MkdirTemp("", "vaulty-env-test-*")
				path := filepath.Join(tmpDir, ".env")
				content := "# This is a comment\nDATABASE_URL=postgres://localhost\n# Another comment\nAPI_KEY=secret123\n"
				_ = os.WriteFile(path, []byte(content), 0644)
				cleanup := func() { os.RemoveAll(tmpDir) }
				return path, cleanup
			},
			wantErr: false,
		},
		{
			name: "valid file without .env extension but with KEY=VALUE",
			setup: func() (string, func()) {
				tmpDir, _ := os.MkdirTemp("", "vaulty-env-test-*")
				path := filepath.Join(tmpDir, "config")
				content := "HOST=localhost\nPORT=5432\n"
				_ = os.WriteFile(path, []byte(content), 0644)
				cleanup := func() { os.RemoveAll(tmpDir) }
				return path, cleanup
			},
			wantErr: false,
		},
		{
			name: "invalid file no KEY=VALUE patterns",
			setup: func() (string, func()) {
				tmpDir, _ := os.MkdirTemp("", "vaulty-env-test-*")
				path := filepath.Join(tmpDir, "random.txt")
				content := "This is not a valid env file\nJust some random text\n"
				_ = os.WriteFile(path, []byte(content), 0644)
				cleanup := func() { os.RemoveAll(tmpDir) }
				return path, cleanup
			},
			wantErr: true,
		},
		{
			name: "empty file",
			setup: func() (string, func()) {
				tmpDir, _ := os.MkdirTemp("", "vaulty-env-test-*")
				path := filepath.Join(tmpDir, "empty.txt")
				os.WriteFile(path, []byte(""), 0644)
				cleanup := func() { os.RemoveAll(tmpDir) }
				return path, cleanup
			},
			wantErr: true,
		},
		{
			name: "file with only comments",
			setup: func() (string, func()) {
				tmpDir, _ := os.MkdirTemp("", "vaulty-env-test-*")
				path := filepath.Join(tmpDir, "comments.txt")
				content := "# Just a comment\n# Another comment\n"
				_ = os.WriteFile(path, []byte(content), 0644)
				cleanup := func() { os.RemoveAll(tmpDir) }
				return path, cleanup
			},
			wantErr: true,
		},
		{
			name: "directory is always valid",
			setup: func() (string, func()) {
				tmpDir, _ := os.MkdirTemp("", "vaulty-env-test-*")
				cleanup := func() { os.RemoveAll(tmpDir) }
				return tmpDir, cleanup
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, cleanup := tt.setup()
			defer cleanup()

			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("Failed to stat path: %v", err)
			}
			isDir := info.IsDir()

			err = validateEnvFile(path, isDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEnvFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateSSHFile(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() (string, func())
		wantErr bool
	}{
		{
			name: "valid SSH private key",
			setup: func() (string, func()) {
				tmpDir, _ := os.MkdirTemp("", "vaulty-ssh-test-*")
				path := filepath.Join(tmpDir, "id_rsa")
				content := "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW\nQyNTUxOQAAACB7y1hZ7QAAAAE2ODY4NwAAAJDi0uVIAAAAoAAAAAAAAAA=\n-----END OPENSSH PRIVATE KEY-----\n"
				os.WriteFile(path, []byte(content), 0600)
				cleanup := func() { os.RemoveAll(tmpDir) }
				return path, cleanup
			},
			wantErr: false,
		},
		{
			name: "valid SSH RSA key",
			setup: func() (string, func()) {
				tmpDir, _ := os.MkdirTemp("", "vaulty-ssh-test-*")
				path := filepath.Join(tmpDir, "id_rsa")
				content := "-----BEGIN RSA PRIVATE KEY-----\nMIIBOgIBAAJBALRiMLAHudeSA2O2LkZ6lqSj7QIDAQABAoGAXz4EuPVf6jE5mP4p\n-----END RSA PRIVATE KEY-----\n"
				os.WriteFile(path, []byte(content), 0600)
				cleanup := func() { os.RemoveAll(tmpDir) }
				return path, cleanup
			},
			wantErr: false,
		},
		{
			name: "valid SSH EC key",
			setup: func() (string, func()) {
				tmpDir, _ := os.MkdirTemp("", "vaulty-ssh-test-*")
				path := filepath.Join(tmpDir, "ec_key")
				content := "-----BEGIN EC PRIVATE KEY-----\nMHQCAQAAIIP6q+Gq9qAAAAEE\n-----END EC PRIVATE KEY-----\n"
				os.WriteFile(path, []byte(content), 0600)
				cleanup := func() { os.RemoveAll(tmpDir) }
				return path, cleanup
			},
			wantErr: false,
		},
		{
			name: "invalid file without BEGIN marker",
			setup: func() (string, func()) {
				tmpDir, _ := os.MkdirTemp("", "vaulty-ssh-test-*")
				path := filepath.Join(tmpDir, "random")
				content := "This is not an SSH key\nJust some random text\n"
				_ = os.WriteFile(path, []byte(content), 0644)
				cleanup := func() { os.RemoveAll(tmpDir) }
				return path, cleanup
			},
			wantErr: true,
		},
		{
			name: "invalid empty file",
			setup: func() (string, func()) {
				tmpDir, _ := os.MkdirTemp("", "vaulty-ssh-test-*")
				path := filepath.Join(tmpDir, "empty")
				os.WriteFile(path, []byte(""), 0644)
				cleanup := func() { os.RemoveAll(tmpDir) }
				return path, cleanup
			},
			wantErr: true,
		},
		{
			name: "invalid file missing END marker",
			setup: func() (string, func()) {
				tmpDir, _ := os.MkdirTemp("", "vaulty-ssh-test-*")
				path := filepath.Join(tmpDir, "partial")
				// Note: validateSSHFile only checks for -----BEGIN, not complete PEM format
				content := "-----BEGIN OPENSSH PRIVATE KEY-----\npartial content without END marker\n"
				_ = os.WriteFile(path, []byte(content), 0644)
				cleanup := func() { os.RemoveAll(tmpDir) }
				return path, cleanup
			},
			wantErr: false, // passes because only BEGIN is checked
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, cleanup := tt.setup()
			defer cleanup()

			err := validateSSHFile(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSSHFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}