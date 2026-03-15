package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/DeadBryam/vaulty/internal/crypto"
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

	return data, nil
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

func (l *LocalStorage) GetRecoverySeed(ctx context.Context, username string) ([]byte, error) {
	path := filepath.Join(l.baseDir, "recovery", fmt.Sprintf("%s.recovery.vty", username))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("recovery seed not found for %s", username)
		}
		return nil, fmt.Errorf("failed to read recovery seed: %w", err)
	}
	return data, nil
}

func (l *LocalStorage) PutRecoverySeed(ctx context.Context, username string, data []byte) error {
	recoveryDir := filepath.Join(l.baseDir, "recovery")
	if err := l.ensureDir(recoveryDir); err != nil {
		return fmt.Errorf("failed to create recovery directory: %w", err)
	}

	path := filepath.Join(recoveryDir, fmt.Sprintf("%s.recovery.vty", username))
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
	l.cleanEmptyDir(userDir)

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
		}
	}

	return envs, nil
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
		l.cleanEmptyDir(envDir)
	}

	return nil
}

func (l *LocalStorage) CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer srcFile.Close()

	dstDir := filepath.Dir(dst)
	if err := l.ensureDir(dstDir); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return dstFile.Sync()
}

func (l *LocalStorage) GetResource(ctx context.Context, path string) ([]byte, error) {
	fullPath := filepath.Join(l.baseDir, path)
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
	fullPath := filepath.Join(l.baseDir, path)
	dir := filepath.Dir(fullPath)

	if err := l.ensureDir(dir); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(fullPath, data, 0600)
}
