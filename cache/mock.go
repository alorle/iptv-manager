package cache

import "time"

// MockStorage is a mock implementation of the Storage interface for testing
type MockStorage struct {
	GetFunc       func(key string) (*Entry, error)
	SetFunc       func(key string, content []byte) error
	IsExpiredFunc func(key string, ttl time.Duration) (bool, error)
}

// Get implements Storage.Get
func (m *MockStorage) Get(key string) (*Entry, error) {
	if m.GetFunc != nil {
		return m.GetFunc(key)
	}
	return nil, nil
}

// Set implements Storage.Set
func (m *MockStorage) Set(key string, content []byte) error {
	if m.SetFunc != nil {
		return m.SetFunc(key, content)
	}
	return nil
}

// IsExpired implements Storage.IsExpired
func (m *MockStorage) IsExpired(key string, ttl time.Duration) (bool, error) {
	if m.IsExpiredFunc != nil {
		return m.IsExpiredFunc(key, ttl)
	}
	return false, nil
}
