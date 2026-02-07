package rewriter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/alorle/iptv-manager/domain"
)

func TestExtractDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple display name",
			input:    "#EXTINF:-1,Channel Name",
			expected: "Channel Name",
		},
		{
			name:     "display name with metadata",
			input:    `#EXTINF:-1 tvg-id="channel1" tvg-name="Channel 1",Channel 1`,
			expected: "Channel 1",
		},
		{
			name:     "display name with complex metadata",
			input:    `#EXTINF:-1 tvg-id="ch1" tvg-name="Name" group-title="Sports" tvg-shift="2",My Channel`,
			expected: "My Channel",
		},
		{
			name:     "display name with trailing spaces",
			input:    `#EXTINF:-1,Channel Name   `,
			expected: "Channel Name",
		},
		{
			name:     "no comma - no display name",
			input:    "#EXTINF:-1",
			expected: "",
		},
		{
			name:     "non-EXTINF line",
			input:    "http://example.com/stream.m3u8",
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "display name with special characters",
			input:    `#EXTINF:-1,Channel: The Best (HD)`,
			expected: "Channel: The Best (HD)",
		},
		{
			name:     "display name starting with lowercase",
			input:    `#EXTINF:-1,abc Channel`,
			expected: "abc Channel",
		},
		{
			name:     "display name with numbers",
			input:    `#EXTINF:-1,24/7 News Channel`,
			expected: "24/7 News Channel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := domain.ExtractDisplayName(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractDisplayName() failed\nInput:    %s\nExpected: %s\nGot:      %s", tt.input, tt.expected, result)
			}
		})
	}
}

func TestExtractGroupTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "extract group-title from metadata",
			input:    `#EXTINF:-1 tvg-id="ch1" group-title="Sports",Channel`,
			expected: "Sports",
		},
		{
			name:     "group-title at the end",
			input:    `#EXTINF:-1 tvg-name="Name" group-title="Movies",Channel`,
			expected: "Movies",
		},
		{
			name:     "no group-title",
			input:    `#EXTINF:-1 tvg-id="ch1",Channel`,
			expected: "",
		},
		{
			name:     "empty group-title",
			input:    `#EXTINF:-1 group-title="",Channel`,
			expected: "",
		},
		{
			name:     "non-EXTINF line",
			input:    `http://example.com/stream.m3u8`,
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "group-title with special characters",
			input:    `#EXTINF:-1 group-title="Sports (HD)",Channel`,
			expected: "Sports (HD)",
		},
		{
			name:     "group-title with spaces",
			input:    `#EXTINF:-1 group-title="Live Events",Channel`,
			expected: "Live Events",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := domain.ExtractGroupTitle(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractGroupTitle() failed\nInput:    %s\nExpected: %s\nGot:      %s", tt.input, tt.expected, result)
			}
		})
	}
}

