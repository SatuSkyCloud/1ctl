// Package context owns the on-disk profile/credential state for the CLI.
//
// All real logic lives on the *Store type (store.go). The functions in
// this file are package-level shims that delegate to a singleton Store
// initialised at package load — they exist so the ~80 existing call
// sites across the codebase keep working unchanged. New code should use
// context.Default() to get a *Store and call methods on it directly;
// tests should construct their own *Store via context.NewTestStore.
//
// The CLIContext type below is the on-disk schema for a single profile's
// credentials. It's the data the *Store reads and writes.
package context

// CLIContext is the JSON-serialised state of a single profile under
// <configDir>/profiles/<name>.json.
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

// --- Package-level shims (delegate to Default()) ---
//
// These exist purely so the existing call-site surface keeps working
// unchanged. Adding new shims here as the codebase grows is fine, but
// the canonical API surface is *Store methods.

// GetToken returns the token from the active profile.
func GetToken() string { return Default().GetToken() }

// SetToken persists the token to the active profile.
func SetToken(token string) error { return Default().SetToken(token) }

// GetUserID returns the user ID from the active profile.
func GetUserID() string { return Default().GetUserID() }

// SetUserID persists the user ID to the active profile.
func SetUserID(userID string) error { return Default().SetUserID(userID) }

// GetEmail returns the user email from the active profile.
func GetEmail() string { return Default().GetEmail() }

// SetEmail persists the user email to the active profile.
func SetEmail(email string) error { return Default().SetEmail(email) }

// GetCurrentNamespace returns the active profile's namespace.
func GetCurrentNamespace() string { return Default().GetCurrentNamespace() }

// GetCurrentNamespaceOrError returns the active namespace or an
// actionable error if no organization is selected.
func GetCurrentNamespaceOrError() (string, error) {
	return Default().GetCurrentNamespaceOrError()
}

// SetCurrentNamespace persists the namespace to the active profile.
func SetCurrentNamespace(namespace string) error {
	return Default().SetCurrentNamespace(namespace)
}

// GetCurrentOrgID returns the active profile's organization ID.
func GetCurrentOrgID() string { return Default().GetCurrentOrgID() }

// SetCurrentOrgID persists the organization ID.
func SetCurrentOrgID(orgID string) error { return Default().SetCurrentOrgID(orgID) }

// GetCurrentOrgName returns the active profile's organization display name.
func GetCurrentOrgName() string { return Default().GetCurrentOrgName() }

// SetCurrentOrgName persists the organization display name.
func SetCurrentOrgName(orgName string) error { return Default().SetCurrentOrgName(orgName) }

// SetCurrentOrganization persists org ID, name, and namespace atomically.
func SetCurrentOrganization(orgID, orgName, namespace string) error {
	return Default().SetCurrentOrganization(orgID, orgName, namespace)
}

// SaveLoginState atomically writes every auth field.
func SaveLoginState(token, userID, email, orgID, orgName, namespace string) error {
	return Default().SaveLoginState(token, userID, email, orgID, orgName, namespace)
}

// ClearAuthState atomically wipes every auth field.
func ClearAuthState() error { return Default().ClearAuthState() }

// GetAPIURL returns the active profile's API URL (or "").
func GetAPIURL() string { return Default().GetAPIURL() }

// SetAPIURL persists an API URL override to the active profile.
func SetAPIURL(apiURL string) error { return Default().SetAPIURL(apiURL) }

// CheckTokenExpiry parses the JWT exp claim from the stored token.
func CheckTokenExpiry() error { return Default().CheckTokenExpiry() }
