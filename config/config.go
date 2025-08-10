package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config func to get env value
func Config(key string) string {
	// load .env file
	err := godotenv.Load(".env.local")
	if err != nil {
		fmt.Print("Error loading .env.local file")
	}
	// Return the value of the variable
	return os.Getenv(key)
}

// GetGiteaBaseURL returns the complete Gitea base URL with configurable scheme
func GetGiteaBaseURL() string {
	gitBaseURL := Config("GIT_BASE_URL")
	if gitBaseURL == "" {
		gitBaseURL = "http://" + Config("GIT_HOST") // Default to http if not provided
	}
	return gitBaseURL
}

// GetOJBaseURL returns the complete OJ base URL with configurable scheme
func GetOJBaseURL() string {
	ojBaseURL := Config("OJ_BASE_URL")
	if ojBaseURL == "" {
		ojBaseURL = "http://" + Config("OJ_HOST") // Default to http if not provided
	}
	return ojBaseURL
}

func GetIsolatePath() string {
	isolatePath := Config("ISOLATE_PATH")
	if isolatePath == "" {
		isolatePath = "/var/local/lib/isolate" // Default path if not provided
	}
	return isolatePath
}
