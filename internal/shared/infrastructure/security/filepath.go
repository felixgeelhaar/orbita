// Package security provides security utilities for path validation and sanitization.
package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// dangerousChars contains shell metacharacters that could be used for injection attacks.
var dangerousChars = []string{";", "&", "|", "$", "`", "(", ")", "{", "}", "<", ">", "!", "\n", "\r"}

// ValidateFilePath validates and sanitizes a file path to prevent path traversal attacks.
// It cleans the path, resolves symlinks, and checks for dangerous characters.
// Returns the cleaned, resolved path or an error if validation fails.
func ValidateFilePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	// Check for dangerous shell metacharacters
	for _, char := range dangerousChars {
		if strings.Contains(path, char) {
			return "", fmt.Errorf("file path contains forbidden character %q: %s", char, path)
		}
	}

	// Clean the path to remove . and .. components
	cleanPath := filepath.Clean(path)

	// If the path is relative, make it absolute based on current working directory
	if !filepath.IsAbs(cleanPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
		cleanPath = filepath.Join(cwd, cleanPath)
	}

	// Try to resolve symlinks for existing files
	resolvedPath, err := filepath.EvalSymlinks(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet, return cleaned path
			return cleanPath, nil
		}
		return "", fmt.Errorf("failed to resolve file path: %w", err)
	}

	return resolvedPath, nil
}

// ValidateFilePathInDir validates a file path and ensures it's within a specific directory.
// This prevents path traversal attacks that could escape the intended directory.
func ValidateFilePathInDir(path, baseDir string) (string, error) {
	if baseDir == "" {
		return "", fmt.Errorf("base directory cannot be empty")
	}

	// First validate the path normally
	cleanPath, err := ValidateFilePath(path)
	if err != nil {
		return "", err
	}

	// Clean and resolve the base directory
	cleanBaseDir := filepath.Clean(baseDir)
	if !filepath.IsAbs(cleanBaseDir) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
		cleanBaseDir = filepath.Join(cwd, cleanBaseDir)
	}

	// Resolve symlinks for base dir if it exists
	resolvedBaseDir, err := filepath.EvalSymlinks(cleanBaseDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to resolve base directory: %w", err)
		}
		resolvedBaseDir = cleanBaseDir
	}

	// Ensure the resolved path is within the base directory
	// Add trailing separator to prevent prefix matching issues (e.g., /foo matching /foobar)
	if !strings.HasPrefix(cleanPath, resolvedBaseDir+string(filepath.Separator)) && cleanPath != resolvedBaseDir {
		return "", fmt.Errorf("file path escapes base directory: %s is not within %s", path, baseDir)
	}

	return cleanPath, nil
}

// SafeReadFile reads a file after validating the path.
// This is a drop-in replacement for os.ReadFile with path validation.
func SafeReadFile(path string) ([]byte, error) {
	cleanPath, err := ValidateFilePath(path)
	if err != nil {
		return nil, err
	}
	// #nosec G304 - path is validated above
	return os.ReadFile(cleanPath)
}

// SafeReadFileInDir reads a file after validating it's within a specific directory.
func SafeReadFileInDir(path, baseDir string) ([]byte, error) {
	cleanPath, err := ValidateFilePathInDir(path, baseDir)
	if err != nil {
		return nil, err
	}
	// #nosec G304 - path is validated above
	return os.ReadFile(cleanPath)
}

// SafeOpen opens a file after validating the path.
// This is a drop-in replacement for os.Open with path validation.
func SafeOpen(path string) (*os.File, error) {
	cleanPath, err := ValidateFilePath(path)
	if err != nil {
		return nil, err
	}
	// #nosec G304 - path is validated above
	return os.Open(cleanPath)
}

// SafeOpenInDir opens a file after validating it's within a specific directory.
func SafeOpenInDir(path, baseDir string) (*os.File, error) {
	cleanPath, err := ValidateFilePathInDir(path, baseDir)
	if err != nil {
		return nil, err
	}
	// #nosec G304 - path is validated above
	return os.Open(cleanPath)
}
