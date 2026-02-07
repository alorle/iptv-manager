package rewriter

import (
	"testing"

	"github.com/alorle/iptv-manager/overrides"
)

func TestRemoveLogoMetadata(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "logo at the beginning",
			input:    `#EXTINF:-1 tvg-logo="http://logo.png" tvg-id="channel.id" tvg-name="Channel Name" group-title="Sports",Channel Name`,
			expected: `#EXTINF:-1 tvg-id="channel.id" tvg-name="Channel Name" group-title="Sports",Channel Name`,
		},
		{
			name:     "logo in the middle",
			input:    `#EXTINF:-1 tvg-id="channel.id" tvg-logo="http://logo.png" tvg-name="Channel Name" group-title="Sports",Channel Name`,
			expected: `#EXTINF:-1 tvg-id="channel.id" tvg-name="Channel Name" group-title="Sports",Channel Name`,
		},
		{
			name:     "logo at the end",
			input:    `#EXTINF:-1 tvg-id="channel.id" tvg-name="Channel Name" group-title="Sports" tvg-logo="http://logo.png",Channel Name`,
			expected: `#EXTINF:-1 tvg-id="channel.id" tvg-name="Channel Name" group-title="Sports",Channel Name`,
		},
		{
			name:     "no logo attribute",
			input:    `#EXTINF:-1 tvg-id="channel.id" tvg-name="Channel Name" group-title="Sports",Channel Name`,
			expected: `#EXTINF:-1 tvg-id="channel.id" tvg-name="Channel Name" group-title="Sports",Channel Name`,
		},
		{
			name:     "logo with https URL",
			input:    `#EXTINF:-1 tvg-logo="https://example.com/logos/channel.jpg" tvg-name="Channel",Channel`,
			expected: `#EXTINF:-1 tvg-name="Channel",Channel`,
		},
		{
			name:     "logo with complex URL",
			input:    `#EXTINF:-1 tvg-logo="https://example.com/path/to/logo?size=large&format=png" group-title="Sports",Channel`,
			expected: `#EXTINF:-1 group-title="Sports",Channel`,
		},
		{
			name:     "only logo attribute",
			input:    `#EXTINF:-1 tvg-logo="http://logo.png",Channel Name`,
			expected: `#EXTINF:-1,Channel Name`,
		},
		{
			name:     "empty logo value",
			input:    `#EXTINF:-1 tvg-logo="" tvg-name="Channel",Channel`,
			expected: `#EXTINF:-1 tvg-name="Channel",Channel`,
		},
		{
			name:     "non-EXTINF line",
			input:    `http://example.com/stream.m3u8`,
			expected: `http://example.com/stream.m3u8`,
		},
		{
			name:     "EXTM3U header",
			input:    `#EXTM3U`,
			expected: `#EXTM3U`,
		},
		{
			name:     "logo with data URI",
			input:    `#EXTINF:-1 tvg-logo="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAUA" tvg-id="ch1",Channel`,
			expected: `#EXTINF:-1 tvg-id="ch1",Channel`,
		},
		{
			name:     "multiple spaces after attribute removal",
			input:    `#EXTINF:-1 tvg-id="ch1"  tvg-logo="http://logo.png"  tvg-name="Channel",Channel Name`,
			expected: `#EXTINF:-1 tvg-id="ch1" tvg-name="Channel",Channel Name`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RemoveLogoMetadata(tt.input)
			if result != tt.expected {
				t.Errorf("RemoveLogoMetadata() failed\nInput:    %s\nExpected: %s\nGot:      %s", tt.input, tt.expected, result)
			}
		})
	}
}

