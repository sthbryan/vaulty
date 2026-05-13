package ports

import "context"

type ResourceEntry struct {
	Name string
	Size int64
	Path string
}

type ResourceStorage interface {
	ListResources(ctx context.Context) ([]string, error)
	PutResource(ctx context.Context, path string, data []byte) error
	GetResource(ctx context.Context, path string) ([]byte, error)
	DeleteResource(ctx context.Context, path string) error
}
