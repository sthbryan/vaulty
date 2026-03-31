package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/DeadBryam/vaulty/internal/config"
	"github.com/DeadBryam/vaulty/internal/crypto"
	"github.com/DeadBryam/vaulty/internal/storage"
)

func TestCompleteRecovery_ReencryptsExistingMasterKeyEnvelope(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("VAULTY_NO_KEYRING", "1")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))

	cfg := &config.Config{StorageType: "local"}
	ctx := context.Background()
	username := "alice"
	seedPhrase := "abandon ability able about above absent absorb abstract academy accent accept access"
	newPassword := "new-password-123"

	s, err := storage.NewLocalStorage()
	if err != nil {
		t.Fatalf("creating local storage: %v", err)
	}

	masterKey, err := crypto.GenerateMasterKey()
	if err != nil {
		t.Fatalf("generating master key: %v", err)
	}

	encryptedMasterKeyBefore, err := crypto.EncryptMasterKeyWithPassword(masterKey, seedPhrase)
	if err != nil {
		t.Fatalf("encrypting master key with recovery seed: %v", err)
	}

	masterKeyEnvelopeBefore := mustCompressEncryptedData(t, encryptedMasterKeyBefore)
	if err := s.PutUserKeys(ctx, username, masterKeyEnvelopeBefore); err != nil {
		t.Fatalf("storing user key envelope: %v", err)
	}

	encryptedSeedBefore, err := crypto.EncryptRecoverySeed(seedPhrase, seedPhrase)
	if err != nil {
		t.Fatalf("encrypting recovery seed: %v", err)
	}
	if err := s.PutRecoverySeed(ctx, username, mustCompressEncryptedData(t, encryptedSeedBefore)); err != nil {
		t.Fatalf("storing recovery seed: %v", err)
	}

	metadata := config.Metadata{
		Repo:    "local",
		Owner:   username,
		Version: "2.1",
		Users: []config.UserEntry{
			{Username: username, Role: "owner"},
		},
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("marshaling metadata: %v", err)
	}
	if err := s.PutMetadata(ctx, metadataJSON); err != nil {
		t.Fatalf("storing metadata: %v", err)
	}

	if err := completeRecovery(ctx, cfg, username, seedPhrase, newPassword); err != nil {
		t.Fatalf("completeRecovery failed: %v", err)
	}

	storedUserKey, err := s.GetUserKeys(ctx, username)
	if err != nil {
		t.Fatalf("reading updated user key envelope: %v", err)
	}
	if bytes.Equal(storedUserKey, masterKeyEnvelopeBefore) {
		t.Fatalf("expected user key envelope to be rotated")
	}

	updatedUserEnvelope := mustReadEncryptedData(t, storedUserKey)
	decryptedMasterKey, err := crypto.DecryptMasterKeyWithPassword(updatedUserEnvelope, newPassword)
	if err != nil {
		t.Fatalf("decrypting updated user key envelope with new password: %v", err)
	}
	if !bytes.Equal(decryptedMasterKey, masterKey) {
		t.Fatalf("master key changed during recovery")
	}

	storedRecovery, err := s.GetRecoverySeed(ctx, username)
	if err != nil {
		t.Fatalf("reading updated recovery seed: %v", err)
	}
	updatedRecoveryEnvelope := mustReadEncryptedData(t, storedRecovery)
	decryptedSeed, err := crypto.DecryptRecoverySeed(updatedRecoveryEnvelope, newPassword)
	if err != nil {
		t.Fatalf("decrypting recovery seed with new password: %v", err)
	}
	if decryptedSeed != seedPhrase {
		t.Fatalf("recovery seed changed during recovery")
	}

	if _, err := os.Stat(filepath.Join(homeDir, ".vty", "config.json")); err != nil {
		t.Fatalf("expected config to be persisted: %v", err)
	}
}

func mustCompressEncryptedData(t *testing.T, data *crypto.EncryptedData) []byte {
	t.Helper()
	payload, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshaling encrypted data: %v", err)
	}
	hexData, err := crypto.CompressHex(payload)
	if err != nil {
		t.Fatalf("compressing encrypted data: %v", err)
	}
	return []byte(hexData)
}

func mustReadEncryptedData(t *testing.T, compressed []byte) *crypto.EncryptedData {
	t.Helper()
	jsonData, err := crypto.DecompressHex(string(compressed))
	if err != nil {
		t.Fatalf("decompressing encrypted data: %v", err)
	}
	var data crypto.EncryptedData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		t.Fatalf("unmarshaling encrypted data: %v", err)
	}
	return &data
}