func TestSortStreamsByName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "alphabetical sorting",
			input: `#EXTM3U
#EXTINF:-1,Zebra Channel
acestream://3333333333333333333333333333333333333333
#EXTINF:-1,Apple Channel
acestream://1111111111111111111111111111111111111111
#EXTINF:-1,Banana Channel
acestream://2222222222222222222222222222222222222222`,
			expected: `#EXTM3U
#EXTINF:-1,Apple Channel
acestream://1111111111111111111111111111111111111111
#EXTINF:-1,Banana Channel
acestream://2222222222222222222222222222222222222222
#EXTINF:-1,Zebra Channel
acestream://3333333333333333333333333333333333333333`,
		},
		{
			name: "case-insensitive sorting",
			input: `#EXTM3U
#EXTINF:-1,zebra channel
acestream://3333333333333333333333333333333333333333
#EXTINF:-1,Apple Channel
acestream://1111111111111111111111111111111111111111
#EXTINF:-1,BANANA CHANNEL
acestream://2222222222222222222222222222222222222222`,
			expected: `#EXTM3U
#EXTINF:-1,Apple Channel
acestream://1111111111111111111111111111111111111111
#EXTINF:-1,BANANA CHANNEL
acestream://2222222222222222222222222222222222222222
#EXTINF:-1,zebra channel
acestream://3333333333333333333333333333333333333333`,
		},
		{
			name: "header preserved at top",
			input: `#EXTM3U
#EXTINF:-1,Z Channel
acestream://1111111111111111111111111111111111111111
#EXTINF:-1,A Channel
acestream://2222222222222222222222222222222222222222`,
			expected: `#EXTM3U
#EXTINF:-1,A Channel
acestream://2222222222222222222222222222222222222222
#EXTINF:-1,Z Channel
acestream://1111111111111111111111111111111111111111`,
		},
		{
			name: "sorting with metadata preserved",
			input: `#EXTM3U
#EXTINF:-1 tvg-id="z1" tvg-name="Z",Z Channel
acestream://1111111111111111111111111111111111111111
#EXTINF:-1 tvg-id="a1" tvg-name="A",A Channel
acestream://2222222222222222222222222222222222222222`,
			expected: `#EXTM3U
#EXTINF:-1 tvg-id="a1" tvg-name="A",A Channel
acestream://2222222222222222222222222222222222222222
#EXTINF:-1 tvg-id="z1" tvg-name="Z",Z Channel
acestream://1111111111111111111111111111111111111111`,
		},
		{
			name: "mixed stream types sorted",
			input: `#EXTM3U
#EXTINF:-1,Z HTTP Stream
http://example.com/z.m3u8
#EXTINF:-1,A Acestream
acestream://1111111111111111111111111111111111111111
#EXTINF:-1,M HTTPS Stream
https://example.com/m.m3u8`,
			expected: `#EXTM3U
#EXTINF:-1,A Acestream
acestream://1111111111111111111111111111111111111111
#EXTINF:-1,M HTTPS Stream
https://example.com/m.m3u8
#EXTINF:-1,Z HTTP Stream
http://example.com/z.m3u8`,
		},
		{
			name: "numbers in names",
			input: `#EXTM3U
#EXTINF:-1,Channel 3
acestream://3333333333333333333333333333333333333333
#EXTINF:-1,Channel 1
acestream://1111111111111111111111111111111111111111
#EXTINF:-1,Channel 2
acestream://2222222222222222222222222222222222222222`,
			expected: `#EXTM3U
#EXTINF:-1,Channel 1
acestream://1111111111111111111111111111111111111111
#EXTINF:-1,Channel 2
acestream://2222222222222222222222222222222222222222
#EXTINF:-1,Channel 3
acestream://3333333333333333333333333333333333333333`,
		},
		{
			name: "special characters in names",
			input: `#EXTM3U
#EXTINF:-1,[UK] BBC One
acestream://3333333333333333333333333333333333333333
#EXTINF:-1,(ES) TVE
acestream://1111111111111111111111111111111111111111
#EXTINF:-1,24/7 News
acestream://2222222222222222222222222222222222222222`,
			expected: `#EXTM3U
#EXTINF:-1,(ES) TVE
acestream://1111111111111111111111111111111111111111
#EXTINF:-1,24/7 News
acestream://2222222222222222222222222222222222222222
#EXTINF:-1,[UK] BBC One
acestream://3333333333333333333333333333333333333333`,
		},
		{
			name:     "empty content",
			input:    "",
			expected: "",
		},
		{
			name:     "only header",
			input:    "#EXTM3U",
			expected: "#EXTM3U",
		},
		{
			name: "already sorted",
			input: `#EXTM3U
#EXTINF:-1,A Channel
acestream://1111111111111111111111111111111111111111
#EXTINF:-1,B Channel
acestream://2222222222222222222222222222222222222222
#EXTINF:-1,C Channel
acestream://3333333333333333333333333333333333333333`,
			expected: `#EXTM3U
#EXTINF:-1,A Channel
acestream://1111111111111111111111111111111111111111
#EXTINF:-1,B Channel
acestream://2222222222222222222222222222222222222222
#EXTINF:-1,C Channel
acestream://3333333333333333333333333333333333333333`,
		},
		{
			name: "single stream",
			input: `#EXTM3U
#EXTINF:-1,Only Channel
acestream://1111111111111111111111111111111111111111`,
			expected: `#EXTM3U
#EXTINF:-1,Only Channel
acestream://1111111111111111111111111111111111111111`,
		},
		{
			name: "sort by group-title first then by name",
			input: `#EXTM3U
#EXTINF:-1 group-title="Sports",Zebra Sports
acestream://3333333333333333333333333333333333333333
#EXTINF:-1 group-title="Movies",Apple Movie
acestream://1111111111111111111111111111111111111111
#EXTINF:-1 group-title="Sports",Alpha Sports
acestream://4444444444444444444444444444444444444444
#EXTINF:-1 group-title="Movies",Banana Movie
acestream://2222222222222222222222222222222222222222`,
			expected: `#EXTM3U
#EXTINF:-1 group-title="Movies",Apple Movie
acestream://1111111111111111111111111111111111111111
#EXTINF:-1 group-title="Movies",Banana Movie
acestream://2222222222222222222222222222222222222222
#EXTINF:-1 group-title="Sports",Alpha Sports
acestream://4444444444444444444444444444444444444444
#EXTINF:-1 group-title="Sports",Zebra Sports
acestream://3333333333333333333333333333333333333333`,
		},
		{
			name: "channels without group-title come last",
			input: `#EXTM3U
#EXTINF:-1,No Group Channel
acestream://1111111111111111111111111111111111111111
#EXTINF:-1 group-title="Sports",Sports Channel
acestream://2222222222222222222222222222222222222222`,
			expected: `#EXTM3U
#EXTINF:-1 group-title="Sports",Sports Channel
acestream://2222222222222222222222222222222222222222
#EXTINF:-1,No Group Channel
acestream://1111111111111111111111111111111111111111`,
		},
		{
			name: "case-insensitive group-title sorting",
			input: `#EXTM3U
#EXTINF:-1 group-title="sports",Lowercase Sports
acestream://1111111111111111111111111111111111111111
#EXTINF:-1 group-title="SPORTS",Uppercase Sports
acestream://2222222222222222222222222222222222222222
#EXTINF:-1 group-title="Movies",Movie Channel
acestream://3333333333333333333333333333333333333333`,
			expected: `#EXTM3U
#EXTINF:-1 group-title="Movies",Movie Channel
acestream://3333333333333333333333333333333333333333
#EXTINF:-1 group-title="sports",Lowercase Sports
acestream://1111111111111111111111111111111111111111
#EXTINF:-1 group-title="SPORTS",Uppercase Sports
acestream://2222222222222222222222222222222222222222`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SortStreamsByName([]byte(tt.input))
			if string(result) != tt.expected {
				t.Errorf("SortStreamsByName() failed\nInput:\n%s\n\nExpected:\n%s\n\nGot:\n%s", tt.input, tt.expected, string(result))
			}
		})
	}
}

func TestSortStreamsByName_LargePlaylist(t *testing.T) {
	// Build a playlist with channels in reverse order
	var input strings.Builder
	input.WriteString("#EXTM3U\n")

	for i := 100; i > 0; i-- {
		input.WriteString(fmt.Sprintf("#EXTINF:-1,Channel %03d\n", i))
		input.WriteString(fmt.Sprintf("acestream://%040d\n", i))
	}

	result := SortStreamsByName([]byte(input.String()))
	resultStr := string(result)

	// Verify header is first
	if !strings.HasPrefix(resultStr, "#EXTM3U") {
		t.Error("Result should start with #EXTM3U header")
	}

	// Verify first channel is "Channel 001"
	if !strings.Contains(resultStr, "#EXTM3U\n#EXTINF:-1,Channel 001\n") {
		t.Error("First stream should be Channel 001 after sorting")
	}

	// Count streams
	streamCount := strings.Count(resultStr, "acestream://")
	if streamCount != 100 {
		t.Errorf("Expected 100 streams, got %d", streamCount)
	}
}
