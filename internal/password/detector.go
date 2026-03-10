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

	fileStorage, err := NewFileStorage()
	if err == nil {
		_, err = fileStorage.Get()
		if err == nil {
			return fileStorage, nil
		}
		testErr := fileStorage.Set("__test__")
		if testErr == nil {
			fileStorage.Delete()
			return fileStorage, nil
		}
	}

	return NewMemoryStorage(), nil
}
