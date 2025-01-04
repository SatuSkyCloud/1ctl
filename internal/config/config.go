package config

import (
	"1ctl/internal/context"
	"1ctl/internal/utils"
	"errors"
	"os"

	"github.com/joho/godotenv"
)

const (
	defaultAPIURL = "https://api.satusky.com/v1/cli"
	devModeEnv    = "SATUSKY_DEV_MODE"
)

type Config struct {
	ApiURL  string
	DevMode bool
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

	// Override API URL if explicitly set
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
