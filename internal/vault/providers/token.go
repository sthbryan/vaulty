package providers

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sthbryan/vaulty/v2/internal/crypto"
	"github.com/sthbryan/vaulty/v2/internal/ui"
)

type TokenManager struct {
	provider string
}

func NewTokenManager(provider string) *TokenManager {
	return &TokenManager{provider: provider}
}

var (
	uiSelect   = ui.Select
	uiPassword = ui.Password
)

func GetTokenForProvider(provider string) (string, error) {
	if provider == "local" {
		return "", nil
	}
	return NewTokenManager(provider).GetToken()
}

func (tm *TokenManager) GetToken() (string, error) {
	var method string
	var encryptedToken string

	config, err := LoadConfig()
	if err == nil && config.Auth.Method != "" {
		token, err := tm.getTokenByMethod(config.Auth.Method, config.Auth.EncryptedToken)
		if err == nil {
			return token, nil
		}
		ui.PrintError(fmt.Sprintf("Token retrieval failed: %v", err))
		ui.PrintInfo("Please re-authenticate...")
	}

	method, err = uiSelect(tm.providerTitle()+" authentication method", []ui.SelectOption{
		{ID: "cli", Label: tm.providerTitle() + " CLI - recommended"},
		{ID: "env", Label: tm.providerEnvVar() + " environment variable"},
		{ID: "manual", Label: "Enter token manually (will be saved encrypted)"},
	})
	if err != nil {
		return "", fmt.Errorf("Cancelled")
	}

	var token string
	switch method {
	case "cli":
		token, err = tm.getTokenFromCLI()
	case "env":
		token, err = tm.getTokenFromEnv()
	case "manual":
		token, err = tm.saveNewToken()
		encryptedToken = token
	}
	if err != nil {
		return "", err
	}

	tm.saveAuthMethod(method, encryptedToken)

	return token, nil
}

func (tm *TokenManager) getTokenByMethod(method string, encryptedToken string) (string, error) {
	switch method {
	case "cli":
		return tm.getTokenFromCLI()
	case "env":
		return tm.getTokenFromEnv()
	case "manual":
		if encryptedToken == "" {
			return "", errors.New("no token saved, please provide master password")
		}
		return tm.decryptToken(encryptedToken)
	default:
		return "", fmt.Errorf("unsupported auth method: %s", method)
	}
}

func (tm *TokenManager) getTokenFromCLI() (string, error) {
	var cmd *exec.Cmd
	switch tm.provider {
	case "github":
		cmd = exec.Command("gh", "auth", "token")
	case "gitlab":
		cmd = exec.Command("glab", "auth", "token")
	default:
		return "", fmt.Errorf("unsupported provider: %s", tm.provider)
	}

	output, err := cmd.Output()
	if err != nil || len(output) == 0 {
		return "", fmt.Errorf("%s CLI not available or not authenticated", tm.provider)
	}
	return strings.TrimSpace(string(output)), nil
}

func (tm *TokenManager) getTokenFromEnv() (string, error) {
	var envVar string
	switch tm.provider {
	case "github":
		envVar = "GITHUB_TOKEN"
	case "gitlab":
		envVar = "GLAB_TOKEN"
	default:
		return "", fmt.Errorf("unsupported provider: %s", tm.provider)
	}

	token := os.Getenv(envVar)
	if token == "" {
		return "", fmt.Errorf("%s environment variable not set", envVar)
	}
	return token, nil
}

func (tm *TokenManager) saveNewToken() (string, error) {
	token, err := uiPassword(tm.providerTitle()+" personal access token", "ghp_••••••••")
	if err != nil {
		return "", fmt.Errorf("Cancelled")
	}

	encrypted, err := tm.encryptTokenForManual(token)
	if err != nil {
		ui.PrintError(fmt.Sprintf("Failed to encrypt token: %v", err))
	}

	return encrypted, nil
}

func (tm *TokenManager) decryptToken(encryptedToken string) (string, error) {
	password, err := uiPassword("Master password to decrypt saved token", "••••••••")
	if err != nil {
		return "", fmt.Errorf("Cancelled")
	}

	data, err := base64.StdEncoding.DecodeString(encryptedToken)
	if err != nil {
		return "", fmt.Errorf("corrupted encrypted token: %w", err)
	}

	parts := strings.SplitN(string(data), ":", 3)
	if len(parts) != 3 {
		return "", errors.New("invalid encrypted token format")
	}

	salt, err := hexToBytes(parts[0])
	if err != nil {
		return "", fmt.Errorf("invalid salt: %w", err)
	}

	iv, err := hexToBytes(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid IV: %w", err)
	}

	ciphertext, err := hexToBytes(parts[2])
	if err != nil {
		return "", fmt.Errorf("invalid ciphertext: %w", err)
	}

	derivedKey, err := crypto.DeriveKey(password, salt)
	if err != nil {
		return "", fmt.Errorf("key derivation failed: %w", err)
	}

	encryptedData := &crypto.EncryptedData{
		Salt:       salt,
		IV:         iv,
		Ciphertext: ciphertext,
	}

	plaintext, err := crypto.DecryptWithKey(encryptedData, derivedKey)
	if err != nil {
		return "", errors.New("invalid password or corrupted token")
	}

	return string(plaintext), nil
}

func (tm *TokenManager) encryptTokenForManual(token string) (string, error) {
	password, err := uiPassword("Master password for encrypting token", "••••••••")
	if err != nil {
		return "", fmt.Errorf("Cancelled")
	}

	encrypted, err := crypto.Encrypt([]byte(token), password)
	if err != nil {
		return "", fmt.Errorf("encryption failed: %w", err)
	}

	format := fmt.Sprintf("%x:%x:%x", encrypted.Salt, encrypted.IV, encrypted.Ciphertext)

	return base64.StdEncoding.EncodeToString([]byte(format)), nil
}

func (tm *TokenManager) saveAuthMethod(method, encryptedToken string) {
	if err := UpdateAuthSettings(ProviderType(tm.provider), method, encryptedToken); err != nil {
		ui.PrintError(fmt.Sprintf("Failed to save auth config: %v", err))
	}
}

func (tm *TokenManager) providerTitle() string {
	switch tm.provider {
	case "github":
		return "GitHub"
	case "gitlab":
		return "GitLab"
	default:
		return tm.provider
	}
}

func (tm *TokenManager) providerEnvVar() string {
	switch tm.provider {
	case "github":
		return "GITHUB_TOKEN"
	case "gitlab":
		return "GLAB_TOKEN"
	default:
		return strings.ToUpper(tm.provider) + "_TOKEN"
	}
}

func hexToBytes(s string) ([]byte, error) {
	var result []byte
	for i := 0; i < len(s); i += 2 {
		if i+2 > len(s) {
			return nil, errors.New("invalid hex length")
		}
		var b byte
		for _, c := range s[i : i+2] {
			cc := byte(c)
			b <<= 4
			switch {
			case cc >= '0' && cc <= '9':
				b |= cc - '0'
			case cc >= 'a' && cc <= 'f':
				b |= cc - 'a' + 10
			case cc >= 'A' && cc <= 'F':
				b |= cc - 'A' + 10
			default:
				return nil, errors.New("invalid hex character")
			}
		}
		result = append(result, b)
	}
	return result, nil
}
