package repoconfig

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/go-git/go-git/v5"
	"gopkg.in/yaml.v2"
)

// Config represents the .signadot/config.yaml file contents
type Config struct {
	SmartTests []string `yaml:"smart_tests"`
}

// TestFile represents a test file found in the tests directory
type TestFile struct {
	Path   string            // Full path relative to repository root
	Labels map[string]string // Labels from all parent directories
}

// generateRunID creates a random alphanumeric string for the run
func generateRunID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	const length = 8

	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Generate random string
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// findGitRepoRoot finds the root of the git repository using go-git
func findGitRepoRoot(startPath string) (string, error) {
	repo, err := git.PlainOpenWithOptions(startPath, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return "", fmt.Errorf("not a git repository (or any parent up to mount point %s): %w", startPath, err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	return worktree.Filesystem.Root(), nil
}

// LoadConfig reads the .signadot/config.yaml file from the git repository root
func LoadConfig(startPath string) (*Config, error) {
	// Find git repository root
	repoRoot, err := findGitRepoRoot(startPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find git repository root: %w", err)
	}

	configPath := filepath.Join(repoRoot, ".signadot", "config.yaml")
	fmt.Printf("Reading config from: %s\n", configPath)
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

// sortPathsByLength sorts paths by their length (shortest first)
func sortPathsByLength(paths []string) []string {
	sorted := make([]string, len(paths))
	copy(sorted, paths)
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i]) < len(sorted[j])
	})
	return sorted
}

// FindTestFiles finds all test files in the tests directories
func FindTestFiles(startPath string, cfg *Config) ([]TestFile, error) {
	// Find git repository root
	repoRoot, err := findGitRepoRoot(startPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find git repository root: %w", err)
	}

	// Generate a run ID for this execution
	runID := generateRunID()

	// Build the directory labels cache
	dirLabelsCache, err := buildLabelsCache(repoRoot)
	if err != nil {
		return nil, err
	}

	// Use a map to deduplicate files by their full path
	testFileMap := make(map[string]TestFile)

	// Sort paths by length to process parent directories first
	sortedPaths := sortPathsByLength(cfg.SmartTests)

	// Collect test files using the cached labels
	for _, testsDir := range sortedPaths {
		absTestsDir := filepath.Join(repoRoot, testsDir)

		// Check if directory exists
		if _, err := os.Stat(absTestsDir); os.IsNotExist(err) {
			return nil, fmt.Errorf("tests directory %q does not exist", absTestsDir)
		}

		err := filepath.Walk(absTestsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip directories and non-test files
			if info.IsDir() || !isTestFile(path) {
				return nil
			}

			// Get relative path from repository root
			relPath, err := filepath.Rel(repoRoot, path)
			if err != nil {
				return fmt.Errorf("failed to get relative path: %w", err)
			}

			// If we've already seen this file, skip it
			if _, exists := testFileMap[relPath]; exists {
				return nil
			}

			// Get directory labels from cache
			dirPath := filepath.Dir(relPath)
			var labels map[string]string
			if dirLabels, exists := dirLabelsCache[dirPath]; exists {
				labels = dirLabels
			} else {
				labels = make(map[string]string)
			}

			// Add the run ID label
			labels["signadot.com/run-id"] = runID

			testFileMap[relPath] = TestFile{
				Path:   relPath,
				Labels: labels,
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to walk tests directory %q: %w", absTestsDir, err)
		}
	}

	// Convert map to slice
	var allTestFiles []TestFile
	for _, tf := range testFileMap {
		allTestFiles = append(allTestFiles, tf)
	}

	return allTestFiles, nil
}

// isTestFile checks if a file is a test file
func isTestFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".star" // Add other test file extensions as needed
}
