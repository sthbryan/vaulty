package cli

import (
	"fmt"
	"strings"
	"unicode"
)

func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("name cannot contain path separators")
	}

	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("name cannot start with a dot")
	}

	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return fmt.Errorf("name must contain only letters, numbers, hyphens, and underscores")
		}
	}

	return nil
}

func ValidateFilePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("path cannot be empty")
	}
	return nil
}
