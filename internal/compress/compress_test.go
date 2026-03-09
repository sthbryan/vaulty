package compress

import (
	"bytes"
	"testing"
)

func TestCompress_Decompress(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "empty data",
			data: []byte{},
		},
		{
			name: "simple text",
			data: []byte("Hello, World!"),
		},
		{
			name: "large data",
			data: bytes.Repeat([]byte("Lorem ipsum dolor sit amet. "), 1000),
		},
		{
			name: "binary data",
			data: []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD, 0xFC},
		},
		{
			name: "unicode text",
			data: []byte("Hello 世界 🌍 Ñoño"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compressed, err := Compress(tt.data)
			if err != nil {
				t.Fatalf("Compress failed: %v", err)
			}

			decompressed, err := Decompress(compressed)
			if err != nil {
				t.Fatalf("Decompress failed: %v", err)
			}

			if !bytes.Equal(tt.data, decompressed) {
				t.Errorf("Data mismatch: got %v, want %v", decompressed, tt.data)
			}
		})
	}
}

func TestCompress_ReducesSize(t *testing.T) {
	data := bytes.Repeat([]byte("This is a repetitive string. "), 1000)
	originalSize := len(data)

	compressed, err := Compress(data)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	compressedSize := len(compressed)

	if compressedSize >= originalSize {
		t.Errorf("Compression did not reduce size: original=%d, compressed=%d", originalSize, compressedSize)
	}

	decompressed, err := Decompress(compressed)
	if err != nil {
		t.Fatalf("Decompress failed: %v", err)
	}

	if !bytes.Equal(data, decompressed) {
		t.Error("Round-trip data mismatch")
	}
}

func TestDecompress_InvalidData(t *testing.T) {
	invalidData := []byte("not valid gzip data")

	_, err := Decompress(invalidData)
	if err == nil {
		t.Error("Expected error when decompressing invalid data")
	}
}
