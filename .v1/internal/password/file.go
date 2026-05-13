package password

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const fileName = ".vty"

type FileStorage struct {
	path string
}

func NewFileStorage() (*FileStorage, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config dir: %w", err)
	}

	vaultyDir := filepath.Join(configDir, "vaulty")
	if err := os.MkdirAll(vaultyDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create vaulty dir: %w", err)
	}

	return &FileStorage{
		path: filepath.Join(vaultyDir, fileName),
	}, nil
}

func (f *FileStorage) Get() (string, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("password not found in file")
		}
		return "", fmt.Errorf("failed to read password file: %w", err)
	}

	var payload struct {
		Password string `json:"password"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", fmt.Errorf("failed to parse password file: %w", err)
	}

	return payload.Password, nil
}

func (f *FileStorage) Set(password string) error {
	payload := struct {
		Password string `json:"password"`
	}{
		Password: password,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal password: %w", err)
	}

	if err := os.WriteFile(f.path, data, 0600); err != nil {
		return fmt.Errorf("failed to write password file: %w", err)
	}

	return nil
}

func (f *FileStorage) Delete() error {
	if err := os.Remove(f.path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to delete password file: %w", err)
	}
	return nil
}

func (f *FileStorage) Type() string {
	return "file"
}
