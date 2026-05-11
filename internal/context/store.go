package context

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"1ctl/internal/utils"
)

// Store encapsulates the on-disk profile/credential state that the CLI
// reads and writes. It replaces the package-level globals (`configDir`,
// `profileOverride`, `cachedCtx`) that used to back the same getters.
//
// Two construction paths:
//
//	NewStore()                 — production: ~/.satusky/, ensured at boot.
//	NewTestStore(tempDir)      — tests: caller-supplied dir, no I/O at construction.
//
// All methods are safe for concurrent use within a single process. The
// in-memory CLIContext cache is guarded by an RWMutex; reads after a
// successful Save invalidate the cache automatically.
type Store struct {
	configDir       string
	profileOverride string

	cacheMu    sync.RWMutex
	cachedCtx  *CLIContext
	cacheValid bool
}

// NewStore constructs a production Store rooted at ~/.satusky/. The
// directory is created with 0750 perms if it doesn't exist. Returns an
// error if the user's home directory can't be resolved or the directory
// can't be created — both fatal-at-boot conditions.
func NewStore() (*Store, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}
	dir := filepath.Join(homeDir, ".satusky")
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("create config directory %s: %w", dir, err)
	}
	return &Store{configDir: dir}, nil
}

// NewTestStore returns a Store rooted at the given directory. Intended
// for unit and integration tests — pass a t.TempDir() result. No I/O
// happens at construction; the caller decides whether to create the
// directory.
func NewTestStore(configDir string) *Store {
	return &Store{configDir: configDir}
}

// ConfigDir returns the absolute path to the Store's root directory.
// Exposed for callers that need to construct paths under it (e.g. tests).
func (s *Store) ConfigDir() string { return s.configDir }

// SetProfileOverride applies a process-scoped profile override. Used by
// the --profile global flag. Not persisted. Resets the in-memory cache
// so subsequent reads pick up the new profile.
func (s *Store) SetProfileOverride(name string) {
	s.cacheMu.Lock()
	s.profileOverride = sanitizeProfileName(name)
	s.cachedCtx = nil
	s.cacheValid = false
	s.cacheMu.Unlock()
}

// invalidateCache clears the in-memory CLIContext cache. Called from
// Save and from anything that mutates the active profile path.
func (s *Store) invalidateCache() {
	s.cacheMu.Lock()
	s.cachedCtx = nil
	s.cacheValid = false
	s.cacheMu.Unlock()
}

// validatePath ensures path traversal can't escape the Store's root.
func (s *Store) validatePath(path string) error {
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return utils.NewError("invalid path: must not contain parent directory references", nil)
	}
	if filepath.IsAbs(cleanPath) && !strings.HasPrefix(cleanPath, s.configDir) {
		return utils.NewError("invalid path: must be within config directory", nil)
	}
	return nil
}

// contextFilePath returns the active profile's CLIContext JSON path,
// or a sentinel non-existent path when no profile is active.
func (s *Store) contextFilePath() string {
	if s.profileOverride != "" {
		return filepath.Join(s.configDir, "profiles", s.profileOverride+".json")
	}
	name := s.ActiveProfileName()
	if name == "" {
		return filepath.Join(s.configDir, "profiles", ".no-active-profile")
	}
	return filepath.Join(s.configDir, "profiles", sanitizeProfileName(name)+".json")
}

// load reads the active profile's CLIContext from disk, cached for the
// rest of the process lifetime. saveContext invalidates the cache after
// a successful write.
func (s *Store) load() *CLIContext {
	s.cacheMu.RLock()
	if s.cacheValid && s.cachedCtx != nil {
		ctx := *s.cachedCtx
		s.cacheMu.RUnlock()
		return &ctx
	}
	s.cacheMu.RUnlock()

	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if s.cacheValid && s.cachedCtx != nil {
		ctx := *s.cachedCtx
		return &ctx
	}

	contextFile := s.contextFilePath()
	if err := s.validatePath(contextFile); err != nil {
		s.cachedCtx = &CLIContext{}
		s.cacheValid = true
		return s.cachedCtx
	}

	loaded := &CLIContext{}
	data, err := os.ReadFile(contextFile) // #nosec G304 -- Path validated above
	if err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, loaded) //nolint:errcheck
	}
	s.cachedCtx = loaded
	s.cacheValid = true
	ctx := *s.cachedCtx
	return &ctx
}

