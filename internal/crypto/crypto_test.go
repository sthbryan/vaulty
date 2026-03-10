package crypto

import (
	"bytes"
	"strings"
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

			for i, chunk := range encrypted.Chunks {
				if chunk.Index != uint32(i) {
					t.Errorf("Chunk %d has index %d", i, chunk.Index)
				}
				if len(chunk.IV) != IVSize {
					t.Errorf("Chunk %d IV size = %d, want %d", i, len(chunk.IV), IVSize)
				}
				if len(chunk.Ciphertext) == 0 {
					t.Errorf("Chunk %d ciphertext is empty", i)
				}
				if i == len(encrypted.Chunks)-1 {
					if !chunk.IsLast {
						t.Errorf("Chunk %d should be marked as last", i)
					}
				} else {
					if chunk.IsLast {
						t.Errorf("Chunk %d should not be marked as last", i)
					}
				}
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

func TestDecryptChunksInvalidData(t *testing.T) {
	password := "testpassword"

	tests := []struct {
		name string
		data *ChunkedEncryptedData
	}{
		{
			name: "wrong salt size",
			data: &ChunkedEncryptedData{
				Salt:   make([]byte, 16),
				Chunks: []EncryptedChunk{{Index: 0, IV: make([]byte, IVSize), Ciphertext: []byte("data")}},
			},
		},
		{
			name: "no chunks",
			data: &ChunkedEncryptedData{
				Salt:   make([]byte, SaltSize),
				Chunks: []EncryptedChunk{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecryptChunks(tt.data, password)
			if err == nil {
				t.Error("DecryptChunks() expected error for invalid data")
			}
		})
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

func TestSerializeDeserializeChunkedEncryptedData(t *testing.T) {
	plaintext := bytes.Repeat([]byte("test"), 1000)
	password := "testpassword"

	encrypted, err := EncryptWithChunks(plaintext, password, 100)
	if err != nil {
		t.Fatalf("EncryptWithChunks() error = %v", err)
	}

	serialized := SerializeChunkedEncryptedData(encrypted)
	if len(serialized) == 0 {
		t.Error("Serialized chunked data is empty")
	}

	deserialized, err := DeserializeChunkedEncryptedData(serialized)
	if err != nil {
		t.Fatalf("DeserializeChunkedEncryptedData() error = %v", err)
	}

	if !bytes.Equal(deserialized.Salt, encrypted.Salt) {
		t.Error("Deserialized salt doesn't match")
	}

	if len(deserialized.Chunks) != len(encrypted.Chunks) {
		t.Fatalf("Deserialized chunk count = %d, expected %d", len(deserialized.Chunks), len(encrypted.Chunks))
	}

	for i := range deserialized.Chunks {
		if deserialized.Chunks[i].Index != encrypted.Chunks[i].Index {
			t.Errorf("Chunk %d index mismatch", i)
		}
		if !bytes.Equal(deserialized.Chunks[i].IV, encrypted.Chunks[i].IV) {
			t.Errorf("Chunk %d IV mismatch", i)
		}
		if !bytes.Equal(deserialized.Chunks[i].Ciphertext, encrypted.Chunks[i].Ciphertext) {
			t.Errorf("Chunk %d ciphertext mismatch", i)
		}
		if deserialized.Chunks[i].IsLast != encrypted.Chunks[i].IsLast {
			t.Errorf("Chunk %d IsLast mismatch", i)
		}
	}

	decrypted, err := DecryptChunks(deserialized, password)
	if err != nil {
		t.Fatalf("DecryptChunks() error = %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Error("Decrypted data doesn't match original")
	}
}

func TestDeserializeChunkedEncryptedDataTooShort(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty data",
			data: []byte{},
		},
		{
			name: "only salt",
			data: make([]byte, SaltSize),
		},
		{
			name: "salt plus 2 bytes",
			data: make([]byte, SaltSize+2),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DeserializeChunkedEncryptedData(tt.data)
			if err == nil {
				t.Error("DeserializeChunkedEncryptedData() expected error for short data")
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

func BenchmarkDeriveKey(b *testing.B) {
	password := "benchmarkpassword"
	salt := make([]byte, SaltSize)
	for i := range salt {
		salt[i] = byte(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DeriveKey(password, salt)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncrypt(b *testing.B) {
	plaintext := bytes.Repeat([]byte("A"), 1024)
	password := "benchmarkpassword"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Encrypt(plaintext, password)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecrypt(b *testing.B) {
	plaintext := bytes.Repeat([]byte("A"), 1024)
	password := "benchmarkpassword"
	encrypted, _ := Encrypt(plaintext, password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Decrypt(encrypted, password)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncryptWithChunks(b *testing.B) {
	plaintext := bytes.Repeat([]byte("A"), 1024*1024)
	password := "benchmarkpassword"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := EncryptWithChunks(plaintext, password, DefaultChunkSize)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecryptChunks(b *testing.B) {
	plaintext := bytes.Repeat([]byte("A"), 1024*1024)
	password := "benchmarkpassword"
	encrypted, _ := EncryptWithChunks(plaintext, password, DefaultChunkSize)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DecryptChunks(encrypted, password)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestGenerateMasterKey(t *testing.T) {
	key, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey() error = %v, wantErr false", err)
	}

	if len(key) != MasterKeySize {
		t.Errorf("GenerateMasterKey() len = %d, want %d", len(key), MasterKeySize)
	}

	key2, _ := GenerateMasterKey()
	if bytes.Equal(key, key2) {
		t.Error("GenerateMasterKey() generated duplicate keys")
	}
}

func TestValidateMasterKey(t *testing.T) {
	tests := []struct {
		name    string
		key     []byte
		wantErr bool
	}{
		{
			name:    "valid 32 byte key",
			key:     make([]byte, 32),
			wantErr: false,
		},
		{
			name:    "invalid 16 byte key",
			key:     make([]byte, 16),
			wantErr: true,
		},
		{
			name:    "invalid 64 byte key",
			key:     make([]byte, 64),
			wantErr: true,
		},
		{
			name:    "invalid empty key",
			key:     []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMasterKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMasterKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEncryptMasterKeyWithPassword(t *testing.T) {
	password := "testpassword123"
	masterKey, _ := GenerateMasterKey()

	encrypted, err := EncryptMasterKeyWithPassword(masterKey, password)
	if err != nil {
		t.Fatalf("EncryptMasterKeyWithPassword() error = %v", err)
	}

	if encrypted.Salt == nil || len(encrypted.Salt) == 0 {
		t.Error("EncryptMasterKeyWithPassword() salt is empty")
	}

	if encrypted.IV == nil || len(encrypted.IV) == 0 {
		t.Error("EncryptMasterKeyWithPassword() IV is empty")
	}

	if encrypted.Ciphertext == nil || len(encrypted.Ciphertext) == 0 {
		t.Error("EncryptMasterKeyWithPassword() ciphertext is empty")
	}
}

func TestEncryptMasterKeyWithPassword_InvalidKey(t *testing.T) {
	password := "testpassword123"
	invalidKey := make([]byte, 16)

	_, err := EncryptMasterKeyWithPassword(invalidKey, password)
	if err == nil {
		t.Error("EncryptMasterKeyWithPassword() with invalid key should error")
	}
}

func TestDecryptMasterKeyWithPassword(t *testing.T) {
	password := "testpassword123"
	originalKey, _ := GenerateMasterKey()

	encrypted, _ := EncryptMasterKeyWithPassword(originalKey, password)
	decrypted, err := DecryptMasterKeyWithPassword(encrypted, password)

	if err != nil {
		t.Fatalf("DecryptMasterKeyWithPassword() error = %v", err)
	}

	if !bytes.Equal(originalKey, decrypted) {
		t.Error("DecryptMasterKeyWithPassword() decrypted key does not match original")
	}
}

func TestDecryptMasterKeyWithPassword_WrongPassword(t *testing.T) {
	password := "testpassword123"
	originalKey, _ := GenerateMasterKey()

	encrypted, _ := EncryptMasterKeyWithPassword(originalKey, password)
	_, err := DecryptMasterKeyWithPassword(encrypted, "wrongpassword")

	if err == nil {
		t.Error("DecryptMasterKeyWithPassword() with wrong password should error")
	}
}

func TestEncryptVaultData(t *testing.T) {
	masterKey, _ := GenerateMasterKey()
	plaintext := []byte("sensitive vault data")

	encrypted, err := EncryptVaultData(plaintext, masterKey)
	if err != nil {
		t.Fatalf("EncryptVaultData() error = %v", err)
	}

	if encrypted.IV == nil || len(encrypted.IV) == 0 {
		t.Error("EncryptVaultData() IV is empty")
	}

	if encrypted.Ciphertext == nil || len(encrypted.Ciphertext) == 0 {
		t.Error("EncryptVaultData() ciphertext is empty")
	}
}

func TestEncryptVaultData_InvalidKey(t *testing.T) {
	invalidKey := make([]byte, 16)
	plaintext := []byte("sensitive data")

	_, err := EncryptVaultData(plaintext, invalidKey)
	if err == nil {
		t.Error("EncryptVaultData() with invalid key should error")
	}
}

func TestDecryptVaultData(t *testing.T) {
	masterKey, _ := GenerateMasterKey()
	plaintext := []byte("sensitive vault data")

	encrypted, _ := EncryptVaultData(plaintext, masterKey)
	decrypted, err := DecryptVaultData(encrypted, masterKey)

	if err != nil {
		t.Fatalf("DecryptVaultData() error = %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Error("DecryptVaultData() decrypted data does not match original")
	}
}

func TestDecryptVaultData_WrongKey(t *testing.T) {
	masterKey, _ := GenerateMasterKey()
	wrongKey, _ := GenerateMasterKey()
	plaintext := []byte("sensitive vault data")

	encrypted, _ := EncryptVaultData(plaintext, masterKey)
	_, err := DecryptVaultData(encrypted, wrongKey)

	if err == nil {
		t.Error("DecryptVaultData() with wrong key should error")
	}
}

func TestMasterKeyRoundTrip(t *testing.T) {
	password := "complexpassword123!@#"
	originalKey, _ := GenerateMasterKey()

	encrypted, _ := EncryptMasterKeyWithPassword(originalKey, password)
	decrypted, _ := DecryptMasterKeyWithPassword(encrypted, password)
	vaultData := []byte("test data")
	encryptedVault, _ := EncryptVaultData(vaultData, decrypted)
	decryptedVault, _ := DecryptVaultData(encryptedVault, decrypted)

	if !bytes.Equal(originalKey, decrypted) {
		t.Error("master key mismatch after encryption/decryption")
	}

	if !bytes.Equal(vaultData, decryptedVault) {
		t.Error("vault data mismatch after encryption/decryption")
	}
}

func TestGeneratePasswordChallenge(t *testing.T) {
	password := "testpassword123"
	challenge, err := GeneratePasswordChallenge(password)
	if err != nil {
		t.Fatalf("GeneratePasswordChallenge() error = %v", err)
	}

	if challenge == "" {
		t.Error("GeneratePasswordChallenge() returned empty challenge")
	}

	if !strings.Contains(challenge, ":") {
		t.Error("GeneratePasswordChallenge() challenge missing colon separator")
	}
}

func TestValidatePasswordChallenge(t *testing.T) {
	password := "testpassword123"
	challenge, _ := GeneratePasswordChallenge(password)

	if !ValidatePasswordChallenge(password, challenge) {
		t.Error("ValidatePasswordChallenge() failed for correct password")
	}

	if ValidatePasswordChallenge("wrongpassword", challenge) {
		t.Error("ValidatePasswordChallenge() should fail for wrong password")
	}

	if ValidatePasswordChallenge(password, "invalid:challenge") {
		t.Error("ValidatePasswordChallenge() should fail for invalid challenge format")
	}
}

func TestValidatePasswordChallenge_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		challenge string
		want      bool
	}{
		{
			name:      "empty challenge",
			password:  "test",
			challenge: "",
			want:      false,
		},
		{
			name:      "missing colon",
			password:  "test",
			challenge: "abcdef",
			want:      false,
		},
		{
			name:      "colon at start",
			password:  "test",
			challenge: ":abcdef",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidatePasswordChallenge(tt.password, tt.challenge)
			if got != tt.want {
				t.Errorf("ValidatePasswordChallenge() = %v, want %v", got, tt.want)
			}
		})
	}
}
