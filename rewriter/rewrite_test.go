package rewriter

import (
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

	rewriter := New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rewriter.RewriteM3U([]byte(tt.input), "http://localhost:8080")
			if string(result) != tt.expected {
				t.Errorf("RewriteM3U() failed\nInput:\n%s\n\nExpected:\n%s\n\nGot:\n%s", tt.input, tt.expected, string(result))
			}
		})
	}
}

func TestRewriteM3U_CustomStreamBaseURL(t *testing.T) {
	customURL := "http://custom-server.local:8080"
	rewriter := New()

	input := `#EXTINF:-1,Test Channel
acestream://1234567890abcdef1234567890abcdef12345678`

	expected := `#EXTINF:-1,Test Channel
http://custom-server.local:8080/stream?id=1234567890abcdef1234567890abcdef12345678`

	result := rewriter.RewriteM3U([]byte(input), customURL)
	if string(result) != expected {
		t.Errorf("Custom stream base URL failed\nExpected:\n%s\n\nGot:\n%s", expected, string(result))
	}
}

func TestRewriteM3U_LargePlaylist(t *testing.T) {
	rewriter := New()

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

	result := rewriter.RewriteM3U([]byte(input), "http://localhost:8080")

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
	rewriter := New()

	input := `#EXTM3U
#EXTINF:-1 tvg-id="channel.id" tvg-name="Channel Name" tvg-logo="http://logo.png" group-title="Sports",Channel Name
acestream://1234567890abcdef1234567890abcdef12345678
#EXTINF:-1 tvg-shift="2" catchup="default",Another Channel
http://example.com/stream.m3u8`

	result := string(rewriter.RewriteM3U([]byte(input), "http://localhost:8080"))

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
	rewriter := New()

	input := `#EXTM3U
#EXTINF:-1,Test Channel
acestream://1234567890abcdef1234567890abcdef12345678`

	expected := `#EXTM3U
#EXTINF:-1,Test Channel
/stream?id=1234567890abcdef1234567890abcdef12345678`

	result := rewriter.RewriteM3U([]byte(input), "")
	if string(result) != expected {
		t.Errorf("Relative URL generation failed\nExpected:\n%s\n\nGot:\n%s", expected, string(result))
	}
}

func TestRewriteM3U_PreservesTranscodeAudio(t *testing.T) {
	rewriter := New()

	// Test with existing rewritten URL that has transcode_audio parameter
	input := `#EXTM3U
#EXTINF:-1,Test Channel
http://127.0.0.1:6878/ace/getstream?id=1234567890abcdef1234567890abcdef12345678&transcode_audio=mp3`

	expected := `#EXTM3U
#EXTINF:-1,Test Channel
http://localhost:8080/stream?id=1234567890abcdef1234567890abcdef12345678&transcode_audio=mp3`

	result := rewriter.RewriteM3U([]byte(input), "http://localhost:8080")
	if string(result) != expected {
		t.Errorf("Transcode audio preservation failed\nExpected:\n%s\n\nGot:\n%s", expected, string(result))
	}
}

func TestRewriteM3U_PreservesTranscodeAudioRelative(t *testing.T) {
	rewriter := New()

	// Test with relative URL and transcode_audio parameter
	input := `#EXTM3U
#EXTINF:-1,Test Channel
/stream?id=1234567890abcdef1234567890abcdef12345678&transcode_audio=aac`

	expected := `#EXTM3U
#EXTINF:-1,Test Channel
/stream?id=1234567890abcdef1234567890abcdef12345678&transcode_audio=aac`

	result := rewriter.RewriteM3U([]byte(input), "")
	if string(result) != expected {
		t.Errorf("Transcode audio preservation with relative URL failed\nExpected:\n%s\n\nGot:\n%s", expected, string(result))
	}
}

func TestRewriteM3U_RewritesOldFormatWithoutTranscode(t *testing.T) {
	rewriter := New()

	// Test rewriting old format URLs without transcode_audio
	input := `#EXTM3U
#EXTINF:-1,Test Channel
http://127.0.0.1:6878/ace/getstream?id=1234567890abcdef1234567890abcdef12345678&network-caching=1000`

	expected := `#EXTM3U
#EXTINF:-1,Test Channel
http://localhost:8080/stream?id=1234567890abcdef1234567890abcdef12345678`

	result := rewriter.RewriteM3U([]byte(input), "http://localhost:8080")
	if string(result) != expected {
		t.Errorf("Old format rewriting failed\nExpected:\n%s\n\nGot:\n%s", expected, string(result))
	}
}

func TestRewriteM3U_RemovesLogos(t *testing.T) {
	rewriter := New()

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

	result := string(rewriter.RewriteM3U([]byte(input), "http://localhost:8080"))

	if result != expected {
		t.Errorf("RewriteM3U() failed to remove logos\nExpected:\n%s\n\nGot:\n%s", expected, result)
	}

	// Verify no logo attributes remain
	if strings.Contains(result, "tvg-logo=") {
		t.Error("Result should not contain any tvg-logo attributes")
	}
}
