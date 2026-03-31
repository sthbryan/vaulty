package compress

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/gzip"
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

func TestDecompressDirectory_RejectsTraversal(t *testing.T) {
	tests := []struct {
		name      string
		entryName string
	}{
		{name: "absolute path", entryName: "/tmp/evil.txt"},
		{name: "parent traversal", entryName: "../evil.txt"},
		{name: "nested traversal", entryName: "dir/../../evil.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archive := buildTarGz(t, []tarEntry{{name: tt.entryName, body: "bad"}})
			dest := t.TempDir()

			err := DecompressDirectory(archive, dest)
			if err == nil {
				t.Fatalf("expected error for entry %q", tt.entryName)
			}
		})
	}
}

func TestDecompressDirectory_AllowsValidArchive(t *testing.T) {
	archive := buildTarGz(t, []tarEntry{
		{name: "project", typeflag: tar.TypeDir, mode: 0755},
		{name: "project/readme.txt", body: "hello"},
	})
	dest := t.TempDir()

	if err := DecompressDirectory(archive, dest); err != nil {
		t.Fatalf("DecompressDirectory failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dest, "project", "readme.txt"))
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected extracted content: %q", string(data))
	}
}

type tarEntry struct {
	name     string
	body     string
	typeflag byte
	mode     int64
}

func buildTarGz(t *testing.T, entries []tarEntry) []byte {
	t.Helper()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	for _, entry := range entries {
		typeflag := entry.typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}
		mode := entry.mode
		if mode == 0 {
			if typeflag == tar.TypeDir {
				mode = 0755
			} else {
				mode = 0644
			}
		}

		header := &tar.Header{
			Name:     entry.name,
			Typeflag: typeflag,
			Mode:     mode,
		}
		if typeflag == tar.TypeReg {
			header.Size = int64(len(entry.body))
		}

		if err := tw.WriteHeader(header); err != nil {
			t.Fatalf("failed to write header: %v", err)
		}
		if typeflag == tar.TypeReg {
			if _, err := tw.Write([]byte(entry.body)); err != nil {
				t.Fatalf("failed to write body: %v", err)
			}
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("failed to close tar writer: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("failed to close gzip writer: %v", err)
	}

	return buf.Bytes()
}
