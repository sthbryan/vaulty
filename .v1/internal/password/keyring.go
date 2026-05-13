package password

import (
	"github.com/zalando/go-keyring"
)

const (
	serviceName = "vaulty"
	accountName = "master-password"
)

type KeyringStorage struct{}

func NewKeyringStorage() *KeyringStorage {
	return &KeyringStorage{}
}

func (k *KeyringStorage) Get() (string, error) {
	return keyring.Get(serviceName, accountName)
}

func (k *KeyringStorage) Set(password string) error {
	return keyring.Set(serviceName, accountName, password)
}

func (k *KeyringStorage) Delete() error {
	return keyring.Delete(serviceName, accountName)
}

func (k *KeyringStorage) Type() string {
	return "keyring"
}
