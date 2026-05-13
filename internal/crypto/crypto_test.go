package crypto

import (
	"bytes"
	"testing"
)

func TestDeriveKey(t *testing.T) {
	tests := []struct {
		name     string
		password string
		saltSize int
		wantErr  bool
	}{
		{
			name:     "valid derivation",
			password: "testpassword123",
			saltSize: SaltSize,
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			saltSize: SaltSize,
			wantErr:  false,
		},
		{
			name:     "long password",
			password: "thisisaverylongpasswordthatexceedsthenormallengthforpasswords123456789",
			saltSize: SaltSize,
			wantErr:  false,
		},
		{
			name:     "invalid salt size too small",
			password: "testpassword",
			saltSize: 16,
			wantErr:  true,
		},
		{
			name:     "invalid salt size too large",
			password: "testpassword",
			saltSize: 64,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			salt := make([]byte, tt.saltSize)
			for i := range salt {
				salt[i] = byte(i)
			}

			key, err := DeriveKey(tt.password, salt)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeriveKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(key) != KeySize {
					t.Errorf("DeriveKey() returned key of size %d, want %d", len(key), KeySize)
				}

				key2, _ := DeriveKey(tt.password, salt)
				if !bytes.Equal(key, key2) {
					t.Error("DeriveKey() not deterministic for same password and salt")
				}

				differentSalt := make([]byte, SaltSize)
				differentSalt[0] = 0xFF
				key3, _ := DeriveKey(tt.password, differentSalt)
				if bytes.Equal(key, key3) {
					t.Error("DeriveKey() returned same key for different salt")
				}
			}
		})
	}
}

func TestEncryptDecrypt(t *testing.T) {
	tests := []struct {
		name      string
		plaintext []byte
		password  string
	}{
		{
			name:      "simple text",
			plaintext: []byte("Hello, World!"),
			password:  "mypassword",
		},
		{
			name:      "empty data",
			plaintext: []byte{},
			password:  "mypassword",
		},
		{
			name:      "large data",
			plaintext: bytes.Repeat([]byte("A"), 1024*1024),
			password:  "mypassword",
		},
		{
			name:      "binary data",
			plaintext: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
			password:  "mypassword",
		},
		{
			name:      "unicode text",
			plaintext: []byte("Hello 世界 🌍 नमस्ते"),
			password:  "mypassword",
		},
		{
			name:      "special characters password",
			plaintext: []byte("secret data"),
			password:  "p@$$w0rd!#$%^&*()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := Encrypt(tt.plaintext, tt.password)
			if err != nil {
				t.Fatalf("Encrypt() error = %v", err)
			}

			if len(encrypted.Salt) != SaltSize {
				t.Errorf("Encrypt() salt size = %d, want %d", len(encrypted.Salt), SaltSize)
			}
			if len(encrypted.IV) != IVSize {
				t.Errorf("Encrypt() IV size = %d, want %d", len(encrypted.IV), IVSize)
			}
			if len(encrypted.Ciphertext) == 0 {
				t.Error("Encrypt() ciphertext is empty")
			}

			decrypted, err := Decrypt(encrypted, tt.password)
			if err != nil {
				t.Fatalf("Decrypt() error = %v", err)
			}

			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Error("Decrypt() returned different data than original")
			}
		})
	}
}

func TestDecryptWrongPassword(t *testing.T) {
	plaintext := []byte("secret message")
	password := "correctpassword"
	wrongPassword := "wrongpassword"

	encrypted, err := Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	_, err = Decrypt(encrypted, wrongPassword)
	if err != ErrDecryptionFailed {
		t.Errorf("Decrypt() with wrong password error = %v, want ErrDecryptionFailed", err)
	}
}

