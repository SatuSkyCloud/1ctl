package config

import (
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"errors"
	"os"

	"github.com/joho/godotenv"
)

// defaultAPIURL is intentionally overridable at build time via -ldflags:
//
//	go build -ldflags "-X '1ctl/internal/config.defaultAPIURL=https://dev-core-api.satusky.com/v1/cli'"
//
// Used by the 1ctl-dev build and for local-dev builds against localhost.
// Active profile and SATUSKY_API_URL env var still override at runtime.
var (
	defaultAPIURL = "https://api.satusky.com/v1/cli"
)

// DefaultAPIURL returns the compiled-in default API URL (prod URL for `1ctl`,
// dev URL for `1ctl-dev`). It ignores both the active profile and the
// SATUSKY_API_URL env var — use GetConfig() for the effective runtime URL.
func DefaultAPIURL() string {
	return defaultAPIURL
}

type Config struct {
	ApiURL string
}

func init() {
	// Load environment variables from .env file
	if os.Getenv("APP_ENV") == "development" {
		err := godotenv.Load()
		if err != nil {
			utils.PrintError("not found .env file: %v", err)
		}
	}
}

func GetConfig() *Config {
	config := &Config{
		ApiURL: defaultAPIURL,
	}

	// Active profile API URL overrides the compiled-in default
	if apiURL := context.GetAPIURL(); apiURL != "" {
		config.ApiURL = apiURL
	}

	// Env var overrides profile (useful for one-off overrides without switching profiles)
	if apiURL := os.Getenv("SATUSKY_API_URL"); apiURL != "" {
		config.ApiURL = apiURL
	}

	return config
}

func ValidateEnvironment() error {
	token := context.GetToken()
	if token == "" {
		utils.PrintError("not authenticated. Please run '1ctl auth login' to authenticate")
		return errors.New("not authenticated")
	}
	return nil
}
