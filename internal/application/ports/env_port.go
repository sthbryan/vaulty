package ports

import "context"

type SecretEntry struct {
	Name string
	Size int64
}

type EnvStorage interface {
	ListEnvs(ctx context.Context) ([]string, error)
	ListEnvSecrets(ctx context.Context, env string) ([]SecretEntry, error)
	PutEnv(ctx context.Context, env, name string, data []byte) error
	GetEnv(ctx context.Context, env, name string) ([]byte, error)
	DeleteEnv(ctx context.Context, env, name string) error
}