// save writes profile data changes to the active profile file. Errors
// if no profile is currently active. Atomic with respect to other
// concurrent reads — cache is invalidated after the write commits.
func (s *Store) save(modifier func(*CLIContext)) error {
	if s.ActiveProfileName() == "" && s.profileOverride == "" {
		return utils.NewError("no profile is active. Create one with '1ctl profile create [--url <url>] <name>' then run '1ctl profile use <name>'", nil)
	}

	contextFile := s.contextFilePath()
	if err := os.MkdirAll(filepath.Dir(contextFile), 0750); err != nil {
		return err
	}

	var ctx CLIContext
	data, err := os.ReadFile(contextFile) // #nosec G304 -- Path resolved via s.contextFilePath()
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
	s.invalidateCache()
	return nil
}

// --- Profile-level state (rootContext, ~/.satusky/context.json) ---

// rootContext is the only thing stored in ~/.satusky/context.json — it
// names the active profile. All credentials live in the profile file.
type rootContext struct {
	ActiveProfile string `json:"active_profile"`
}

// ActiveProfileName returns the active profile name (or "" if none).
// Honours an in-process profileOverride first.
func (s *Store) ActiveProfileName() string {
	if s.profileOverride != "" {
		return s.profileOverride
	}
	data, err := os.ReadFile(filepath.Join(s.configDir, "context.json")) // #nosec G304
	if err != nil {
		return ""
	}
	var root rootContext
	if err := json.Unmarshal(data, &root); err != nil {
		return ""
	}
	return root.ActiveProfile
}

// SetActiveProfileName writes the active profile name to context.json.
// Invalidates the cache so subsequent reads see the new profile.
func (s *Store) SetActiveProfileName(name string) error {
	defer s.invalidateCache()
	root := rootContext{ActiveProfile: name}
	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.configDir, "context.json"), data, 0600)
}

// --- Per-field accessors (delegate to the cached CLIContext load) ---

// GetToken returns the API token from the active profile.
func (s *Store) GetToken() string { return s.load().Token }

// SetToken persists the API token to the active profile.
func (s *Store) SetToken(token string) error {
	return s.save(func(ctx *CLIContext) { ctx.Token = token })
}

// GetUserID returns the user ID from the active profile.
func (s *Store) GetUserID() string { return s.load().UserID }

// SetUserID persists the user ID to the active profile.
func (s *Store) SetUserID(userID string) error {
	return s.save(func(ctx *CLIContext) { ctx.UserID = userID })
}

// GetEmail returns the user email from the active profile.
func (s *Store) GetEmail() string { return s.load().Email }

// SetEmail persists the user email to the active profile.
func (s *Store) SetEmail(email string) error {
	return s.save(func(ctx *CLIContext) { ctx.Email = email })
}

// GetCurrentNamespace returns the current namespace from the active profile.
// For paths where an empty namespace would silently misroute, prefer
// GetCurrentNamespaceOrError.
func (s *Store) GetCurrentNamespace() string { return s.load().CurrentNamespace }

// GetCurrentNamespaceOrError returns the current namespace, or an
// actionable error if no namespace is set. Use at API call sites where
// forwarding an empty namespace to the backend would silently produce
// wrong results.
func (s *Store) GetCurrentNamespaceOrError() (string, error) {
	ns := s.load().CurrentNamespace
	if ns == "" {
		return "", utils.NewError("no organization is selected — run '1ctl auth login' or '1ctl org switch <name>'", nil)
	}
	return ns, nil
}

