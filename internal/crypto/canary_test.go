package crypto

import (
	"testing"
)

func TestEncryptDecryptPasswordWithSeed(t *testing.T) {
	password := "lemoande"
	seed := "return betray mother hood eager abandon hair arrange orient inherit steak nominee"

	encrypted, err := EncryptPasswordWithSeed(password, seed)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}
	t.Logf("Encrypted length: %d bytes", len(encrypted))

	decrypted, err := DecryptPasswordWithSeed(encrypted, seed)
	if err != nil {
		t.Fatalf("Decrypt error: %v", err)
	}

	if decrypted != password {
		t.Errorf("Password mismatch: expected %q, got %q", password, decrypted)
	}
}

func TestEncryptDecryptWithWrongSeed(t *testing.T) {
	password := "lemoande"
	correctSeed := "return betray mother hood eager abandon hair arrange orient inherit steak nominee"
	wrongSeed := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

	encrypted, err := EncryptPasswordWithSeed(password, correctSeed)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	_, err = DecryptPasswordWithSeed(encrypted, wrongSeed)
	if err == nil {
		t.Error("Expected error when decrypting with wrong seed, got nil")
	}
}
