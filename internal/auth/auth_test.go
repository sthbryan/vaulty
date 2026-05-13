package auth

import (
	"testing"
)

func TestAuth_GenerateSalt(t *testing.T) {
	a := New()

	salt1, err := a.GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() error = %v", err)
	}

	if len(salt1) != 64 {
		t.Errorf("GenerateSalt() length = %d, want 64", len(salt1))
	}

	salt2, _ := a.GenerateSalt()
	if salt1 == salt2 {
		t.Error("GenerateSalt() returned same salt twice")
	}
}

func TestAuth_DeriveKey(t *testing.T) {
	a := New()

	salt, _ := a.GenerateSalt()

	key1, err := a.DeriveKey("testpassword", salt)
	if err != nil {
		t.Fatalf("DeriveKey() error = %v", err)
	}

	if len(key1) != 32 {
		t.Errorf("DeriveKey() length = %d, want 32", len(key1))
	}

	key2, _ := a.DeriveKey("testpassword", salt)
	if string(key1) != string(key2) {
		t.Error("DeriveKey() not deterministic")
	}

	key3, _ := a.DeriveKey("differentpassword", salt)
	if string(key1) == string(key3) {
		t.Error("DeriveKey() same key for different password")
	}
}

func TestAuth_DeriveKey_InvalidSalt(t *testing.T) {
	a := New()

	_, err := a.DeriveKey("password", "invalid-hex")
	if err == nil {
		t.Error("DeriveKey() expected error for invalid salt")
	}
}

func TestAuth_ValidatePassword(t *testing.T) {
	a := New()

	tests := []struct {
		name    string
		password string
		wantErr bool
	}{
		{"valid 8 chars", "password1", false},
		{"valid long", "thisisavalidpassword", false},
		{"too short", "pass", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := a.ValidatePassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuth_GenerateSessionDuration(t *testing.T) {
	a := New()

	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"8h", "8h", 8 * 60 * 60},
		{"24h", "24h", 24 * 60 * 60},
		{"7d", "7d", 7 * 24 * 60 * 60},
		{"30d", "30d", 30 * 24 * 60 * 60},
		{"invalid defaults to 8h", "invalid", 8 * 60 * 60},
		{"empty defaults to 8h", "", 8 * 60 * 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, err := a.GenerateSessionDuration(tt.input)
			if err != nil {
				t.Errorf("GenerateSessionDuration() error = %v", err)
			}
			if duration != tt.expected {
				t.Errorf("GenerateSessionDuration() = %d, want %d", duration, tt.expected)
			}
		})
	}
}

func TestAuth_GenerateVaultKey(t *testing.T) {
	a := New()

	key1, err := a.GenerateVaultKey()
	if err != nil {
		t.Fatalf("GenerateVaultKey() error = %v", err)
	}

	if len(key1) != 32 {
		t.Errorf("GenerateVaultKey() length = %d, want 32", len(key1))
	}

	key2, _ := a.GenerateVaultKey()
	if string(key1) == string(key2) {
		t.Error("GenerateVaultKey() returned same key twice")
	}
}