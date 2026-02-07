package rewriter

// Rewriter handles URL rewriting for M3U playlists
type Rewriter struct {
}

// New creates a new Rewriter with the specified stream base URL
func New() Interface {
	return &Rewriter{}
}

// Stream represents a single M3U playlist entry
type Stream struct {
	Metadata string
	URL      string
	AceID    string
}
