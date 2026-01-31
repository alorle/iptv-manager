package overrides

// ChannelOverride defines optional overrides for a channel's metadata.
// All fields are pointers to distinguish between "not configured" (nil)
// and "set to empty value" (pointer to empty string or false).
type ChannelOverride struct {
	// Enabled controls whether the channel is enabled or disabled.
	// nil means use default behavior, otherwise use the specified value.
	Enabled *bool `yaml:"enabled,omitempty"`

	// TvgID overrides the tvg-id attribute for the channel.
	// nil means no override, empty string means explicitly set to empty.
	TvgID *string `yaml:"tvg_id,omitempty"`

	// TvgName overrides the tvg-name attribute for the channel.
	// nil means no override, empty string means explicitly set to empty.
	TvgName *string `yaml:"tvg_name,omitempty"`

	// TvgLogo overrides the tvg-logo attribute for the channel.
	// nil means no override, empty string means explicitly set to empty.
	TvgLogo *string `yaml:"tvg_logo,omitempty"`

	// GroupTitle overrides the group-title attribute for the channel.
	// nil means no override, empty string means explicitly set to empty.
	GroupTitle *string `yaml:"group_title,omitempty"`
}

// OverridesConfig is a map from acestream content ID to channel override configuration.
// The key is the acestream content ID (hash), and the value contains the override settings.
type OverridesConfig map[string]ChannelOverride
