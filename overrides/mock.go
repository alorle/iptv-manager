package overrides

// MockManager is a mock implementation of the Interface for testing
type MockManager struct {
	GetFunc          func(acestreamID string) *ChannelOverride
	SetFunc          func(acestreamID string, override ChannelOverride) error
	DeleteFunc       func(acestreamID string) error
	ListFunc         func() map[string]ChannelOverride
	CleanOrphansFunc func(validIDs []string) (int, error)
	BulkUpdateFunc   func(acestreamIDs []string, field string, value interface{}, atomic bool) (*BulkUpdateResult, error)
}

// Get implements Interface.Get
func (m *MockManager) Get(acestreamID string) *ChannelOverride {
	if m.GetFunc != nil {
		return m.GetFunc(acestreamID)
	}
	return nil
}

// Set implements Interface.Set
func (m *MockManager) Set(acestreamID string, override ChannelOverride) error {
	if m.SetFunc != nil {
		return m.SetFunc(acestreamID, override)
	}
	return nil
}

// Delete implements Interface.Delete
func (m *MockManager) Delete(acestreamID string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(acestreamID)
	}
	return nil
}

// List implements Interface.List
func (m *MockManager) List() map[string]ChannelOverride {
	if m.ListFunc != nil {
		return m.ListFunc()
	}
	return make(map[string]ChannelOverride)
}

// CleanOrphans implements Interface.CleanOrphans
func (m *MockManager) CleanOrphans(validIDs []string) (int, error) {
	if m.CleanOrphansFunc != nil {
		return m.CleanOrphansFunc(validIDs)
	}
	return 0, nil
}

// BulkUpdate implements Interface.BulkUpdate
func (m *MockManager) BulkUpdate(acestreamIDs []string, field string, value interface{}, atomic bool) (*BulkUpdateResult, error) {
	if m.BulkUpdateFunc != nil {
		return m.BulkUpdateFunc(acestreamIDs, field, value, atomic)
	}
	return &BulkUpdateResult{Updated: len(acestreamIDs), Failed: 0}, nil
}
