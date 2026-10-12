package context

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// EnsureProfile backs `1ctl profile create`, which CI runs as a bootstrap —
// often in more than one step of the same job. Failing on the second call made
// every workflow in the organisation write `|| true`, so the first case below
// is the behaviour those jobs depend on.
func TestEnsureProfileCreatesThenToleratesRepeat(t *testing.T) {
	dir := setupTestContext(t)

	created, err := Default().EnsureProfile("ci", "https://api.example.test")
	if err != nil {
		t.Fatalf("first call: unexpected error: %v", err)
	}
	if !created {
		t.Fatal("first call: expected the profile to be written")
	}

	// The second bootstrap in the same job must succeed and report that it
	// wrote nothing, so the caller can avoid claiming "created".
	created, err = Default().EnsureProfile("ci", "https://api.example.test")
	if err != nil {
		t.Fatalf("repeat call: expected success, got: %v", err)
	}
	if created {
		t.Fatal("repeat call: expected created=false")
	}

	profilePath := filepath.Join(dir, "profiles", "ci.json")
	data, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatalf("profile file missing: %v", err)
	}
	var stored CLIContext
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("profile file is not valid JSON: %v", err)
	}
	if stored.APIURL != "https://api.example.test" {
		t.Fatalf("API URL was rewritten: got %q", stored.APIURL)
	}
}

// An existing profile that points somewhere else is a real conflict, not a
// repeat of the same bootstrap, and must stay an error.
func TestEnsureProfileRefusesConflictingAPIURL(t *testing.T) {
	setupTestContext(t)

	if _, err := Default().EnsureProfile("ci", "https://api.example.test"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	_, err := Default().EnsureProfile("ci", "https://other.example.test")
	if err == nil {
		t.Fatal("expected a conflict error when the API URL differs")
	}
	for _, want := range []string{"already exists", "https://api.example.test", "profile delete"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error should name %q so the fix is obvious; got: %v", want, err)
		}
	}
}

// `profile create name` with no --url must not clobber an existing profile's URL.
func TestEnsureProfileWithoutURLKeepsExisting(t *testing.T) {
	dir := setupTestContext(t)

	if _, err := Default().EnsureProfile("ci", "https://api.example.test"); err != nil {
		t.Fatalf("setup: %v", err)
	}
	created, err := Default().EnsureProfile("ci", "")
	if err != nil {
		t.Fatalf("expected success, got: %v", err)
	}
	if created {
		t.Fatal("expected created=false")
	}
	_, err = os.ReadFile(filepath.Join(dir, "profiles", "ci.json"))
	if err != nil {
		t.Fatalf("profile file should be untouched: %v", err)
	}
}

func TestEnsureProfileRejectsInvalidInput(t *testing.T) {
	setupTestContext(t)

	if _, err := Default().EnsureProfile("!!!", ""); err == nil {
		t.Error("expected a name that sanitizes to empty to be rejected")
	}
	if _, err := Default().EnsureProfile("ci", "http://insecure.example.test"); err == nil {
		t.Error("expected a non-HTTPS error for a non-localhost URL")
	}
}
