package epg

// Repository defines the interface for EPG data access
type Repository interface {
	// GetAll returns all EPG channels
	GetAll() []EPGChannel

	// Search returns channels matching the query
	Search(query string) []EPGChannel

	// FindByID returns a channel by ID, or nil if not found
	FindByID(id string) *EPGChannel

	// IsAvailable returns true if EPG data is available
	IsAvailable() bool
}

// Ensure Cache implements Repository
var _ Repository = (*Cache)(nil)
