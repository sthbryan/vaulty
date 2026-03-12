package models

import (
	"time"

	"github.com/DeadBryam/vaulty/internal/crypto"
)

type SecretType string

const (
	SecretTypeEnv      SecretType = "env"
	SecretTypeSSH      SecretType = "ssh"
	SecretTypeResource SecretType = "resource"
	SecretTypeConfig   SecretType = "config"
)

type SecretMetadata struct {
	Name        string     `json:"name"`
	Type        SecretType `json:"type"`
	Environment string     `json:"environment"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Size        int64      `json:"size"`
	Checksum    string     `json:"checksum"`
}

type VaultFile struct {
	Metadata SecretMetadata       `json:"metadata"`
	Data     crypto.EncryptedData `json:"data"`
}

type SecretInfo struct {
	Name        string     `json:"name"`
	Type        SecretType `json:"type"`
	Environment string     `json:"environment"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Size        int64      `json:"size"`
}

type ResourceMetadata struct {
	Name        string     `json:"name"`
	Type        SecretType `json:"type"`
	Tag         string     `json:"tag"`
	IsEncrypted bool       `json:"is_encrypted"`
	IsDirectory bool       `json:"is_directory"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Size        int64      `json:"size"`
	Checksum    string     `json:"checksum"`
}
