package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sthbryan/vaulty/v2/internal/vault/providers"
	"github.com/sthbryan/vaulty/v2/pkg/models"
)

type testVaultDir struct {
	path   string
	oldDir string
}

func newTestVaultDir(t *testing.T) *testVaultDir {
	path, err := os.MkdirTemp("", "vaulty-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return &testVaultDir{path: path}
}

func (td *testVaultDir) Set(t *testing.T) {
	td.oldDir = vaultDir
	_ = os.MkdirAll(td.path, 0700)
	vaultDir = td.path
}

func (td *testVaultDir) Restore() {
	vaultDir = td.oldDir
	os.RemoveAll(td.path)
}

func TestLoadConfig(t *testing.T) {
	td := newTestVaultDir(t)
	defer td.Restore()
	td.Set(t)

	_, err := LoadConfig()
	if err == nil {
		t.Error("LoadConfig() expected error for non-existent config")
	}

	now := time.Now()
	config := &models.VaultConfig{
		Username:    "testuser",
		VaultID:     "test-vault",
		StorageType: "local",
		StoragePath: "testuser/test-vault",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := SaveConfig(config); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loaded.Username != config.Username {
		t.Errorf("LoadConfig() Username = %s, want %s", loaded.Username, config.Username)
	}
	if loaded.VaultID != config.VaultID {
		t.Errorf("LoadConfig() VaultID = %s, want %s", loaded.VaultID, config.VaultID)
	}
	if loaded.StorageType != config.StorageType {
		t.Errorf("LoadConfig() StorageType = %s, want %s", loaded.StorageType, config.StorageType)
	}
}

func TestSaveConfig(t *testing.T) {
	td := newTestVaultDir(t)
	defer td.Restore()
	td.Set(t)

	config := &models.VaultConfig{
		Username:    "testuser",
		VaultID:     "test-vault",
		StorageType: "github",
		StoragePath: "testuser/test-vault",
	}

	if err := SaveConfig(config); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	configPath := ConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("SaveConfig() config file was not created")
	}

	if _, err := os.Stat(vaultDir); os.IsNotExist(err) {
		t.Error("SaveConfig() vault directory was not created")
	}
}

func TestConfigExists(t *testing.T) {
	td := newTestVaultDir(t)
	defer td.Restore()
	td.Set(t)

	if ConfigExists() {
		t.Error("ConfigExists() should return false when config does not exist")
	}

	config := &models.VaultConfig{
		Username:    "testuser",
		VaultID:     "test-vault",
		StorageType: "local",
	}
	SaveConfig(config)

	if !ConfigExists() {
		t.Error("ConfigExists() should return true when config exists")
	}
}

func TestLoadSession(t *testing.T) {
	td := newTestVaultDir(t)
	defer td.Restore()
	td.Set(t)

	_, err := LoadSession()
	if err == nil {
		t.Error("LoadSession() expected error for non-existent session")
	}

	session := &models.Session{
		Username:    "testuser",
		VaultID:     "test-vault",
		StorageType: "local",
		ExpiresAt:   time.Now().Add(8 * time.Hour),
	}

	if err := SaveSession(session); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	loaded, err := LoadSession()
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}

	if loaded.Username != session.Username {
		t.Errorf("LoadSession() Username = %s, want %s", loaded.Username, session.Username)
	}
	if loaded.VaultID != session.VaultID {
		t.Errorf("LoadSession() VaultID = %s, want %s", loaded.VaultID, session.VaultID)
	}
	if loaded.StorageType != session.StorageType {
		t.Errorf("LoadSession() StorageType = %s, want %s", loaded.StorageType, session.StorageType)
	}
}

func TestSaveSession(t *testing.T) {
	td := newTestVaultDir(t)
	defer td.Restore()
	td.Set(t)

	session := &models.Session{
		Username:    "testuser",
		VaultID:     "test-vault",
		StorageType: "local",
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	if err := SaveSession(session); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	sessionPath := SessionPath()
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		t.Error("SaveSession() session file was not created")
	}
}

