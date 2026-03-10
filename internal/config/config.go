package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Base64Bytes []byte

func (b Base64Bytes) MarshalJSON() ([]byte, error) {
	if b == nil {
		return []byte("null"), nil
	}
	encoded := base64.StdEncoding.EncodeToString(b)
	return json.Marshal(encoded)
}

func (b *Base64Bytes) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*b = nil
		return nil
	}
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	decoded, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return err
	}
	*b = decoded
	return nil
}

type Config struct {
	Repo          string      `json:"repo"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	DeviceSalt    Base64Bytes `json:"device_salt"`
	CacheDuration string      `json:"cache_duration"`
	StorageType   string      `json:"storage_type"`
}

func DefaultPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".vty", "config.json")
}

func Load(path string) (*Config, error) {
	if path == "" {
		path = DefaultPath()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Save(path string) error {
	if path == "" {
		path = DefaultPath()
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	if c.DeviceSalt == nil || len(c.DeviceSalt) == 0 {
		if err := c.GenerateDeviceSalt(); err != nil {
			return fmt.Errorf("generating device salt: %w", err)
		}
	}

	if c.CacheDuration == "" {
		c.CacheDuration = "15m"
	}

	if c.StorageType == "" {
		c.StorageType = "auto"
	}

	c.UpdatedAt = time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = c.UpdatedAt
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

func (c *Config) GenerateDeviceSalt() error {
	salt := make([]byte, 32)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("generating random bytes: %w", err)
	}
	c.DeviceSalt = salt
	return nil
}

func (c *Config) Validate() error {
	if c.Repo == "" {
		return fmt.Errorf("repo is required")
	}
	return nil
}

func (c *Config) SetRepo(repo string) {
	c.Repo = repo
	c.UpdatedAt = time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = c.UpdatedAt
	}
}
