package utils

import (
	"encoding/json"
	"io"
	"os"
	"testing"
)

type listJSONThing struct {
	Name string `json:"name"`
}

// captureStdout runs fn and returns everything it wrote to stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	fn()
	if closeErr := w.Close(); closeErr != nil {
		t.Fatalf("close pipe: %v", closeErr)
	}
	os.Stdout = original
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	return string(out)
}

func withJSONOutput(t *testing.T, fn func()) {
	t.Helper()
	SetOutputFormat("json")
	t.Cleanup(func() { SetOutputFormat("table") })
	fn()
}

// A nil slice marshals to `null`, which made `jq '.[]'` fail with "Cannot
// iterate over null" for any list command that happened to return no rows.
// List commands must always emit an array.
func TestPrintListOrJSONEmitsArrayForNilSlice(t *testing.T) {
	var out string
	withJSONOutput(t, func() {
		out = captureStdout(t, func() {
			var items []listJSONThing // nil, not empty
			if !PrintListOrJSON(items, "No things found") {
				t.Error("PrintListOrJSON must report handled in JSON mode")
			}
		})
	})

	var decoded []listJSONThing
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("output %q is not valid JSON: %v", out, err)
	}
	if decoded == nil {
		t.Fatalf("nil slice must encode as [], got %q", out)
	}
	if len(decoded) != 0 {
		t.Fatalf("expected empty array, got %d items", len(decoded))
	}
}

func TestPrintListOrJSONPreservesPopulatedSlice(t *testing.T) {
	var out string
	withJSONOutput(t, func() {
		out = captureStdout(t, func() {
			PrintListOrJSON([]listJSONThing{{Name: "one"}, {Name: "two"}}, "No things found")
		})
	})

	var decoded []listJSONThing
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("output %q is not valid JSON: %v", out, err)
	}
	if len(decoded) != 2 || decoded[0].Name != "one" || decoded[1].Name != "two" {
		t.Fatalf("populated slice was altered: %q", out)
	}
}

// Table mode must still fall through to the empty message rather than being
// swallowed by the normalization.
func TestPrintListOrJSONTableModeStillReportsEmpty(t *testing.T) {
	out := captureStdout(t, func() {
		var items []listJSONThing
		if !PrintListOrJSON(items, "No things found") {
			t.Error("empty list in table mode must be handled")
		}
	})
	if out != "No things found\n" {
		t.Fatalf("expected empty message, got %q", out)
	}
}

func TestPrintListOrJSONTableModeDefersNonEmpty(t *testing.T) {
	out := captureStdout(t, func() {
		if PrintListOrJSON([]listJSONThing{{Name: "one"}}, "No things found") {
			t.Error("non-empty list in table mode must defer to the caller's table")
		}
	})
	if out != "" {
		t.Fatalf("expected no output, got %q", out)
	}
}
