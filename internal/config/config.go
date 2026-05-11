package config

import (
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"errors"
	"os"

	"github.com/joho/godotenv"
)

// defaultAPIURL is the baked-in production endpoint. It can be overridden at
// build time via -ldflags (for forks pointing at a different production
// deployment) and at runtime via the --api-url flag, the SATUSKY_API_URL
// environment variable, or a named profile's api_url field.
var (
	defaultAPIURL = "https://api.satusky.com/v1/cli"
)

// DefaultAPIURL returns the compiled-in default API URL. It ignores both the
// active profile and the SATUSKY_API_URL env var — use GetConfig() for the
// effective runtime URL.
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
