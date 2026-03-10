package crypto

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

var (
	ErrInvalidMasterKeySize = errors.New("master key must be 32 bytes")
)

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

func EncryptRecoverySeed(seed string, password string) (*EncryptedData, error) {
	return Encrypt([]byte(seed), password)
}

func DecryptRecoverySeed(data *EncryptedData, password string) (string, error) {
	seedBytes, err := Decrypt(data, password)
	if err != nil {
		return "", err
	}
	return string(seedBytes), nil
}
