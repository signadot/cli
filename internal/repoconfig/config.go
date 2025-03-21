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
	repoRoot, err := findGitRepoRoot(startPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find git repository root: %w", err)
	}

	configPath := filepath.Join(repoRoot, ".signadot", "config.yaml")
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

// processTestFile handles a single test file, adding it to the testFileMap if valid
func processTestFile(repoRoot string, path string, dirLabelsCache map[string]map[string]string, runID string, testFileMap map[string]TestFile) error {
	relPath, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	if _, exists := testFileMap[relPath]; exists {
		return nil
	}

	dirPath := filepath.Dir(relPath)
	var labels map[string]string
	if dirLabels, exists := dirLabelsCache[dirPath]; exists {
		labels = dirLabels
	} else {
		labels = make(map[string]string)
	}

	labels["signadot.com/run-id"] = runID

	testFileMap[relPath] = TestFile{
		Path:   relPath,
		Labels: labels,
	}

	return nil
}

// walkTestDirectory processes all test files in a directory
func walkTestDirectory(repoRoot, testsDir string, dirLabelsCache map[string]map[string]string, runID string, testFileMap map[string]TestFile) error {
	absTestsDir := filepath.Join(repoRoot, testsDir)

	if _, err := os.Stat(absTestsDir); os.IsNotExist(err) {
		return fmt.Errorf("tests directory %q does not exist: %w", absTestsDir, err)
	}

	err := filepath.Walk(absTestsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !isTestFile(path) {
			return nil
		}

		return processTestFile(repoRoot, path, dirLabelsCache, runID, testFileMap)
	})

	if err != nil {
		return fmt.Errorf("failed to walk tests directory %q: %w", absTestsDir, err)
	}

	return nil
}

// FindTestFiles finds all test files in the tests directories
func FindTestFiles(startPath string, cfg *Config) ([]TestFile, error) {
	repoRoot, err := findGitRepoRoot(startPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find git repository root: %w", err)
	}

	runID := generateRunID()

	dirLabelsCache, err := buildLabelsCache(repoRoot)
	if err != nil {
		return nil, err
	}

	testFileMap := make(map[string]TestFile)

	cleanPaths := make([]string, len(cfg.SmartTests))
	for i, path := range cfg.SmartTests {
		cleanPaths[i] = filepath.Clean(path)
	}
	sortedPaths := sortPathsByLength(cleanPaths)

	for _, testsDir := range sortedPaths {
		if err := walkTestDirectory(repoRoot, testsDir, dirLabelsCache, runID, testFileMap); err != nil {
			return nil, err
		}
	}

	var allTestFiles []TestFile
	for _, tf := range testFileMap {
		allTestFiles = append(allTestFiles, tf)
	}

	return allTestFiles, nil
}

func isTestFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".star"
}
