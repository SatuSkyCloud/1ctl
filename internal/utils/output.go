package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
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

// PrintListOrJSON handles both JSON and table output for list commands.
//
//	When --output json is set: prints items as JSON (empty array or populated) and returns true.
//	In table mode with empty items: prints emptyMsg and returns true.
//	In table mode with non-empty items: returns false so the caller can render the table.
//
// Usage:
//
//	items, _ := api.ListThings()
//	if utils.PrintListOrJSON(items, "No things found") {
//	    return nil
//	}
//	utils.PrintTable(headers, rows)
func PrintListOrJSON(items interface{}, emptyMsg string) bool {
	if TryPrintJSON(items) {
		return true
	}
	v := reflect.ValueOf(items)
	if v.Kind() == reflect.Slice && v.Len() == 0 && emptyMsg != "" {
		fmt.Println(emptyMsg)
		return true
	}
	return false
}
