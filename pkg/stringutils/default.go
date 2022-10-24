package stringutils

// OrDefault returns the string if it's not empty, or the default.
func OrDefault(s, defaultValue string) string {
	if s != "" {
		return s
	}
	return defaultValue
}
