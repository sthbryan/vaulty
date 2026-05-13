package models

import "time"

type VaultConfig struct {
	StorageType  string    `yaml:"storage_type"`
	StoragePath  string    `yaml:"storage_path"`
	EncryptedKey string    `yaml:"encrypted_key"`
	Salt         string    `yaml:"salt"`
	Username     string    `yaml:"username"`
	VaultID      string    `yaml:"vault_id"`
	CreatedAt    time.Time `yaml:"created_at"`
	UpdatedAt    time.Time `yaml:"updated_at"`
}

type Session struct {
	Username    string    `yaml:"username"`
	VaultID     string    `yaml:"vault_id"`
	StorageType string    `yaml:"storage_type"`
	ExpiresAt   time.Time `yaml:"expires_at"`
}

type SecretMetadata struct {
	Type      SecretType `json:"type"`
	Name      string     `json:"name"`
	Env       string     `json:"env"`
	IsDir     bool       `json:"is_dir"`
	Encrypted bool       `json:"encrypted"`
	Size      int64      `json:"size"`
	Checksum  string     `json:"checksum"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type SecretFile struct {
	Metadata SecretMetadata `json:"metadata"`
	Data     []byte         `json:"data"`
}

type SecretType string

const (
	SecretTypeEnv       SecretType = "env"
	SecretTypeConfig    SecretType = "config"
	SecretTypeSSH       SecretType = "ssh"
	SecretTypeResources SecretType = "resources"
)

func (t SecretType) IsValid() bool {
	switch t {
	case SecretTypeEnv, SecretTypeConfig, SecretTypeSSH, SecretTypeResources:
		return true
	}
	return false
}

func (t SecretType) FolderName() string {
	return string(t)
}
