package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sthbryan/vaulty/v2/pkg/models"
)

func TestSessionManager_VaultDir(t *testing.T) {
	sm, err := NewSessionManager()
	if err != nil {
		t.Fatalf("NewSessionManager() error = %v", err)
	}

	expectedDir := filepath.Join(getTestHomeDir(), VaultDirName)
	if sm.GetVaultDir() != expectedDir {
		t.Errorf("GetVaultDir() = %s, want %s", sm.GetVaultDir(), expectedDir)
	}
}

func TestSessionManager_SessionPath(t *testing.T) {
	sm, _ := NewSessionManager()

	sessionPath := sm.GetSessionPath()
	if !filepath.IsAbs(sessionPath) {
		t.Error("GetSessionPath() should return absolute path")
	}
	if filepath.Base(sessionPath) != SessionFileName {
		t.Errorf("GetSessionPath() = %s, want %s", filepath.Base(sessionPath), SessionFileName)
	}
}

func TestSessionManager_SaveLoadSession(t *testing.T) {
	sm, _ := NewSessionManager()

	session := &models.Session{
		Username:    "testuser",
		VaultID:     "test-vault",
		StorageType: "local",
		ExpiresAt:   time.Now().Add(8 * time.Hour),
	}

	if err := sm.SaveSession(session); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	loaded, err := sm.LoadSession()
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}

	if loaded.Username != session.Username {
		t.Errorf("Username = %s, want %s", loaded.Username, session.Username)
	}
	if loaded.VaultID != session.VaultID {
		t.Errorf("VaultID = %s, want %s", loaded.VaultID, session.VaultID)
	}
	if loaded.StorageType != session.StorageType {
		t.Errorf("StorageType = %s, want %s", loaded.StorageType, session.StorageType)
	}

	_ = sm.DeleteSession()
}

func TestSessionManager_DeleteSession(t *testing.T) {
	sm, _ := NewSessionManager()

	session := &models.Session{
		Username:  "testuser",
		VaultID:   "test-vault",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	_ = sm.SaveSession(session)

	if err := sm.DeleteSession(); err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}

	if sm.SessionExists() {
		t.Error("SessionExists() after DeleteSession() = true, want false")
	}
}

func TestSessionManager_SessionExpired(t *testing.T) {
	sm, _ := NewSessionManager()

	session := &models.Session{
		Username:  "testuser",
		VaultID:   "test-vault",
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	_ = sm.SaveSession(session)

	_, err := sm.LoadSession()
	if err == nil {
		t.Error("LoadSession() should error for expired session")
	}

	_ = sm.DeleteSession()
}

func TestSessionManager_VaultExists(t *testing.T) {
	sm, _ := NewSessionManager()

	// Save a config
	config := &models.VaultConfig{
		StorageType: "local",
		StoragePath: "/tmp/test",
		Username:    "test",
		VaultID:     "test",
	}
	_ = sm.SaveConfig(config)

	if !sm.VaultExists() {
		t.Error("VaultExists() = false, want true (config exists)")
	}

	// Verify we can load it
	loadedConfig, err := sm.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if loadedConfig.StorageType != "local" {
		t.Errorf("StorageType = %s, want local", loadedConfig.StorageType)
	}

	// Clean up
	os.Remove(sm.GetConfigPath())
}

func TestSessionManager_IsSessionValid(t *testing.T) {
	sm, _ := NewSessionManager()

	if sm.IsSessionValid() {
		t.Error("IsSessionValid() = true, want false (no session)")
	}

	session := &models.Session{
		Username:  "testuser",
		VaultID:   "test-vault",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	_ = sm.SaveSession(session)

	if !sm.IsSessionValid() {
		t.Error("IsSessionValid() = false, want true (valid session)")
	}

	_ = sm.DeleteSession()
}

func getTestHomeDir() string {
	home, _ := os.UserHomeDir()
	return home
}