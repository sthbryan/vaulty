package password

import (
	"os"
)

const allowFilePasswordFallbackEnv = "VAULTY_ALLOW_FILE_PASSWORD_FALLBACK"

var (
	newKeyringStorage = func() Storage { return NewKeyringStorage() }
	newFileStorage    = func() (Storage, error) { return NewFileStorage() }
)

func NewStorage() (Storage, error) {
	keyringStorage := newKeyringStorage()

	_, err := keyringStorage.Get()
	if err == nil {
		return keyringStorage, nil
	}

	if os.Getenv("VAULTY_NO_KEYRING") != "" {
		return NewMemoryStorage(), nil
	}

	testErr := keyringStorage.Set("__test__")
	if testErr == nil {
		_ = keyringStorage.Delete()
		return keyringStorage, nil
	}

	if os.Getenv(allowFilePasswordFallbackEnv) == "1" {
		fileStorage, err := newFileStorage()
		if err == nil {
			_, err = fileStorage.Get()
			if err == nil {
				return fileStorage, nil
			}
			testErr := fileStorage.Set("__test__")
			if testErr == nil {
				_ = fileStorage.Delete()
				return fileStorage, nil
			}
		}
	}

	return NewMemoryStorage(), nil
}
