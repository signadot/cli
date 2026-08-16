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
	// SmartTests lists directories, or individual .star files, holding the
	// smart tests this repository runs by default.
	SmartTests []string `yaml:"smart_tests"`
	// Plans lists the plan tags this repository runs by default. Entries are
	// matched against tag names as globs, so `checkout-*` selects every plan
	// tagged with a name beginning that way.
	//
	// Plans, unlike smart tests, do not live in the repository: a tag names a
	// plan the control plane already holds. What the repository contributes is
	// which of them this repository's changes should be checked against.
	Plans []string `yaml:"plans"`
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

	// Which keys are required depends on what is being run, so each caller
	// checks for its own. A repository that configures only plans is a
	// perfectly good repository as far as this file is concerned.
	return &cfg, nil
}
