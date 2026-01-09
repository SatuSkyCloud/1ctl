package helpers

import (
	"os"
	"path/filepath"
	"testing"
)

// SetupTestContext creates a temporary context directory for testing
func SetupTestContext(t *testing.T) string {
	t.Helper()
	dir := CreateTempDir(t)
	t.Setenv("HOME", dir) // Override HOME for test context
	return dir
}

// CreateContextFile creates a test context file with given content
func CreateContextFile(t *testing.T, dir, content string) string {
	t.Helper()
	configDir := filepath.Join(dir, ".satusky")
	if err := os.MkdirAll(configDir, 0750); err != nil { // #nosec G301 -- test directory permissions
		t.Fatalf("Failed to create config dir: %v", err)
	}
	return CreateTestFile(t, configDir, "context.json", content)
}
