package epg

// EPGChannel represents a channel from the EPG guide
type EPGChannel struct {
	ID   string // Channel ID from EPG (e.g., "dazn.f1.hd")
	Name string // Display name (e.g., "DAZN F1 HD")
	Logo string // Logo URL from EPG
}
