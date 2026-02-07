package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const testKey = "test-key"

func TestNewFileStorage(t *testing.T) {
	t.Run("creates cache directory if it doesn't exist", func(t *testing.T) {
		tempDir := filepath.Join(t.TempDir(), "cache")

		storage, err := NewFileStorage(tempDir)
		if err != nil {
			t.Fatalf("NewFileStorage failed: %v", err)
		}

		if storage == nil {
			t.Fatal("Expected non-nil storage")
		}

		// Verify directory was created
		info, err := os.Stat(tempDir)
		if err != nil {
			t.Fatalf("Cache directory was not created: %v", err)
		}

		if !info.IsDir() {
			t.Error("Expected cache path to be a directory")
		}
	})

	t.Run("works with existing cache directory", func(t *testing.T) {
		tempDir := filepath.Join(t.TempDir(), "cache")

		// Create directory first
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}

		storage, err := NewFileStorage(tempDir)
		if err != nil {
			t.Fatalf("NewFileStorage failed with existing directory: %v", err)
		}

		if storage == nil {
			t.Fatal("Expected non-nil storage")
		}
	})

	t.Run("returns error for empty directory path", func(t *testing.T) {
		storage, err := NewFileStorage("")
		if err == nil {
			t.Error("Expected error for empty directory path")
		}

		if storage != nil {
			t.Error("Expected nil storage on error")
		}
	})
}

// setAndGetContent is a test helper that sets and retrieves content
func setAndGetContent(t *testing.T, storage *FileStorage, key string, content []byte) *Entry {
	t.Helper()

	if err := storage.Set(key, content); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	entry, err := storage.Get(key)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	return entry
}

// verifyContentMatch checks if entry content matches expected content
func verifyContentMatch(t *testing.T, entry *Entry, expected []byte) {
	t.Helper()

	if string(entry.Content) != string(expected) {
		t.Errorf("Expected content %q, got %q", expected, entry.Content)
	}
}

// verifyTimestampValid checks if entry has a valid recent timestamp
func verifyTimestampValid(t *testing.T, entry *Entry) {
	t.Helper()

	if entry.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}

	if time.Since(entry.Timestamp) > time.Second {
		t.Error("Timestamp should be recent")
	}
}

func TestFileStorage_SetAndGet(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("NewFileStorage failed: %v", err)
	}

	t.Run("stores and retrieves content", func(t *testing.T) {
		key := testKey
		content := []byte("test content")

		entry := setAndGetContent(t, storage, key, content)
		verifyContentMatch(t, entry, content)
		verifyTimestampValid(t, entry)
	})

	t.Run("overwrites existing content", func(t *testing.T) {
		key := "overwrite-key"
		firstContent := []byte("first content")
		secondContent := []byte("second content")

		setAndGetContent(t, storage, key, firstContent)
		time.Sleep(10 * time.Millisecond)
		entry := setAndGetContent(t, storage, key, secondContent)
		verifyContentMatch(t, entry, secondContent)
	})

	t.Run("returns error for non-existent key", func(t *testing.T) {
		entry, err := storage.Get("non-existent-key")
		if err == nil {
			t.Error("Expected error for non-existent key")
		}

		if entry != nil {
			t.Error("Expected nil entry for non-existent key")
		}
	})

	t.Run("handles empty content", func(t *testing.T) {
		key := "empty-key"
		content := []byte("")

		entry := setAndGetContent(t, storage, key, content)

		if len(entry.Content) != 0 {
			t.Errorf("Expected empty content, got %d bytes", len(entry.Content))
		}
	})

	t.Run("handles binary content", func(t *testing.T) {
		key := "binary-key"
		content := []byte{0x00, 0x01, 0xFF, 0xFE}

		entry := setAndGetContent(t, storage, key, content)

		if len(entry.Content) != len(content) {
			t.Errorf("Expected content length %d, got %d", len(content), len(entry.Content))
		}

		for i, b := range content {
			if entry.Content[i] != b {
				t.Errorf("Byte mismatch at position %d: expected %x, got %x", i, b, entry.Content[i])
			}
		}
	})
}