func TestSessionExists(t *testing.T) {
	td := newTestVaultDir(t)
	defer td.Restore()
	td.Set(t)

	if SessionExists() {
		t.Error("SessionExists() should return false when session does not exist")
	}

	session := &models.Session{
		Username:    "testuser",
		VaultID:     "test-vault",
		StorageType: "local",
		ExpiresAt:   time.Now().Add(8 * time.Hour),
	}
	SaveSession(session)

	if !SessionExists() {
		t.Error("SessionExists() should return true when session exists")
	}
}

func TestDeleteSession(t *testing.T) {
	td := newTestVaultDir(t)
	defer td.Restore()
	td.Set(t)

	session := &models.Session{
		Username:    "testuser",
		VaultID:     "test-vault",
		StorageType: "local",
		ExpiresAt:   time.Now().Add(8 * time.Hour),
	}
	SaveSession(session)

	if err := DeleteSession(); err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}

	if SessionExists() {
		t.Error("DeleteSession() session still exists")
	}
}

func TestCreateSession(t *testing.T) {
	td := newTestVaultDir(t)
	defer td.Restore()
	td.Set(t)

	if err := CreateSession("testuser", "test-vault", "local", 8); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	session, err := LoadSession()
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}

	if session.Username != "testuser" {
		t.Errorf("CreateSession() Username = %s, want testuser", session.Username)
	}
	if session.VaultID != "test-vault" {
		t.Errorf("CreateSession() VaultID = %s, want test-vault", session.VaultID)
	}
	if session.StorageType != "local" {
		t.Errorf("CreateSession() StorageType = %s, want local", session.StorageType)
	}

	expectedExpiry := time.Now().Add(8 * time.Hour)
	diff := session.ExpiresAt.Sub(expectedExpiry)
	if diff > time.Minute || diff < -time.Minute {
		t.Errorf("CreateSession() ExpiresAt not within expected range")
	}
}

