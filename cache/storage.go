package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Storage defines the interface for cache operations
type Storage interface {
	Get(key string) (*Entry, error)
	Set(key string, content []byte) error
	IsExpired(key string, ttl time.Duration) (bool, error)
}

// Entry represents a cached item with its metadata
type Entry struct {
	Content   []byte    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// FileStorage implements Storage using the file system
type FileStorage struct {
	baseDir string
}

// NewFileStorage creates a new file-based cache storage
// It ensures the cache directory exists before returning
func NewFileStorage(baseDir string) (*FileStorage, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("cache directory cannot be empty")
	}

	// Ensure the cache directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &FileStorage{
		baseDir: baseDir,
	}, nil
}

// Get retrieves a cached entry by key
func (fs *FileStorage) Get(key string) (*Entry, error) {
	filePath := fs.getFilePath(key)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("cache entry not found: %w", err)
		}
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var entry Entry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache entry: %w", err)
	}

	return &entry, nil
}

// Set stores content in the cache with the current timestamp
func (fs *FileStorage) Set(key string, content []byte) error {
	entry := Entry{
		Content:   content,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	filePath := fs.getFilePath(key)

	// Ensure parent directory exists (defensive, should already exist)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create cache subdirectory: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// IsExpired checks if a cache entry has exceeded the TTL
func (fs *FileStorage) IsExpired(key string, ttl time.Duration) (bool, error) {
	entry, err := fs.Get(key)
	if err != nil {
		// If entry doesn't exist, consider it expired
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return false, fmt.Errorf("failed to check expiration: %w", err)
	}

	age := time.Since(entry.Timestamp)
	return age > ttl, nil
}

// getFilePath generates a file path from a cache key
// The key is hashed to create a safe filename
func (fs *FileStorage) getFilePath(key string) string {
	hash := sha256.Sum256([]byte(key))
	filename := hex.EncodeToString(hash[:]) + ".json"
	return filepath.Join(fs.baseDir, filename)
}

// DeriveKeyFromURL creates a cache key from a source URL
func DeriveKeyFromURL(url string) string {
	return url
}
