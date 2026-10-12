// Profile-related package-level shims. Real logic lives on *Store
// (store.go). These exist so existing call sites keep working unchanged.
package context

// SetProfileOverride applies a process-scoped profile override on the
// default Store. Used by the --profile global flag.
func SetProfileOverride(name string) { Default().SetProfileOverride(name) }

// GetActiveProfileName returns the active profile name (or "").
func GetActiveProfileName() string { return Default().ActiveProfileName() }

// SetActiveProfileName writes the active profile name to context.json.
func SetActiveProfileName(name string) error { return Default().SetActiveProfileName(name) }

// ListProfiles returns all profiles in the default Store.
func ListProfiles() ([]ProfileInfo, error) { return Default().ListProfiles() }

// CreateProfile creates a new profile in the default Store.
func CreateProfile(name, apiURL string) error { return Default().CreateProfile(name, apiURL) }

// EnsureProfile creates the profile on the default Store if it does not exist,
// and reports whether a new profile was written. See Store.EnsureProfile: an
// existing profile pointing at the same API URL is success, not an error, so a
// bootstrap can run more than once in the same job without `|| true`.
func EnsureProfile(name, apiURL string) (bool, error) {
	return Default().EnsureProfile(name, apiURL)
}

// UseProfile switches the default Store's active profile.
func UseProfile(name string) error { return Default().UseProfile(name) }

// DeleteProfile removes the named profile from the default Store.
func DeleteProfile(name string) error { return Default().DeleteProfile(name) }