func TestDecryptInvalidData(t *testing.T) {
	password := "testpassword"

	tests := []struct {
		name string
		data *EncryptedData
	}{
		{
			name: "wrong salt size",
			data: &EncryptedData{
				Salt:       make([]byte, 16),
				IV:         make([]byte, IVSize),
				Ciphertext: []byte("ciphertext"),
			},
		},
		{
			name: "wrong IV size",
			data: &EncryptedData{
				Salt:       make([]byte, SaltSize),
				IV:         make([]byte, 16),
				Ciphertext: []byte("ciphertext"),
			},
		},
		{
			name: "empty ciphertext",
			data: &EncryptedData{
				Salt:       make([]byte, SaltSize),
				IV:         make([]byte, IVSize),
				Ciphertext: []byte{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Decrypt(tt.data, password)
			if err == nil {
				t.Error("Decrypt() expected error for invalid data")
			}
		})
	}
}

func TestEncryptWithChunks(t *testing.T) {
	tests := []struct {
		name      string
		plaintext []byte
		chunkSize int
	}{
		{
			name:      "small data single chunk",
			plaintext: []byte("Hello, World!"),
			chunkSize: DefaultChunkSize,
		},
		{
			name:      "exact chunk size",
			plaintext: bytes.Repeat([]byte("A"), DefaultChunkSize),
			chunkSize: DefaultChunkSize,
		},
		{
			name:      "multiple chunks",
			plaintext: bytes.Repeat([]byte("B"), DefaultChunkSize*3+100),
			chunkSize: DefaultChunkSize,
		},
		{
			name:      "custom chunk size",
			plaintext: bytes.Repeat([]byte("C"), 1000),
			chunkSize: 100,
		},
		{
			name:      "empty data",
			plaintext: []byte{},
			chunkSize: DefaultChunkSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password := "testpassword"
			encrypted, err := EncryptWithChunks(tt.plaintext, password, tt.chunkSize)
			if err != nil {
				t.Fatalf("EncryptWithChunks() error = %v", err)
			}

			if len(encrypted.Salt) != SaltSize {
				t.Errorf("EncryptWithChunks() salt size = %d, want %d", len(encrypted.Salt), SaltSize)
			}

			if len(tt.plaintext) == 0 {
				if len(encrypted.Chunks) != 0 {
					t.Error("EncryptWithChunks() should return no chunks for empty data")
				}
				return
			}

			expectedChunks := (len(tt.plaintext) + tt.chunkSize - 1) / tt.chunkSize
			if len(encrypted.Chunks) != expectedChunks {
				t.Errorf("EncryptWithChunks() chunk count = %d, want %d", len(encrypted.Chunks), expectedChunks)
			}

			decrypted, err := DecryptChunks(encrypted, password)
			if err != nil {
				t.Fatalf("DecryptChunks() error = %v", err)
			}

			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Error("DecryptChunks() returned different data than original")
			}
		})
	}
}

func TestDecryptChunksWrongPassword(t *testing.T) {
	plaintext := bytes.Repeat([]byte("test data"), 1000)
	password := "correctpassword"
	wrongPassword := "wrongpassword"

	encrypted, err := EncryptWithChunks(plaintext, password, 100)
	if err != nil {
		t.Fatalf("EncryptWithChunks() error = %v", err)
	}

	_, err = DecryptChunks(encrypted, wrongPassword)
	if err == nil {
		t.Error("DecryptChunks() expected error with wrong password")
	}
}

func TestSerializeDeserializeEncryptedData(t *testing.T) {
	plaintext := []byte("test data for serialization")
	password := "testpassword"

	encrypted, err := Encrypt(plaintext, password)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	serialized := SerializeEncryptedData(encrypted)
	if len(serialized) != SaltSize+IVSize+len(encrypted.Ciphertext) {
		t.Errorf("Serialized data size = %d, expected %d", len(serialized), SaltSize+IVSize+len(encrypted.Ciphertext))
	}

	deserialized, err := DeserializeEncryptedData(serialized)
	if err != nil {
		t.Fatalf("DeserializeEncryptedData() error = %v", err)
	}

	if !bytes.Equal(deserialized.Salt, encrypted.Salt) {
		t.Error("Deserialized salt doesn't match")
	}
	if !bytes.Equal(deserialized.IV, encrypted.IV) {
		t.Error("Deserialized IV doesn't match")
	}
	if !bytes.Equal(deserialized.Ciphertext, encrypted.Ciphertext) {
		t.Error("Deserialized ciphertext doesn't match")
	}

	decrypted, err := Decrypt(deserialized, password)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Error("Decrypted data doesn't match original")
	}
}

