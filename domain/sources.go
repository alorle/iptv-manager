package domain

import (
	"fmt"
	"strings"
)

// GetSourceName derives a source name from a URL or uses a fallback index-based name
func GetSourceName(url string, index int) string {
	// Try to extract a meaningful name from the URL
	// Look for common patterns in IPFS URLs
	if strings.Contains(url, "k51qzi5uqu5di462t7j4vu4akwfhvtjhy88qbupktvoacqfqe9uforjvhyi4wr") {
		return "elcano"
	}
	if strings.Contains(url, "k2k4r8oqlcjxsritt5mczkcn4mmvcmymbqw7113fz2flkrerfwfps004") {
		return "newera"
	}

	// Fallback to generic source name
	return fmt.Sprintf("source%d", index+1)
}
