package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken string
	AllowedUsers  []int64
	Debug         bool
	DatabasePath  string
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

	// Parse ALLOWED_USERS into list of IDs
	allowedUsers := parseAllowedUsers(os.Getenv("ALLOWED_USERS"))

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./shopping.db"
	}

	return &Config{
		TelegramToken: token,
		AllowedUsers:  allowedUsers,
		Debug:         debugEnabled,
		DatabasePath:  dbPath,
	}
}

// parseAllowedUsers parses a comma-separated string of user IDs into a slice of int64
func parseAllowedUsers(usersStr string) []int64 {
	if usersStr == "" {
		return []int64{}
	}

	parts := strings.Split(usersStr, ",")
	userIDs := make([]int64, 0, len(parts))

	for _, part := range parts {
		idStr := strings.TrimSpace(part)
		if idStr == "" {
			continue
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Printf("Warning: failed to parse user ID '%s': %v", idStr, err)
			continue
		}

		userIDs = append(userIDs, id)
	}

	return userIDs
}
