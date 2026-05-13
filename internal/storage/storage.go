package storage

import (
	"context"
	"time"
)

type Storage interface {
	Ping(ctx context.Context) error
	Upload(ctx context.Context, path string, data []byte) error
	Download(ctx context.Context, path string) ([]byte, error)
	Delete(ctx context.Context, path string) error
	List(ctx context.Context, prefix string) ([]string, error)
	Exists(ctx context.Context, path string) (bool, error)
}

func NewLocal(path string) Storage {
	return &localStorage{basePath: path}
}

func NewGitHub(token, owner, repo string) Storage {
	return &githubStorage{
		token:      token,
		owner:      owner,
		repo:       repo,
		httpClient: &httpClient{timeout: 30 * time.Second},
	}
}

type httpClient struct {
	timeout time.Duration
}

type localStorage struct {
	basePath string
}

func (s *localStorage) Ping(ctx context.Context) error {
	return nil
}

func (s *localStorage) Upload(ctx context.Context, path string, data []byte) error {
	return nil
}

func (s *localStorage) Download(ctx context.Context, path string) ([]byte, error) {
	return nil, nil
}

func (s *localStorage) Delete(ctx context.Context, path string) error {
	return nil
}

func (s *localStorage) List(ctx context.Context, prefix string) ([]string, error) {
	return nil, nil
}

func (s *localStorage) Exists(ctx context.Context, path string) (bool, error) {
	return false, nil
}

type githubStorage struct {
	token      string
	owner      string
	repo       string
	httpClient *httpClient
}

func (s *githubStorage) Ping(ctx context.Context) error {
	return nil
}

func (s *githubStorage) Upload(ctx context.Context, path string, data []byte) error {
	return nil
}

func (s *githubStorage) Download(ctx context.Context, path string) ([]byte, error) {
	return nil, nil
}

func (s *githubStorage) Delete(ctx context.Context, path string) error {
	return nil
}

func (s *githubStorage) List(ctx context.Context, prefix string) ([]string, error) {
	return nil, nil
}

func (s *githubStorage) Exists(ctx context.Context, path string) (bool, error) {
	return false, nil
}
