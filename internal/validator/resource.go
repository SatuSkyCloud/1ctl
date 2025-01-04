package validator

import (
	"fmt"
	"regexp"
	"strconv"
)

var memoryPattern = regexp.MustCompile(`^(\d+)(Mi|Gi)?$`)
var cpuPattern = regexp.MustCompile(`^(\d+)(m)?$`)
var domainPattern = regexp.MustCompile(`^(\*\.)?[a-zA-Z0-9][-a-zA-Z0-9]*(\.[a-zA-Z0-9][-a-zA-Z0-9]*)*\.[a-zA-Z]{2,}$`)

func ValidateCPU(cpu string) error {
	if !cpuPattern.MatchString(cpu) {
		return fmt.Errorf("CPU must be in format: <number> or <number>m (e.g., 1 or 100m)")
	}

	if cpu[len(cpu)-1] == 'm' {
		// Handle millicpu format
		millicpu, err := strconv.Atoi(cpu[:len(cpu)-1])
		if err != nil {
			return fmt.Errorf("invalid millicpu value")
		}
		if millicpu <= 0 {
			return fmt.Errorf("CPU millicores must be greater than 0")
		}
		if millicpu < 10 {
			return fmt.Errorf("CPU millicores must be at least 10m")
		}
	} else {
		// Handle whole CPU units
		cpuValue, err := strconv.Atoi(cpu)
		if err != nil {
			return fmt.Errorf("invalid CPU value")
		}
		if cpuValue <= 0 {
			return fmt.Errorf("CPU must be greater than 0")
		}
	}
	return nil
}

func ValidateMemory(memory string) error {
	if memory == "" {
		return fmt.Errorf("memory value cannot be empty")
	}

	if !memoryPattern.MatchString(memory) {
		return fmt.Errorf("memory must be in format: <number>Mi or <number>Gi (e.g., 512Mi, 2Gi)")
	}

	matches := memoryPattern.FindStringSubmatch(memory)
	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return fmt.Errorf("invalid memory value: %v", err)
	}

	if value <= 0 {
		return fmt.Errorf("memory value must be positive")
	}

	return nil
}

func ValidateDomain(domain string) error {
	if domain == "" {
		return nil
	}

	if !domainPattern.MatchString(domain) {
		return fmt.Errorf("invalid domain format")
	}

	return nil
}
