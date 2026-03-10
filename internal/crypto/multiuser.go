package crypto

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
)

var (
	ErrInvalidMasterKeySize = errors.New("master key must be 32 bytes")
	ErrInvalidRecoverySeed  = errors.New("recovery seed must be exactly 12 words")
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

func GenerateRecoverySeed() (string, error) {
	words := []string{
		"abandon", "ability", "able", "about", "above", "absent", "absorb", "abstract",
		"academy", "accent", "accept", "access", "accident", "account", "accuse", "achieve",
		"acid", "acoustic", "acquire", "across", "act", "action", "actor", "actual",
		"acumen", "acute", "ad", "add", "addict", "added", "adder", "addicted",
		"adding", "addition", "additive", "address", "adds", "adept", "adequate", "adieu",
		"admin", "admire", "admit", "adobe", "adopt", "adore", "adorn", "adult",
		"advance", "advent", "adverb", "adverse", "advertise", "advice", "advise", "advocate",
		"aegis", "aeon", "aerial", "affair", "afford", "afraid", "africa", "after",
		"after", "again", "against", "age", "agency", "agent", "ages", "agile",
	}

	selected := make([]string, 12)
	for i := 0; i < 12; i++ {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(words))))
		if err != nil {
			return "", fmt.Errorf("generating random word: %w", err)
		}
		selected[i] = words[idx.Int64()]
	}

	return strings.Join(selected, " "), nil
}

func ValidateRecoverySeed(seed string) (string, error) {
	seed = strings.TrimSpace(strings.ToLower(seed))
	words := strings.Fields(seed)

	if len(words) != 12 {
		return "", fmt.Errorf("seed must contain 12 words, got %d", len(words))
	}

	return seed, nil
}

func ValidateOwnerPassword(ownerPassword, challenge string) error {
	if !ValidatePasswordChallenge(ownerPassword, challenge) {
		return errors.New("invalid owner password")
	}
	return nil
}
