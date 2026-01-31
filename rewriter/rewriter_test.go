package rewriter

import (
	"fmt"
	"strings"
	"testing"
)

func TestRewriteM3U(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "single acestream URL",
			input: `#EXTM3U
#EXTINF:-1,Example Channel
acestream://1234567890abcdef1234567890abcdef12345678`,
			expected: `#EXTM3U
#EXTINF:-1,Example Channel
http://localhost:8080/stream?id=1234567890abcdef1234567890abcdef12345678`,
		},
		{
			name: "multiple acestream URLs with metadata",
			input: `#EXTM3U
#EXTINF:-1 tvg-id="channel1" tvg-name="Channel 1",Channel 1
acestream://1111111111111111111111111111111111111111
#EXTINF:-1 tvg-id="channel2" tvg-name="Channel 2",Channel 2
acestream://2222222222222222222222222222222222222222`,
			expected: `#EXTM3U
#EXTINF:-1 tvg-id="channel1" tvg-name="Channel 1",Channel 1
http://localhost:8080/stream?id=1111111111111111111111111111111111111111
#EXTINF:-1 tvg-id="channel2" tvg-name="Channel 2",Channel 2
http://localhost:8080/stream?id=2222222222222222222222222222222222222222`,
		},
		{
			name: "mixed acestream and regular URLs",
			input: `#EXTM3U
#EXTINF:-1,Regular HTTP Stream
http://example.com/stream.m3u8
#EXTINF:-1,Acestream Channel
acestream://abcdef1234567890abcdef1234567890abcdef12
#EXTINF:-1,Another HTTP Stream
https://example.com/another.m3u8`,
			expected: `#EXTM3U
#EXTINF:-1,Regular HTTP Stream
http://example.com/stream.m3u8
#EXTINF:-1,Acestream Channel
http://localhost:8080/stream?id=abcdef1234567890abcdef1234567890abcdef12
#EXTINF:-1,Another HTTP Stream
https://example.com/another.m3u8`,
		},
		{
			name: "acestream URL with trailing whitespace",
			input: `#EXTINF:-1,Test Channel
acestream://1234567890abcdef1234567890abcdef12345678   `,
			expected: `#EXTINF:-1,Test Channel
http://localhost:8080/stream?id=1234567890abcdef1234567890abcdef12345678`,
		},
		{
			name:     "acestream URL with Windows line ending",
			input:    "#EXTINF:-1,Test Channel\nacestream://1234567890abcdef1234567890abcdef12345678",
			expected: "#EXTINF:-1,Test Channel\nhttp://localhost:8080/stream?id=1234567890abcdef1234567890abcdef12345678",
		},
		{
			name:     "empty content",
			input:    "",
			expected: "",
		},
		{
			name: "empty lines preserved",
			input: `#EXTM3U

#EXTINF:-1,Channel
acestream://1234567890abcdef1234567890abcdef12345678

`,
			expected: `#EXTM3U

#EXTINF:-1,Channel
http://localhost:8080/stream?id=1234567890abcdef1234567890abcdef12345678

`,
		},
		{
			name:     "only metadata (no URLs)",
			input:    `#EXTM3U\n#EXTINF:-1,Test`,
			expected: `#EXTM3U\n#EXTINF:-1,Test`,
		},
		{
			name: "acestream with short ID (edge case)",
			input: `#EXTINF:-1,Short ID
acestream://abc`,
			expected: `#EXTINF:-1,Short ID
http://localhost:8080/stream?id=abc`,
		},
		{
			name: "line that contains acestream:// but doesn't start with it",
			input: `#EXTM3U
#EXTINF:-1,Not an acestream URL: acestream://should-not-be-rewritten
http://example.com/stream.m3u8`,
			expected: `#EXTM3U
#EXTINF:-1,Not an acestream URL: acestream://should-not-be-rewritten
http://example.com/stream.m3u8`,
		},
		{
			name:     "malformed acestream URL (just the protocol)",
			input:    "acestream://",
			expected: "http://localhost:8080/stream?id=",
		},
	}

	rewriter := New("http://localhost:8080")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rewriter.RewriteM3U([]byte(tt.input))
			if string(result) != tt.expected {
				t.Errorf("RewriteM3U() failed\nInput:\n%s\n\nExpected:\n%s\n\nGot:\n%s", tt.input, tt.expected, string(result))
			}
		})
	}
}

