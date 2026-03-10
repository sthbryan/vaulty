package password

import (
	"os"
)

func NewStorage() (Storage, error) {
	keyringStorage := NewKeyringStorage()

	_, err := keyringStorage.Get()
	if err == nil {
		return keyringStorage, nil
	}

	if os.Getenv("VAULTY_NO_KEYRING") != "" {
		return NewMemoryStorage(), nil
	}

	testErr := keyringStorage.Set("__test__")
	if testErr == nil {
		keyringStorage.Delete()
		return keyringStorage, nil
	}

	return NewMemoryStorage(), nil
}
