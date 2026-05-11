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

// UseProfile switches the default Store's active profile.
func UseProfile(name string) error { return Default().UseProfile(name) }

// DeleteProfile removes the named profile from the default Store.
func DeleteProfile(name string) error { return Default().DeleteProfile(name) }
