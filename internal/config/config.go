package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds application configuration
type Config struct {
	// ProjectRepositories maps project ID to default repository ID
	ProjectRepositories map[string]string `json:"project_repositories"`
}

// New creates a new empty config
func New() *Config {
	return &Config{
		ProjectRepositories: make(map[string]string),
	}
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	
	configDir := filepath.Join(homeDir, ".config", "ghptui")
	
	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}
	
	return filepath.Join(configDir, "config.json"), nil
}

// Load reads the config from disk
// If the config file doesn't exist or cannot be read, returns a new empty config
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		// Can't determine config path, return empty config
		fmt.Fprintf(os.Stderr, "Warning: unable to determine config path: %v\n", err)
		return New(), nil
	}
	
	// If config doesn't exist, return new config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return New(), nil
	}
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Can't read config file, return empty config
		fmt.Fprintf(os.Stderr, "Warning: unable to read config file: %v\n", err)
		return New(), nil
	}
	
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		// Invalid JSON, return empty config
		fmt.Fprintf(os.Stderr, "Warning: config file is corrupted, using defaults: %v\n", err)
		return New(), nil
	}
	
	// Ensure map is initialized
	if cfg.ProjectRepositories == nil {
		cfg.ProjectRepositories = make(map[string]string)
	}
	
	return &cfg, nil
}

// Save writes the config to disk
// Returns an error if saving fails, but the application can continue without saving
func (c *Config) Save() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return fmt.Errorf("unable to determine config path: %w", err)
	}
	
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// GetDefaultRepository returns the default repository ID for a project
func (c *Config) GetDefaultRepository(projectID string) (string, bool) {
	repoID, ok := c.ProjectRepositories[projectID]
	return repoID, ok
}

// SetDefaultRepository sets the default repository for a project
func (c *Config) SetDefaultRepository(projectID, repositoryID string) {
	c.ProjectRepositories[projectID] = repositoryID
}

// ClearDefaultRepository removes the default repository for a project
func (c *Config) ClearDefaultRepository(projectID string) {
	delete(c.ProjectRepositories, projectID)
}
