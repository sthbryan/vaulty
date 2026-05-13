package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/sthbryan/vaulty/v2/internal/crypto"
	"github.com/sthbryan/vaulty/v2/pkg/models"
)

const (
	DefaultSessionDuration = 8 * 60 * 60
)

type Auth struct{}

func New() *Auth {
	return &Auth{}
}

func (a *Auth) GenerateSalt() (string, error) {
	salt, err := crypto.GenerateSalt()
	if err != nil {
		return "", fmt.Errorf("generating salt: %w", err)
	}
	return hex.EncodeToString(salt), nil
}

func (a *Auth) DeriveKey(password string, saltHex string) ([]byte, error) {
	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return nil, fmt.Errorf("decoding salt: %w", err)
	}

	key, err := crypto.DeriveKey(password, salt)
	if err != nil {
		return nil, fmt.Errorf("deriving key: %w", err)
	}

	return key, nil
}

func (a *Auth) EncryptVaultKey(vaultKey []byte, password string) (string, string, error) {

	salt, err := crypto.GenerateSalt()
	if err != nil {
		return "", "", fmt.Errorf("generating salt: %w", err)
	}

	derivedKey, err := crypto.DeriveKey(password, salt)
	if err != nil {
		return "", "", fmt.Errorf("deriving key: %w", err)
	}

	encrypted, err := crypto.EncryptWithKey(vaultKey, derivedKey)
	if err != nil {
		return "", "", fmt.Errorf("encrypting vault key: %w", err)
	}

	iv := hex.EncodeToString(encrypted.IV)
	ciphertext := hex.EncodeToString(encrypted.Ciphertext)

	return hex.EncodeToString(salt) + ":" + iv + ":" + ciphertext, hex.EncodeToString(salt), nil
}

func (a *Auth) DecryptVaultKey(encryptedKeyHex string, password string) ([]byte, error) {
	parts := splitEncryptedKey(encryptedKeyHex)
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid encrypted key format")
	}

	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decoding salt: %w", err)
	}

	iv, err := hex.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decoding iv: %w", err)
	}

	ciphertext, err := hex.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("decoding ciphertext: %w", err)
	}

	derivedKey, err := crypto.DeriveKey(password, salt)
	if err != nil {
		return nil, fmt.Errorf("deriving key: %w", err)
	}

	encryptedData := &crypto.EncryptedData{
		IV:         iv,
		Ciphertext: ciphertext,
	}

	vaultKey, err := crypto.DecryptWithKey(encryptedData, derivedKey)
	if err != nil {
		return nil, fmt.Errorf("decrypting vault key: %w", err)
	}

	return vaultKey, nil
}

func (a *Auth) ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	return nil
}

func (a *Auth) GenerateSessionDuration(duration string) (int64, error) {
	switch duration {
	case "8h":
		return 8 * 60 * 60, nil
	case "24h":
		return 24 * 60 * 60, nil
	case "7d":
		return 7 * 24 * 60 * 60, nil
	case "30d":
		return 30 * 24 * 60 * 60, nil
	default:
		return DefaultSessionDuration, nil
	}
}

func (a *Auth) CreateSession(username, vaultID, storageType string, expiresInSeconds int64) *models.Session {
	return &models.Session{
		Username:    username,
		VaultID:     vaultID,
		StorageType: storageType,
	}
}

func (a *Auth) GenerateVaultKey() ([]byte, error) {
	key := make([]byte, crypto.KeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generating vault key: %w", err)
	}
	return key, nil
}

func splitEncryptedKey(encrypted string) []string {
	var parts []string
	current := ""
	for _, c := range encrypted {
		if c == ':' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