// SetCurrentNamespace persists the namespace to the active profile.
func (s *Store) SetCurrentNamespace(namespace string) error {
	return s.save(func(ctx *CLIContext) { ctx.CurrentNamespace = namespace })
}

// GetCurrentOrgID returns the current organization ID.
func (s *Store) GetCurrentOrgID() string { return s.load().CurrentOrgID }

// SetCurrentOrgID persists the organization ID.
func (s *Store) SetCurrentOrgID(orgID string) error {
	return s.save(func(ctx *CLIContext) { ctx.CurrentOrgID = orgID })
}

// GetCurrentOrgName returns the current organization display name.
func (s *Store) GetCurrentOrgName() string { return s.load().CurrentOrgName }

// SetCurrentOrgName persists the organization display name.
func (s *Store) SetCurrentOrgName(orgName string) error {
	return s.save(func(ctx *CLIContext) { ctx.CurrentOrgName = orgName })
}

// SetCurrentOrganization persists org ID, display name, and namespace
// in a single atomic write.
func (s *Store) SetCurrentOrganization(orgID, orgName, namespace string) error {
	return s.save(func(ctx *CLIContext) {
		ctx.CurrentOrgID = orgID
		ctx.CurrentOrgName = orgName
		ctx.CurrentNamespace = namespace
	})
}

// SaveLoginState writes every auth field in a single atomic write so a
// crash mid-login can't leave the on-disk state inconsistent.
func (s *Store) SaveLoginState(token, userID, email, orgID, orgName, namespace string) error {
	return s.save(func(ctx *CLIContext) {
		ctx.Token = token
		ctx.UserID = userID
		ctx.Email = email
		ctx.CurrentOrgID = orgID
		ctx.CurrentOrgName = orgName
		ctx.CurrentNamespace = namespace
	})
}

// ClearAuthState wipes every auth field in a single atomic write.
func (s *Store) ClearAuthState() error {
	return s.save(func(ctx *CLIContext) {
		ctx.Token = ""
		ctx.UserID = ""
		ctx.Email = ""
		ctx.CurrentOrgID = ""
		ctx.CurrentOrgName = ""
		ctx.CurrentNamespace = ""
	})
}

// GetAPIURL returns the API URL stored on the active profile (or "" if
// the profile inherits the platform default).
func (s *Store) GetAPIURL() string { return s.load().APIURL }

// SetAPIURL persists an API URL override to the active profile.
func (s *Store) SetAPIURL(apiURL string) error {
	return s.save(func(ctx *CLIContext) { ctx.APIURL = apiURL })
}

// --- Profile management (ListProfiles, CreateProfile, UseProfile, DeleteProfile) ---

// ProfileInfo holds displayable metadata for a profile.
type ProfileInfo struct {
	Name     string
	APIURL   string
	Email    string
	OrgName  string
	IsActive bool
}

// ListProfiles returns all profiles found under <configDir>/profiles/.
func (s *Store) ListProfiles() ([]ProfileInfo, error) {
	profilesDir := filepath.Join(s.configDir, "profiles")
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	activeName := s.ActiveProfileName()
	var profiles []ProfileInfo

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		profilePath := filepath.Join(profilesDir, entry.Name())

		data, err := os.ReadFile(profilePath) // #nosec G304
		if err != nil {
			continue
		}
		var ctx CLIContext
		if err := json.Unmarshal(data, &ctx); err != nil {
			continue
		}
		profiles = append(profiles, ProfileInfo{
			Name:     name,
			APIURL:   ctx.APIURL,
			Email:    ctx.Email,
			OrgName:  ctx.CurrentOrgName,
			IsActive: name == activeName,
		})
	}
	return profiles, nil
}

