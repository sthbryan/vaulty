package crypto

import (
	"crypto/rand"
	"errors"
	"io"

	"github.com/tyler-smith/go-bip39"
)

const canaryText = "vaulty-check"

func GenerateCanary(password string, deviceSalt []byte) ([]byte, error) {
	compositePassword := password + string(deviceSalt)
	encrypted, err := Encrypt([]byte(canaryText), compositePassword)
	if err != nil {
		return nil, err
	}
	return SerializeEncryptedData(encrypted), nil
}

func ValidateCanary(data []byte, password string, deviceSalt []byte) error {
	encrypted, err := DeserializeEncryptedData(data)
	if err != nil {
		return err
	}
	compositePassword := password + string(deviceSalt)
	plaintext, err := Decrypt(encrypted, compositePassword)
	if err != nil {
		return err
	}
	if string(plaintext) != canaryText {
		return errors.New("invalid canary content")
	}
	return nil
}

func GenerateRecoverySeed() (string, error) {
	entropy := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, entropy); err != nil {
		return "", err
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", err
	}
	return mnemonic, nil
}

func ValidateRecoverySeed(seed string) ([]byte, error) {
	if !bip39.IsMnemonicValid(seed) {
		return nil, errors.New("invalid mnemonic")
	}
	return bip39.EntropyFromMnemonic(seed)
}
