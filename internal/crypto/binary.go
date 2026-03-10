package crypto

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"fmt"
	"io"
)

func EncryptBinary(jsonData []byte, masterKey []byte) (string, error) {
	var compressedBuf bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressedBuf)
	if _, err := gzipWriter.Write(jsonData); err != nil {
		return "", fmt.Errorf("failed to gzip compress: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return "", fmt.Errorf("failed to close gzip writer: %w", err)
	}
	compressedData := compressedBuf.Bytes()

	encryptedData, err := EncryptWithKey(compressedData, masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt: %w", err)
	}

	serialized := SerializeEncryptedData(encryptedData)
	hexString := hex.EncodeToString(serialized)

	return hexString, nil
}

func DecryptBinary(hexData string, masterKey []byte) ([]byte, error) {
	serialized, err := hex.DecodeString(hexData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex: %w", err)
	}

	encryptedData, err := DeserializeEncryptedData(serialized)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize data: %w", err)
	}

	compressedData, err := DecryptWithKey(encryptedData, masterKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	jsonData, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress gzip: %w", err)
	}

	return jsonData, nil
}

func CompressHex(data []byte) (string, error) {
	var compressedBuf bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressedBuf)
	if _, err := gzipWriter.Write(data); err != nil {
		return "", fmt.Errorf("failed to gzip compress: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return "", fmt.Errorf("failed to close gzip writer: %w", err)
	}
	return hex.EncodeToString(compressedBuf.Bytes()), nil
}

func DecompressHex(hexStr string) ([]byte, error) {
	compressedData, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex: %w", err)
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	data, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress gzip: %w", err)
	}

	return data, nil
}
