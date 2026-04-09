package context

import (
	"1ctl/internal/utils"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type CLIContext struct {
	APIURL           string `json:"api_url,omitempty"`
	CurrentNamespace string `json:"organization"`
	CurrentOrgID     string `json:"current_org_id,omitempty"`
	CurrentOrgName   string `json:"current_org_name,omitempty"`
	Email            string `json:"email,omitempty"`
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
		// Verify it's within the configured config directory
		// This allows tests to override configDir while still being secure
		if !strings.HasPrefix(cleanPath, configDir) {
			return utils.NewError("invalid path: must be within config directory", nil)
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

// SetConfigDir overrides the config directory (for testing only)
func SetConfigDir(dir string) {
	configDir = dir
}

// GetToken returns the token from the active profile (or legacy context.json).
func GetToken() string {
	contextFile := getContextFilePath()
	if err := validatePath(contextFile); err != nil {
		return ""
	}

	data, err := os.ReadFile(contextFile) // #nosec G304 -- Path validated above
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
	data, err := os.ReadFile(getContextFilePath()) // #nosec G304 -- Path resolved via getContextFilePath
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

// GetEmail returns the user email from context.json
func GetEmail() string {
	data, err := os.ReadFile(getContextFilePath()) // #nosec G304 -- Path resolved via getContextFilePath
	if err != nil {
		return ""
	}

	var ctx CLIContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return ""
	}

	return ctx.Email
}

// SetEmail saves the user email to context.json
func SetEmail(email string) error {
	return saveContext(func(ctx *CLIContext) {
		ctx.Email = email
	})
}

// GetCurrentNamespace returns the current namespace from context.json
func GetCurrentNamespace() string {
	data, err := os.ReadFile(getContextFilePath()) // #nosec G304 -- Path resolved via getContextFilePath
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
	data, err := os.ReadFile(getContextFilePath()) // #nosec G304 -- Path resolved via getContextFilePath
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

// GetCurrentOrgID returns the current organization ID from context.json
func GetCurrentOrgID() string {
	data, err := os.ReadFile(getContextFilePath()) // #nosec G304 -- Path resolved via getContextFilePath
	if err != nil {
		return ""
	}

	var ctx CLIContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return ""
	}

	return ctx.CurrentOrgID
}

// SetCurrentOrgID saves the organization ID to context.json
func SetCurrentOrgID(orgID string) error {
	return saveContext(func(ctx *CLIContext) {
		ctx.CurrentOrgID = orgID
	})
}

// GetCurrentOrgName returns the current organization name from context.json
func GetCurrentOrgName() string {
	data, err := os.ReadFile(getContextFilePath()) // #nosec G304 -- Path resolved via getContextFilePath
	if err != nil {
		return ""
	}

	var ctx CLIContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return ""
	}

	return ctx.CurrentOrgName
}

// SetCurrentOrgName saves the organization name to context.json
func SetCurrentOrgName(orgName string) error {
	return saveContext(func(ctx *CLIContext) {
		ctx.CurrentOrgName = orgName
	})
}

// SetCurrentOrganization saves both org ID, name, and namespace to context.json
func SetCurrentOrganization(orgID, orgName, namespace string) error {
	return saveContext(func(ctx *CLIContext) {
		ctx.CurrentOrgID = orgID
		ctx.CurrentOrgName = orgName
		ctx.CurrentNamespace = namespace
	})
}

// saveContext writes profile data changes to the active profile file.
// Returns an error if no profile is currently active.
func saveContext(modifier func(*CLIContext)) error {
	if GetActiveProfileName() == "" && profileOverride == "" {
		return utils.NewError("no profile is active. Create one with '1ctl profile create [--url <url>] <name>' then run '1ctl profile use <name>'", nil)
	}

	contextFile := getContextFilePath()

	// Ensure ~/.satusky/profiles/ exists
	if err := os.MkdirAll(filepath.Dir(contextFile), 0750); err != nil {
		return err
	}

	var ctx CLIContext
	data, err := os.ReadFile(contextFile) // #nosec G304 -- Path resolved via getContextFilePath
	if err == nil && len(data) > 0 {
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

// GetAPIURL returns the API URL stored in the active profile, or "" if none is set.
func GetAPIURL() string {
	data, err := os.ReadFile(getContextFilePath()) // #nosec G304
	if err != nil {
		return ""
	}

	var ctx CLIContext
	if err := json.Unmarshal(data, &ctx); err != nil {
		return ""
	}

	return ctx.APIURL
}

// SetAPIURL saves an API URL override to the active profile.
func SetAPIURL(apiURL string) error {
	return saveContext(func(ctx *CLIContext) {
		ctx.APIURL = apiURL
	})
}

