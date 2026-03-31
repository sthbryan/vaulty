package ports

import "context"

type UserStorage interface {
	GetUserKeys(ctx context.Context, username string) ([]byte, error)
	PutUserKeys(ctx context.Context, username string, data []byte) error
	GetRecoverySeed(ctx context.Context, username string) ([]byte, error)
	PutRecoverySeed(ctx context.Context, username string, data []byte) error
	GetUserList(ctx context.Context) ([]byte, error)
}
