package helpers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// SetupTestContext creates a temporary home directory, sets HOME to it, and
// initialises a "test" profile as the active profile under ~/.satusky/.
// Returns the temp dir (not the .satusky subdirectory).
func SetupTestContext(t *testing.T) string {
	t.Helper()
	dir := CreateTempDir(t)
	t.Setenv("HOME", dir)
	SetupTestProfile(t, dir, "test")
	return dir
}

// SetupTestProfile creates a named profile in dir/.satusky/profiles/ and
// sets it as the active profile in dir/.satusky/context.json.
func SetupTestProfile(t *testing.T, homeDir, profileName string) {
	t.Helper()
	satuDir := filepath.Join(homeDir, ".satusky")
	profilesDir := filepath.Join(satuDir, "profiles")
	if err := os.MkdirAll(profilesDir, 0750); err != nil {
		t.Fatalf("failed to create profiles dir: %v", err)
	}

	// Empty profile file
	profilePath := filepath.Join(profilesDir, profileName+".json")
	if err := os.WriteFile(profilePath, []byte("{}"), 0600); err != nil {
		t.Fatalf("failed to create profile file: %v", err)
	}

	// context.json points at the profile
	type rootCtx struct {
		ActiveProfile string `json:"active_profile"`
	}
	data, err := json.MarshalIndent(rootCtx{ActiveProfile: profileName}, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal context json: %v", err)
	}
	contextPath := filepath.Join(satuDir, "context.json")
	if err := os.WriteFile(contextPath, data, 0600); err != nil {
		t.Fatalf("failed to write context.json: %v", err)
	}
}

// CreateContextFile writes content into the active profile file
// (dir/.satusky/profiles/test.json) so setters can read it back.
// Kept for test backward compatibility — content must be valid CLIContext JSON.
func CreateContextFile(t *testing.T, dir, content string) string {
	t.Helper()
	profilePath := filepath.Join(dir, ".satusky", "profiles", "test.json")
	if err := os.MkdirAll(filepath.Dir(profilePath), 0750); err != nil {
		t.Fatalf("failed to create profiles dir: %v", err)
	}
	if err := os.WriteFile(profilePath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write profile file: %v", err)
	}
	return profilePath
}
