package config

import (
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"errors"
	"os"

	"github.com/joho/godotenv"
)

// These defaults are intentionally vars (not consts) so the 1ctl-dev build
// can override them at link time via -ldflags "-X ...defaultAPIURL=...".
// SATUSKY_API_URL / SATUSKY_DOCKER_API_URL env vars still override at runtime.
var (
	defaultAPIURL          = "https://api.satusky.com/v1/cli"
	defaultDockerUploadURL = "http://docker-upload.api.satusky.com"
)

// DefaultAPIURL returns the compiled-in default API URL (prod for `1ctl`,
// dev for `1ctl-dev`). Ignores both env vars — use GetConfig() for the
// effective runtime URL.
func DefaultAPIURL() string {
	return defaultAPIURL
}

type Config struct {
	ApiURL       string
	DockerApiURL string
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
		ApiURL:       defaultAPIURL,
		DockerApiURL: defaultDockerUploadURL,
	}

	// Override API URL if explicitly set
	if apiURL := os.Getenv("SATUSKY_API_URL"); apiURL != "" {
		config.ApiURL = apiURL
	}

	// Override Docker API URL if explicitly set
	if dockerURL := os.Getenv("SATUSKY_DOCKER_API_URL"); dockerURL != "" {
		config.DockerApiURL = dockerURL
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
