package overrides

// Interface defines the contract for managing channel overrides
type Interface interface {
	// Get retrieves the override configuration for a specific acestream ID
	// Returns nil if no override exists for the given ID
	Get(acestreamID string) *ChannelOverride

	// Set updates or creates an override for a specific acestream ID
	// and immediately persists the changes to disk
	Set(acestreamID string, override ChannelOverride) error

	// Delete removes an override for a specific acestream ID
	// and immediately persists the changes to disk
	Delete(acestreamID string) error

	// List returns a copy of all current overrides
	List() map[string]ChannelOverride

	// CleanOrphans removes overrides for acestream IDs not in the provided validIDs list
	// Returns the number of deleted overrides and any error
	CleanOrphans(validIDs []string) (int, error)

	// BulkUpdate updates a specific field across multiple acestream IDs
	// Returns a BulkUpdateResult with counts and any errors encountered
	BulkUpdate(acestreamIDs []string, field string, value interface{}, atomic bool) (*BulkUpdateResult, error)
}
