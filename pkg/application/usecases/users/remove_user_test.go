package users

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/sthbryan/vaulty/internal/config"
	"github.com/sthbryan/vaulty/internal/crypto"
)

func TestParseEncryptedDataFromStorage_CompressedHex(t *testing.T) {
	password := "super-secret"
	masterKey, err := crypto.GenerateMasterKey()
	if err != nil {
		t.Fatalf("generate master key: %v", err)
	}

	encrypted, err := crypto.EncryptMasterKeyWithPassword(masterKey, password)
	if err != nil {
		t.Fatalf("encrypt master key: %v", err)
	}

	payload, err := json.Marshal(encrypted)
	if err != nil {
		t.Fatalf("marshal encrypted: %v", err)
	}

	hexPayload, err := crypto.CompressHex(payload)
	if err != nil {
		t.Fatalf("compress hex: %v", err)
	}

	parsed, err := parseEncryptedDataFromStorage([]byte(hexPayload))
	if err != nil {
		t.Fatalf("parse encrypted data: %v", err)
	}

	decrypted, err := crypto.DecryptMasterKeyWithPassword(parsed, password)
	if err != nil {
		t.Fatalf("decrypt parsed key: %v", err)
	}

	if string(decrypted) != string(masterKey) {
		t.Fatalf("decrypted key mismatch")
	}
}

func TestParseEncryptedDataFromStorage_RawJSONCompatibility(t *testing.T) {
	masterKey, err := crypto.GenerateMasterKey()
	if err != nil {
		t.Fatalf("generate master key: %v", err)
	}

	encrypted, err := crypto.EncryptMasterKeyWithPassword(masterKey, "pw")
	if err != nil {
		t.Fatalf("encrypt master key: %v", err)
	}

	payload, err := json.Marshal(encrypted)
	if err != nil {
		t.Fatalf("marshal encrypted: %v", err)
	}

	if _, err := parseEncryptedDataFromStorage(payload); err != nil {
		t.Fatalf("expected raw JSON compatibility, got error: %v", err)
	}
}

func TestReencryptBinaryPayload(t *testing.T) {
	oldKey, err := crypto.GenerateMasterKey()
	if err != nil {
		t.Fatalf("generate old key: %v", err)
	}
	newKey, err := crypto.GenerateMasterKey()
	if err != nil {
		t.Fatalf("generate new key: %v", err)
	}

	originalPlaintext := []byte("secret-value")
	encryptedHex, err := crypto.EncryptBinary(originalPlaintext, oldKey)
	if err != nil {
		t.Fatalf("encrypt with old key: %v", err)
	}

	reencrypted, err := reencryptBinaryPayload([]byte(encryptedHex), oldKey, newKey)
	if err != nil {
		t.Fatalf("reencrypt payload: %v", err)
	}

	if _, err := crypto.DecryptBinary(string(reencrypted), oldKey); err == nil {
		t.Fatalf("expected old key to fail after re-encryption")
	}

	decryptedWithNewKey, err := crypto.DecryptBinary(string(reencrypted), newKey)
	if err != nil {
		t.Fatalf("decrypt with new key: %v", err)
	}

	if string(decryptedWithNewKey) != string(originalPlaintext) {
		t.Fatalf("plaintext mismatch after re-encryption")
	}
}

func TestNormalizeResourcePath(t *testing.T) {
	if got := normalizeResourcePath("resources/a.vty"); got != "resources/a.vty" {
		t.Fatalf("unexpected resources path: %s", got)
	}
	if got := normalizeResourcePath("config/a.vty"); got != "config/a.vty" {
		t.Fatalf("unexpected config path: %s", got)
	}
	if got := normalizeResourcePath("a.vty"); got != "resources/a.vty" {
		t.Fatalf("unexpected normalized path: %s", got)
	}
}

func TestBuildRotatedUserKeys_ReencryptsForAllRemainingUsers(t *testing.T) {
	oldMasterKey, err := crypto.GenerateMasterKey()
	if err != nil {
		t.Fatalf("generate old master key: %v", err)
	}
	newMasterKey, err := crypto.GenerateMasterKey()
	if err != nil {
		t.Fatalf("generate new master key: %v", err)
	}

	aliceEncryptedPassword, err := crypto.EncryptBinary([]byte("alice-password"), oldMasterKey)
	if err != nil {
		t.Fatalf("encrypt alice password: %v", err)
	}

	users := []config.UserEntry{
		{Username: "owner"},
		{Username: "alice", EncryptedPassword: aliceEncryptedPassword},
	}

	rotatedKeys, err := buildRotatedUserKeys(users, "owner", "owner-password", oldMasterKey, newMasterKey)
	if err != nil {
		t.Fatalf("build rotated keys: %v", err)
	}

	if len(rotatedKeys) != len(users) {
		t.Fatalf("expected %d rotated keys, got %d", len(users), len(rotatedKeys))
	}

	for username, password := range map[string]string{"owner": "owner-password", "alice": "alice-password"} {
		payload, ok := rotatedKeys[username]
		if !ok {
			t.Fatalf("missing rotated key for %s", username)
		}

		keyJSON, err := crypto.DecompressHex(string(payload))
		if err != nil {
			t.Fatalf("decompress rotated key for %s: %v", username, err)
		}

		encryptedData := &crypto.EncryptedData{}
		if err := json.Unmarshal(keyJSON, encryptedData); err != nil {
			t.Fatalf("unmarshal rotated key for %s: %v", username, err)
		}

		decryptedKey, err := crypto.DecryptMasterKeyWithPassword(encryptedData, password)
		if err != nil {
			t.Fatalf("decrypt rotated key for %s: %v", username, err)
		}

		if string(decryptedKey) != string(newMasterKey) {
			t.Fatalf("rotated key mismatch for %s", username)
		}
	}
}

func TestBuildRotatedUserKeys_FailsWhenEncryptedPasswordMissing(t *testing.T) {
	oldMasterKey, err := crypto.GenerateMasterKey()
	if err != nil {
		t.Fatalf("generate old master key: %v", err)
	}
	newMasterKey, err := crypto.GenerateMasterKey()
	if err != nil {
		t.Fatalf("generate new master key: %v", err)
	}

	_, err = buildRotatedUserKeys(
		[]config.UserEntry{{Username: "owner"}, {Username: "alice"}},
		"owner",
		"owner-password",
		oldMasterKey,
		newMasterKey,
	)
	if err == nil {
		t.Fatalf("expected error when encrypted password is missing")
	}

	if !strings.Contains(err.Error(), "missing encrypted password") {
		t.Fatalf("unexpected error: %v", err)
	}
}
