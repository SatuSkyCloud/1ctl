package context

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

type CLIContext struct {
	CurrentNamespace string `json:"organization"`
	Token            string `json:"token"`
	UserConfigKey    string `json:"user_config_key"`
	UserID           string `json:"user_id"`
}

var configDir string

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Could not get user home directory:", err)
	}
	configDir = filepath.Join(homeDir, ".satusky")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Fatal("Could not create config directory:", err)
	}
}

// GetToken returns the token from context.json
func GetToken() string {
	contextFile := filepath.Join(configDir, "context.json")
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
