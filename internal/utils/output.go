package utils

import (
	"encoding/json"
	"fmt"
)

var outputFormat = "table"

// SetOutputFormat sets the global output format ("table" or "json").
func SetOutputFormat(format string) {
	outputFormat = format
}

// IsJSONOutput returns true when --output json was requested.
func IsJSONOutput() bool {
	return outputFormat == "json"
}

// TryPrintJSON marshals data to indented JSON and prints it if JSON output is enabled.
// Returns true so callers can do: if utils.TryPrintJSON(data) { return nil }
func TryPrintJSON(data interface{}) bool {
	if !IsJSONOutput() {
		return false
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("{\"error\": %q}\n", err.Error())
	} else {
		fmt.Println(string(b))
	}
	return true
}
