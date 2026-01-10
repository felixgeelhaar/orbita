package security

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFilePath(t *testing.T) {
	t.Run("rejects empty path", func(t *testing.T) {
		_, err := ValidateFilePath("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("rejects dangerous shell characters", func(t *testing.T) {
		for _, char := range dangerousChars {
			path := "/tmp/test" + char + "file"
			_, err := ValidateFilePath(path)
			assert.Error(t, err, "expected error for character %q", char)
			assert.Contains(t, err.Error(), "forbidden character")
		}
	})

	t.Run("accepts valid absolute path", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

		result, err := ValidateFilePath(testFile)
		assert.NoError(t, err)

		// On macOS, /var is a symlink to /private/var, so compare resolved paths
		expectedResolved, _ := filepath.EvalSymlinks(testFile)
		assert.Equal(t, expectedResolved, result)
	})

	t.Run("converts relative path to absolute", func(t *testing.T) {
		result, err := ValidateFilePath("test.txt")
		assert.NoError(t, err)
		assert.True(t, filepath.IsAbs(result))
	})

	t.Run("resolves symlinks", func(t *testing.T) {
		tmpDir := t.TempDir()
		realFile := filepath.Join(tmpDir, "real.txt")
		require.NoError(t, os.WriteFile(realFile, []byte("test"), 0644))

		linkFile := filepath.Join(tmpDir, "link.txt")
		require.NoError(t, os.Symlink(realFile, linkFile))

		result, err := ValidateFilePath(linkFile)
		assert.NoError(t, err)

		// Result should be the resolved real file path
		expectedResolved, _ := filepath.EvalSymlinks(realFile)
		assert.Equal(t, expectedResolved, result)
	})

	t.Run("handles non-existent file gracefully", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonExistent := filepath.Join(tmpDir, "nonexistent.txt")

		result, err := ValidateFilePath(nonExistent)
		assert.NoError(t, err)
		assert.Contains(t, result, "nonexistent.txt")
	})

	t.Run("cleans path traversal attempts", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "subdir", "..", "test.txt")
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644))

		result, err := ValidateFilePath(testFile)
		assert.NoError(t, err)
		// Path should be cleaned, not contain ".."
		assert.NotContains(t, result, "..")
	})
}

func TestValidateFilePathInDir(t *testing.T) {
	t.Run("rejects empty base directory", func(t *testing.T) {
		_, err := ValidateFilePathInDir("/tmp/test.txt", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "base directory cannot be empty")
	})

	t.Run("accepts file within base directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

		result, err := ValidateFilePathInDir(testFile, tmpDir)
		assert.NoError(t, err)

		// Compare resolved paths
		expectedResolved, _ := filepath.EvalSymlinks(testFile)
		assert.Equal(t, expectedResolved, result)
	})

	t.Run("accepts subdirectory file within base directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		subDir := filepath.Join(tmpDir, "subdir")
		require.NoError(t, os.MkdirAll(subDir, 0755))

		testFile := filepath.Join(subDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

		result, err := ValidateFilePathInDir(testFile, tmpDir)
		assert.NoError(t, err)

		expectedResolved, _ := filepath.EvalSymlinks(testFile)
		assert.Equal(t, expectedResolved, result)
	})

	t.Run("rejects path traversal escape", func(t *testing.T) {
		tmpDir := t.TempDir()
		parentFile := filepath.Join(tmpDir, "..", "escape.txt")

		_, err := ValidateFilePathInDir(parentFile, tmpDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "escapes base directory")
	})

	t.Run("rejects sibling directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		baseDir := filepath.Join(tmpDir, "base")
		siblingDir := filepath.Join(tmpDir, "sibling")
		require.NoError(t, os.MkdirAll(baseDir, 0755))
		require.NoError(t, os.MkdirAll(siblingDir, 0755))

		siblingFile := filepath.Join(siblingDir, "test.txt")
		require.NoError(t, os.WriteFile(siblingFile, []byte("test"), 0644))

		_, err := ValidateFilePathInDir(siblingFile, baseDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "escapes base directory")
	})

	t.Run("rejects prefix attack", func(t *testing.T) {
		tmpDir := t.TempDir()
		baseDir := filepath.Join(tmpDir, "foo")
		prefixDir := filepath.Join(tmpDir, "foobar")
		require.NoError(t, os.MkdirAll(baseDir, 0755))
		require.NoError(t, os.MkdirAll(prefixDir, 0755))

		prefixFile := filepath.Join(prefixDir, "test.txt")
		require.NoError(t, os.WriteFile(prefixFile, []byte("test"), 0644))

		_, err := ValidateFilePathInDir(prefixFile, baseDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "escapes base directory")
	})
}

func TestSafeReadFile(t *testing.T) {
	t.Run("reads valid file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		content := []byte("hello world")
		require.NoError(t, os.WriteFile(testFile, content, 0644))

		data, err := SafeReadFile(testFile)
		assert.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("rejects path with dangerous characters", func(t *testing.T) {
		_, err := SafeReadFile("/tmp/test;file.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden character")
	})

	t.Run("returns error for non-existent file", func(t *testing.T) {
		tmpDir := t.TempDir()
		_, err := SafeReadFile(filepath.Join(tmpDir, "nonexistent.txt"))
		assert.Error(t, err)
	})
}

func TestSafeReadFileInDir(t *testing.T) {
	t.Run("reads file within directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		content := []byte("hello world")
		require.NoError(t, os.WriteFile(testFile, content, 0644))

		data, err := SafeReadFileInDir(testFile, tmpDir)
		assert.NoError(t, err)
		assert.Equal(t, content, data)
	})

	t.Run("rejects file outside directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		outsideFile := filepath.Join(tmpDir, "..", "outside.txt")

		_, err := SafeReadFileInDir(outsideFile, tmpDir)
		assert.Error(t, err)
	})
}

func TestSafeOpen(t *testing.T) {
	t.Run("opens valid file", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

		file, err := SafeOpen(testFile)
		assert.NoError(t, err)
		assert.NotNil(t, file)
		file.Close()
	})

	t.Run("rejects path with dangerous characters", func(t *testing.T) {
		_, err := SafeOpen("/tmp/test|file.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "forbidden character")
	})
}

func TestSafeOpenInDir(t *testing.T) {
	t.Run("opens file within directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

		file, err := SafeOpenInDir(testFile, tmpDir)
		assert.NoError(t, err)
		assert.NotNil(t, file)
		file.Close()
	})

	t.Run("rejects file outside directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		outsideFile := filepath.Join(tmpDir, "..", "outside.txt")

		_, err := SafeOpenInDir(outsideFile, tmpDir)
		assert.Error(t, err)
	})
}
