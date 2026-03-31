package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalEnvStorage_ListEnvSecrets_UsesRequestedEnvSubdirectory(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	store, err := NewLocalEnvStorage(baseDir)
	if err != nil {
		t.Fatalf("NewLocalEnvStorage() error = %v", err)
	}

	mustWrite := func(path string, data []byte) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", path, err)
		}
	}

	mustWrite(filepath.Join(baseDir, "envs", "dev", "API_KEY.vty"), []byte("dev-api"))
	mustWrite(filepath.Join(baseDir, "envs", "dev", "DB_URL.vty"), []byte("dev-db"))
	mustWrite(filepath.Join(baseDir, "envs", "prod", "API_KEY.vty"), []byte("prod-api"))
	mustWrite(filepath.Join(baseDir, "envs", "API_KEY.vty"), []byte("root-api"))

	secrets, err := store.ListEnvSecrets(context.Background(), "dev")
	if err != nil {
		t.Fatalf("ListEnvSecrets() error = %v", err)
	}

	if len(secrets) != 2 {
		t.Fatalf("ListEnvSecrets() len = %d, want 2", len(secrets))
	}

	got := map[string]int64{}
	for _, secret := range secrets {
		got[secret.Name] = secret.Size
	}

	if got["API_KEY"] != int64(len("dev-api")) {
		t.Fatalf("API_KEY size = %d, want %d", got["API_KEY"], len("dev-api"))
	}
	if got["DB_URL"] != int64(len("dev-db")) {
		t.Fatalf("DB_URL size = %d, want %d", got["DB_URL"], len("dev-db"))
	}
	if _, ok := got["prod/API_KEY"]; ok {
		t.Fatalf("unexpected prod secret listed: %+v", secrets)
	}
}
