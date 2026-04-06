package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sthbryan/vaulty/internal/crypto"
	"github.com/sthbryan/vaulty/internal/github"
)

type LocalStorage struct {
	baseDir string
}

func NewLocalStorage() (*LocalStorage, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	baseDir := filepath.Join(homeDir, ".vty", "vault")

	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create vault directory: %w", err)
	}

	return &LocalStorage{
		baseDir: baseDir,
	}, nil
}

func (l *LocalStorage) ensureDir(dir string) error {
	return os.MkdirAll(dir, 0700)
}

func (l *LocalStorage) GetVault(ctx context.Context) ([]byte, error) {
	path := filepath.Join(l.baseDir, "vault.vty")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("vault not found")
		}
		return nil, fmt.Errorf("failed to read vault: %w", err)
	}
	return data, nil
}

func (l *LocalStorage) PutVault(ctx context.Context, data []byte) error {
	path := filepath.Join(l.baseDir, "vault.vty")
	return os.WriteFile(path, data, 0600)
}

func (l *LocalStorage) GetMetadata(ctx context.Context) ([]byte, error) {
	path := filepath.Join(l.baseDir, "metadata.vty")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("metadata not found")
		}
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	return crypto.DecompressHex(string(data))
}

func (l *LocalStorage) PutMetadata(ctx context.Context, data []byte) error {
	path := filepath.Join(l.baseDir, "metadata.vty")

	compressed, err := crypto.CompressHex(data)
	if err != nil {
		return fmt.Errorf("failed to compress metadata: %w", err)
	}

	return os.WriteFile(path, []byte(compressed), 0600)
}

func (l *LocalStorage) GetUserKeys(ctx context.Context, username string) ([]byte, error) {
	path := filepath.Join(l.baseDir, "keys", fmt.Sprintf("%s.vty", username))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("user keys not found for %s", username)
		}
		return nil, fmt.Errorf("failed to read user keys: %w", err)
	}
	return data, nil
}

func (l *LocalStorage) PutUserKeys(ctx context.Context, username string, data []byte) error {
	keysDir := filepath.Join(l.baseDir, "keys")
	if err := l.ensureDir(keysDir); err != nil {
		return fmt.Errorf("failed to create keys directory: %w", err)
	}

	path := filepath.Join(keysDir, fmt.Sprintf("%s.vty", username))
	return os.WriteFile(path, data, 0600)
}

func (l *LocalStorage) ListSSHKeys(ctx context.Context, username string) ([]SSHKeyInfo, error) {
	sshDir := filepath.Join(l.baseDir, "ssh", username)

	entries, err := os.ReadDir(sshDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []SSHKeyInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read SSH directory: %w", err)
	}

	var keys []SSHKeyInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) == ".vty" {
			keyName := name[:len(name)-len(".vty")]

			info, err := entry.Info()
			if err != nil {
				continue
			}

			keys = append(keys, SSHKeyInfo{
				Username: username,
				KeyName:  keyName,
				Size:     int(info.Size()),
			})
		}
	}

	return keys, nil
}

func (l *LocalStorage) PutSSHKey(ctx context.Context, username, keyName string, data []byte) error {
	sshDir := filepath.Join(l.baseDir, "ssh", username)
	if err := l.ensureDir(sshDir); err != nil {
		return fmt.Errorf("failed to create SSH directory: %w", err)
	}

	path := filepath.Join(sshDir, fmt.Sprintf("%s.vty", keyName))
	return os.WriteFile(path, data, 0600)
}

func (l *LocalStorage) GetSSHKey(ctx context.Context, username, keyName string) ([]byte, error) {
	path := filepath.Join(l.baseDir, "ssh", username, fmt.Sprintf("%s.vty", keyName))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("SSH key not found: %s/%s", username, keyName)
		}
		return nil, fmt.Errorf("failed to read SSH key: %w", err)
	}
	return data, nil
}

