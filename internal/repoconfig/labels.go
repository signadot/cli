package repoconfig

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type LabelsCache interface {
	ForFile(p string) (map[string]string, error)
}

type labelsCache struct {
	root      string
	labelsMap map[string]map[string]string
}

// ForFile: finds the labels for a given path relative to the labels cache root.
func (l *labelsCache) ForFile(relFile string) (map[string]string, error) {
	relDir := filepath.Dir(relFile)
	return l.labelsMap[relDir], nil
}

func NewLabelsCache(rootPath string) (LabelsCache, error) {
	cache, err := buildLabelsCache(rootPath)
	if err != nil {
		return nil, err
	}
	return &labelsCache{root: rootPath, labelsMap: cache}, nil
}

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
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("invalid label format in %s: %q", filePath, line)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		labels[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading .labels file: %w", err)
	}

	return labels, nil
}

// buildLabelsCache walks the repository root and builds a cache of directory labels
func buildLabelsCache(repoRoot string) (map[string]map[string]string, error) {
	dirLabelsCache := make(map[string]map[string]string)

	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		relDir, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		if _, exists := dirLabelsCache[relDir]; exists {
			return nil
		}

		labels := make(map[string]string)

		// If this is not the root directory, get parent's labels first
		if relDir != "." {
			parentDir := filepath.Dir(relDir)
			if parentLabels, exists := dirLabelsCache[parentDir]; exists {
				for k, v := range parentLabels {
					labels[k] = v
				}
			}
		}

		// Read and merge this directory's labels
		labelsFile := filepath.Join(path, ".labels")
		currentLabels, err := readLabels(labelsFile)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to read labels from %s: %w", labelsFile, err)
		}
		for k, v := range currentLabels {
			labels[k] = v
		}

		dirLabelsCache[relDir] = labels
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to build labels cache: %w", err)
	}

	return dirLabelsCache, nil
}
