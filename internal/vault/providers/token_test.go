package providers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sthbryan/vaulty/v2/internal/ui"
)

func TestGetTokenForProvider_Local(t *testing.T) {
	token, err := GetTokenForProvider("local")
	if err != nil {
		t.Fatalf("GetTokenForProvider(\"local\") error = %v", err)
	}
	if token != "" {
		t.Errorf("GetTokenForProvider(\"local\") = %q, want empty string", token)
	}
}

func TestTokenManager_SavesAuthMethod(t *testing.T) {

	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	os.MkdirAll(homeDir, 0755)
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", originalHome)

	originalConfigPath := configPath
	configPath = func() string {
		return filepath.Join(homeDir, ".vaulty", "config.yaml")
	}
	defer func() {
		configPath = originalConfigPath
	}()

	origSelect := uiSelect
	origPassword := uiPassword

	methodSelected := ""
	uiSelect = func(title string, options []ui.SelectOption) (string, error) {
		for _, opt := range options {
			if opt.ID == "manual" {
				methodSelected = opt.ID
				return opt.ID, nil
			}
		}
		return "", nil
	}

	passwordCallCount := 0
	uiPassword = func(title, placeholder string) (string, error) {
		passwordCallCount++
		if passwordCallCount == 1 {

			return "ghp_test123456789", nil
		}

		return "masterpass123", nil
	}
	defer func() {
		uiSelect = origSelect
		uiPassword = origPassword
	}()

	tm := NewTokenManager("github")
	token, err := tm.GetToken()
	if err != nil {
		t.Fatalf("TokenManager.GetToken() error = %v", err)
	}
	if token == "" {
		t.Error("TokenManager.GetToken() returned empty token")
	}
	if methodSelected != "manual" {
		t.Errorf("Expected method 'manual', got %q", methodSelected)
	}

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if config.Auth.Method != "manual" {
		t.Errorf("Config.Auth.Method = %q, want 'manual'", config.Auth.Method)
	}
	if config.Auth.Provider != "github" {
		t.Errorf("Config.Auth.Provider = %q, want 'github'", config.Auth.Provider)
	}
	if config.Auth.EncryptedToken == "" {
		t.Error("Config.Auth.EncryptedToken should not be empty")
	}
}

func TestTokenManager_RetrievesSavedMethod(t *testing.T) {

	tempDir := t.TempDir()
	homeDir := filepath.Join(tempDir, "home")
	os.MkdirAll(homeDir, 0755)
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", originalHome)

	originalConfigPath := configPath
	configPath = func() string {
		return filepath.Join(homeDir, ".vaulty", "config.yaml")
	}
	defer func() {
		configPath = originalConfigPath
	}()

	origSelect := uiSelect
	origPassword := uiPassword

	var validEncryptedToken string
	var getTokenDone bool

	uiPassword = func(title, placeholder string) (string, error) {
		if !getTokenDone {
			getTokenDone = true
			return "testtoken123", nil
		}
		return "masterpass123", nil
	}

	uiSelect = func(title string, options []ui.SelectOption) (string, error) {
		for _, opt := range options {
			if opt.ID == "manual" {
				return opt.ID, nil
			}
		}
		return "", nil
	}

	tm := NewTokenManager("github")
	_, err := tm.GetToken()
	if err != nil {
		t.Fatalf("Failed to get initial encrypted token: %v", err)
	}

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	validEncryptedToken = config.Auth.EncryptedToken

	uiSelect = origSelect
	uiPassword = origPassword

	err = UpdateAuthSettings(ProviderType("github"), "manual", validEncryptedToken)
	if err != nil {
		t.Fatalf("Failed to update auth settings: %v", err)
	}

	uiPassword = func(title, placeholder string) (string, error) {
		return "masterpass123", nil
	}

	selectCalled := false
	uiSelect = func(title string, options []ui.SelectOption) (string, error) {
		selectCalled = true
		return "", nil
	}
	defer func() {
		uiSelect = origSelect
		uiPassword = origPassword
	}()

	tm2 := NewTokenManager("github")
	token, err := tm2.GetToken()
	if err != nil {
		t.Fatalf("TokenManager.GetToken() error = %v", err)
	}

	if selectCalled {
		t.Error("ui.Select should not be called when saved method exists and decrypts successfully")
	}

	if token != "testtoken123" {
		t.Errorf("Token = %q, want %q", token, "testtoken123")
	}
}
