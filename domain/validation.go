package domain

// IsValidAcestreamID validates that a content ID is exactly 40 hexadecimal characters
func IsValidAcestreamID(id string) bool {
	if len(id) != AcestreamIDLength {
		return false
	}
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
