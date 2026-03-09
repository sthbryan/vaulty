package models

import (
	"time"

	"github.com/sthbryan/vaulty/internal/crypto"
)

// SecretType represents the type of secret
type SecretType string

const (
	// SecretTypeEnv is for environment variable secrets
	SecretTypeEnv SecretType = "env"
	// SecretTypeSSH is for SSH key secrets
	SecretTypeSSH SecretType = "ssh"
)

// SecretMetadata holds metadata about a secret
type SecretMetadata struct {
	Name      string     `json:"name"`
	Type      SecretType `json:"type"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Size      int64      `json:"size"`
	Checksum  string     `json:"checksum"`
}

// VaultFile represents a complete vault file with metadata and encrypted data
type VaultFile struct {
	Metadata SecretMetadata       `json:"metadata"`
	Data     crypto.EncryptedData `json:"data"`
}

// SecretInfo represents a secret for listing (without the encrypted data)
type SecretInfo struct {
	Name      string     `json:"name"`
	Type      SecretType `json:"type"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Size      int64      `json:"size"`
}