func TestFileStorage_IsExpired(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("NewFileStorage failed: %v", err)
	}

	t.Run("returns false for fresh cache", func(t *testing.T) {
		key := "fresh-key"
		content := []byte("fresh content")

		if err := storage.Set(key, content); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		expired, err := storage.IsExpired(key, 1*time.Hour)
		if err != nil {
			t.Fatalf("IsExpired failed: %v", err)
		}

		if expired {
			t.Error("Expected cache to not be expired")
		}
	})

	t.Run("returns true for expired cache", func(t *testing.T) {
		key := "expired-key"
		content := []byte("old content")

		if err := storage.Set(key, content); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		// Use a very short TTL to make it expired
		expired, err := storage.IsExpired(key, 1*time.Nanosecond)
		if err != nil {
			t.Fatalf("IsExpired failed: %v", err)
		}

		if !expired {
			t.Error("Expected cache to be expired")
		}
	})

	t.Run("returns true for non-existent cache", func(t *testing.T) {
		expired, err := storage.IsExpired("non-existent", 1*time.Hour)
		if err != nil {
			t.Fatalf("IsExpired failed: %v", err)
		}

		if !expired {
			t.Error("Expected non-existent cache to be considered expired")
		}
	})

	t.Run("handles zero TTL", func(t *testing.T) {
		key := "zero-ttl-key"
		content := []byte("content")

		if err := storage.Set(key, content); err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		expired, err := storage.IsExpired(key, 0)
		if err != nil {
			t.Fatalf("IsExpired failed: %v", err)
		}

		if !expired {
			t.Error("Expected cache to be expired with zero TTL")
		}
	})
}

func TestDeriveKeyFromURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "simple URL",
			url:  "https://example.com/file.m3u",
			want: "https://example.com/file.m3u",
		},
		{
			name: "URL with query parameters",
			url:  "https://example.com/file.m3u?param=value",
			want: "https://example.com/file.m3u?param=value",
		},
		{
			name: "IPFS URL",
			url:  "https://ipfs.io/ipfs/QmHash123",
			want: "https://ipfs.io/ipfs/QmHash123",
		},
		{
			name: "empty URL",
			url:  "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveKeyFromURL(tt.url)
			if got != tt.want {
				t.Errorf("DeriveKeyFromURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileStorage_getFilePath(t *testing.T) {
	tempDir := t.TempDir()
	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("NewFileStorage failed: %v", err)
	}

	t.Run("generates consistent paths for same key", func(t *testing.T) {
		key := testKey
		path1 := storage.getFilePath(key)
		path2 := storage.getFilePath(key)

		if path1 != path2 {
			t.Errorf("Expected consistent paths, got %q and %q", path1, path2)
		}
	})

	t.Run("generates different paths for different keys", func(t *testing.T) {
		path1 := storage.getFilePath("key1")
		path2 := storage.getFilePath("key2")

		if path1 == path2 {
			t.Error("Expected different paths for different keys")
		}
	})

	t.Run("generates paths within base directory", func(t *testing.T) {
		key := testKey
		path := storage.getFilePath(key)

		// Check if path is absolute or relative to tempDir
		rel, err := filepath.Rel(tempDir, path)
		if err != nil || filepath.IsAbs(rel) {
			t.Errorf("Expected path to be within %q, got %q", tempDir, path)
		}
	})

	t.Run("generates safe filenames", func(t *testing.T) {
		unsafeKey := "../../etc/passwd"
		path := storage.getFilePath(unsafeKey)

		// The path should not escape the base directory
		if filepath.Dir(path) != tempDir {
			t.Errorf("Path should be directly in base directory, got %q", path)
		}
	})
}
