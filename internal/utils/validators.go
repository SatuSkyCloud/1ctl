package utils

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var digitRE = regexp.MustCompile(`^\d+$`)

// ValidateUUID returns an error if s is not a valid UUID.
func ValidateUUID(s string) error {
	if _, err := uuid.Parse(s); err != nil {
		return fmt.Errorf("invalid UUID: %s", s)
	}
	return nil
}

// ValidateRFC3339 returns an error if s is not a valid RFC3339 timestamp.
func ValidateRFC3339(s string) error {
	if _, err := time.Parse(time.RFC3339, s); err != nil {
		return fmt.Errorf("invalid RFC3339 timestamp: %s", s)
	}
	return nil
}

// ValidateDigits returns an error if s contains non-digit characters.
func ValidateDigits(s string) error {
	if !digitRE.MatchString(s) {
		return fmt.Errorf("must be numeric: %s", s)
	}
	return nil
}

// ValidateEnum returns an error if val is not one of the allowed options.
func ValidateEnum(val string, options []string) error {
	for _, o := range options {
		if val == o {
			return nil
		}
	}
	return fmt.Errorf("invalid value %q: must be one of [%s]", val, JoinOptions(options))
}

// ValidateTimeRange returns an error if the RFC3339 start time is not
// before the RFC3339 end time.
func ValidateTimeRange(start, end string) error {
	st, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return fmt.Errorf("invalid start time: %w", err)
	}
	et, err := time.Parse(time.RFC3339, end)
	if err != nil {
		return fmt.Errorf("invalid end time: %w", err)
	}
	if !st.Before(et) {
		return fmt.Errorf("start time must be before end time")
	}
	return nil
}

// JoinOptions joins a list of options into a comma-separated string.
func JoinOptions(opts []string) string {
	return strings.Join(opts, ", ")
}
