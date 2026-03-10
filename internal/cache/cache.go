package cache

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/password"
)

const CacheTTL = 24 * time.Hour

type CacheManager struct {
	storage password.Storage
}

func NewCacheManager(storage password.Storage) *CacheManager {
	return &CacheManager{
		storage: storage,
	}
}

func (c *CacheManager) cacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(home, ".vty", "cache")
	return cacheDir, nil
}

func (c *CacheManager) cachePath(username string) (string, error) {
	cacheDir, err := c.cacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, username+".vault.enc"), nil
}

func (c *CacheManager) Save(username string, vaultData []byte) error {
	pwd, err := c.storage.Get()
	if err != nil {
		return fmt.Errorf("failed to get password: %w", err)
	}

	encrypted, err := crypto.Encrypt(vaultData, pwd)
	if err != nil {
		return fmt.Errorf("failed to encrypt vault data: %w", err)
	}

	cacheDir, err := c.cacheDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cacheDir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	cachePath, err := c.cachePath(username)
	if err != nil {
		return err
	}

	timestamp := time.Now().Unix()
	timestampBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBytes, uint64(timestamp))

	fileData := append(timestampBytes, crypto.SerializeEncryptedData(encrypted)...)

	if err := os.WriteFile(cachePath, fileData, 0600); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

func (c *CacheManager) Load(username string) ([]byte, error) {
	cachePath, err := c.cachePath(username)
	if err != nil {
		return nil, err
	}

	fileData, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("cache not found")
		}
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	if len(fileData) < 8 {
		return nil, errors.New("invalid cache file format")
	}

	timestamp := int64(binary.BigEndian.Uint64(fileData[:8]))
	cacheTime := time.Unix(timestamp, 0)

	if time.Since(cacheTime) > CacheTTL {
		return nil, errors.New("cache expired")
	}

	encryptedData, err := crypto.DeserializeEncryptedData(fileData[8:])
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize encrypted data: %w", err)
	}

	pwd, err := c.storage.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get password: %w", err)
	}

	plaintext, err := crypto.Decrypt(encryptedData, pwd)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt vault data: %w", err)
	}

	return plaintext, nil
}

func (c *CacheManager) IsValid(username string) (bool, error) {
	cachePath, err := c.cachePath(username)
	if err != nil {
		return false, err
	}

	fileData, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to read cache file: %w", err)
	}

	if len(fileData) < 8 {
		return false, nil
	}

	timestamp := int64(binary.BigEndian.Uint64(fileData[:8]))
	cacheTime := time.Unix(timestamp, 0)

	if time.Since(cacheTime) > CacheTTL {
		return false, nil
	}

	return true, nil
}

func (c *CacheManager) Delete(username string) error {
	cachePath, err := c.cachePath(username)
	if err != nil {
		return err
	}

	if err := os.Remove(cachePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to delete cache file: %w", err)
	}

	return nil
}

func (c *CacheManager) GetTimestamp(username string) (time.Time, error) {
	cachePath, err := c.cachePath(username)
	if err != nil {
		return time.Time{}, err
	}

	fileData, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, errors.New("cache not found")
		}
		return time.Time{}, fmt.Errorf("failed to read cache file: %w", err)
	}

	if len(fileData) < 8 {
		return time.Time{}, errors.New("invalid cache file format")
	}

	timestamp := int64(binary.BigEndian.Uint64(fileData[:8]))
	return time.Unix(timestamp, 0), nil
}
