package config

import (
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

type UserEntry struct {
	Username          string    `json:"username"`
	Role              string    `json:"role"`
	CreatedAt         time.Time `json:"created_at"`
	PasswordChallenge string    `json:"password_challenge,omitempty"`
}

type Metadata struct {
	Repo    string      `json:"repo"`
	Owner   string      `json:"owner"`
	Version string      `json:"version"`
	Users   []UserEntry `json:"users"`
}

type Config struct {
	Repo            string      `json:"repo"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
	DeviceSalt      Base64Bytes `json:"device_salt,omitempty"`
	CacheDuration   string      `json:"cache_duration"`
	StorageType     string      `json:"storage_type"`
	CurrentUser     string      `json:"current_user,omitempty"`
	CurrentUserRole string      `json:"current_user_role,omitempty"`
	Metadata        *Metadata   `json:"metadata,omitempty"`
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

func (c *Config) IsOwner() bool {
	return c.CurrentUserRole == "owner"
}

func (c *Config) GetRole() string {
	if c.CurrentUserRole == "" {
		return ""
	}
	return c.CurrentUserRole
}

func (c *Config) SetCurrentUser(username, role string) {
	c.CurrentUser = username
	c.CurrentUserRole = role
	c.UpdatedAt = time.Now()
}

func (c *Config) ClearCurrentUser() {
	c.CurrentUser = ""
	c.CurrentUserRole = ""
	c.UpdatedAt = time.Now()
}

func (c *Config) FindUser(username string) (*UserEntry, error) {
	if c.Metadata == nil || len(c.Metadata.Users) == 0 {
		return nil, fmt.Errorf("no users found in metadata")
	}

	for i := range c.Metadata.Users {
		if c.Metadata.Users[i].Username == username {
			return &c.Metadata.Users[i], nil
		}
	}

	return nil, fmt.Errorf("user %q not found", username)
}

func (c *Config) ValidateAndRefreshSession() error {
	if c.CurrentUser == "" {
		return fmt.Errorf("no active user session")
	}
	if c.CurrentUserRole == "" {
		return fmt.Errorf("user role is not set")
	}
	user, err := c.FindUser(c.CurrentUser)
	if err != nil {
		return fmt.Errorf("membership validation failed: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user %q not found in repository members", c.CurrentUser)
	}
	return nil
}
