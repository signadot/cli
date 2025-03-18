package repoconfig

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// readLabels reads labels from a .labels file
func readLabels(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to open .labels file: %w", err)
	}
	defer file.Close()

	labels := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid label format in %s: %q", filePath, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		labels[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading .labels file: %w", err)
	}

	return labels, nil
}

// mergeLabels merges two label maps, with the second map taking precedence
func mergeLabels(base, override map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		result[k] = v
	}
	return result
}

// buildLabelsCache walks the repository root and builds a cache of directory labels
func buildLabelsCache(repoRoot string) (map[string]map[string]string, error) {
	dirLabelsCache := make(map[string]map[string]string)

	err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		relDir, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Skip if we've already processed this directory
		if _, exists := dirLabelsCache[relDir]; exists {
			return nil
		}

		// Start with an empty label set
		labels := make(map[string]string)

		// If this is not the root directory, get parent's labels first
		if relDir != "." {
			parentDir := filepath.Dir(relDir)
			if parentLabels, exists := dirLabelsCache[parentDir]; exists {
				labels = mergeLabels(make(map[string]string), parentLabels)
			}
		}

		// Read and merge this directory's labels
		labelsFile := filepath.Join(path, ".labels")
		currentLabels, err := readLabels(labelsFile)
		if err != nil {
			return fmt.Errorf("failed to read labels from %s: %w", labelsFile, err)
		}
		if currentLabels != nil {
			labels = mergeLabels(labels, currentLabels)
		}

		// Store in cache
		dirLabelsCache[relDir] = labels
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to build labels cache: %w", err)
	}

	return dirLabelsCache, nil
}
