package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken string
	AllowedUsers  string
	Debug         bool
}

func Load() *Config {
	// Load .env file (ignore error if file doesn't exist)
	godotenv.Load()

	token := os.Getenv("TG_TOKEN")
	if token == "" {
		log.Fatal("TG_TOKEN environment variable is required")
	}

	debug := os.Getenv("DEBUG")
	debugEnabled := false
	if debug != "" {
		debugEnabled = true
	}

	// TODO: Parse users into list of IDs
	allowedUsers := os.Getenv("ALLOWED_USERS")

	return &Config{
		TelegramToken: token,
		AllowedUsers:  allowedUsers,
		Debug:         debugEnabled,
	}
}
