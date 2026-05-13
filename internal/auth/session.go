package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sthbryan/vaulty/v2/pkg/models"
	"gopkg.in/yaml.v3"
)

const (
	SessionFileName = "session.yml"
	VaultDirName    = ".vaulty"
	ConfigFileName  = "config.yaml"
	MetaFileName    = "vault.meta"
)

type SessionManager struct {
	vaultDir string
}

func NewSessionManager() (*SessionManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}

	vaultDir := filepath.Join(homeDir, VaultDirName)
	if err := os.MkdirAll(vaultDir, 0700); err != nil {
		return nil, fmt.Errorf("creating vault directory: %w", err)
	}

	return &SessionManager{vaultDir: vaultDir}, nil
}

func (sm *SessionManager) GetVaultDir() string {
	return sm.vaultDir
}

func (sm *SessionManager) GetSessionPath() string {
	return filepath.Join(sm.vaultDir, SessionFileName)
}

func (sm *SessionManager) GetConfigPath() string {
	return filepath.Join(sm.vaultDir, ConfigFileName)
}

func (sm *SessionManager) GetMetaPath() string {
	return filepath.Join(sm.vaultDir, MetaFileName)
}

func (sm *SessionManager) LoadSession() (*models.Session, error) {
	sessionPath := sm.GetSessionPath()

	data, err := os.ReadFile(sessionPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no active session")
		}
		return nil, fmt.Errorf("reading session file: %w", err)
	}

	var session models.Session
	if err := yaml.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("parsing session file: %w", err)
	}

	if time.Now().After(session.ExpiresAt) {
		_ = sm.DeleteSession()
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

func (sm *SessionManager) SaveSession(session *models.Session) error {
	sessionPath := sm.GetSessionPath()

	data, err := yaml.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshaling session: %w", err)
	}

	if err := os.WriteFile(sessionPath, data, 0600); err != nil {
		return fmt.Errorf("writing session file: %w", err)
	}

	return nil
}

func (sm *SessionManager) DeleteSession() error {
	sessionPath := sm.GetSessionPath()

	if err := os.Remove(sessionPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("deleting session file: %w", err)
	}

	return nil
}

func (sm *SessionManager) LoadConfig() (*models.VaultConfig, error) {
	configPath := sm.GetConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("vault not initialized")
		}
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var config models.VaultConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	return &config, nil
}

func (sm *SessionManager) SaveConfig(config *models.VaultConfig) error {
	configPath := sm.GetConfigPath()

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

func (sm *SessionManager) LoadMeta() (*models.VaultMeta, error) {
	metaPath := sm.GetMetaPath()

	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("vault meta not found")
		}
		return nil, fmt.Errorf("reading meta file: %w", err)
	}

	var meta models.VaultMeta
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parsing meta file: %w", err)
	}

	return &meta, nil
}

func (sm *SessionManager) SaveMeta(meta *models.VaultMeta) error {
	metaPath := sm.GetMetaPath()


	data, err := yaml.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshaling meta: %w", err)
	}

	if err := os.WriteFile(metaPath, data, 0600); err != nil {
		return fmt.Errorf("writing meta file: %w", err)
	}

	return nil
}

func (sm *SessionManager) VaultExists() bool {
	_, err := sm.LoadConfig()
	return err == nil
}

func (sm *SessionManager) SessionExists() bool {
	session, _ := sm.LoadSession()
	return session != nil
}

func (sm *SessionManager) IsSessionValid() bool {
	session, err := sm.LoadSession()
	if err != nil {
		return false
	}
	return time.Now().Before(session.ExpiresAt)
}