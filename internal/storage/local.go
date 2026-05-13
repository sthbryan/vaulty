package storage

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sthbryan/vaulty/v2/pkg/models"
)

type LocalStorage struct {
	basePath string
}

func NewLocalStorage(path string) *LocalStorage {
	return &LocalStorage{basePath: path}
}

func (s *LocalStorage) Ping(ctx context.Context) error {
	if err := os.MkdirAll(s.basePath, 0755); err != nil {
		return fmt.Errorf("creating storage directory: %w", err)
	}
	return nil
}

func (s *LocalStorage) Upload(ctx context.Context, path string, data []byte) error {
	fullPath := filepath.Join(s.basePath, path)

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	if err := os.WriteFile(fullPath, data, 0600); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	return nil
}

func (s *LocalStorage) Download(ctx context.Context, path string) ([]byte, error) {
	fullPath := filepath.Join(s.basePath, path)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("reading file: %w", err)
	}

	return data, nil
}

func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(s.basePath, path)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", path)
		}
		return fmt.Errorf("deleting file: %w", err)
	}

	return nil
}

func (s *LocalStorage) List(ctx context.Context, prefix string) ([]string, error) {
	fullPrefix := filepath.Join(s.basePath, prefix)

	entries, err := os.ReadDir(fullPrefix)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	var files []string
	for _, entry := range entries {
		relPath, _ := filepath.Rel(s.basePath, filepath.Join(fullPrefix, entry.Name()))
		files = append(files, relPath)
	}

	return files, nil
}

func (s *LocalStorage) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(s.basePath, path)

	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("checking file: %w", err)
	}

	return true, nil
}

func (s *LocalStorage) GetVaultMetaPath() string {
	return filepath.Join(s.basePath, "vault.meta")
}

func (s *LocalStorage) SaveVaultMeta(meta *models.VaultConfig) error {
	metaPath := s.GetVaultMetaPath()

	data, err := os.ReadFile(metaPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading vault meta: %w", err)
	}

	if err := os.WriteFile(metaPath, data, 0600); err != nil {
		return fmt.Errorf("writing vault meta: %w", err)
	}

	return nil
}

func BuildSecretPath(secretType models.SecretType, env, name string) string {
	if env == "" {
		env = "default"
	}

	baseName := name
	if strings.HasSuffix(name, ".vty") {
		baseName = name
	} else {
		baseName = name + ".vty"
	}

	return filepath.Join(secretType.FolderName(), env, baseName)
}

func ParseSecretPath(path string) (models.SecretType, string, string, error) {
	parts := strings.Split(path, string(filepath.Separator))
	if len(parts) < 3 {
		return "", "", "", fmt.Errorf("invalid secret path: %s", path)
	}

	secretType := models.SecretType(parts[0])
	if !secretType.IsValid() {
		return "", "", "", fmt.Errorf("invalid secret type: %s", parts[0])
	}

	env := parts[1]
	name := strings.TrimSuffix(parts[2], ".vty")

	return secretType, env, name, nil
}

func CompressDirectory(dirPath string) ([]byte, error) {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	dirName := filepath.Base(dirPath)

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(filepath.Dir(dirPath), path)
		if err != nil {
			return err
		}

		relPath = filepath.Join(dirName, relPath)
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if _, err := tw.Write(data); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	if err := gzw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func DecompressDirectory(data []byte, destPath string) error {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destPath, header.Name)

		if strings.Contains(target, "..") {
			return fmt.Errorf("path traversal not allowed: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			parentDir := filepath.Dir(target)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return err
			}

			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

func ComputeChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
