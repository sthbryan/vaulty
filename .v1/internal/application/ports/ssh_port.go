package ports

import "context"

type SSHKeyInfo struct {
	Name string
	Data []byte
}

type SSHStorage interface {
	ListSSHKeys(ctx context.Context, username string) ([]SSHKeyInfo, error)
	PutSSHKey(ctx context.Context, username, keyName string, data []byte) error
	GetSSHKey(ctx context.Context, username, keyName string) ([]byte, error)
	DeleteSSHKey(ctx context.Context, username, keyName, sha string) error
}
