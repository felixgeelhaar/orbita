package registry

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoader(t *testing.T) {
	t.Run("creates loader with logger", func(t *testing.T) {
		logger := slog.Default()
		loader := NewLoader(logger)

		require.NotNil(t, loader)
		assert.NotNil(t, loader.logger)
		assert.NotNil(t, loader.clients)
	})

	t.Run("uses default logger when nil", func(t *testing.T) {
		loader := NewLoader(nil)

		require.NotNil(t, loader)
		assert.NotNil(t, loader.logger)
	})
}

func TestLoader_validateBinaryPath(t *testing.T) {
	loader := NewLoader(nil)

	t.Run("accepts valid absolute path", func(t *testing.T) {
		// Create a temporary file to test with
		tmpDir := t.TempDir()
		binaryPath := filepath.Join(tmpDir, "test-binary")
		require.NoError(t, os.WriteFile(binaryPath, []byte("#!/bin/sh\necho hello"), 0755))

		result, err := loader.validateBinaryPath(binaryPath)

		require.NoError(t, err)
		// On macOS, /var is a symlink to /private/var, so we compare resolved paths
		expectedResolved, _ := filepath.EvalSymlinks(binaryPath)
		assert.Equal(t, expectedResolved, result)
	})

	t.Run("rejects empty path", func(t *testing.T) {
		_, err := loader.validateBinaryPath("")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("rejects relative path", func(t *testing.T) {
		_, err := loader.validateBinaryPath("./relative/path")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be absolute")
	})

	t.Run("rejects path with semicolon", func(t *testing.T) {
		_, err := loader.validateBinaryPath("/path/to/binary;rm -rf /")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden character")
		assert.Contains(t, err.Error(), ";")
	})

	t.Run("rejects path with ampersand", func(t *testing.T) {
		_, err := loader.validateBinaryPath("/path/to/binary&malicious")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden character")
		assert.Contains(t, err.Error(), "&")
	})

	t.Run("rejects path with pipe", func(t *testing.T) {
		_, err := loader.validateBinaryPath("/path/to/binary|cat /etc/passwd")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden character")
		assert.Contains(t, err.Error(), "|")
	})

	t.Run("rejects path with dollar sign", func(t *testing.T) {
		_, err := loader.validateBinaryPath("/path/to/$USER/binary")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden character")
		assert.Contains(t, err.Error(), "$")
	})

	t.Run("rejects path with backtick", func(t *testing.T) {
		_, err := loader.validateBinaryPath("/path/to/`whoami`/binary")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden character")
		assert.Contains(t, err.Error(), "`")
	})

	t.Run("rejects path with parentheses", func(t *testing.T) {
		_, err := loader.validateBinaryPath("/path/to/(subshell)")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden character")
	})

	t.Run("rejects path with newline", func(t *testing.T) {
		_, err := loader.validateBinaryPath("/path/to/binary\nmalicious")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden character")
	})

	t.Run("rejects path with quotes", func(t *testing.T) {
		_, err := loader.validateBinaryPath("/path/to/'quoted'")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden character")
	})

	t.Run("cleans path traversal attempts", func(t *testing.T) {
		// Create a temporary structure
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "sub")
		require.NoError(t, os.MkdirAll(subDir, 0755))
		binaryPath := filepath.Join(tmpDir, "test-binary")
		require.NoError(t, os.WriteFile(binaryPath, []byte("test"), 0755))

		// Path with ".." should be cleaned
		traversalPath := filepath.Join(subDir, "..", "test-binary")
		result, err := loader.validateBinaryPath(traversalPath)

		require.NoError(t, err)
		// On macOS, /var is a symlink to /private/var, so we compare resolved paths
		expectedResolved, _ := filepath.EvalSymlinks(binaryPath)
		assert.Equal(t, expectedResolved, result)
	})

	t.Run("returns cleaned path for nonexistent file", func(t *testing.T) {
		nonexistentPath := "/nonexistent/path/to/binary"

		result, err := loader.validateBinaryPath(nonexistentPath)

		require.NoError(t, err)
		assert.Equal(t, nonexistentPath, result)
	})

	t.Run("resolves symlinks", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create the actual binary
		actualBinary := filepath.Join(tmpDir, "actual-binary")
		require.NoError(t, os.WriteFile(actualBinary, []byte("test"), 0755))

		// Create a symlink to it
		symlinkPath := filepath.Join(tmpDir, "symlink-binary")
		require.NoError(t, os.Symlink(actualBinary, symlinkPath))

		result, err := loader.validateBinaryPath(symlinkPath)

		require.NoError(t, err)
		// On macOS, /var is a symlink to /private/var, so we compare resolved paths
		expectedResolved, _ := filepath.EvalSymlinks(actualBinary)
		assert.Equal(t, expectedResolved, result)
	})

	t.Run("rejects all dangerous shell characters", func(t *testing.T) {
		dangerousChars := []string{";", "&", "|", "$", "`", "(", ")", "{", "}", "<", ">", "!", "'", "\""}

		for _, char := range dangerousChars {
			path := "/path/to/binary" + char + "malicious"
			_, err := loader.validateBinaryPath(path)

			assert.Error(t, err, "expected error for character %q", char)
			assert.Contains(t, err.Error(), "forbidden character", "character %q should be forbidden", char)
		}
	})
}