func TestApplyMetadataOverrides(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		override *overrides.ChannelOverride
		expected string
	}{
		{
			name:     "nil override",
			input:    `#EXTINF:-1,Channel Name`,
			override: nil,
			expected: `#EXTINF:-1,Channel Name`,
		},
		{
			name:  "override tvg-id",
			input: `#EXTINF:-1 tvg-id="old-id",Channel Name`,
			override: &overrides.ChannelOverride{
				TvgID: stringPtr("new-id"),
			},
			expected: `#EXTINF:-1 tvg-id="new-id",Channel Name`,
		},
		{
			name:  "add tvg-id when not present",
			input: `#EXTINF:-1,Channel Name`,
			override: &overrides.ChannelOverride{
				TvgID: stringPtr("added-id"),
			},
			expected: `#EXTINF:-1 tvg-id="added-id",Channel Name`,
		},
		{
			name:  "override tvg-name and display name",
			input: `#EXTINF:-1 tvg-name="Old Name",Old Name`,
			override: &overrides.ChannelOverride{
				TvgName: stringPtr("New Name"),
			},
			expected: `#EXTINF:-1 tvg-name="New Name",New Name`,
		},
		{
			name:  "override tvg-logo",
			input: `#EXTINF:-1 tvg-logo="http://old-logo.png",Channel Name`,
			override: &overrides.ChannelOverride{
				TvgLogo: stringPtr("http://new-logo.png"),
			},
			expected: `#EXTINF:-1 tvg-logo="http://new-logo.png",Channel Name`,
		},
		{
			name:  "override group-title",
			input: `#EXTINF:-1 group-title="Old Group",Channel Name`,
			override: &overrides.ChannelOverride{
				GroupTitle: stringPtr("New Group"),
			},
			expected: `#EXTINF:-1 group-title="New Group",Channel Name`,
		},
		{
			name:  "override multiple attributes",
			input: `#EXTINF:-1 tvg-id="old-id" tvg-name="Old Name" group-title="Old Group",Old Name`,
			override: &overrides.ChannelOverride{
				TvgID:      stringPtr("new-id"),
				TvgName:    stringPtr("New Name"),
				GroupTitle: stringPtr("New Group"),
			},
			expected: `#EXTINF:-1 tvg-id="new-id" tvg-name="New Name" group-title="New Group",New Name`,
		},
		{
			name:  "remove attribute with empty string",
			input: `#EXTINF:-1 tvg-logo="http://logo.png",Channel Name`,
			override: &overrides.ChannelOverride{
				TvgLogo: stringPtr(""),
			},
			expected: `#EXTINF:-1,Channel Name`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyMetadataOverrides(tt.input, tt.override)
			if result != tt.expected {
				t.Errorf("applyMetadataOverrides() failed\nInput:    %s\nExpected: %s\nGot:      %s", tt.input, tt.expected, result)
			}
		})
	}
}

func TestReplaceOrAddAttribute(t *testing.T) {
	tests := []struct {
		name      string
		extinf    string
		attrName  string
		attrValue string
		expected  string
	}{
		{
			name:      "replace existing attribute",
			extinf:    `#EXTINF:-1 tvg-id="old-id",Channel`,
			attrName:  "tvg-id",
			attrValue: "new-id",
			expected:  `#EXTINF:-1 tvg-id="new-id",Channel`,
		},
		{
			name:      "add new attribute",
			extinf:    `#EXTINF:-1,Channel`,
			attrName:  "tvg-id",
			attrValue: "new-id",
			expected:  `#EXTINF:-1 tvg-id="new-id",Channel`,
		},
		{
			name:      "remove attribute with empty value",
			extinf:    `#EXTINF:-1 tvg-id="some-id",Channel`,
			attrName:  "tvg-id",
			attrValue: "",
			expected:  `#EXTINF:-1,Channel`,
		},
		{
			name:      "add to existing metadata",
			extinf:    `#EXTINF:-1 tvg-name="Name",Channel`,
			attrName:  "tvg-id",
			attrValue: "new-id",
			expected:  `#EXTINF:-1 tvg-name="Name" tvg-id="new-id",Channel`,
		},
		{
			name:      "non-EXTINF line unchanged",
			extinf:    `http://example.com/stream.m3u8`,
			attrName:  "tvg-id",
			attrValue: "new-id",
			expected:  `http://example.com/stream.m3u8`,
		},
		{
			name:      "no comma - cannot add attribute",
			extinf:    `#EXTINF:-1`,
			attrName:  "tvg-id",
			attrValue: "new-id",
			expected:  `#EXTINF:-1`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceOrAddAttribute(tt.extinf, tt.attrName, tt.attrValue)
			if result != tt.expected {
				t.Errorf("replaceOrAddAttribute() failed\nInput:    %s\nExpected: %s\nGot:      %s", tt.extinf, tt.expected, result)
			}
		})
	}
}

func TestReplaceDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		extinf   string
		newName  string
		expected string
	}{
		{
			name:     "replace simple display name",
			extinf:   `#EXTINF:-1,Old Name`,
			newName:  "New Name",
			expected: `#EXTINF:-1,New Name`,
		},
		{
			name:     "replace display name with metadata",
			extinf:   `#EXTINF:-1 tvg-id="id" tvg-name="Name",Old Name`,
			newName:  "New Name",
			expected: `#EXTINF:-1 tvg-id="id" tvg-name="Name",New Name`,
		},
		{
			name:     "non-EXTINF line unchanged",
			extinf:   `http://example.com/stream.m3u8`,
			newName:  "New Name",
			expected: `http://example.com/stream.m3u8`,
		},
		{
			name:     "no comma - cannot replace",
			extinf:   `#EXTINF:-1`,
			newName:  "New Name",
			expected: `#EXTINF:-1`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceDisplayName(tt.extinf, tt.newName)
			if result != tt.expected {
				t.Errorf("replaceDisplayName() failed\nInput:    %s\nExpected: %s\nGot:      %s", tt.extinf, tt.expected, result)
			}
		})
	}
}

// Helper function for tests
func stringPtr(s string) *string {
	return &s
}
