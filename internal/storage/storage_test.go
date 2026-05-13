package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sthbryan/vaulty/v2/pkg/models"
)

func TestLocalStorage_Ping(t *testing.T) {
	storage := NewLocalStorage(t.TempDir())
	err := storage.Ping(context.Background())
	if err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

func TestLocalStorage_UploadDownload(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewLocalStorage(tmpDir)
	ctx := context.Background()

	path := "test/file.txt"
	data := []byte("hello world")

	if err := storage.Upload(ctx, path, data); err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	downloaded, err := storage.Download(ctx, path)
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	if string(downloaded) != string(data) {
		t.Errorf("Download() = %s, want %s", string(downloaded), string(data))
	}
}

func TestLocalStorage_DownloadNotFound(t *testing.T) {
	storage := NewLocalStorage(t.TempDir())
	ctx := context.Background()

	_, err := storage.Download(ctx, "nonexistent.txt")
	if err == nil {
		t.Error("Download() should error for nonexistent file")
	}
}

func TestLocalStorage_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewLocalStorage(tmpDir)
	ctx := context.Background()

	path := "test/file.txt"
	_ = storage.Upload(ctx, path, []byte("data"))

	if err := storage.Delete(ctx, path); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := storage.Download(ctx, path)
	if err == nil {
		t.Error("Delete() should make file not found")
	}
}

func TestLocalStorage_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	storage := NewLocalStorage(tmpDir)
	ctx := context.Background()

	path := "test/file.txt"
	_ = storage.Upload(ctx, path, []byte("data"))

	exists, err := storage.Exists(ctx, path)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Exists() = false, want true")
	}

	exists, _ = storage.Exists(ctx, "nonexistent.txt")
	if exists {
		t.Error("Exists() = true for nonexistent file, want false")
	}
}

func TestBuildSecretPath(t *testing.T) {
	tests := []struct {
		name     string
		secretType models.SecretType
		env      string
		fileName string
		want     string
	}{
		{"env default", models.SecretTypeEnv, "default", "api", "env/default/api.vty"},
		{"env production", models.SecretTypeEnv, "production", "api", "env/production/api.vty"},
		{"ssh default", models.SecretTypeSSH, "default", "github", "ssh/default/github.vty"},
		{"config production", models.SecretTypeConfig, "production", "db", "config/production/db.vty"},
		{"resources default", models.SecretTypeResources, "default", "key", "resources/default/key.vty"},
		{"empty env defaults to default", models.SecretTypeEnv, "", "api", "env/default/api.vty"},
		{"already has .vty", models.SecretTypeEnv, "default", "api.vty", "env/default/api.vty"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildSecretPath(tt.secretType, tt.env, tt.fileName)
			if got != tt.want {
				t.Errorf("BuildSecretPath() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestParseSecretPath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantType   models.SecretType
		wantEnv    string
		wantName   string
		wantErr    bool
	}{
		{"valid env", "env/default/api.vty", models.SecretTypeEnv, "default", "api", false},
		{"valid ssh", "ssh/production/github.vty", models.SecretTypeSSH, "production", "github", false},
		{"invalid type", "invalid/default/api.vty", "", "", "", true},
		{"too short", "env.vty", "", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secretType, env, name, err := ParseSecretPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSecretPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if secretType != tt.wantType {
					t.Errorf("secretType = %s, want %s", secretType, tt.wantType)
				}
				if env != tt.wantEnv {
					t.Errorf("env = %s, want %s", env, tt.wantEnv)
				}
				if name != tt.wantName {
					t.Errorf("name = %s, want %s", name, tt.wantName)
				}
			}
		})
	}
}

func TestComputeChecksum(t *testing.T) {
	data := []byte("hello world")

	checksum := ComputeChecksum(data)
	if len(checksum) != 64 {
		t.Errorf("ComputeChecksum() length = %d, want 64", len(checksum))
	}

	checksum2 := ComputeChecksum(data)
	if checksum != checksum2 {
		t.Error("ComputeChecksum() not deterministic")
	}

	differentChecksum := ComputeChecksum([]byte("different"))
	if checksum == differentChecksum {
		t.Error("ComputeChecksum() same for different data")
	}
}

func TestCompressDecompressDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	dirPath := filepath.Join(tmpDir, "testdir")

	if err := os.MkdirAll(filepath.Join(dirPath, "subdir"), 0755); err != nil {
		t.Fatalf("creating test directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dirPath, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("creating test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dirPath, "subdir", "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatalf("creating nested test file: %v", err)
	}

	compressed, err := CompressDirectory(dirPath)
	if err != nil {
		t.Fatalf("CompressDirectory() error = %v", err)
	}

	if len(compressed) == 0 {
		t.Error("CompressDirectory() returned empty data")
	}

	destDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		t.Fatalf("creating dest directory: %v", err)
	}

	if err := DecompressDirectory(compressed, destDir); err != nil {
		t.Fatalf("DecompressDirectory() error = %v", err)
	}

	files := []string{}
	filepath.Walk(destDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			rel, _ := filepath.Rel(destDir, path)
			files = append(files, rel)
		}
		return nil
	})

	found := false
	for _, f := range files {
		if strings.HasSuffix(f, "file1.txt") {
			data, err := os.ReadFile(filepath.Join(destDir, f))
			if err != nil {
				t.Fatalf("reading file: %v", err)
			}
			if string(data) != "content1" {
				t.Errorf("file1 content = %s, want content1", string(data))
			}
			found = true
		}
	}
	if !found {
		t.Errorf("file1.txt not found in extracted files: %v", files)
	}

	found = false
	for _, f := range files {
		if strings.HasSuffix(f, "file2.txt") {
			data, err := os.ReadFile(filepath.Join(destDir, f))
			if err != nil {
				t.Fatalf("reading file: %v", err)
			}
			if string(data) != "content2" {
				t.Errorf("file2 content = %s, want content2", string(data))
			}
			found = true
		}
	}
	if !found {
		t.Errorf("file2.txt not found in extracted files: %v", files)
	}
}