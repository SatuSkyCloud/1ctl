package context

import (
	"1ctl/internal/utils"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type CLIContext struct {
	CurrentNamespace string `json:"organization"`
	Token            string `json:"token"`
	UserConfigKey    string `json:"user_config_key"`
	UserID           string `json:"user_id"`
}

var configDir string

// validatePath checks if the path is safe to use
func validatePath(path string) error {
	// Clean the path
	cleanPath := filepath.Clean(path)

	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") {
		return utils.NewError("invalid path: must not contain parent directory references", nil)
	}

	// Check if path is absolute
	if filepath.IsAbs(cleanPath) {
		// Verify it's within the user's home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return utils.NewError(fmt.Sprintf("failed to get home directory: %s", err.Error()), nil)
		}
		if !strings.HasPrefix(cleanPath, homeDir) {
			return utils.NewError("invalid path: must be within user's home directory", nil)
		}
	}

	return nil
}

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Could not get home directory:", err)
	}

	configDir = filepath.Join(homeDir, ".satusky")
	if err := os.MkdirAll(configDir, 0750); err != nil {
		log.Fatal("Could not create config directory:", err)
	}
}

// GetToken returns the token from context.json
func GetToken() string {
	contextFile := filepath.Join(configDir, "context.json")
	if err := validatePath(contextFile); err != nil {
		return ""
	}

	data, err := os.ReadFile(contextFile)
	if err != nil {
		return ""
	}

	var ctx CLIContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return ""
	}

	return ctx.Token
}

// SetToken saves the token to context.json
func SetToken(token string) error {
	return saveContext(func(ctx *CLIContext) {
		ctx.Token = token
	})
}

// GetUserID returns the user ID from context.json
func GetUserID() string {
	contextFile := filepath.Join(configDir, "context.json")
	data, err := os.ReadFile(contextFile)
	if err != nil {
		return ""
	}

	var ctx CLIContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return ""
	}

	return ctx.UserID
}

// SetUserID saves the user ID to context.json
func SetUserID(userID string) error {
	return saveContext(func(ctx *CLIContext) {
		ctx.UserID = userID
	})
}

// GetCurrentNamespace returns the current namespace from context.json
func GetCurrentNamespace() string {
	contextFile := filepath.Join(configDir, "context.json")
	data, err := os.ReadFile(contextFile)
	if err != nil {
		return ""
	}

	var ctx CLIContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return ""
	}

	return ctx.CurrentNamespace
}

// SetCurrentNamespace saves the namespace to context.json
func SetCurrentNamespace(namespace string) error {
	return saveContext(func(ctx *CLIContext) {
		ctx.CurrentNamespace = namespace
	})
}

// GetUserConfigKey returns the user config key from context.json
func GetUserConfigKey() string {
	contextFile := filepath.Join(configDir, "context.json")
	data, err := os.ReadFile(contextFile)
	if err != nil {
		return ""
	}

	var ctx CLIContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return ""
	}

	return ctx.UserConfigKey
}

// SetUserConfigKey saves the user config key to context.json
func SetUserConfigKey(userConfigKey string) error {
	return saveContext(func(ctx *CLIContext) {
		ctx.UserConfigKey = userConfigKey
	})
}

// saveContext is a helper function to save context changes
func saveContext(modifier func(*CLIContext)) error {
	contextFile := filepath.Join(configDir, "context.json")

	var ctx CLIContext
	data, err := os.ReadFile(contextFile)
	if err == nil {
		if err := json.Unmarshal(data, &ctx); err != nil {
			return err
		}
	}

	modifier(&ctx)

	data, err = json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(contextFile, data, 0600)
}
