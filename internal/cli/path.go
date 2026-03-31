package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

func ResolveOutputPath(outputPath, defaultName string) (string, error) {
	if outputPath == "" {
		outputPath = defaultName
	}

	if filepath.IsAbs(outputPath) {
		return outputPath, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	return filepath.Join(cwd, outputPath), nil
}

func EnsureParentDir(path string) error {
	parentDir := filepath.Dir(path)
	if parentDir == "" || parentDir == "." {
		return nil
	}
	return os.MkdirAll(parentDir, 0755)
}