func (l *LocalStorage) DeleteSSHKey(ctx context.Context, username, keyName, sha string) error {
	path := filepath.Join(l.baseDir, "ssh", username, fmt.Sprintf("%s.vty", keyName))
	err := os.Remove(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("SSH key not found: %s/%s", username, keyName)
		}
		return fmt.Errorf("failed to delete SSH key: %w", err)
	}

	userDir := filepath.Join(l.baseDir, "ssh", username)
	_ = l.cleanEmptyDir(userDir)

	return nil
}

func (l *LocalStorage) cleanEmptyDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	if len(entries) == 0 {
		return os.Remove(dir)
	}
	return nil
}

func (l *LocalStorage) IsLocal() bool {
	return true
}

func (l *LocalStorage) GetRepo() string {
	return l.baseDir
}

func (l *LocalStorage) ListEnvs(ctx context.Context) ([]string, error) {
	envsDir := filepath.Join(l.baseDir, "envs")

	entries, err := os.ReadDir(envsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read envs directory: %w", err)
	}

	var envs []string
	for _, entry := range entries {
		if entry.IsDir() {
			envs = append(envs, entry.Name())
		} else if strings.HasSuffix(entry.Name(), ".vty") {
			name := strings.TrimSuffix(entry.Name(), ".vty")
			envs = append(envs, name)
		}
	}

	return envs, nil
}

func (l *LocalStorage) ListEnvSecrets(ctx context.Context, env string) ([]string, error) {
	if env == "" {
		return []string{}, nil
	}

	envPath := filepath.Join(l.baseDir, "envs", env)
	entries, err := os.ReadDir(envPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read env directory: %w", err)
	}

	var secrets []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".vty") {
			secrets = append(secrets, strings.TrimSuffix(name, ".vty"))
		}
	}

	return secrets, nil
}

func (l *LocalStorage) PutEnv(ctx context.Context, env, name string, data []byte) error {
	var envDir string
	if env == "" {
		envDir = filepath.Join(l.baseDir, "envs")
	} else {
		envDir = filepath.Join(l.baseDir, "envs", env)
	}

	if err := l.ensureDir(envDir); err != nil {
		return fmt.Errorf("failed to create env directory: %w", err)
	}

	path := filepath.Join(envDir, fmt.Sprintf("%s.vty", name))
	return os.WriteFile(path, data, 0600)
}

func (l *LocalStorage) GetEnv(ctx context.Context, env, name string) ([]byte, error) {
	var path string
	if env == "" {
		path = filepath.Join(l.baseDir, "envs", fmt.Sprintf("%s.vty", name))
	} else {
		path = filepath.Join(l.baseDir, "envs", env, fmt.Sprintf("%s.vty", name))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("env not found: %s/%s", env, name)
		}
		return nil, fmt.Errorf("failed to read env: %w", err)
	}

	return data, nil
}

func (l *LocalStorage) DeleteEnv(ctx context.Context, env, name string) error {
	var path string
	if env == "" {
		path = filepath.Join(l.baseDir, "envs", fmt.Sprintf("%s.vty", name))
	} else {
		path = filepath.Join(l.baseDir, "envs", env, fmt.Sprintf("%s.vty", name))
	}

	err := os.Remove(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("env not found: %s/%s", env, name)
		}
		return fmt.Errorf("failed to delete env: %w", err)
	}

	if env != "" {
		envDir := filepath.Join(l.baseDir, "envs", env)
		_ = l.cleanEmptyDir(envDir)
	}

	return nil
}

func (l *LocalStorage) CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer func() { _ = srcFile.Close() }()

	dstDir := filepath.Dir(dst)
	if err := l.ensureDir(dstDir); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer func() { _ = dstFile.Close() }()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return dstFile.Sync()
}