func TestLoadMeta(t *testing.T) {
	td := newTestVaultDir(t)
	defer td.Restore()
	td.Set(t)

	_, err := LoadMeta()
	if err == nil {
		t.Error("LoadMeta() expected error for non-existent meta")
	}

	now := time.Now()
	meta := &models.VaultMeta{
		Salt:         "testsalt123",
		EncryptedKey: "testencryptedkey",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := SaveMeta(meta); err != nil {
		t.Fatalf("SaveMeta() error = %v", err)
	}

	loaded, err := LoadMeta()
	if err != nil {
		t.Fatalf("LoadMeta() error = %v", err)
	}

	if loaded.Salt != meta.Salt {
		t.Errorf("LoadMeta() Salt = %s, want %s", loaded.Salt, meta.Salt)
	}
	if loaded.EncryptedKey != meta.EncryptedKey {
		t.Errorf("LoadMeta() EncryptedKey = %s, want %s", loaded.EncryptedKey, meta.EncryptedKey)
	}
}

func TestMetaExists(t *testing.T) {
	td := newTestVaultDir(t)
	defer td.Restore()
	td.Set(t)

	if MetaExists() {
		t.Error("MetaExists() should return false when meta does not exist")
	}

	meta := &models.VaultMeta{
		Salt:         "testsalt",
		EncryptedKey: "testkey",
	}
	SaveMeta(meta)

	if !MetaExists() {
		t.Error("MetaExists() should return true when meta exists")
	}
}

func TestCreateVault(t *testing.T) {
	td := newTestVaultDir(t)
	defer td.Restore()
	td.Set(t)

	info := VaultInfoWithPassword{
		VaultInfo: VaultInfo{
			Username:    "testuser",
			VaultID:     "test-vault",
			StorageType: "local",
		},
		Password: "testpassword123",
	}

	if err := CreateVault(info); err != nil {
		t.Fatalf("CreateVault() error = %v", err)
	}

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if config.Username != info.Username {
		t.Errorf("CreateVault() config.Username = %s, want %s", config.Username, info.Username)
	}
	if config.VaultID != info.VaultID {
		t.Errorf("CreateVault() config.VaultID = %s, want %s", config.VaultID, info.VaultID)
	}
	if config.StorageType != info.StorageType {
		t.Errorf("CreateVault() config.StorageType = %s, want %s", config.StorageType, info.StorageType)
	}

	meta, err := LoadMeta()
	if err != nil {
		t.Fatalf("LoadMeta() error = %v", err)
	}
	if meta.Salt == "" {
		t.Error("CreateVault() meta.Salt should not be empty")
	}
	if meta.EncryptedKey == "" {
		t.Error("CreateVault() meta.EncryptedKey should not be empty")
	}
}

func TestSetupStorage_Local(t *testing.T) {
	td := newTestVaultDir(t)
	defer td.Restore()
	td.Set(t)

	info := VaultInfo{
		Username:    "testuser",
		VaultID:     "test-vault",
		StorageType: "local",
	}

	if err := SetupStorage(info); err != nil {
		t.Fatalf("SetupStorage() error = %v", err)
	}
}

func TestLoadMetaFromStorage_Local(t *testing.T) {
	td := newTestVaultDir(t)
	defer td.Restore()
	td.Set(t)

	meta := &models.VaultMeta{
		Salt:         "testsalt456",
		EncryptedKey: "testencryptedkey456",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := SaveMeta(meta); err != nil {
		t.Fatalf("SaveMeta() error = %v", err)
	}

	info := VaultInfo{
		Username:    "testuser",
		VaultID:     "test-vault",
		StorageType: "local",
	}

	loaded, err := LoadMetaFromStorage(info)
	if err != nil {
		t.Fatalf("LoadMetaFromStorage() error = %v", err)
	}

	if loaded.Salt != meta.Salt {
		t.Errorf("LoadMetaFromStorage() Salt = %s, want %s", loaded.Salt, meta.Salt)
	}
	if loaded.EncryptedKey != meta.EncryptedKey {
		t.Errorf("LoadMetaFromStorage() EncryptedKey = %s, want %s", loaded.EncryptedKey, meta.EncryptedKey)
	}
}

func TestCheckVaultAvailability_Local(t *testing.T) {
	td := newTestVaultDir(t)
	defer td.Restore()
	td.Set(t)

	provider := providers.NewProvider(providers.ProviderLocal, "testuser", "test-vault")
	if provider == nil {
		t.Fatal("NewProvider() returned nil")
	}

	info := VaultInfo{
		Username:    "testuser",
		VaultID:     "test-vault",
		StorageType: "local",
	}

	available, err := CheckVaultAvailability(info)
	if err != nil {
		t.Fatalf("CheckVaultAvailability() error = %v", err)
	}
	_ = available

	if err := provider.CreateRepo(nil); err != nil {
		t.Fatalf("CreateRepo() error = %v", err)
	}

	available, err = CheckVaultAvailability(info)
	if err != nil {
		t.Fatalf("CheckVaultAvailability() error = %v", err)
	}

	if !available {
		t.Error("CheckVaultAvailability() should return true after vault is created")
	}
}

func TestCheckVault(t *testing.T) {

	if CheckVault(providers.ProviderType("invalid"), "testuser", "test-vault") {
		t.Error("CheckVault() should return false for invalid provider")
	}
}

func TestVaultDir(t *testing.T) {
	td := newTestVaultDir(t)
	defer td.Restore()
	td.Set(t)

	dir := VaultDir()
	if dir == "" {
		t.Error("VaultDir() returned empty string")
	}
	if dir == ".vaulty" {
		t.Error("VaultDir() should not return the default constant")
	}
}

func TestStoragePath(t *testing.T) {
	path := StoragePath("testuser", "test-vault")
	if path != "testuser/test-vault" {
		t.Errorf("StoragePath() = %s, want testuser/test-vault", path)
	}
}

func TestCurrentUser(t *testing.T) {
	user := CurrentUser()
	if user == "" {
		t.Error("CurrentUser() returned empty string")
	}
}

func TestNewProvider(t *testing.T) {

	githubProvider := NewProvider(providers.ProviderGitHub, "token", "owner", "repo")
	if githubProvider == nil {
		t.Error("NewProvider() returned nil for GitHub provider")
	}

	localProvider := NewProvider(providers.ProviderLocal, "", "owner", "repo")
	if localProvider == nil {
		t.Error("NewProvider() returned nil for Local provider")
	}

	nilProvider := NewProvider(providers.ProviderGitHub, "token")
	if nilProvider != nil {
		t.Error("NewProvider() should return nil with insufficient params")
	}
}

func TestPathHelpers(t *testing.T) {
	td := newTestVaultDir(t)
	defer td.Restore()
	td.Set(t)

	configPath := ConfigPath()
	if !filepath.IsAbs(configPath) {
		t.Errorf("ConfigPath() should return absolute path, got %s", configPath)
	}
	if filepath.Base(configPath) != "config.yaml" {
		t.Errorf("ConfigPath() should end with config.yaml, got %s", configPath)
	}

	metaPath := MetaPath()
	if !filepath.IsAbs(metaPath) {
		t.Errorf("MetaPath() should return absolute path, got %s", metaPath)
	}
	if filepath.Base(metaPath) != "vault.meta" {
		t.Errorf("MetaPath() should end with vault.meta, got %s", metaPath)
	}

	sessionPath := SessionPath()
	if !filepath.IsAbs(sessionPath) {
		t.Errorf("SessionPath() should return absolute path, got %s", sessionPath)
	}
	if filepath.Base(sessionPath) != "session.yaml" {
		t.Errorf("SessionPath() should end with session.yaml, got %s", sessionPath)
	}

	vaultPath := VaultPath("my-vault")
	if !filepath.IsAbs(vaultPath) {
		t.Errorf("VaultPath() should return absolute path, got %s", vaultPath)
	}
	if !strings.HasSuffix(vaultPath, "/my-vault") {
		t.Errorf("VaultPath() should end with vault ID, got %s", vaultPath)
	}
}

func TestConfigPersistence(t *testing.T) {
	td := newTestVaultDir(t)
	defer td.Restore()
	td.Set(t)

	original := &models.VaultConfig{
		Username:    "persistuser",
		VaultID:     "persist-vault",
		StorageType: "local",
		StoragePath: "persistuser/persist-vault",
	}

	if err := SaveConfig(original); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if loaded.Username != original.Username {
		t.Errorf("Config persistence Username = %s, want %s", loaded.Username, original.Username)
	}
	if loaded.VaultID != original.VaultID {
		t.Errorf("Config persistence VaultID = %s, want %s", loaded.VaultID, original.VaultID)
	}
}

func TestSessionPersistence(t *testing.T) {
	td := newTestVaultDir(t)
	defer td.Restore()
	td.Set(t)

	expiresAt := time.Now().Add(24 * time.Hour)
	original := &models.Session{
		Username:    "persistuser",
		VaultID:     "persist-vault",
		StorageType: "local",
		ExpiresAt:   expiresAt,
	}

	if err := SaveSession(original); err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	loaded, err := LoadSession()
	if err != nil {
		t.Fatalf("LoadSession() error = %v", err)
	}

	if loaded.Username != original.Username {
		t.Errorf("Session persistence Username = %s, want %s", loaded.Username, original.Username)
	}
	if loaded.VaultID != original.VaultID {
		t.Errorf("Session persistence VaultID = %s, want %s", loaded.VaultID, original.VaultID)
	}
}

func TestMetaPersistence(t *testing.T) {
	td := newTestVaultDir(t)
	defer td.Restore()
	td.Set(t)

	original := &models.VaultMeta{
		Salt:         "persistentsalt1234567890123456789012345678901234567890123456",
		EncryptedKey: "persistentencryptedkey",
	}

	if err := SaveMeta(original); err != nil {
		t.Fatalf("SaveMeta() error = %v", err)
	}

	loaded, err := LoadMeta()
	if err != nil {
		t.Fatalf("LoadMeta() error = %v", err)
	}

	if loaded.Salt != original.Salt {
		t.Errorf("Meta persistence Salt = %s, want %s", loaded.Salt, original.Salt)
	}
	if loaded.EncryptedKey != original.EncryptedKey {
		t.Errorf("Meta persistence EncryptedKey = %s, want %s", loaded.EncryptedKey, original.EncryptedKey)
	}
}
