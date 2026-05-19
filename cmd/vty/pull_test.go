package main

import (
	"encoding/json"
	"testing"

	"github.com/sthbryan/vaulty/v2/internal/crypto"
	"github.com/sthbryan/vaulty/v2/pkg/models"
)

func TestPullArgsValidation(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid args",
			args:    []string{"env", "api-key"},
			wantErr: false,
		},
		{
			name:    "valid type ssh",
			args:    []string{"ssh", "deploy-key"},
			wantErr: false,
		},
		{
			name:    "valid type config",
			args:    []string{"config", "settings"},
			wantErr: false,
		},
		{
			name:    "valid type resources",
			args:    []string{"resources", "assets"},
			wantErr: false,
		},
		{
			name:    "missing name arg",
			args:    []string{"env"},
			wantErr: true,
			errMsg:  "requires 2 arguments",
		},
		{
			name:    "missing both args",
			args:    []string{},
			wantErr: true,
			errMsg:  "requires 2 arguments",
		},
		{
			name:    "invalid type",
			args:    []string{"invalid", "name"},
			wantErr: true,
			errMsg:  "invalid secret type",
		},
		{
			name:    "empty type",
			args:    []string{"", "name"},
			wantErr: true,
			errMsg:  "invalid secret type",
		},
		{
			name:    "numbers type",
			args:    []string{"123", "name"},
			wantErr: true,
			errMsg:  "invalid secret type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pullCmd.Args(pullCmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("pullCmd.Args() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errMsg != "" {
				if cmdErr, ok := err.(*CommandError); ok {
					if !contains(cmdErr.Message, tt.errMsg) {
						t.Errorf("pullCmd.Args() error message = %v, want contain %v", cmdErr.Message, tt.errMsg)
					}
				} else if !contains(err.Error(), tt.errMsg) {
					t.Errorf("pullCmd.Args() error = %v, want contain %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestTryDecrypt(t *testing.T) {
	masterKey := make([]byte, 32)
	copy(masterKey, []byte("test-master-key-32-bytes-long!!"))

	wrongKey := make([]byte, 32)
	copy(wrongKey, []byte("wrong-master-key-32-bytes-l!"))

	validSecretFile := &models.SecretFile{
		Metadata: models.SecretMetadata{
			Name:  "test-secret",
			Type:  models.SecretTypeEnv,
			Env:   "default",
			IsDir: false,
		},
		Data: []byte("SECRET_VALUE=test"),
	}

	t.Run("valid encrypted data", func(t *testing.T) {
		data, err := json.Marshal(validSecretFile)
		if err != nil {
			t.Fatalf("Failed to marshal secret file: %v", err)
		}

		encryptedData, err := crypto.EncryptWithKey(data, masterKey)
		if err != nil {
			t.Fatalf("EncryptWithKey() error = %v", err)
		}

		serialized := crypto.SerializeEncryptedData(encryptedData)

		result, err := tryDecrypt(serialized, masterKey)
		if err != nil {
			t.Fatalf("tryDecrypt() error = %v", err)
		}

		if result.Metadata.Name != validSecretFile.Metadata.Name {
			t.Errorf("tryDecrypt() Name = %s, want %s", result.Metadata.Name, validSecretFile.Metadata.Name)
		}
		if result.Metadata.Type != validSecretFile.Metadata.Type {
			t.Errorf("tryDecrypt() Type = %s, want %s", result.Metadata.Type, validSecretFile.Metadata.Type)
		}
	})

	t.Run("wrong key", func(t *testing.T) {
		data, err := json.Marshal(validSecretFile)
		if err != nil {
			t.Fatalf("Failed to marshal secret file: %v", err)
		}

		encryptedData, err := crypto.EncryptWithKey(data, masterKey)
		if err != nil {
			t.Fatalf("EncryptWithKey() error = %v", err)
		}

		serialized := crypto.SerializeEncryptedData(encryptedData)

		_, err = tryDecrypt(serialized, wrongKey)
		if err == nil {
			t.Error("tryDecrypt() expected error with wrong key")
		}
	})

	t.Run("invalid data format", func(t *testing.T) {
		invalidData := []byte("not encrypted data at all")

		_, err := tryDecrypt(invalidData, masterKey)
		if err == nil {
			t.Error("tryDecrypt() expected error for invalid data format")
		}
	})

	t.Run("empty data", func(t *testing.T) {
		_, err := tryDecrypt([]byte{}, masterKey)
		if err == nil {
			t.Error("tryDecrypt() expected error for empty data")
		}
	})

	t.Run("truncated encrypted data", func(t *testing.T) {
		data, err := json.Marshal(validSecretFile)
		if err != nil {
			t.Fatalf("Failed to marshal secret file: %v", err)
		}

		encryptedData, err := crypto.EncryptWithKey(data, masterKey)
		if err != nil {
			t.Fatalf("EncryptWithKey() error = %v", err)
		}

		serialized := crypto.SerializeEncryptedData(encryptedData)
		truncated := serialized[:len(serialized)/2]

		_, err = tryDecrypt(truncated, masterKey)
		if err == nil {
			t.Error("tryDecrypt() expected error for truncated data")
		}
	})

	t.Run("valid data with different type", func(t *testing.T) {
		dirSecretFile := &models.SecretFile{
			Metadata: models.SecretMetadata{
				Name:  "backup.tar.gz",
				Type:  models.SecretTypeResources,
				Env:   "production",
				IsDir: true,
			},
			Data: []byte("compressed data"),
		}

		data, err := json.Marshal(dirSecretFile)
		if err != nil {
			t.Fatalf("Failed to marshal secret file: %v", err)
		}

		encryptedData, err := crypto.EncryptWithKey(data, masterKey)
		if err != nil {
			t.Fatalf("EncryptWithKey() error = %v", err)
		}

		serialized := crypto.SerializeEncryptedData(encryptedData)

		result, err := tryDecrypt(serialized, masterKey)
		if err != nil {
			t.Fatalf("tryDecrypt() error = %v", err)
		}

		if !result.Metadata.IsDir {
			t.Error("tryDecrypt() IsDir = false, want true")
		}
		if result.Metadata.Type != models.SecretTypeResources {
			t.Errorf("tryDecrypt() Type = %s, want %s", result.Metadata.Type, models.SecretTypeResources)
		}
	})
}

func TestCommandError(t *testing.T) {
	t.Run("file not found error", func(t *testing.T) {
		err := &CommandError{
			Message: "secret 'nonexistent' not found in env/default",
			Hint:    "Use 'vty show' to check available secrets",
		}

		if err.Message == "" {
			t.Error("CommandError should have a Message")
		}
		if err.Hint == "" {
			t.Error("CommandError should have a Hint")
		}
	})

	t.Run("requires args error", func(t *testing.T) {
		err := &CommandError{
			Message: "requires 2 arguments: <type> <name>",
			Hint:    "Usage: vty pull <type> <name> [-e env] [-o path]",
		}

		if err.Message == "" {
			t.Error("CommandError should have a Message")
		}
	})

	t.Run("invalid type error", func(t *testing.T) {
		err := &CommandError{
			Message: "invalid secret type: invalid",
			Hint:    "Valid types: env, config, ssh, resources",
		}

		if err.Message == "" {
			t.Error("CommandError should have a Message")
		}
		if err.Hint == "" {
			t.Error("CommandError should have a Hint")
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}