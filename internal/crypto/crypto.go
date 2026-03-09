package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	SaltSize         = 32
	IVSize           = 12
	KeySize          = 32
	DefaultChunkSize = 64 * 1024 // 64KB
	PBKDF2Iterations = 100000
)

var (
	ErrInvalidSaltSize   = errors.New("invalid salt size")
	ErrInvalidIVSize     = errors.New("invalid IV size")
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
	ErrDecryptionFailed  = errors.New("decryption failed")
	ErrChunkTooSmall     = errors.New("chunk size too small")
	ErrInvalidChunkData  = errors.New("invalid chunk data")
)

// EncryptedData represents the full encrypted data structure
type EncryptedData struct {
	Salt       []byte `json:"salt"`
	IV         []byte `json:"iv"`
	Ciphertext []byte `json:"ciphertext"`
}

// EncryptedChunk represents a single encrypted chunk
type EncryptedChunk struct {
	Index      uint32 `json:"index"`
	IV         []byte `json:"iv"`
	Ciphertext []byte `json:"ciphertext"`
	IsLast     bool   `json:"is_last"`
}

// ChunkedEncryptedData represents data encrypted in chunks
type ChunkedEncryptedData struct {
	Salt   []byte           `json:"salt"`
	Chunks []EncryptedChunk `json:"chunks"`
}

// pbkdf2Key derives a key from password and salt using PBKDF2 with HMAC-SHA256
// Implementation using only standard library packages
func pbkdf2Key(password, salt []byte, iter, keyLen int) []byte {
	prf := hmac.New(sha256.New, password)
	dkLen := keyLen
	prfLen := prf.Size()
	numBlocks := (dkLen + prfLen - 1) / prfLen

	var buf [4]byte
	dk := make([]byte, 0, numBlocks*prfLen)
	u := make([]byte, prfLen)
	for block := 1; block <= numBlocks; block++ {
		prf.Reset()
		prf.Write(salt)
		buf[0] = byte(block >> 24)
		buf[1] = byte(block >> 16)
		buf[2] = byte(block >> 8)
		buf[3] = byte(block)
		prf.Write(buf[:4])
		dk = prf.Sum(dk)
		t := dk[len(dk)-prfLen:]
		copy(u, t)

		for n := 2; n <= iter; n++ {
			prf.Reset()
			prf.Write(u)
			u = u[:0]
			u = prf.Sum(u)
			for x := range u {
				t[x] ^= u[x]
			}
		}
	}
	return dk[:dkLen]
}

// DeriveKey derives a 256-bit key from a password and salt using PBKDF2
func DeriveKey(password string, salt []byte) ([]byte, error) {
	if len(salt) != SaltSize {
		return nil, ErrInvalidSaltSize
	}

	key := pbkdf2Key([]byte(password), salt, PBKDF2Iterations, KeySize)
	return key, nil
}

