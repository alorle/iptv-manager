package domain

// IsValidContentID validates that a content ID is exactly 40 hexadecimal characters
func IsValidContentID(id string) bool {
	if len(id) != ContentIDLength {
		return false
	}
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// IsValidAcestreamID is deprecated. Use IsValidContentID instead.
func IsValidAcestreamID(id string) bool {
	return IsValidContentID(id)
}
