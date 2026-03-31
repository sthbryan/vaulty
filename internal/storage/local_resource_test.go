package storage

import (
	"context"
	"strings"
	"testing"
)

func TestLocalResourceStorage_PathTraversalRejected(t *testing.T) {
	t.Parallel()

	store, err := NewLocalResourceStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalResourceStorage() error = %v", err)
	}

	ctx := context.Background()
	paths := []string{
		"../outside.vty",
		"../../etc/passwd",
		"/tmp/absolute.vty",
	}

	for _, path := range paths {
		path := path
		t.Run(path, func(t *testing.T) {
			t.Parallel()

			if err := store.PutResource(ctx, path, []byte("secret")); err == nil {
				t.Fatalf("PutResource(%q) expected error, got nil", path)
			}

			if _, err := store.GetResource(ctx, path); err == nil {
				t.Fatalf("GetResource(%q) expected error, got nil", path)
			}

			if err := store.DeleteResource(ctx, path); err == nil {
				t.Fatalf("DeleteResource(%q) expected error, got nil", path)
			}
		})
	}
}

func TestLocalResourceStorage_ValidPathStillWorks(t *testing.T) {
	t.Parallel()

	store, err := NewLocalResourceStorage(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocalResourceStorage() error = %v", err)
	}

	ctx := context.Background()
	resourcePath := "resources/app/config.vty"
	data := []byte("ok")

	if err := store.PutResource(ctx, resourcePath, data); err != nil {
		t.Fatalf("PutResource() error = %v", err)
	}

	got, err := store.GetResource(ctx, resourcePath)
	if err != nil {
		t.Fatalf("GetResource() error = %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("GetResource() = %q, want %q", got, data)
	}

	if err := store.DeleteResource(ctx, resourcePath); err != nil {
		t.Fatalf("DeleteResource() error = %v", err)
	}

	_, err = store.GetResource(ctx, resourcePath)
	if err == nil {
		t.Fatal("expected not found error after delete")
	}
	if !strings.Contains(err.Error(), "resource not found") {
		t.Fatalf("expected resource not found error, got %v", err)
	}
}
