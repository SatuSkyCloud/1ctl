package validator

import (
	"1ctl/internal/utils"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var memoryPattern = regexp.MustCompile(`^(\d+)(Mi|Gi)?$`)
var cpuPattern = regexp.MustCompile(`^(\d+)(m)?$`)
var domainPattern = regexp.MustCompile(`^(\*\.)?[a-zA-Z0-9][-a-zA-Z0-9]*(\.[a-zA-Z0-9][-a-zA-Z0-9]*)*\.[a-zA-Z]{2,}$`)

func ValidateCPU(cpu string) error {
	if !cpuPattern.MatchString(cpu) {
		return utils.NewError("CPU must be in format: <number> or <number>m (e.g., 1 or 100m)", nil)
	}

	if cpu[len(cpu)-1] == 'm' {
		// Handle millicpu format
		millicpu, err := strconv.Atoi(cpu[:len(cpu)-1])
		if err != nil {
			return utils.NewError("invalid millicpu value", err)
		}
		if millicpu <= 0 {
			return utils.NewError("CPU millicores must be greater than 0", nil)
		}
		if millicpu < 10 {
			return utils.NewError("CPU millicores must be at least 10m", nil)
		}
	} else {
		// Handle whole CPU units
		cpuValue, err := strconv.Atoi(cpu)
		if err != nil {
			return utils.NewError("invalid CPU value", err)
		}
		if cpuValue <= 0 {
			return utils.NewError("CPU must be greater than 0", nil)
		}
	}
	return nil
}

func ValidateMemory(memory string) error {
	if memory == "" {
		return utils.NewError("memory value cannot be empty", nil)
	}

	if !memoryPattern.MatchString(memory) {
		return utils.NewError("memory must be in format: <number>Mi or <number>Gi (e.g., 512Mi, 2Gi)", nil)
	}

	matches := memoryPattern.FindStringSubmatch(memory)
	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return utils.NewError("invalid memory value: %v", err)
	}

	if value <= 0 {
		return utils.NewError("memory value must be positive", nil)
	}

	return nil
}

func ValidateDomain(domain string) error {
	if domain == "" {
		return nil
	}

	if !domainPattern.MatchString(domain) {
		return utils.NewError("invalid domain format", nil)
	}

	return nil
}

// ValidateWaitFor validates a --wait-for value in "host:port" format.
func ValidateWaitFor(value string) (host string, port int32, err error) {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return "", 0, utils.NewError(fmt.Sprintf("--wait-for %q must be in host:port format (e.g., postgres:5432)", value), nil)
	}
	host = strings.TrimSpace(parts[0])
	if host == "" {
		return "", 0, utils.NewError(fmt.Sprintf("--wait-for %q: host cannot be empty", value), nil)
	}
	p, parseErr := strconv.Atoi(strings.TrimSpace(parts[1]))
	if parseErr != nil || p < 1 || p > 65535 {
		return "", 0, utils.NewError(fmt.Sprintf("--wait-for %q: port must be a number between 1 and 65535", value), nil)
	}
	return host, int32(p), nil //nolint:gosec // port is validated to be within 1–65535
}
