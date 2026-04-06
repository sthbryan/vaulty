package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
)

var ErrInvalidMasterKeySize = errors.New("master key must be 32 bytes")

const (
	MasterKeySize = 32
)

func GenerateMasterKey() ([]byte, error) {
	masterKey := make([]byte, MasterKeySize)
	if _, err := io.ReadFull(rand.Reader, masterKey); err != nil {
		return nil, fmt.Errorf("failed to generate master key: %w", err)
	}
	return masterKey, nil
}

func ValidateMasterKey(key []byte) error {
	if len(key) != MasterKeySize {
		return ErrInvalidMasterKeySize
	}
	return nil
}

func EncryptMasterKeyWithPassword(masterKey []byte, password string) (*EncryptedData, error) {
	if err := ValidateMasterKey(masterKey); err != nil {
		return nil, err
	}

	return Encrypt(masterKey, password)
}

func DecryptMasterKeyWithPassword(data *EncryptedData, password string) ([]byte, error) {
	masterKey, err := Decrypt(data, password)
	if err != nil {
		return nil, err
	}

	if err := ValidateMasterKey(masterKey); err != nil {
		return nil, err
	}

	return masterKey, nil
}

func EncryptVaultData(plaintext []byte, masterKey []byte) (*EncryptedData, error) {
	if err := ValidateMasterKey(masterKey); err != nil {
		return nil, err
	}

	return EncryptWithKey(plaintext, masterKey)
}

func DecryptVaultData(data *EncryptedData, masterKey []byte) ([]byte, error) {
	if err := ValidateMasterKey(masterKey); err != nil {
		return nil, err
	}

	return DecryptWithKey(data, masterKey)
}


func ValidatePasswordWithChallenge(password string, salt, challenge []byte) bool {
	if len(salt) == 0 || len(challenge) == 0 {
		return false
	}

	h := hmac.New(sha256.New, []byte(password))
	h.Write(salt)
	computed := h.Sum(nil)

	return hmac.Equal(computed, challenge)
}

func GeneratePasswordChallengeStruct(password string) ([]byte, []byte, error) {
	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	h := hmac.New(sha256.New, []byte(password))
	h.Write(salt)
	challenge := h.Sum(nil)

	return salt, challenge, nil
}
