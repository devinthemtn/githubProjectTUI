package auth

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GetToken retrieves the GitHub authentication token
// First tries to use gh CLI, then falls back to GITHUB_TOKEN env var
func GetToken() (string, error) {
	// Try gh CLI first
	token, err := getTokenFromGH()
	if err == nil && token != "" {
		return token, nil
	}

	// Fall back to environment variable
	token = os.Getenv("GITHUB_TOKEN")
	if token != "" {
		return token, nil
	}

	return "", fmt.Errorf("no GitHub token found. Please run 'gh auth login' or set GITHUB_TOKEN environment variable")
}

// getTokenFromGH retrieves token from GitHub CLI
func getTokenFromGH() (string, error) {
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", fmt.Errorf("gh returned empty token")
	}

	return token, nil
}

// CheckAuthentication verifies that we can authenticate with GitHub
func CheckAuthentication() error {
	_, err := GetToken()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	return nil
}

// GetAuthenticatedUser retrieves the currently authenticated user's login
func GetAuthenticatedUser() (string, error) {
	cmd := exec.Command("gh", "api", "user", "--jq", ".login")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get authenticated user: %w", err)
	}

	user := strings.TrimSpace(string(output))
	return user, nil
}
