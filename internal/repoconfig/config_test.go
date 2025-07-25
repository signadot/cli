package repoconfig

import (
	"path/filepath"
	"testing"
)

func TestLabelAggregation(t *testing.T) {
	// Create a test finder
	tf, err := NewTestFinder("../../tests/fixtures/smart-tests-git/backend", nil, nil)
	if err != nil {
		t.Fatalf("NewTestFinder failed: %v", err)
	}
	// get the git repo
	repo := tf.GetGitRepo()
	if repo == nil {
		t.Fatal("empty git repo")
	}

	// Find test files
	testFiles, err := tf.FindTestFiles()
	if err != nil {
		t.Fatalf("FindTestFiles failed: %v", err)
	}

	// Create a map of expected labels for each file
	expectedLabels := map[string]map[string]string{
		filepath.Join(repo.Path, "tests/fixtures/smart-tests-git/backend/test.star"): {
			"area":  "backend",
			"suite": "integration",
		},
		filepath.Join(repo.Path, "tests/fixtures/smart-tests-git/backend/team1/file1.star"): {
			"area":  "backend",
			"team":  "team1",
			"suite": "integration",
		},
		filepath.Join(repo.Path, "tests/fixtures/smart-tests-git/backend/team2/file2.star"): {
			"area":  "backend",
			"team":  "team2",
			"suite": "integration",
		},
	}

	// Create a map of actual test files for easier lookup
	actualFiles := make(map[string]TestFile)
	for _, tf := range testFiles {
		actualFiles[tf.Path] = tf
	}

	// Verify each expected file has the correct labels
	for expectedPath, expectedLabels := range expectedLabels {
		tf, exists := actualFiles[expectedPath]
		if !exists {
			t.Errorf("Expected file %s not found", expectedPath)
			continue
		}

		// Verify each expected label
		for key, value := range expectedLabels {
			if actualValue, exists := tf.Labels[key]; !exists || actualValue != value {
				t.Errorf("File %s: Expected label %s=%s, got %s=%s",
					expectedPath, key, value, key, actualValue)
			}
		}
	}
}

func TestFileDiscovery(t *testing.T) {
	// Create a test finder
	tf, err := NewTestFinder("", nil, nil)
	if err != nil {
		t.Fatalf("NewTestFinder failed: %v", err)
	}
	// overwrite the config with both backend and frontend directories
	tf.cfg = &Config{
		SmartTests: []string{
			"tests/fixtures/smart-tests-git/backend",
			"tests/fixtures/smart-tests-git/frontend",
		},
	}
	// get the git repo
	repo := tf.GetGitRepo()
	if repo == nil {
		t.Fatal("empty git repo")
	}

	// Find test files
	testFiles, err := tf.FindTestFiles()
	if err != nil {
		t.Fatalf("FindTestFiles failed: %v", err)
	}

	// Create a map of actual test files for easier lookup
	actualFiles := make(map[string]TestFile)
	for _, tf := range testFiles {
		actualFiles[tf.Path] = tf
	}

	// List of all expected test files
	expectedFiles := []string{
		filepath.Join(repo.Path, "tests/fixtures/smart-tests-git/backend/test.star"),
		filepath.Join(repo.Path, "tests/fixtures/smart-tests-git/backend/team1/file1.star"),
		filepath.Join(repo.Path, "tests/fixtures/smart-tests-git/backend/team2/file2.star"),
		filepath.Join(repo.Path, "tests/fixtures/smart-tests-git/frontend/file3.star"),
	}

	// Verify all expected files are found
	for _, expectedPath := range expectedFiles {
		if _, exists := actualFiles[expectedPath]; !exists {
			t.Errorf("Expected file %s not found", expectedPath)
		}
	}

	// Verify no unexpected files are found
	if len(actualFiles) != len(expectedFiles) {
		t.Errorf("Found %d files, expected %d", len(actualFiles), len(expectedFiles))
		for path := range actualFiles {
			found := false
			for _, expected := range expectedFiles {
				if path == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Unexpected file found: %s", path)
			}
		}
	}

	// Verify file extensions
	for path := range actualFiles {
		if filepath.Ext(path) != ".star" {
			t.Errorf("File %s has unexpected extension", path)
		}
	}
}

func TestDirectFilePathInConfig(t *testing.T) {
	// Create a test finder
	tf, err := NewTestFinder("", nil, nil)
	if err != nil {
		t.Fatalf("NewTestFinder failed: %v", err)
	}
	// overwrite the config with only a direct file path
	tf.cfg = &Config{
		SmartTests: []string{
			"tests/fixtures/smart-tests-git/backend/test.star",
		},
	}
	// get the git repo
	repo := tf.GetGitRepo()
	if repo == nil {
		t.Fatal("empty git repo")
	}

	// Find test files
	testFiles, err := tf.FindTestFiles()
	if err != nil {
		t.Fatalf("FindTestFiles failed: %v", err)
	}

	// Create a map of actual test files for easier lookup
	actualFiles := make(map[string]struct{})
	for _, tf := range testFiles {
		t.Logf("Found file: %s", tf.Path)
		actualFiles[tf.Path] = struct{}{}
	}

	// We expect to find only the directly specified file
	expectedFiles := []string{
		filepath.Join(repo.Path, "tests/fixtures/smart-tests-git/backend/test.star"),
	}

	// Verify the expected file is found
	for _, expectedPath := range expectedFiles {
		if _, exists := actualFiles[expectedPath]; !exists {
			t.Errorf("Expected file %s not found", expectedPath)
		}
	}

	// Verify no unexpected files are found
	if len(actualFiles) != len(expectedFiles) {
		t.Errorf("Found %d files, expected exactly %d file (only the directly specified file)",
			len(actualFiles), len(expectedFiles))

		// Print unexpected files
		for path := range actualFiles {
			if path != expectedFiles[0] {
				t.Errorf("Unexpected file found: %s", path)
			}
		}
	}

	// Verify file extension
	for path := range actualFiles {
		if filepath.Ext(path) != ".star" {
			t.Errorf("File %s has unexpected extension", path)
		}
	}
}