// Encrypt encrypts plaintext using AES-256-GCM with a derived key
func Encrypt(plaintext []byte, password string) (*EncryptedData, error) {
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	iv := make([]byte, IVSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("failed to generate IV: %w", err)
	}

	key, err := DeriveKey(password, salt)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	ciphertext := aead.Seal(nil, iv, plaintext, nil)

	return &EncryptedData{
		Salt:       salt,
		IV:         iv,
		Ciphertext: ciphertext,
	}, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM with a derived key
func Decrypt(data *EncryptedData, password string) ([]byte, error) {
	if len(data.Salt) != SaltSize {
		return nil, ErrInvalidSaltSize
	}

	if len(data.IV) != IVSize {
		return nil, ErrInvalidIVSize
	}

	if len(data.Ciphertext) == 0 {
		return nil, ErrInvalidCiphertext
	}

	key, err := DeriveKey(password, data.Salt)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := aead.Open(nil, data.IV, data.Ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// EncryptWithChunks encrypts plaintext in chunks using AES-256-GCM
func EncryptWithChunks(plaintext []byte, password string, chunkSize int) (*ChunkedEncryptedData, error) {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}

	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	key, err := DeriveKey(password, salt)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	var chunks []EncryptedChunk
	totalLen := len(plaintext)
	numChunks := (totalLen + chunkSize - 1) / chunkSize

	for i := 0; i < numChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > totalLen {
			end = totalLen
		}

		iv := make([]byte, IVSize)
		if _, err := io.ReadFull(rand.Reader, iv); err != nil {
			return nil, fmt.Errorf("failed to generate IV for chunk %d: %w", i, err)
		}

		ciphertext := aead.Seal(nil, iv, plaintext[start:end], nil)

		chunk := EncryptedChunk{
			Index:      uint32(i),
			IV:         iv,
			Ciphertext: ciphertext,
			IsLast:     i == numChunks-1,
		}
		chunks = append(chunks, chunk)
	}

	return &ChunkedEncryptedData{
		Salt:   salt,
		Chunks: chunks,
	}, nil
}

// DecryptChunks decrypts chunked encrypted data
func DecryptChunks(data *ChunkedEncryptedData, password string) ([]byte, error) {
	if len(data.Salt) != SaltSize {
		return nil, ErrInvalidSaltSize
	}

	if len(data.Chunks) == 0 {
		return nil, ErrInvalidChunkData
	}

	key, err := DeriveKey(password, data.Salt)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	var plaintext []byte
	for i, chunk := range data.Chunks {
		if len(chunk.IV) != IVSize {
			return nil, fmt.Errorf("invalid IV size for chunk %d", i)
		}

		if len(chunk.Ciphertext) == 0 {
			return nil, fmt.Errorf("empty ciphertext for chunk %d", i)
		}

		chunkPlaintext, err := aead.Open(nil, chunk.IV, chunk.Ciphertext, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt chunk %d: %w", i, ErrDecryptionFailed)
		}

		plaintext = append(plaintext, chunkPlaintext...)
	}

	return plaintext, nil
}

// SerializeEncryptedData serializes EncryptedData to bytes
func SerializeEncryptedData(data *EncryptedData) []byte {
	result := make([]byte, 0, SaltSize+IVSize+len(data.Ciphertext))
	result = append(result, data.Salt...)
	result = append(result, data.IV...)
	result = append(result, data.Ciphertext...)
	return result
}

// DeserializeEncryptedData deserializes bytes to EncryptedData
func DeserializeEncryptedData(data []byte) (*EncryptedData, error) {
	if len(data) < SaltSize+IVSize {
		return nil, errors.New("data too short")
	}

	return &EncryptedData{
		Salt:       data[:SaltSize],
		IV:         data[SaltSize : SaltSize+IVSize],
		Ciphertext: data[SaltSize+IVSize:],
	}, nil
}

// SerializeChunkedEncryptedData serializes ChunkedEncryptedData to bytes
func SerializeChunkedEncryptedData(data *ChunkedEncryptedData) []byte {
	var result []byte

	result = append(result, data.Salt...)

	numChunks := uint32(len(data.Chunks))
	numChunksBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(numChunksBytes, numChunks)
	result = append(result, numChunksBytes...)

	for _, chunk := range data.Chunks {
		indexBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(indexBytes, chunk.Index)
		result = append(result, indexBytes...)

		result = append(result, chunk.IV...)

		ciphertextLen := uint32(len(chunk.Ciphertext))
		ciphertextLenBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(ciphertextLenBytes, ciphertextLen)
		result = append(result, ciphertextLenBytes...)

		result = append(result, chunk.Ciphertext...)

		if chunk.IsLast {
			result = append(result, 1)
		} else {
			result = append(result, 0)
		}
	}

	return result
}

// DeserializeChunkedEncryptedData deserializes bytes to ChunkedEncryptedData
func DeserializeChunkedEncryptedData(data []byte) (*ChunkedEncryptedData, error) {
	if len(data) < SaltSize+4 {
		return nil, errors.New("data too short")
	}

	offset := 0
	salt := data[offset : offset+SaltSize]
	offset += SaltSize

	numChunks := binary.BigEndian.Uint32(data[offset : offset+4])
	offset += 4

	var chunks []EncryptedChunk
	for i := uint32(0); i < numChunks; i++ {
		if offset+4 > len(data) {
			return nil, errors.New("invalid chunk data")
		}

		index := binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4

		if offset+IVSize > len(data) {
			return nil, errors.New("invalid chunk IV")
		}
		iv := data[offset : offset+IVSize]
		offset += IVSize

		if offset+4 > len(data) {
			return nil, errors.New("invalid chunk length")
		}
		ciphertextLen := binary.BigEndian.Uint32(data[offset : offset+4])
		offset += 4

		if offset+int(ciphertextLen) > len(data) {
			return nil, errors.New("chunk ciphertext truncated")
		}
		ciphertext := data[offset : offset+int(ciphertextLen)]
		offset += int(ciphertextLen)

		if offset+1 > len(data) {
			return nil, errors.New("missing is_last flag")
		}
		isLast := data[offset] == 1
		offset++

		chunks = append(chunks, EncryptedChunk{
			Index:      index,
			IV:         iv,
			Ciphertext: ciphertext,
			IsLast:     isLast,
		})
	}

	return &ChunkedEncryptedData{
		Salt:   salt,
		Chunks: chunks,
	}, nil
}

// GenerateSalt generates a cryptographically secure random salt
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// GenerateIV generates a cryptographically secure random IV
func GenerateIV() ([]byte, error) {
	iv := make([]byte, IVSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("failed to generate IV: %w", err)
	}
	return iv, nil
}

// EncryptWithKey encrypts plaintext using AES-256-GCM with a provided key
func EncryptWithKey(plaintext, key []byte) (*EncryptedData, error) {
	if len(key) != KeySize {
		return nil, errors.New("key must be 32 bytes")
	}

	iv, err := GenerateIV()
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	ciphertext := aead.Seal(nil, iv, plaintext, nil)

	return &EncryptedData{
		IV:         iv,
		Ciphertext: ciphertext,
	}, nil
}

// DecryptWithKey decrypts ciphertext using AES-256-GCM with a provided key
func DecryptWithKey(data *EncryptedData, key []byte) ([]byte, error) {
	if len(key) != KeySize {
		return nil, errors.New("key must be 32 bytes")
	}

	if len(data.IV) != IVSize {
		return nil, ErrInvalidIVSize
	}

	if len(data.Ciphertext) == 0 {
		return nil, ErrInvalidCiphertext
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := aead.Open(nil, data.IV, data.Ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}
