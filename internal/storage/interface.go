package storage

import (
	"context"
)

type SSHKeyInfo struct {
	Username string
	KeyName  string
	Size     int
}

type Storage interface {
	GetVault(ctx context.Context) ([]byte, error)
	PutVault(ctx context.Context, data []byte) error

	GetMetadata(ctx context.Context) ([]byte, error)
	PutMetadata(ctx context.Context, data []byte) error

	GetUserKeys(ctx context.Context, username string) ([]byte, error)
	PutUserKeys(ctx context.Context, username string, data []byte) error

	GetRecoverySeed(ctx context.Context, username string) ([]byte, error)
	PutRecoverySeed(ctx context.Context, username string, data []byte) error

	ListSSHKeys(ctx context.Context, username string) ([]SSHKeyInfo, error)
	PutSSHKey(ctx context.Context, username, keyName string, data []byte) error
	GetSSHKey(ctx context.Context, username, keyName string) ([]byte, error)
	DeleteSSHKey(ctx context.Context, username, keyName, sha string) error

	ListEnvs(ctx context.Context) ([]string, error)
	ListEnvSecrets(ctx context.Context, env string) ([]string, error)
	PutEnv(ctx context.Context, env, name string, data []byte) error
	GetEnv(ctx context.Context, env, name string) ([]byte, error)
	DeleteEnv(ctx context.Context, env, name string) error

	GetResource(ctx context.Context, path string) ([]byte, error)
	PutResource(ctx context.Context, path string, data []byte) error
	DeleteResource(ctx context.Context, path string) error
	ListResources(ctx context.Context) ([]string, error)

	ListMetadata(ctx context.Context) ([]string, error)

	IsLocal() bool
	GetRepo() string
}
