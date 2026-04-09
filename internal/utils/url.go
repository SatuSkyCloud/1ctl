package utils

import "strings"

// IsLocalhostURL returns true if the URL points to localhost or 127.0.0.1.
func IsLocalhostURL(rawURL string) bool {
	return strings.HasPrefix(rawURL, "http://localhost") ||
		strings.HasPrefix(rawURL, "http://127.0.0.1") ||
		strings.HasPrefix(rawURL, "http://[::1]")
}
