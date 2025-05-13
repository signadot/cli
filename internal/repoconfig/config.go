package repoconfig

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// Config represents the .signadot/config.yaml file contents
type Config struct {
	SmartTests []string `yaml:"smart_tests"`
}

// TestFile represents a test file found in the tests directory
type TestFile struct {
	Name   string            `json:"name"`   // Test name
	Path   string            `json:"path"`   // Full path relative to base directory
	Reader io.Reader         `json:"-"`      // if Path is empty, may be a Reader
	Labels map[string]string `json:"labels"` // Labels from all parent directories
}

// LoadConfig reads the .signadot/config.yaml file from the git repository root
func LoadConfig(repo *GitRepo) (*Config, error) {
	configPath := filepath.Join(repo.Path, ".signadot", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read .signadot/config.yaml: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse .signadot/config.yaml: %w", err)
	}

	if len(cfg.SmartTests) == 0 {
		return nil, fmt.Errorf("smart_tests is required in .signadot/config.yaml")
	}
	return &cfg, nil
}