func TestLoader_IsLoaded(t *testing.T) {
	t.Run("returns false when plugin not loaded", func(t *testing.T) {
		loader := NewLoader(nil)

		assert.False(t, loader.IsLoaded("unknown.plugin"))
	})
}

func TestLoader_Unload(t *testing.T) {
	t.Run("returns nil when plugin not loaded", func(t *testing.T) {
		loader := NewLoader(nil)

		err := loader.Unload("unknown.plugin")

		assert.NoError(t, err)
	})
}

func TestLoader_UnloadAll(t *testing.T) {
	t.Run("clears all clients", func(t *testing.T) {
		loader := NewLoader(nil)

		loader.UnloadAll()

		assert.Empty(t, loader.clients)
	})
}

func TestLoader_verifyChecksum(t *testing.T) {
	loader := NewLoader(nil)

	t.Run("verifies valid checksum", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "testfile")
		content := []byte("hello world")
		require.NoError(t, os.WriteFile(filePath, content, 0644))

		// SHA256 of "hello world" is: b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9
		err := loader.verifyChecksum(filePath, "sha256:b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9")

		assert.NoError(t, err)
	})

	t.Run("verifies checksum without algorithm prefix", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "testfile")
		content := []byte("hello world")
		require.NoError(t, os.WriteFile(filePath, content, 0644))

		err := loader.verifyChecksum(filePath, "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9")

		assert.NoError(t, err)
	})

	t.Run("fails on checksum mismatch", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "testfile")
		require.NoError(t, os.WriteFile(filePath, []byte("hello world"), 0644))

		err := loader.verifyChecksum(filePath, "sha256:invalidhash")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "checksum mismatch")
	})

	t.Run("fails on unsupported algorithm", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "testfile")
		require.NoError(t, os.WriteFile(filePath, []byte("test"), 0644))

		err := loader.verifyChecksum(filePath, "md5:somehash")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported checksum algorithm")
	})

	t.Run("fails on nonexistent file", func(t *testing.T) {
		err := loader.verifyChecksum("/nonexistent/file", "sha256:abc123")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open file")
	})

	t.Run("case insensitive hash comparison", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "testfile")
		require.NoError(t, os.WriteFile(filePath, []byte("hello world"), 0644))

		// Use uppercase hash
		err := loader.verifyChecksum(filePath, "sha256:B94D27B9934D3E08A52E52D7DA7DABFAC484EFE37A5380EE9088F7ACE2EFCDE9")

		assert.NoError(t, err)
	})
}

func TestHclogAdapter(t *testing.T) {
	t.Run("creates adapter with name", func(t *testing.T) {
		logger := slog.Default()
		adapter := newHclogAdapter(logger)

		assert.Equal(t, "orbita", adapter.Name())
	})

	t.Run("Named returns new adapter with prefixed name", func(t *testing.T) {
		logger := slog.Default()
		adapter := newHclogAdapter(logger)

		named := adapter.Named("plugin")

		assert.Equal(t, "orbita.plugin", named.Name())
	})

	t.Run("ResetNamed returns adapter with new name", func(t *testing.T) {
		logger := slog.Default()
		adapter := newHclogAdapter(logger)

		reset := adapter.ResetNamed("new-name")

		assert.Equal(t, "new-name", reset.Name())
	})

	t.Run("With returns same adapter", func(t *testing.T) {
		logger := slog.Default()
		adapter := newHclogAdapter(logger)

		withArgs := adapter.With("key", "value")

		assert.Equal(t, adapter, withArgs)
	})

	t.Run("ImpliedArgs returns nil", func(t *testing.T) {
		adapter := newHclogAdapter(slog.Default())

		assert.Nil(t, adapter.ImpliedArgs())
	})

	t.Run("level checks", func(t *testing.T) {
		adapter := newHclogAdapter(slog.Default())

		assert.False(t, adapter.IsTrace())
		assert.True(t, adapter.IsDebug())
		assert.True(t, adapter.IsInfo())
		assert.True(t, adapter.IsWarn())
		assert.True(t, adapter.IsError())
	})

	t.Run("StandardLogger returns default", func(t *testing.T) {
		adapter := newHclogAdapter(slog.Default())

		standardLogger := adapter.StandardLogger(nil)

		assert.NotNil(t, standardLogger)
	})

	t.Run("StandardWriter returns stderr", func(t *testing.T) {
		adapter := newHclogAdapter(slog.Default())

		writer := adapter.StandardWriter(nil)

		assert.Equal(t, os.Stderr, writer)
	})
}