func TestRewriteM3U_CustomStreamBaseURL(t *testing.T) {
	customURL := "http://custom-server.local:8080"
	rewriter := New(customURL)

	input := `#EXTINF:-1,Test Channel
acestream://1234567890abcdef1234567890abcdef12345678`

	expected := `#EXTINF:-1,Test Channel
http://custom-server.local:8080/stream?id=1234567890abcdef1234567890abcdef12345678`

	result := rewriter.RewriteM3U([]byte(input))
	if string(result) != expected {
		t.Errorf("Custom stream base URL failed\nExpected:\n%s\n\nGot:\n%s", expected, string(result))
	}
}

func TestRewriteM3U_LargePlaylist(t *testing.T) {
	rewriter := New("http://localhost:8080")

	// Build a large playlist with 1000 channels
	var input string
	input = "#EXTM3U\n"
	for i := 0; i < 1000; i++ {
		input += "#EXTINF:-1,Channel " + string(rune('0'+i%10)) + "\n"
		if i%2 == 0 {
			input += "acestream://1234567890abcdef123456789000000000000000\n"
		} else {
			input += "http://example.com/stream" + string(rune('0'+i%10)) + ".m3u8\n"
		}
	}

	result := rewriter.RewriteM3U([]byte(input))

	// Verify result is not empty and contains rewritten URLs
	if len(result) == 0 {
		t.Error("Result should not be empty for large playlist")
	}

	resultStr := string(result)
	if !strings.Contains(resultStr, "http://localhost:8080/stream?id=") {
		t.Error("Result should contain rewritten acestream URLs")
	}

	// Count the number of rewritten URLs (should be 500)
	count := strings.Count(resultStr, "http://localhost:8080/stream?id=")
	if count != 500 {
		t.Errorf("Expected 500 rewritten URLs, got %d", count)
	}
}

func TestRewriteM3U_PreservesAllMetadata(t *testing.T) {
	rewriter := New("http://localhost:8080")

	input := `#EXTM3U
#EXTINF:-1 tvg-id="channel.id" tvg-name="Channel Name" tvg-logo="http://logo.png" group-title="Sports",Channel Name
acestream://1234567890abcdef1234567890abcdef12345678
#EXTINF:-1 tvg-shift="2" catchup="default",Another Channel
http://example.com/stream.m3u8`

	result := string(rewriter.RewriteM3U([]byte(input)))

	// Check that logo metadata is removed but other metadata is preserved
	if strings.Contains(result, `tvg-logo="http://logo.png"`) {
		t.Error("Logo metadata should be removed")
	}

	if !strings.Contains(result, `#EXTINF:-1 tvg-id="channel.id" tvg-name="Channel Name" group-title="Sports",Channel Name`) {
		t.Error("First metadata line not preserved correctly (without logo)")
	}

	if !strings.Contains(result, `#EXTINF:-1 tvg-shift="2" catchup="default",Another Channel`) {
		t.Error("Second metadata line not preserved correctly")
	}

	if !strings.Contains(result, "http://example.com/stream.m3u8") {
		t.Error("Regular URL not preserved correctly")
	}

	if !strings.Contains(result, "http://localhost:8080/stream?id=1234567890abcdef1234567890abcdef12345678") {
		t.Error("Acestream URL not rewritten correctly")
	}
}

func TestRewriteM3U_RelativeURLs(t *testing.T) {
	// Test with empty stream base URL - should generate relative URLs
	rewriter := New("")

	input := `#EXTM3U
#EXTINF:-1,Test Channel
acestream://1234567890abcdef1234567890abcdef12345678`

	expected := `#EXTM3U
#EXTINF:-1,Test Channel
/stream?id=1234567890abcdef1234567890abcdef12345678`

	result := rewriter.RewriteM3U([]byte(input))
	if string(result) != expected {
		t.Errorf("Relative URL generation failed\nExpected:\n%s\n\nGot:\n%s", expected, string(result))
	}
}

