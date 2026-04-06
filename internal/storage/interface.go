package storage

import (
	"context"

	"github.com/DeadBryam/vaulty/internal/github"
)

type SSHKeyInfo struct {
	Username string
	KeyName  string
	Size     int
}

type ContentInfo struct {
	Name string
	Sha  string
}

type Storage interface {
	GetVault(ctx context.Context) ([]byte, error)
	PutVault(ctx context.Context, data []byte) error

	GetMetadata(ctx context.Context) ([]byte, error)
	PutMetadata(ctx context.Context, data []byte) error

	GetUserKeys(ctx context.Context, username string) ([]byte, error)
	PutUserKeys(ctx context.Context, username string, data []byte) error

	GetOwner() string
	GetOwnerAndRepo() (string, string, error)
	PutContent(ctx context.Context, path string, content string) error
	GetContent(ctx context.Context, path string) (*github.ContentResponse, error)
	DecodeContent(content *github.ContentResponse) ([]byte, error)
	DeleteContent(ctx context.Context, path string, sha string) error
	ListDirectory(ctx context.Context, path string) ([]ContentInfo, error)

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
