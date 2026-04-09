package context

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// setupTestContext creates a temp config dir with a "test" profile already active.
// All getters/setters operate on the profile file, not context.json.
func setupTestContext(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	testConfigDir := filepath.Join(dir, ".satusky")
	if err := os.MkdirAll(testConfigDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Create profiles directory and an empty "test" profile
	profilesDir := filepath.Join(testConfigDir, "profiles")
	if err := os.MkdirAll(profilesDir, 0750); err != nil {
		t.Fatalf("Failed to create profiles dir: %v", err)
	}
	profileFile := filepath.Join(profilesDir, "test.json")
	if err := os.WriteFile(profileFile, []byte("{}"), 0600); err != nil {
		t.Fatalf("Failed to create test profile: %v", err)
	}

	// Point context.json at the "test" profile
	contextFile := filepath.Join(testConfigDir, "context.json")
	if err := os.WriteFile(contextFile, []byte(`{"active_profile":"test"}`), 0600); err != nil {
		t.Fatalf("Failed to write context.json: %v", err)
	}

	t.Setenv("HOME", dir)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", dir)
	}
	return dir
}

func TestContextOperations(t *testing.T) {
	homeDir := setupTestContext(t)
	configDir = filepath.Join(homeDir, ".satusky")

	testToken := "test-token-123"
	testNamespace := "test-org"
	testUserID := "user-123"
	testConfigKey := "config-key-123"

	t.Run("token operations", func(t *testing.T) {
		if err := SetToken(testToken); err != nil {
			t.Fatalf("SetToken() error = %v", err)
		}
		if got := GetToken(); got != testToken {
			t.Errorf("GetToken() = %v, want %v", got, testToken)
		}
	})

	t.Run("namespace operations", func(t *testing.T) {
		if err := SetCurrentNamespace(testNamespace); err != nil {
			t.Fatalf("SetCurrentNamespace() error = %v", err)
		}
		if got := GetCurrentNamespace(); got != testNamespace {
			t.Errorf("GetCurrentNamespace() = %v, want %v", got, testNamespace)
		}
	})

	t.Run("user ID operations", func(t *testing.T) {
		if err := SetUserID(testUserID); err != nil {
			t.Fatalf("SetUserID() error = %v", err)
		}
		if got := GetUserID(); got != testUserID {
			t.Errorf("GetUserID() = %v, want %v", got, testUserID)
		}
	})

	t.Run("config key operations", func(t *testing.T) {
		if err := SetUserConfigKey(testConfigKey); err != nil {
			t.Fatalf("SetUserConfigKey() error = %v", err)
		}
		if got := GetUserConfigKey(); got != testConfigKey {
			t.Errorf("GetUserConfigKey() = %v, want %v", got, testConfigKey)
		}
	})

	t.Run("organization ID operations", func(t *testing.T) {
		testOrgID := "org-123-uuid"
		if err := SetCurrentOrgID(testOrgID); err != nil {
			t.Fatalf("SetCurrentOrgID() error = %v", err)
		}
		if got := GetCurrentOrgID(); got != testOrgID {
			t.Errorf("GetCurrentOrgID() = %v, want %v", got, testOrgID)
		}
	})

	t.Run("organization name operations", func(t *testing.T) {
		testOrgName := "Test Organization"
		if err := SetCurrentOrgName(testOrgName); err != nil {
			t.Fatalf("SetCurrentOrgName() error = %v", err)
		}
		if got := GetCurrentOrgName(); got != testOrgName {
			t.Errorf("GetCurrentOrgName() = %v, want %v", got, testOrgName)
		}
	})

	t.Run("set current organization", func(t *testing.T) {
		testOrgID := "org-456-uuid"
		testOrgName := "Complete Org"
		testNS := "complete-org-namespace"

		if err := SetCurrentOrganization(testOrgID, testOrgName, testNS); err != nil {
			t.Fatalf("SetCurrentOrganization() error = %v", err)
		}
		if got := GetCurrentOrgID(); got != testOrgID {
			t.Errorf("GetCurrentOrgID() = %v, want %v", got, testOrgID)
		}
		if got := GetCurrentOrgName(); got != testOrgName {
			t.Errorf("GetCurrentOrgName() = %v, want %v", got, testOrgName)
		}
		if got := GetCurrentNamespace(); got != testNS {
			t.Errorf("GetCurrentNamespace() = %v, want %v", got, testNS)
		}
	})

	// Data is written to the profile file, not context.json
	t.Run("profile file persistence", func(t *testing.T) {
		profileFile := filepath.Join(configDir, "profiles", "test.json")
		data, err := os.ReadFile(profileFile)
		if err != nil {
			t.Fatalf("Failed to read profile file: %v", err)
		}

		var ctx CLIContext
		if err := json.Unmarshal(data, &ctx); err != nil {
			t.Fatalf("Failed to unmarshal profile: %v", err)
		}

		if ctx.Token != testToken {
			t.Errorf("Persisted token = %v, want %v", ctx.Token, testToken)
		}
		if ctx.CurrentNamespace != "complete-org-namespace" {
			t.Errorf("Persisted namespace = %v, want complete-org-namespace", ctx.CurrentNamespace)
		}
		if ctx.CurrentOrgID != "org-456-uuid" {
			t.Errorf("Persisted org ID = %v, want org-456-uuid", ctx.CurrentOrgID)
		}
		if ctx.CurrentOrgName != "Complete Org" {
			t.Errorf("Persisted org name = %v, want Complete Org", ctx.CurrentOrgName)
		}
		if ctx.UserID != testUserID {
			t.Errorf("Persisted user ID = %v, want %v", ctx.UserID, testUserID)
		}
		if ctx.UserConfigKey != testConfigKey {
			t.Errorf("Persisted config key = %v, want %v", ctx.UserConfigKey, testConfigKey)
		}
	})

	t.Run("clear context", func(t *testing.T) {
		if err := SetToken(""); err != nil {
			t.Fatalf("SetToken() error = %v", err)
		}
		if err := SetCurrentNamespace(""); err != nil {
			t.Fatalf("SetCurrentNamespace() error = %v", err)
		}
		if got := GetToken(); got != "" {
			t.Errorf("GetToken() after clear = %v, want empty", got)
		}
		if got := GetCurrentNamespace(); got != "" {
			t.Errorf("GetCurrentNamespace() after clear = %v, want empty", got)
		}
	})

	t.Run("no profile returns empty", func(t *testing.T) {
		// Temporarily clear the active profile
		orig := configDir
		emptyDir := t.TempDir()
		configDir = emptyDir
		if err := os.MkdirAll(emptyDir, 0750); err != nil {
			t.Fatal(err)
		}
		// Write context.json with no active_profile
		if err := os.WriteFile(filepath.Join(emptyDir, "context.json"), []byte(`{}`), 0600); err != nil {
			t.Fatal(err)
		}
		defer func() { configDir = orig }()

		if got := GetToken(); got != "" {
			t.Errorf("GetToken() with no profile = %v, want empty", got)
		}
	})
}

func TestContextFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping file permissions test on Windows")
	}

	originalConfigDir := configDir
	defer func() { configDir = originalConfigDir }()

	tempDir := t.TempDir()
	configDir = tempDir

	// Set up a "test" profile so saveContext has somewhere to write
	profilesDir := filepath.Join(tempDir, "profiles")
	if err := os.MkdirAll(profilesDir, 0750); err != nil {
		t.Fatalf("Failed to create profiles dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(profilesDir, "test.json"), []byte("{}"), 0600); err != nil {
		t.Fatalf("Failed to create test profile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "context.json"), []byte(`{"active_profile":"test"}`), 0600); err != nil {
		t.Fatalf("Failed to write context.json: %v", err)
	}

	if err := SetToken("test-token"); err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}

	profileFile := filepath.Join(configDir, "profiles", "test.json")
	info, err := os.Stat(profileFile)
	if err != nil {
		t.Fatalf("Failed to stat profile file: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("Profile file permissions = %v, want %v", info.Mode().Perm(), 0600)
	}
}
