package password

import (
	"errors"
	"testing"
)

type fakeStorage struct {
	storageType string
	getErr      error
	setErr      error
	deleteErr   error
}

func (f *fakeStorage) Get() (string, error) {
	if f.getErr != nil {
		return "", f.getErr
	}
	return "ok", nil
}

func (f *fakeStorage) Set(_ string) error {
	return f.setErr
}

func (f *fakeStorage) Delete() error {
	return f.deleteErr
}

func (f *fakeStorage) Type() string {
	return f.storageType
}

func TestNewStorage_KeyringAvailable_ReturnsKeyring(t *testing.T) {
	originalNewKeyringStorage := newKeyringStorage
	originalNewFileStorage := newFileStorage
	defer func() {
		newKeyringStorage = originalNewKeyringStorage
		newFileStorage = originalNewFileStorage
	}()

	newKeyringStorage = func() Storage {
		return &fakeStorage{storageType: "keyring"}
	}
	newFileStorage = func() (Storage, error) {
		t.Fatal("file storage should not be initialized when keyring is available")
		return nil, nil
	}

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	if storage.Type() != "keyring" {
		t.Fatalf("NewStorage() type = %q, want %q", storage.Type(), "keyring")
	}
}

func TestNewStorage_NoKeyringEnv_ReturnsMemory(t *testing.T) {
	originalNewKeyringStorage := newKeyringStorage
	originalNewFileStorage := newFileStorage
	defer func() {
		newKeyringStorage = originalNewKeyringStorage
		newFileStorage = originalNewFileStorage
	}()

	t.Setenv("VAULTY_NO_KEYRING", "1")
	t.Setenv(allowFilePasswordFallbackEnv, "1")

	newKeyringStorage = func() Storage {
		return &fakeStorage{storageType: "keyring", getErr: errors.New("keyring unavailable"), setErr: errors.New("keyring unavailable")}
	}
	newFileStorage = func() (Storage, error) {
		t.Fatal("file storage should not be initialized when VAULTY_NO_KEYRING is set")
		return nil, nil
	}

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	if storage.Type() != "memory" {
		t.Fatalf("NewStorage() type = %q, want %q", storage.Type(), "memory")
	}
}

func TestNewStorage_KeyringUnavailable_NoFileFallbackFlag_ReturnsMemory(t *testing.T) {
	originalNewKeyringStorage := newKeyringStorage
	originalNewFileStorage := newFileStorage
	defer func() {
		newKeyringStorage = originalNewKeyringStorage
		newFileStorage = originalNewFileStorage
	}()

	fileInitCalls := 0
	newKeyringStorage = func() Storage {
		return &fakeStorage{storageType: "keyring", getErr: errors.New("keyring unavailable"), setErr: errors.New("keyring unavailable")}
	}
	newFileStorage = func() (Storage, error) {
		fileInitCalls++
		return &fakeStorage{storageType: "file"}, nil
	}

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	if storage.Type() != "memory" {
		t.Fatalf("NewStorage() type = %q, want %q", storage.Type(), "memory")
	}
	if fileInitCalls != 0 {
		t.Fatalf("file storage init calls = %d, want 0", fileInitCalls)
	}
}

func TestNewStorage_KeyringUnavailable_FileFallbackAllowed_ReturnsFile(t *testing.T) {
	originalNewKeyringStorage := newKeyringStorage
	originalNewFileStorage := newFileStorage
	defer func() {
		newKeyringStorage = originalNewKeyringStorage
		newFileStorage = originalNewFileStorage
	}()

	t.Setenv(allowFilePasswordFallbackEnv, "1")

	newKeyringStorage = func() Storage {
		return &fakeStorage{storageType: "keyring", getErr: errors.New("keyring unavailable"), setErr: errors.New("keyring unavailable")}
	}
	newFileStorage = func() (Storage, error) {
		return &fakeStorage{storageType: "file", getErr: errors.New("password not found")}, nil
	}

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	if storage.Type() != "file" {
		t.Fatalf("NewStorage() type = %q, want %q", storage.Type(), "file")
	}
}

func TestNewStorage_KeyringUnavailable_FileFallbackAllowedButUnusable_ReturnsMemory(t *testing.T) {
	originalNewKeyringStorage := newKeyringStorage
	originalNewFileStorage := newFileStorage
	defer func() {
		newKeyringStorage = originalNewKeyringStorage
		newFileStorage = originalNewFileStorage
	}()

	t.Setenv(allowFilePasswordFallbackEnv, "1")

	newKeyringStorage = func() Storage {
		return &fakeStorage{storageType: "keyring", getErr: errors.New("keyring unavailable"), setErr: errors.New("keyring unavailable")}
	}
	newFileStorage = func() (Storage, error) {
		return &fakeStorage{storageType: "file", getErr: errors.New("password not found"), setErr: errors.New("cannot write")}, nil
	}

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	if storage.Type() != "memory" {
		t.Fatalf("NewStorage() type = %q, want %q", storage.Type(), "memory")
	}
}
