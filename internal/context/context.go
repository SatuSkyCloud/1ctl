package context

import (
	"1ctl/internal/utils"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type CLIContext struct {
	APIURL string `json:"api_url,omitempty"`
	// CurrentNamespace's JSON tag is "organization" for legacy compatibility:
	// older context.json files on disk use that key. Renaming would silently
	// drop the value on first read. Do not change.
	CurrentNamespace string `json:"organization"`
	CurrentOrgID     string `json:"current_org_id,omitempty"`
	CurrentOrgName   string `json:"current_org_name,omitempty"`
	Email            string `json:"email,omitempty"`
	Token            string `json:"token"`
	UserID           string `json:"user_id"`
}

var configDir string

// In-memory cache of the loaded context. Populated lazily on first access via
// loadContext(); invalidated by saveContext() and SetConfigDir(). Single-process
// CLI assumption: no concurrent writers from other processes during a command.
var (
	cachedCtx  *CLIContext
	cacheMu    sync.RWMutex
	cacheValid bool
)

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

	// Single config dir for all environments. Per-environment isolation
	// (prod, dev, local) is handled by named profiles, not by binary variant.
	// Internal devs configure a dev profile via internal onboarding docs.
	configDir = filepath.Join(homeDir, ".satusky")
	if err := os.MkdirAll(configDir, 0750); err != nil {
		log.Fatal("Could not create config directory:", err)
	}
}

// SetConfigDir overrides the config directory (for testing only).
// Resets the in-memory cache so subsequent reads pick up the new path.
func SetConfigDir(dir string) {
	cacheMu.Lock()
	configDir = dir
	cachedCtx = nil
	cacheValid = false
	cacheMu.Unlock()
}

// loadContext returns the cached CLIContext, reading from disk on first call
// per active profile. saveContext invalidates the cache after a successful write.
func loadContext() *CLIContext {
	cacheMu.RLock()
	if cacheValid && cachedCtx != nil {
		ctx := *cachedCtx
		cacheMu.RUnlock()
		return &ctx
	}
	cacheMu.RUnlock()

	cacheMu.Lock()
	defer cacheMu.Unlock()
	// Re-check after acquiring write lock.
	if cacheValid && cachedCtx != nil {
		ctx := *cachedCtx
		return &ctx
	}

	contextFile := getContextFilePath()
	if err := validatePath(contextFile); err != nil {
		cachedCtx = &CLIContext{}
		cacheValid = true
		return cachedCtx
	}

	loaded := &CLIContext{}
	data, err := os.ReadFile(contextFile) // #nosec G304 -- Path validated above
	if err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, loaded) //nolint:errcheck
	}
	cachedCtx = loaded
	cacheValid = true
	ctx := *cachedCtx
	return &ctx
}

// invalidateCache clears the in-memory context cache. Called from saveContext
// after a successful write so the next read returns fresh data.
func invalidateCache() {
	cacheMu.Lock()
	cachedCtx = nil
	cacheValid = false
	cacheMu.Unlock()
}

// GetToken returns the token from the active profile (or legacy context.json).
func GetToken() string {
	return loadContext().Token
}

// SetToken saves the token to context.json
func SetToken(token string) error {
	return saveContext(func(ctx *CLIContext) {
		ctx.Token = token
	})
}

// GetUserID returns the user ID from context.json
func GetUserID() string {
	return loadContext().UserID
}

// SetUserID saves the user ID to context.json
func SetUserID(userID string) error {
	return saveContext(func(ctx *CLIContext) {
		ctx.UserID = userID
	})
}

// GetEmail returns the user email from context.json
func GetEmail() string {
	return loadContext().Email
}

// SetEmail saves the user email to context.json
func SetEmail(email string) error {
	return saveContext(func(ctx *CLIContext) {
		ctx.Email = email
	})
}

// GetCurrentNamespace returns the current namespace from context.json.
// For paths where an empty namespace would silently misroute or hide the
// failure, prefer GetCurrentNamespaceOrError.
func GetCurrentNamespace() string {
	return loadContext().CurrentNamespace
}

// GetCurrentNamespaceOrError returns the current namespace, or an actionable
// error if no namespace is set. Use this at API call sites where forwarding
// an empty namespace to the backend would silently produce wrong results.
func GetCurrentNamespaceOrError() (string, error) {
	ns := loadContext().CurrentNamespace
	if ns == "" {
		return "", utils.NewError("no organization is selected — run '1ctl auth login' or '1ctl org switch <name>'", nil)
	}
	return ns, nil
}

// SetCurrentNamespace saves the namespace to context.json
func SetCurrentNamespace(namespace string) error {
	return saveContext(func(ctx *CLIContext) {
		ctx.CurrentNamespace = namespace
	})
}

// GetCurrentOrgID returns the current organization ID from context.json
func GetCurrentOrgID() string {
	return loadContext().CurrentOrgID
}

// SetCurrentOrgID saves the organization ID to context.json
func SetCurrentOrgID(orgID string) error {
	return saveContext(func(ctx *CLIContext) {
		ctx.CurrentOrgID = orgID
	})
}

// GetCurrentOrgName returns the current organization name from context.json
func GetCurrentOrgName() string {
	return loadContext().CurrentOrgName
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

// SaveLoginState writes all auth fields in a single atomic write to prevent
// corrupted state from a crash between separate writes.
func SaveLoginState(token, userID, email, orgID, orgName, namespace string) error {
	return saveContext(func(ctx *CLIContext) {
		ctx.Token = token
		ctx.UserID = userID
		ctx.Email = email
		ctx.CurrentOrgID = orgID
		ctx.CurrentOrgName = orgName
		ctx.CurrentNamespace = namespace
	})
}

// ClearAuthState removes all auth fields in a single atomic write.
func ClearAuthState() error {
	return saveContext(func(ctx *CLIContext) {
		ctx.Token = ""
		ctx.UserID = ""
		ctx.Email = ""
		ctx.CurrentOrgID = ""
		ctx.CurrentOrgName = ""
		ctx.CurrentNamespace = ""
	})
}

// CheckTokenExpiry parses the JWT exp claim from the stored token.
// Returns an error with a clear message if the token is expired or malformed.
func CheckTokenExpiry() error {
	token := GetToken()
	if token == "" {
		return utils.NewError("not authenticated. Please run '1ctl auth login' to authenticate", nil)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		// Not a JWT — can't check expiry, let the backend decide
		return nil
	}

	// Decode the payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Can't decode — let the backend decide
		return nil
	}

	var claims struct {
		Exp float64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		// No exp claim — let the backend decide
		return nil
	}

	expiry := time.Unix(int64(claims.Exp), 0)
	if time.Now().After(expiry) {
		return utils.NewError(fmt.Sprintf("session expired at %s. Please run '1ctl auth login' to re-authenticate", expiry.Format("Jan 2, 2006 15:04 MST")), nil)
	}

	return nil
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

	if err := os.WriteFile(contextFile, data, 0600); err != nil {
		return err
	}
	// Cache will be repopulated on the next read with the new state.
	invalidateCache()
	return nil
}

// GetAPIURL returns the API URL stored in the active profile, or "" if none is set.
func GetAPIURL() string {
	return loadContext().APIURL
}

// SetAPIURL saves an API URL override to the active profile.
func SetAPIURL(apiURL string) error {
	return saveContext(func(ctx *CLIContext) {
		ctx.APIURL = apiURL
	})
}