func TestDeserializeEncryptedDataTooShort(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty data",
			data: []byte{},
		},
		{
			name: "too short",
			data: make([]byte, SaltSize+IVSize-1),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DeserializeEncryptedData(tt.data)
			if err == nil {
				t.Error("DeserializeEncryptedData() expected error for short data")
			}
		})
	}
}

func TestGenerateSalt(t *testing.T) {
	salt1, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() error = %v", err)
	}
	if len(salt1) != SaltSize {
		t.Errorf("GenerateSalt() size = %d, want %d", len(salt1), SaltSize)
	}

	salt2, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() error = %v", err)
	}

	if bytes.Equal(salt1, salt2) {
		t.Error("GenerateSalt() returned same salt twice")
	}
}

func TestGenerateIV(t *testing.T) {
	iv1, err := GenerateIV()
	if err != nil {
		t.Fatalf("GenerateIV() error = %v", err)
	}
	if len(iv1) != IVSize {
		t.Errorf("GenerateIV() size = %d, want %d", len(iv1), IVSize)
	}

	iv2, err := GenerateIV()
	if err != nil {
		t.Fatalf("GenerateIV() error = %v", err)
	}

	if bytes.Equal(iv1, iv2) {
		t.Error("GenerateIV() returned same IV twice")
	}
}

func TestEncryptWithKeyDecryptWithKey(t *testing.T) {
	tests := []struct {
		name      string
		plaintext []byte
	}{
		{
			name:      "simple text",
			plaintext: []byte("Hello, World!"),
		},
		{
			name:      "large data",
			plaintext: bytes.Repeat([]byte("A"), 10000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, KeySize)
			for i := range key {
				key[i] = byte(i)
			}

			encrypted, err := EncryptWithKey(tt.plaintext, key)
			if err != nil {
				t.Fatalf("EncryptWithKey() error = %v", err)
			}

			if len(encrypted.IV) != IVSize {
				t.Errorf("EncryptWithKey() IV size = %d, want %d", len(encrypted.IV), IVSize)
			}
			if len(encrypted.Ciphertext) == 0 {
				t.Error("EncryptWithKey() ciphertext is empty")
			}

			decrypted, err := DecryptWithKey(encrypted, key)
			if err != nil {
				t.Fatalf("DecryptWithKey() error = %v", err)
			}

			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Error("DecryptWithKey() returned different data than original")
			}
		})
	}
}

func TestEncryptWithKeyWrongKeySize(t *testing.T) {
	plaintext := []byte("test")
	key := make([]byte, 16)

	_, err := EncryptWithKey(plaintext, key)
	if err == nil {
		t.Error("EncryptWithKey() expected error for wrong key size")
	}
}

func TestDecryptWithKeyWrongKey(t *testing.T) {
	plaintext := []byte("secret")
	key := make([]byte, KeySize)
	wrongKey := make([]byte, KeySize)
	wrongKey[0] = 0xFF

	encrypted, err := EncryptWithKey(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptWithKey() error = %v", err)
	}

	_, err = DecryptWithKey(encrypted, wrongKey)
	if err != ErrDecryptionFailed {
		t.Errorf("DecryptWithKey() error = %v, want ErrDecryptionFailed", err)
	}
}

func TestConstants(t *testing.T) {
	if SaltSize != 32 {
		t.Errorf("SaltSize = %d, want 32", SaltSize)
	}
	if IVSize != 12 {
		t.Errorf("IVSize = %d, want 12", IVSize)
	}
	if KeySize != 32 {
		t.Errorf("KeySize = %d, want 32", KeySize)
	}
	if DefaultChunkSize != 64*1024 {
		t.Errorf("DefaultChunkSize = %d, want 65536", DefaultChunkSize)
	}
	if PBKDF2Iterations != 100000 {
		t.Errorf("PBKDF2Iterations = %d, want 100000", PBKDF2Iterations)
	}
}

func TestErrors(t *testing.T) {
	if ErrInvalidSaltSize == nil {
		t.Error("ErrInvalidSaltSize is nil")
	}
	if ErrInvalidIVSize == nil {
		t.Error("ErrInvalidIVSize is nil")
	}
	if ErrInvalidCiphertext == nil {
		t.Error("ErrInvalidCiphertext is nil")
	}
	if ErrDecryptionFailed == nil {
		t.Error("ErrDecryptionFailed is nil")
	}
}