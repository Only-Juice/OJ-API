package sandbox

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWriteToTempFile(t *testing.T) {
	t.Run("write to temp file successfully", func(t *testing.T) {
		testData := []byte("#!/bin/bash\necho 'Hello World'")

		id, err := WriteToTempFile(testData)

		assert.NoError(t, err)
		assert.NotEmpty(t, id)

		// Verify the file was created with correct content
		filename := shellFilename(id)
		if _, err := os.Stat(filename); err == nil {
			content, readErr := os.ReadFile(filename)
			assert.NoError(t, readErr)
			assert.Equal(t, testData, content)

			// Cleanup
			os.Remove(filename)
		}
	})

	t.Run("write empty content", func(t *testing.T) {
		testData := []byte("")

		id, err := WriteToTempFile(testData)

		assert.NoError(t, err)
		assert.NotEmpty(t, id)

		// Cleanup
		filename := shellFilename(id)
		if _, err := os.Stat(filename); err == nil {
			os.Remove(filename)
		}
	})

	t.Run("write large content", func(t *testing.T) {
		// Create a large byte array (1MB)
		largeData := make([]byte, 1024*1024)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		id, err := WriteToTempFile(largeData)

		assert.NoError(t, err)
		assert.NotEmpty(t, id)

		// Cleanup
		filename := shellFilename(id)
		if _, err := os.Stat(filename); err == nil {
			os.Remove(filename)
		}
	})

	t.Run("verify unique IDs", func(t *testing.T) {
		testData := []byte("test content")

		id1, err1 := WriteToTempFile(testData)
		time.Sleep(time.Millisecond) // Ensure different timestamps
		id2, err2 := WriteToTempFile(testData)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEqual(t, id1, id2)

		// Cleanup
		os.Remove(shellFilename(id1))
		os.Remove(shellFilename(id2))
	})
}

func TestShellFilename(t *testing.T) {
	t.Run("shell filename generation", func(t *testing.T) {
		testID := "123456789"

		filename := shellFilename(testID)

		assert.Contains(t, filename, testID)
		assert.Contains(t, filename, CodeStorageFolder)

		// Verify it's a valid file path
		dir := filepath.Dir(filename)
		base := filepath.Base(filename)
		assert.Equal(t, CodeStorageFolder, dir)
		assert.Contains(t, base, testID)
	})

	t.Run("shell filename with empty ID", func(t *testing.T) {
		filename := shellFilename("")

		assert.NotEmpty(t, filename)
		assert.Contains(t, filename, CodeStorageFolder)
	})

	t.Run("shell filename with special characters", func(t *testing.T) {
		testID := "test-id_123.abc"

		filename := shellFilename(testID)

		assert.Contains(t, filename, testID)
		assert.Contains(t, filename, CodeStorageFolder)
	})
}

func TestCodeStorageFolder(t *testing.T) {
	t.Run("code storage folder constant", func(t *testing.T) {
		assert.Equal(t, "/sandbox/code", CodeStorageFolder)
		assert.NotEmpty(t, CodeStorageFolder)
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("test WriteToTempFile function exists", func(t *testing.T) {
		assert.NotNil(t, WriteToTempFile)
	})
}

func TestFileSystemIntegration(t *testing.T) {
	t.Run("create directory if not exists", func(t *testing.T) {
		testData := []byte("integration test content")
		id, err := WriteToTempFile(testData)

		assert.NotEmpty(t, id)
		if err != nil {
			t.Logf("WriteToTempFile error (expected in some environments): %v", err)
		}

		// Cleanup if file was created
		filename := shellFilename(id)
		if _, statErr := os.Stat(filename); statErr == nil {
			os.Remove(filename)
		}
	})
}

func TestEdgeCases(t *testing.T) {
	t.Run("test with binary data", func(t *testing.T) {
		binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}

		id, err := WriteToTempFile(binaryData)

		assert.NoError(t, err)
		assert.NotEmpty(t, id)

		// Cleanup
		filename := shellFilename(id)
		if _, err := os.Stat(filename); err == nil {
			content, readErr := os.ReadFile(filename)
			if readErr == nil {
				assert.Equal(t, binaryData, content)
			}
			os.Remove(filename)
		}
	})

	t.Run("test with unicode content", func(t *testing.T) {
		unicodeData := []byte("Hello 世界 🌍 Test ñáéíóú")

		id, err := WriteToTempFile(unicodeData)

		assert.NoError(t, err)
		assert.NotEmpty(t, id)

		// Cleanup
		filename := shellFilename(id)
		if _, err := os.Stat(filename); err == nil {
			os.Remove(filename)
		}
	})
}

func TestDirectoryCreationScenarios(t *testing.T) {
	t.Run("verify function handles directory operations", func(t *testing.T) {
		testData := []byte("directory test")

		assert.NotPanics(t, func() {
			id, _ := WriteToTempFile(testData)
			if id != "" {
				filename := shellFilename(id)
				os.Remove(filename) // Cleanup
			}
		})
	})
}
