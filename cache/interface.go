package cache

// Note: The primary interface 'Storage' is already defined in storage.go
// This file provides documentation and any additional cache-related interfaces

// Provider is an alias for Storage to make the naming clearer in consumer code
// It provides the contract for cache operations with get, set, and expiration checking
type Provider interface {
	Storage
}