// CreateProfile creates a new profile with the given name and optional API URL.
// Does not switch to the new profile automatically.
func (s *Store) CreateProfile(name, apiURL string) error {
	name = sanitizeProfileName(name)
	if name == "" {
		return utils.NewError("invalid profile name: only letters, numbers, dashes, and underscores are allowed", nil)
	}
	if apiURL != "" && !utils.IsLocalhostURL(apiURL) && !strings.HasPrefix(apiURL, "https://") {
		return utils.NewError("API URL must use HTTPS for non-localhost endpoints. Use http://localhost for local development", nil)
	}
	profilesDir := filepath.Join(s.configDir, "profiles")
	if err := os.MkdirAll(profilesDir, 0750); err != nil {
		return fmt.Errorf("failed to create profiles directory: %w", err)
	}
	profilePath := filepath.Join(profilesDir, name+".json")
	if _, err := os.Stat(profilePath); err == nil {
		return utils.NewError(fmt.Sprintf("profile '%s' already exists", name), nil)
	}
	ctx := CLIContext{APIURL: apiURL}
	data, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(profilePath, data, 0600)
}

// UseProfile switches the active profile. Errors if the profile doesn't exist.
func (s *Store) UseProfile(name string) error {
	name = sanitizeProfileName(name)
	profilePath := filepath.Join(s.configDir, "profiles", name+".json")
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		return utils.NewError(
			fmt.Sprintf("profile '%s' not found. Create it first with '1ctl profile create [--url <url>] %s'", name, name),
			nil,
		)
	}
	return s.SetActiveProfileName(name)
}

// DeleteProfile removes the named profile file. Refuses to delete the
// active profile.
func (s *Store) DeleteProfile(name string) error {
	name = sanitizeProfileName(name)
	profilePath := filepath.Join(s.configDir, "profiles", name+".json")
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		return utils.NewError(fmt.Sprintf("profile '%s' not found", name), nil)
	}
	if s.ActiveProfileName() == name {
		return utils.NewError(
			fmt.Sprintf("cannot delete the active profile '%s'. Switch to another profile first with '1ctl profile use <name>'", name),
			nil,
		)
	}
	return os.Remove(profilePath)
}

// CheckTokenExpiry parses the JWT exp claim from the stored token. Returns
// a clear error if expired; returns nil for non-JWT tokens (the backend
// is the authority on those).
func (s *Store) CheckTokenExpiry() error {
	token := s.GetToken()
	if token == "" {
		return utils.NewError("not authenticated. Please run '1ctl auth login' to authenticate", nil)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	var claims struct {
		Exp float64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil || claims.Exp == 0 {
		return nil
	}
	expiry := time.Unix(int64(claims.Exp), 0)
	if time.Now().After(expiry) {
		return utils.NewError(fmt.Sprintf("session expired at %s. Please run '1ctl auth login' to re-authenticate", expiry.Format("Jan 2, 2006 15:04 MST")), nil)
	}
	return nil
}

// --- Package-level default + delegation ---

// defaultStore is the singleton Store used by the package-level helper
// functions below. Initialised by init() at package load. Tests that
// need isolation should construct their own Store via NewTestStore and
// call methods on it directly — they should not touch defaultStore.
//
// Production code only mutates defaultStore in two places: the init()
// call below, and SetDefault (used by main.go to swap in a freshly-
// constructed Store at startup if needed). Mutex-guarded.
var (
	defaultMu sync.RWMutex
	defaultS  *Store
)

func init() {
	s, err := NewStore()
	if err != nil {
		log.Fatal(err)
	}
	defaultS = s
}

// SetDefault replaces the package-level default Store. Used by main.go
// if it wants to inject an alternate Store at startup; rarely needed
// outside that one call site.
func SetDefault(s *Store) {
	defaultMu.Lock()
	defaultS = s
	defaultMu.Unlock()
}

// Default returns the package-level default Store. Always non-nil after
// package init (init log.Fatals if it can't construct one).
func Default() *Store {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultS
}

// sanitizeProfileName allows only alphanumeric, dash, and underscore.
// Shared by Store methods and profile-package helpers.
func sanitizeProfileName(name string) string {
	return regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(name, "")
}
