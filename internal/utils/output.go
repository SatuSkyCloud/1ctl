package utils

import (
	"encoding/json"
	"fmt"
	"reflect"
)

var outputFormat = "table"

// SetOutputFormat sets the global output format ("table", "wide" or "json").
func SetOutputFormat(format string) {
	outputFormat = format
}

// IsJSONOutput returns true when --output json was requested.
func IsJSONOutput() bool {
	return outputFormat == "json"
}

// IsWideOutput returns true when --output wide was requested.
//
// Wide is a table variant, not a third rendering mode: a command that has no
// extra columns to show renders its normal table, the same way kubectl does for
// a resource with no wide columns. Only commands that opt in read this.
func IsWideOutput() bool {
	return outputFormat == "wide"
}

// TryPrintJSON marshals data to indented JSON and prints it if JSON output is enabled.
// Returns true so callers can do: if utils.TryPrintJSON(data) { return nil }
//
// A nil slice is emitted as `[]` rather than `null`. Scripts pipe this output
// into `jq '.[]'`, which fails with "Cannot iterate over null" on an empty
// result instead of iterating zero times. Normalizing here rather than in
// PrintListOrJSON makes the guarantee unconditional: several commands print
// arrays through this function directly, and an exception would have to be
// documented and remembered at every one of them.
func TryPrintJSON(data interface{}) bool {
	if !IsJSONOutput() {
		return false
	}
	b, err := json.MarshalIndent(emptySliceForNil(data), "", "  ")
	if err != nil {
		fmt.Printf("{\"error\": %q}\n", err.Error())
	} else {
		fmt.Println(string(b))
	}
	return true
}

// emptySliceForNil replaces a nil slice with an empty one of the same type and
// leaves every other value untouched.
func emptySliceForNil(data interface{}) interface{} {
	v := reflect.ValueOf(data)
	if v.IsValid() && v.Kind() == reflect.Slice && v.IsNil() {
		return reflect.MakeSlice(v.Type(), 0, 0).Interface()
	}
	return data
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
	// TryPrintJSON already emits `[]` for a nil slice.
	if TryPrintJSON(items) {
		return true
	}
	v := reflect.ValueOf(items)
	if v.IsValid() && v.Kind() == reflect.Slice && v.Len() == 0 && emptyMsg != "" {
		fmt.Println(emptyMsg)
		return true
	}
	return false
}
