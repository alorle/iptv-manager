package domain

import "testing"

func TestExtractDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		extinf   string
		expected string
	}{
		{
			name:     "simple display name",
			extinf:   `#EXTINF:-1,Channel Name`,
			expected: "Channel Name",
		},
		{
			name:     "display name with metadata",
			extinf:   `#EXTINF:-1 tvg-id="channel1" tvg-name="Channel 1",Channel 1`,
			expected: "Channel 1",
		},
		{
			name:     "display name with leading/trailing whitespace",
			extinf:   `#EXTINF:-1 tvg-id="channel1",  Channel Name  `,
			expected: "Channel Name",
		},
		{
			name:     "display name with special characters",
			extinf:   `#EXTINF:-1,La 1 HD España`,
			expected: "La 1 HD España",
		},
		{
			name:     "display name with emoji",
			extinf:   `#EXTINF:-1,Channel 🎬 Movies`,
			expected: "Channel 🎬 Movies",
		},
		{
			name:     "no comma in line",
			extinf:   `#EXTINF:-1 tvg-id="channel1"`,
			expected: "",
		},
		{
			name:     "not an EXTINF line",
			extinf:   `#EXTM3U`,
			expected: "",
		},
		{
			name:     "empty string",
			extinf:   "",
			expected: "",
		},
		{
			name:     "comma in metadata attribute",
			extinf:   `#EXTINF:-1 tvg-name="Channel, with comma",Real Channel Name`,
			expected: "Real Channel Name",
		},
		{
			name:     "empty display name after comma",
			extinf:   `#EXTINF:-1 tvg-id="channel1",`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractDisplayName(tt.extinf)
			if result != tt.expected {
				t.Errorf("ExtractDisplayName(%q) = %q, want %q", tt.extinf, result, tt.expected)
			}
		})
	}
}

func TestExtractGroupTitle(t *testing.T) {
	tests := []struct {
		name     string
		extinf   string
		expected string
	}{
		{
			name:     "simple group-title",
			extinf:   `#EXTINF:-1 group-title="Sports",Channel Name`,
			expected: "Sports",
		},
		{
			name:     "group-title with special characters",
			extinf:   `#EXTINF:-1 group-title="España HD",Channel Name`,
			expected: "España HD",
		},
		{
			name:     "group-title with multiple attributes",
			extinf:   `#EXTINF:-1 tvg-id="ch1" group-title="Movies" tvg-name="Channel",Channel Name`,
			expected: "Movies",
		},
		{
			name:     "empty group-title",
			extinf:   `#EXTINF:-1 group-title="",Channel Name`,
			expected: "",
		},
		{
			name:     "no group-title attribute",
			extinf:   `#EXTINF:-1 tvg-id="channel1",Channel Name`,
			expected: "",
		},
		{
			name:     "not an EXTINF line",
			extinf:   `#EXTM3U`,
			expected: "",
		},
		{
			name:     "empty string",
			extinf:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractGroupTitle(tt.extinf)
			if result != tt.expected {
				t.Errorf("ExtractGroupTitle(%q) = %q, want %q", tt.extinf, result, tt.expected)
			}
		})
	}
}

func TestExtractTvgID(t *testing.T) {
	tests := []struct {
		name     string
		extinf   string
		expected string
	}{
		{
			name:     "simple tvg-id",
			extinf:   `#EXTINF:-1 tvg-id="channel1",Channel Name`,
			expected: "channel1",
		},
		{
			name:     "tvg-id with multiple attributes",
			extinf:   `#EXTINF:-1 tvg-id="la1.es" tvg-name="La 1" group-title="España",La 1`,
			expected: "la1.es",
		},
		{
			name:     "empty tvg-id",
			extinf:   `#EXTINF:-1 tvg-id="",Channel Name`,
			expected: "",
		},
		{
			name:     "no tvg-id attribute",
			extinf:   `#EXTINF:-1 group-title="Sports",Channel Name`,
			expected: "",
		},
		{
			name:     "not an EXTINF line",
			extinf:   `#EXTM3U`,
			expected: "",
		},
		{
			name:     "empty string",
			extinf:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractTvgID(tt.extinf)
			if result != tt.expected {
				t.Errorf("ExtractTvgID(%q) = %q, want %q", tt.extinf, result, tt.expected)
			}
		})
	}
}

func TestExtractTvgLogo(t *testing.T) {
	tests := []struct {
		name     string
		extinf   string
		expected string
	}{
		{
			name:     "simple tvg-logo",
			extinf:   `#EXTINF:-1 tvg-logo="http://example.com/logo.png",Channel Name`,
			expected: "http://example.com/logo.png",
		},
		{
			name:     "tvg-logo with multiple attributes",
			extinf:   `#EXTINF:-1 tvg-id="ch1" tvg-logo="https://cdn.example.com/logo.jpg" tvg-name="Channel",Channel Name`,
			expected: "https://cdn.example.com/logo.jpg",
		},
		{
			name:     "empty tvg-logo",
			extinf:   `#EXTINF:-1 tvg-logo="",Channel Name`,
			expected: "",
		},
		{
			name:     "no tvg-logo attribute",
			extinf:   `#EXTINF:-1 tvg-id="channel1",Channel Name`,
			expected: "",
		},
		{
			name:     "not an EXTINF line",
			extinf:   `#EXTM3U`,
			expected: "",
		},
		{
			name:     "empty string",
			extinf:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractTvgLogo(tt.extinf)
			if result != tt.expected {
				t.Errorf("ExtractTvgLogo(%q) = %q, want %q", tt.extinf, result, tt.expected)
			}
		})
	}
}

func TestExtractMetadata(t *testing.T) {
	tests := []struct {
		name              string
		extinf            string
		expectedTvgID     string
		expectedTvgLogo   string
		expectedGroupTitle string
	}{
		{
			name:               "all metadata present",
			extinf:             `#EXTINF:-1 tvg-id="la1.es" tvg-logo="http://example.com/logo.png" group-title="España",La 1`,
			expectedTvgID:      "la1.es",
			expectedTvgLogo:    "http://example.com/logo.png",
			expectedGroupTitle: "España",
		},
		{
			name:               "partial metadata",
			extinf:             `#EXTINF:-1 tvg-id="ch1" group-title="Sports",Channel Name`,
			expectedTvgID:      "ch1",
			expectedTvgLogo:    "",
			expectedGroupTitle: "Sports",
		},
		{
			name:               "no metadata",
			extinf:             `#EXTINF:-1,Channel Name`,
			expectedTvgID:      "",
			expectedTvgLogo:    "",
			expectedGroupTitle: "",
		},
		{
			name:               "empty string",
			extinf:             "",
			expectedTvgID:      "",
			expectedTvgLogo:    "",
			expectedGroupTitle: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tvgID, tvgLogo, groupTitle := ExtractMetadata(tt.extinf)
			if tvgID != tt.expectedTvgID {
				t.Errorf("ExtractMetadata(%q) tvgID = %q, want %q", tt.extinf, tvgID, tt.expectedTvgID)
			}
			if tvgLogo != tt.expectedTvgLogo {
				t.Errorf("ExtractMetadata(%q) tvgLogo = %q, want %q", tt.extinf, tvgLogo, tt.expectedTvgLogo)
			}
			if groupTitle != tt.expectedGroupTitle {
				t.Errorf("ExtractMetadata(%q) groupTitle = %q, want %q", tt.extinf, groupTitle, tt.expectedGroupTitle)
			}
		})
	}
}
