package api

import (
	"1ctl/internal/utils"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

// FormatTimeAgo returns a human-readable string representing how long ago a time was
func FormatTimeAgo(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 48*time.Hour:
		return "yesterday"
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	case diff < 365*24*time.Hour:
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(diff.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

// ToUUID parses a UUID string, returning uuid.Nil on parse failure.
//
// Contract: the caller MUST have validated the input upstream (e.g., from a
// successful API response or via ParseUUID at the system boundary). Use this
// only when a parse error is structurally impossible — never on user input,
// where it would turn a malformed UUID into a silent backend "not found".
func ToUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}

// ParseUUID parses a UUID string and returns a descriptive error on failure.
// Use at system boundaries (CLI args, env vars, context state) where a
// malformed value should surface as a clear error, not silently degrade.
func ParseUUID(s string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid UUID %q: %w", s, err)
	}
	return id, nil
}

// SafeInt32 safely converts an int to int32, checking for overflow
func SafeInt32(n int) (int32, error) {
	if n > math.MaxInt32 || n < math.MinInt32 {
		return 0, utils.NewError(fmt.Sprintf("integer overflow: value %d out of int32 range", n), nil)
	}
	return int32(n), nil
}

// GenerateDomainName calls the backend's domain generator endpoint and returns
// the auto-assigned domain name (adjective+animal-suffix.satusky.com).
// projectName is unused — the backend owns domain assignment.
func GenerateDomainName(_ string) (string, error) {
	var resp struct {
		Error bool   `json:"error"`
		Data  string `json:"data"`
	}
	if err := makeRequest("GET", "/ingresses/domainNameGenerator", nil, &resp); err != nil {
		return "", fmt.Errorf("failed to get auto-assigned domain: %w", err)
	}
	if resp.Data == "" {
		return "", fmt.Errorf("backend returned an empty domain name")
	}
	return resp.Data, nil
}
