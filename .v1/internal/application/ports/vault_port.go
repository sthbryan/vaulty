package ports

import "context"

type VaultStorage interface {
	GetVault(ctx context.Context) ([]byte, error)
	PutVault(ctx context.Context, data []byte) error
	GetMetadata(ctx context.Context) ([]byte, error)
	PutMetadata(ctx context.Context, data []byte) error
}