func TestRewriteM3U_PreservesTranscodeAudio(t *testing.T) {
	rewriter := New("http://localhost:8080")

	// Test with existing rewritten URL that has transcode_audio parameter
	input := `#EXTM3U
#EXTINF:-1,Test Channel
http://127.0.0.1:6878/ace/getstream?id=1234567890abcdef1234567890abcdef12345678&transcode_audio=mp3`

	expected := `#EXTM3U
#EXTINF:-1,Test Channel
http://localhost:8080/stream?id=1234567890abcdef1234567890abcdef12345678&transcode_audio=mp3`

	result := rewriter.RewriteM3U([]byte(input))
	if string(result) != expected {
		t.Errorf("Transcode audio preservation failed\nExpected:\n%s\n\nGot:\n%s", expected, string(result))
	}
}

func TestRewriteM3U_PreservesTranscodeAudioRelative(t *testing.T) {
	rewriter := New("")

	// Test with relative URL and transcode_audio parameter
	input := `#EXTM3U
#EXTINF:-1,Test Channel
/stream?id=1234567890abcdef1234567890abcdef12345678&transcode_audio=aac`

	expected := `#EXTM3U
#EXTINF:-1,Test Channel
/stream?id=1234567890abcdef1234567890abcdef12345678&transcode_audio=aac`

	result := rewriter.RewriteM3U([]byte(input))
	if string(result) != expected {
		t.Errorf("Transcode audio preservation with relative URL failed\nExpected:\n%s\n\nGot:\n%s", expected, string(result))
	}
}

func TestRewriteM3U_RewritesOldFormatWithoutTranscode(t *testing.T) {
	rewriter := New("http://localhost:8080")

	// Test rewriting old format URLs without transcode_audio
	input := `#EXTM3U
#EXTINF:-1,Test Channel
http://127.0.0.1:6878/ace/getstream?id=1234567890abcdef1234567890abcdef12345678&network-caching=1000`

	expected := `#EXTM3U
#EXTINF:-1,Test Channel
http://localhost:8080/stream?id=1234567890abcdef1234567890abcdef12345678`

	result := rewriter.RewriteM3U([]byte(input))
	if string(result) != expected {
		t.Errorf("Old format rewriting failed\nExpected:\n%s\n\nGot:\n%s", expected, string(result))
	}
}

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

func TestRewriteM3U_RemovesLogos(t *testing.T) {
	rewriter := New("http://localhost:8080")

	input := `#EXTM3U
#EXTINF:-1 tvg-id="channel1" tvg-logo="http://example.com/logo1.png" tvg-name="Channel 1" group-title="Sports",Channel 1
acestream://1111111111111111111111111111111111111111
#EXTINF:-1 tvg-logo="https://example.com/logo2.jpg" tvg-id="channel2" tvg-name="Channel 2",Channel 2
acestream://2222222222222222222222222222222222222222
#EXTINF:-1 tvg-id="channel3" tvg-name="Channel 3",Channel 3
http://example.com/stream.m3u8`

	expected := `#EXTM3U
#EXTINF:-1 tvg-id="channel1" tvg-name="Channel 1" group-title="Sports",Channel 1
http://localhost:8080/stream?id=1111111111111111111111111111111111111111
#EXTINF:-1 tvg-id="channel2" tvg-name="Channel 2",Channel 2
http://localhost:8080/stream?id=2222222222222222222222222222222222222222
#EXTINF:-1 tvg-id="channel3" tvg-name="Channel 3",Channel 3
http://example.com/stream.m3u8`

	result := string(rewriter.RewriteM3U([]byte(input)))

	if result != expected {
		t.Errorf("RewriteM3U() failed to remove logos\nExpected:\n%s\n\nGot:\n%s", expected, result)
	}

	// Verify no logo attributes remain
	if strings.Contains(result, "tvg-logo=") {
		t.Error("Result should not contain any tvg-logo attributes")
	}
}

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
			name: "empty content",
			input:    "",
			expected: "",
		},
		{
			name: "only header",
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
			result := ExtractDisplayName(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractDisplayName() failed\nInput:    %s\nExpected: %s\nGot:      %s", tt.input, tt.expected, result)
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
