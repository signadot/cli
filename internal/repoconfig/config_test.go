package repoconfig

import (
	"path/filepath"
	"testing"
)

func TestLabelAggregation(t *testing.T) {
	// Create a config with the fixtures directory
	cfg := &Config{
		SmartTests: []string{"tests/fixtures/smart-tests-git/backend"},
	}

	// Find test files
	testFiles, err := FindTestFiles("tests/fixtures/smart-tests-git/backend", cfg)
	if err != nil {
		t.Fatalf("FindTestFiles failed: %v", err)
	}

	// Create a map of expected labels for each file
	expectedLabels := map[string]map[string]string{
		"tests/fixtures/smart-tests-git/backend/test.star": {
			"area":  "backend",
			"suite": "integration",
		},
		"tests/fixtures/smart-tests-git/backend/team1/file1.star": {
			"area":  "backend",
			"team":  "team1",
			"suite": "integration",
		},
		"tests/fixtures/smart-tests-git/backend/team2/file2.star": {
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

		// Verify run-id label exists
		if _, exists := tf.Labels["signadot.com/run-id"]; !exists {
			t.Errorf("File %s: Missing run-id label", expectedPath)
		}
	}
}

func TestFileDiscovery(t *testing.T) {
	// Create a config with both backend and frontend directories
	cfg := &Config{
		SmartTests: []string{
			"tests/fixtures/smart-tests-git/backend",
			"tests/fixtures/smart-tests-git/frontend",
		},
	}

	// Find test files
	testFiles, err := FindTestFiles("tests/fixtures/smart-tests-git", cfg)
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
		"tests/fixtures/smart-tests-git/backend/test.star",
		"tests/fixtures/smart-tests-git/backend/team1/file1.star",
		"tests/fixtures/smart-tests-git/backend/team2/file2.star",
		"tests/fixtures/smart-tests-git/frontend/file3.star",
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
	// Create a config with only a direct file path
	cfg := &Config{
		SmartTests: []string{
			"tests/fixtures/smart-tests-git/backend/test.star",
		},
	}

	// Find test files
	testFiles, err := FindTestFiles("tests/fixtures/smart-tests-git", cfg)
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
		"tests/fixtures/smart-tests-git/backend/test.star",
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