func (l *LocalStorage) resolveSafeVaultPath(path string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(path))
	if clean == "." || clean == "" {
		return "", fmt.Errorf("invalid empty path")
	}
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid path: %s", path)
	}

	fullPath := filepath.Join(l.baseDir, clean)
	rel, err := filepath.Rel(l.baseDir, fullPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes vault directory")
	}

	return fullPath, nil
}

func (l *LocalStorage) GetResource(ctx context.Context, path string) ([]byte, error) {
	fullPath, err := l.resolveSafeVaultPath(path)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("resource not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read resource: %w", err)
	}
	return data, nil
}

func (l *LocalStorage) PutResource(ctx context.Context, path string, data []byte) error {
	fullPath, err := l.resolveSafeVaultPath(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(fullPath)

	if err := l.ensureDir(dir); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(fullPath, data, 0600)
}

func (l *LocalStorage) DeleteResource(ctx context.Context, path string) error {
	fullPath, err := l.resolveSafeVaultPath(path)
	if err != nil {
		return err
	}

	err = os.Remove(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("resource not found: %s", path)
		}
		return fmt.Errorf("failed to delete resource: %w", err)
	}

	dir := filepath.Dir(fullPath)
	_ = l.cleanEmptyDir(dir)

	return nil
}

func (l *LocalStorage) ListResources(ctx context.Context) ([]string, error) {
	var resources []string

	resourcesDir := filepath.Join(l.baseDir, "resources")
	err := filepath.Walk(resourcesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Ext(path) == ".vty" {
			rel, _ := filepath.Rel(resourcesDir, path)
			resources = append(resources, filepath.Join("resources", rel))
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}

	configDir := filepath.Join(l.baseDir, "config")
	err = filepath.Walk(configDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Ext(path) == ".vty" {
			rel, _ := filepath.Rel(configDir, path)
			resources = append(resources, filepath.Join("config", rel))
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to list config: %w", err)
	}

	return resources, nil
}

func (l *LocalStorage) ListMetadata(ctx context.Context) ([]string, error) {
	var files []string

	metadataPath := filepath.Join(l.baseDir, "metadata.vty")
	if _, err := os.Stat(metadataPath); err == nil {
		files = append(files, "metadata.vty")
	}

	keysDir := filepath.Join(l.baseDir, "keys")
	entries, err := os.ReadDir(keysDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".vty" {
				files = append(files, filepath.Join("keys", entry.Name()))
			}
		}
	}

	return files, nil
}

func (l *LocalStorage) GetOwner() string {
	return "local"
}

func (l *LocalStorage) GetOwnerAndRepo() (string, string, error) {
	return "local", "local", nil
}

func (l *LocalStorage) PutContent(ctx context.Context, path string, content string) error {
	fullPath := filepath.Join(l.baseDir, path)
	dir := filepath.Dir(fullPath)
	if err := l.ensureDir(dir); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return os.WriteFile(fullPath, []byte(content), 0600)
}

func (l *LocalStorage) GetContent(ctx context.Context, path string) (*github.ContentResponse, error) {
	fullPath := filepath.Join(l.baseDir, path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	return &github.ContentResponse{
		Content: string(data),
		Sha:     "",
	}, nil
}

func (l *LocalStorage) DecodeContent(content *github.ContentResponse) ([]byte, error) {
	return []byte(content.Content), nil
}

func (l *LocalStorage) DeleteContent(ctx context.Context, path string, sha string) error {
	fullPath := filepath.Join(l.baseDir, path)
	return os.Remove(fullPath)
}

func (l *LocalStorage) ListDirectory(ctx context.Context, path string) ([]ContentInfo, error) {
	fullPath := filepath.Join(l.baseDir, path)
	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory: %w", err)
	}

	var result []ContentInfo
	for _, entry := range entries {
		info, _ := entry.Info()
		sha := ""
		if info != nil {
			sha = fmt.Sprintf("%x", info.Size())
		}
		result = append(result, ContentInfo{
			Name: entry.Name(),
			Sha:  sha,
		})
	}
	return result, nil
}
