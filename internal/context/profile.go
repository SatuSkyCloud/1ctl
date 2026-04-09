package context

import (
	"1ctl/internal/utils"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// rootContext is the only thing stored in ~/.satusky/context.json.
// It points at the active profile. All credentials and settings live in the profile file.
type rootContext struct {
	ActiveProfile string `json:"active_profile"`
}

// profileOverride is set by the --profile flag for the current process only (not persisted).
var profileOverride string

// SetProfileOverride temporarily overrides the active profile for the current process.
// Used by the --profile global flag. Not persisted to disk.
func SetProfileOverride(name string) {
	profileOverride = sanitizeProfileName(name)
}

// sanitizeProfileName allows only alphanumeric, dash, and underscore characters.
func sanitizeProfileName(name string) string {
	return regexp.MustCompile(`[^a-zA-Z0-9_-]`).ReplaceAllString(name, "")
}

// getContextFilePath returns the path to the active profile file.
// Returns a sentinel non-existent path when no profile is set so that all
// getters return "" gracefully (same behaviour as a missing file).
func getContextFilePath() string {
	if profileOverride != "" {
		return filepath.Join(configDir, "profiles", profileOverride+".json")
	}

	name := GetActiveProfileName()
	if name == "" {
		// No profile set — return a path that won't exist so callers return "" naturally.
		return filepath.Join(configDir, "profiles", ".no-active-profile")
	}

	return filepath.Join(configDir, "profiles", sanitizeProfileName(name)+".json")
}

// GetActiveProfileName returns the name of the currently active profile, or "" if none is set.
func GetActiveProfileName() string {
	if profileOverride != "" {
		return profileOverride
	}

	data, err := os.ReadFile(filepath.Join(configDir, "context.json")) // #nosec G304
	if err != nil {
		return ""
	}

	var root rootContext
	_ = json.Unmarshal(data, &root)
	return root.ActiveProfile
}

// SetActiveProfileName writes only the active profile name to context.json.
// context.json contains nothing else — all credentials are in the profile file.
func SetActiveProfileName(name string) error {
	root := rootContext{ActiveProfile: name}
	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(configDir, "context.json"), data, 0600)
}

// ProfileInfo holds displayable metadata for a profile.
type ProfileInfo struct {
	Name     string
	APIURL   string
	Email    string
	OrgName  string
	IsActive bool
}

// ListProfiles returns all profiles found in ~/.satusky/profiles/.
func ListProfiles() ([]ProfileInfo, error) {
	profilesDir := filepath.Join(configDir, "profiles")
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	activeName := GetActiveProfileName()
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
		_ = json.Unmarshal(data, &ctx)

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
func CreateProfile(name, apiURL string) error {
	name = sanitizeProfileName(name)
	if name == "" {
		return utils.NewError("invalid profile name: only letters, numbers, dashes, and underscores are allowed", nil)
	}

	profilesDir := filepath.Join(configDir, "profiles")
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

// UseProfile switches the active profile to the named profile.
// Returns an error if the profile does not exist.
func UseProfile(name string) error {
	name = sanitizeProfileName(name)
	profilePath := filepath.Join(configDir, "profiles", name+".json")

	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		return utils.NewError(
			fmt.Sprintf("profile '%s' not found. Create it first with '1ctl profile create [--url <url>] %s'", name, name),
			nil,
		)
	}

	return SetActiveProfileName(name)
}

// DeleteProfile removes the named profile file.
// Returns an error if the profile is currently active.
func DeleteProfile(name string) error {
	name = sanitizeProfileName(name)
	profilePath := filepath.Join(configDir, "profiles", name+".json")

	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		return utils.NewError(fmt.Sprintf("profile '%s' not found", name), nil)
	}

	if GetActiveProfileName() == name {
		return utils.NewError(
			fmt.Sprintf("cannot delete the active profile '%s'. Switch to another profile first with '1ctl profile use <name>'", name),
			nil,
		)
	}

	return os.Remove(profilePath)
}
