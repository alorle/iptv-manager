package rewriter

import (
	"fmt"
	"strings"
	"testing"
)

func TestDeduplicateStreams(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "duplicate acestream IDs - keeps first",
			input: `#EXTM3U
#EXTINF:-1,Channel A
acestream://1234567890abcdef1234567890abcdef12345678
#EXTINF:-1,Channel B
acestream://1234567890abcdef1234567890abcdef12345678
#EXTINF:-1,Channel C
acestream://abcdefabcdefabcdefabcdefabcdefabcdefabcd`,
			expected: `#EXTM3U
#EXTINF:-1,Channel A
acestream://1234567890abcdef1234567890abcdef12345678
#EXTINF:-1,Channel C
acestream://abcdefabcdefabcdefabcdefabcdefabcdefabcd`,
		},
		{
			name: "no duplicates",
			input: `#EXTM3U
#EXTINF:-1,Channel A
acestream://1111111111111111111111111111111111111111
#EXTINF:-1,Channel B
acestream://2222222222222222222222222222222222222222`,
			expected: `#EXTM3U
#EXTINF:-1,Channel A
acestream://1111111111111111111111111111111111111111
#EXTINF:-1,Channel B
acestream://2222222222222222222222222222222222222222`,
		},
		{
			name: "multiple duplicates - keeps first of each",
			input: `#EXTM3U
#EXTINF:-1,First A
acestream://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
#EXTINF:-1,First B
acestream://bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
#EXTINF:-1,Duplicate A
acestream://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
#EXTINF:-1,Duplicate B
acestream://bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb`,
			expected: `#EXTM3U
#EXTINF:-1,First A
acestream://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
#EXTINF:-1,First B
acestream://bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb`,
		},
		{
			name: "non-acestream URLs are preserved - no deduplication",
			input: `#EXTM3U
#EXTINF:-1,HTTP Stream 1
http://example.com/stream.m3u8
#EXTINF:-1,HTTP Stream 2
http://example.com/stream.m3u8
#EXTINF:-1,HTTPS Stream
https://example.com/stream.m3u8`,
			expected: `#EXTM3U
#EXTINF:-1,HTTP Stream 1
http://example.com/stream.m3u8
#EXTINF:-1,HTTP Stream 2
http://example.com/stream.m3u8
#EXTINF:-1,HTTPS Stream
https://example.com/stream.m3u8`,
		},
		{
			name: "mixed acestream and non-acestream",
			input: `#EXTM3U
#EXTINF:-1,Acestream A
acestream://1234567890abcdef1234567890abcdef12345678
#EXTINF:-1,HTTP Stream
http://example.com/stream.m3u8
#EXTINF:-1,Duplicate Acestream A
acestream://1234567890abcdef1234567890abcdef12345678
#EXTINF:-1,Another HTTP Stream
http://example.com/another.m3u8`,
			expected: `#EXTM3U
#EXTINF:-1,Acestream A
acestream://1234567890abcdef1234567890abcdef12345678
#EXTINF:-1,HTTP Stream
http://example.com/stream.m3u8
#EXTINF:-1,Another HTTP Stream
http://example.com/another.m3u8`,
		},
		{
			name: "acestream with trailing whitespace",
			input: `#EXTM3U
#EXTINF:-1,Channel A
acestream://1234567890abcdef1234567890abcdef12345678
#EXTINF:-1,Channel B
acestream://1234567890abcdef1234567890abcdef12345678`,
			expected: `#EXTM3U
#EXTINF:-1,Channel A
acestream://1234567890abcdef1234567890abcdef12345678`,
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
			name: "three duplicates - keeps only first",
			input: `#EXTM3U
#EXTINF:-1,First
acestream://1234567890abcdef1234567890abcdef12345678
#EXTINF:-1,Second (duplicate)
acestream://1234567890abcdef1234567890abcdef12345678
#EXTINF:-1,Third (duplicate)
acestream://1234567890abcdef1234567890abcdef12345678`,
			expected: `#EXTM3U
#EXTINF:-1,First
acestream://1234567890abcdef1234567890abcdef12345678`,
		},
		{
			name: "preserves metadata attributes",
			input: `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" group-title="Sports",First
acestream://1234567890abcdef1234567890abcdef12345678
#EXTINF:-1 tvg-id="ch2" tvg-name="Channel 2" group-title="Movies",Duplicate
acestream://1234567890abcdef1234567890abcdef12345678`,
			expected: `#EXTM3U
#EXTINF:-1 tvg-id="ch1" tvg-name="Channel 1" group-title="Sports",First
acestream://1234567890abcdef1234567890abcdef12345678`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeduplicateStreams([]byte(tt.input))
			if string(result) != tt.expected {
				t.Errorf("DeduplicateStreams() failed\nInput:\n%s\n\nExpected:\n%s\n\nGot:\n%s", tt.input, tt.expected, string(result))
			}
		})
	}
}

func TestDeduplicateStreams_LargePlaylist(t *testing.T) {
	// Build a large playlist with duplicates
	var input strings.Builder
	input.WriteString("#EXTM3U\n")

	// Add 100 unique streams
	for i := 0; i < 100; i++ {
		input.WriteString(fmt.Sprintf("#EXTINF:-1,Channel %d\n", i))
		input.WriteString(fmt.Sprintf("acestream://%040d\n", i))
	}

	// Add 100 duplicate streams
	for i := 0; i < 100; i++ {
		input.WriteString(fmt.Sprintf("#EXTINF:-1,Duplicate Channel %d\n", i))
		input.WriteString(fmt.Sprintf("acestream://%040d\n", i))
	}

	result := DeduplicateStreams([]byte(input.String()))
	resultStr := string(result)

	// Count stream entries (should be 100 unique + header)
	streamCount := strings.Count(resultStr, "acestream://")
	if streamCount != 100 {
		t.Errorf("Expected 100 unique streams after deduplication, got %d", streamCount)
	}

	// Verify header is preserved
	if !strings.HasPrefix(resultStr, "#EXTM3U") {
		t.Error("Header should be preserved")
	}
}

func TestDeduplicateStreams_PreservesOrder(t *testing.T) {
	input := `#EXTM3U
#EXTINF:-1,Channel A
acestream://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
#EXTINF:-1,Channel B
acestream://bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
#EXTINF:-1,Channel C
acestream://cccccccccccccccccccccccccccccccccccccccc
#EXTINF:-1,Duplicate B
acestream://bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb`

	expected := `#EXTM3U
#EXTINF:-1,Channel A
acestream://aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
#EXTINF:-1,Channel B
acestream://bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb
#EXTINF:-1,Channel C
acestream://cccccccccccccccccccccccccccccccccccccccc`

	result := DeduplicateStreams([]byte(input))
	if string(result) != expected {
		t.Errorf("DeduplicateStreams() should preserve order\nExpected:\n%s\n\nGot:\n%s", expected, string(result))
	}
}
