package rewriter

// MockRewriter is a mock implementation of the Interface for testing
type MockRewriter struct {
	RewriteM3UFunc func(content []byte, baseURL string) []byte
}

// RewriteM3U implements Interface.RewriteM3U
func (m *MockRewriter) RewriteM3U(content []byte, baseURL string) []byte {
	if m.RewriteM3UFunc != nil {
		return m.RewriteM3UFunc(content, baseURL)
	}
	return content
}
