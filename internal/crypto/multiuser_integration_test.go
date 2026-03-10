package crypto

import (
	"bytes"
	"testing"
)

func TestMultiUserVaultFlow(t *testing.T) {
	password1 := "owner-password"
	password2 := "editor-password"

	masterKey, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey failed: %v", err)
	}

	if len(masterKey) != 32 {
		t.Errorf("masterKey size = %d, want 32", len(masterKey))
	}

	encryptedMasterKey1, err := EncryptMasterKeyWithPassword(masterKey, password1)
	if err != nil {
		t.Fatalf("EncryptMasterKeyWithPassword failed: %v", err)
	}

	encryptedMasterKey2, err := EncryptMasterKeyWithPassword(masterKey, password2)
	if err != nil {
		t.Fatalf("EncryptMasterKeyWithPassword failed: %v", err)
	}

	decryptedMasterKey1, err := DecryptMasterKeyWithPassword(encryptedMasterKey1, password1)
	if err != nil {
		t.Fatalf("DecryptMasterKeyWithPassword failed: %v", err)
	}

	if !bytes.Equal(decryptedMasterKey1, masterKey) {
		t.Error("Decrypted masterKey1 doesn't match original")
	}

	decryptedMasterKey2, err := DecryptMasterKeyWithPassword(encryptedMasterKey2, password2)
	if err != nil {
		t.Fatalf("DecryptMasterKeyWithPassword failed: %v", err)
	}

	if !bytes.Equal(decryptedMasterKey2, masterKey) {
		t.Error("Decrypted masterKey2 doesn't match original")
	}

	secretData := []byte("my-api-key=secret123")

	encryptedVault, err := EncryptVaultData(secretData, masterKey)
	if err != nil {
		t.Fatalf("EncryptVaultData failed: %v", err)
	}

	decryptedVault1, err := DecryptVaultData(encryptedVault, decryptedMasterKey1)
	if err != nil {
		t.Fatalf("DecryptVaultData with masterKey1 failed: %v", err)
	}

	if !bytes.Equal(decryptedVault1, secretData) {
		t.Error("Decrypted vault data doesn't match original")
	}

	decryptedVault2, err := DecryptVaultData(encryptedVault, decryptedMasterKey2)
	if err != nil {
		t.Fatalf("DecryptVaultData with masterKey2 failed: %v", err)
	}

	if !bytes.Equal(decryptedVault2, secretData) {
		t.Error("Decrypted vault data doesn't match with second user")
	}

	_, err = DecryptMasterKeyWithPassword(encryptedMasterKey1, "wrong-password")
	if err == nil {
		t.Error("Expected error with wrong password")
	}
}
