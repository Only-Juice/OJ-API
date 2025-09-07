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

// GetExternalURL returns the complete external OJ URL with configurable scheme
func GetGiteaExternalURL() string {
	externalURL := Config("GIT_EXTERNAL_URL")
	if externalURL == "" {
		externalURL = GetGiteaBaseURL() // Default to Gitea base URL if not provided
	}
	return externalURL
}

// GetOJBaseURL returns the complete OJ base URL with configurable scheme
func GetOJBaseURL() string {
	ojBaseURL := Config("OJ_BASE_URL")
	if ojBaseURL == "" {
		ojBaseURL = "http://" + Config("OJ_HOST") // Default to http if not provided
	}
	return ojBaseURL
}

// OJ_EXTERNAL_URL returns the external host for generating user links
func GetOJExternalURL() string {
	externalHost := Config("OJ_EXTERNAL_URL")
	if externalHost == "" {
		externalHost = GetOJBaseURL() // Default to OJ base URL if not provided
	}
	return externalHost
}

func GetFrontendURL() string {
	frontendURL := Config("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "https://oj.is1ab.com/" // Default to a specific URL if not provided
	}
	return frontendURL
}

func GetIsolatePath() string {
	isolatePath := Config("ISOLATE_PATH")
	if isolatePath == "" {
		isolatePath = "/var/local/lib/isolate" // Default path if not provided
	}
	return isolatePath
}

// GetGiteaOAuthConfig returns the Gitea OAuth configuration
func GetGiteaOAuthConfig() struct {
	URL          string
	ClientID     string
	ClientSecret string
	Issuer       string
} {
	return struct {
		URL          string
		ClientID     string
		ClientSecret string
		Issuer       string
	}{
		URL:          Config("GITEA_URL"),
		ClientID:     Config("GITEA_CLIENT_ID"),
		ClientSecret: Config("GITEA_CLIENT_SECRET"),
		Issuer:       Config("GITEA_ISSUER"),
	}
}
