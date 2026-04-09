package config

import (
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"errors"
	"os"

	"github.com/joho/godotenv"
)

// These vars are intentionally overridable at build time via -ldflags
// e.g. go build -ldflags "-X '1ctl/internal/config.defaultAPIURL=http://localhost:8080/v1/cli'"
var (
	defaultAPIURL = "https://api.satusky.com/v1/cli"
)

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
