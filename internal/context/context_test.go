package context

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func setupTestContext(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create .satusky directory
	configDir := filepath.Join(dir, ".satusky")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Create empty context file
	contextFile := filepath.Join(configDir, "context.json")
	if err := os.WriteFile(contextFile, []byte("{}"), 0600); err != nil {
		t.Fatalf("Failed to create context file: %v", err)
	}

	// Set HOME environment variable
	t.Setenv("HOME", dir)
	return dir
}

func TestContextOperations(t *testing.T) {
	// Create test home directory
	homeDir := setupTestContext(t)
	configDir = filepath.Join(homeDir, ".satusky")

	// Test data
	testToken := "test-token-123"
	testNamespace := "test-org"
	testUserID := "user-123"
	testConfigKey := "config-key-123"

	// Test SetToken and GetToken
	t.Run("token operations", func(t *testing.T) {
		if err := SetToken(testToken); err != nil {
			t.Fatalf("SetToken() error = %v", err)
		}

		if got := GetToken(); got != testToken {
			t.Errorf("GetToken() = %v, want %v", got, testToken)
		}
	})

	// Test SetCurrentNamespace and GetCurrentNamespace
	t.Run("namespace operations", func(t *testing.T) {
		if err := SetCurrentNamespace(testNamespace); err != nil {
			t.Fatalf("SetCurrentNamespace() error = %v", err)
		}

		if got := GetCurrentNamespace(); got != testNamespace {
			t.Errorf("GetCurrentNamespace() = %v, want %v", got, testNamespace)
		}
	})

	// Test SetUserID and GetUserID
	t.Run("user ID operations", func(t *testing.T) {
		if err := SetUserID(testUserID); err != nil {
			t.Fatalf("SetUserID() error = %v", err)
		}

		if got := GetUserID(); got != testUserID {
			t.Errorf("GetUserID() = %v, want %v", got, testUserID)
		}
	})

	// Test SetUserConfigKey and GetUserConfigKey
	t.Run("config key operations", func(t *testing.T) {
		if err := SetUserConfigKey(testConfigKey); err != nil {
			t.Fatalf("SetUserConfigKey() error = %v", err)
		}

		if got := GetUserConfigKey(); got != testConfigKey {
			t.Errorf("GetUserConfigKey() = %v, want %v", got, testConfigKey)
		}
	})

	// Test file persistence
	t.Run("context file persistence", func(t *testing.T) {
		contextFile := filepath.Join(configDir, "context.json")
		data, err := os.ReadFile(contextFile)
		if err != nil {
			t.Fatalf("Failed to read context file: %v", err)
		}

		var ctx CLIContext
		if err := json.Unmarshal(data, &ctx); err != nil {
			t.Fatalf("Failed to unmarshal context: %v", err)
		}

		if ctx.Token != testToken {
			t.Errorf("Persisted token = %v, want %v", ctx.Token, testToken)
		}
		if ctx.CurrentNamespace != testNamespace {
			t.Errorf("Persisted namespace = %v, want %v", ctx.CurrentNamespace, testNamespace)
		}
		if ctx.UserID != testUserID {
			t.Errorf("Persisted user ID = %v, want %v", ctx.UserID, testUserID)
		}
		if ctx.UserConfigKey != testConfigKey {
			t.Errorf("Persisted config key = %v, want %v", ctx.UserConfigKey, testConfigKey)
		}
	})

	// Test clearing context
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
}

func TestContextFilePermissions(t *testing.T) {
	// Save original configDir and restore after test
	originalConfigDir := configDir
	defer func() { configDir = originalConfigDir }()

	// Create temporary directory for test
	tempDir := t.TempDir()
	configDir = tempDir

	// Set some data to create the file
	if err := SetToken("test-token"); err != nil {
		t.Fatalf("SetToken() error = %v", err)
	}

	// Check file permissions
	contextFile := filepath.Join(configDir, "context.json")
	info, err := os.Stat(contextFile)
	if err != nil {
		t.Fatalf("Failed to stat context file: %v", err)
	}

	// Check if file permissions are 0600 (user read/write only)
	if info.Mode().Perm() != 0600 {
		t.Errorf("Context file permissions = %v, want %v", info.Mode().Perm(), 0600)
	}
}
