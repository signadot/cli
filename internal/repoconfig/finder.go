package repoconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type TestFinder struct {
	cfg      *Config
	repo     *GitRepo
	basePath string

	dirLabelsCache LabelsCache
	withLabels     map[string]string
	withoutLabels  map[string]string
}

func NewTestFinder(inputPath string, withLabels, withoutLabels map[string]string) (*TestFinder, error) {
	var tf *TestFinder

	if inputPath != "" {
		// use the provided dir as the base for finding tests

		// make the input dir absolute
		basePath, err := filepath.Abs(inputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to convert input dir into an absolute dir: %w", err)
		}

		// try to find the base git repo based on the dir
		gitRepo, _ := FindGitRepo(basePath)

		// define test directory
		testDir := "."
		if gitRepo != nil {
			// there is a git repo, therefore convert the test directory
			// relative to the root of the git repo
			testDir, err = GetRelativePathFromGitRoot(gitRepo.Path, basePath)
			if err != nil {
				return nil, fmt.Errorf("failed to convert input dir relative to git root: %w", err)
			}
			basePath = gitRepo.Path
		}

		tf = &TestFinder{
			cfg: &Config{
				SmartTests: []string{testDir},
			},
			repo:          gitRepo,
			basePath:      basePath,
			withLabels:    withLabels,
			withoutLabels: withoutLabels,
		}
	} else {
		// try to read from .signadot/config in the git repo (if any)

		// get current directory
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}

		// try to find the base git repo based on current directory
		gitRepo, err := FindGitRepo(cwd)
		if err != nil {
			return nil, fmt.Errorf("failed to find git repository root: %w", err)
		}

		// try to read from .signadot/config from the git repo
		repoConf, err := LoadConfig(gitRepo)
		if err != nil {
			return nil, fmt.Errorf("failed to load .signadot/config.yaml: %w", err)
		}

		tf = &TestFinder{
			cfg:           repoConf,
			repo:          gitRepo,
			basePath:      gitRepo.Path,
			withLabels:    withLabels,
			withoutLabels: withoutLabels,
		}
	}

	// build the label cache
	dirLabelsCache, err := NewLabelsCache(tf.basePath)
	if err != nil {
		return nil, err
	}
	tf.dirLabelsCache = dirLabelsCache

	return tf, nil
}

func (tf *TestFinder) GetGitRepo() *GitRepo {
	return tf.repo
}

// FindTestFiles finds all test files in the tests directories
func (tf *TestFinder) FindTestFiles() ([]TestFile, error) {
	testFileMap := make(map[string]TestFile)
	cleanPaths := make([]string, len(tf.cfg.SmartTests))
	for i, path := range tf.cfg.SmartTests {
		cleanPaths[i] = filepath.Clean(path)
	}
	sortedPaths := sortPathsByLength(cleanPaths)

	for _, path := range sortedPaths {
		absPath := filepath.Join(tf.basePath, path)
		fileInfo, err := os.Stat(absPath)
		if err != nil {
			return nil, fmt.Errorf("failed to stat path %q: %w", path, err)
		}

		if fileInfo.IsDir() {
			if err := tf.walkTestDirectory(path, testFileMap); err != nil {
				return nil, err
			}
		} else {
			if !isTestFile(absPath) {
				continue
			}
			if err := tf.processTestFile(absPath, testFileMap); err != nil {
				return nil, err
			}
		}
	}

	var allTestFiles []TestFile
	for _, tFile := range testFileMap {
		allMatch, err := allLabelsMatch(tf.withLabels, tFile.Labels)
		if err != nil {
			return nil, err
		}
		if !allMatch {
			continue
		}
		someMatch, err := someLabelMatches(tf.withoutLabels, tFile.Labels)
		if err != nil {
			return nil, err
		}
		if someMatch {
			continue
		}
		allTestFiles = append(allTestFiles, tFile)
	}

	return allTestFiles, nil
}

func allLabelsMatch(filter, labels map[string]string) (bool, error) {
	for key, valGlob := range filter {
		lVal, ok := labels[key]
		if !ok {
			return false, nil
		}
		match, err := filepath.Match(valGlob, lVal)
		if err != nil {
			return false, err
		}
		if !match {
			return false, nil
		}
	}
	return true, nil
}

func someLabelMatches(filter, labels map[string]string) (bool, error) {
	for key, valGlob := range filter {
		lVal, ok := labels[key]
		if !ok {
			continue
		}
		match, err := filepath.Match(valGlob, lVal)
		if err != nil {
			return false, err
		}
		if match {
			return true, nil
		}
	}
	return false, nil
}

// processTestFile handles a single test file, adding it to the testFileMap if valid
func (tf *TestFinder) processTestFile(path string, testFileMap map[string]TestFile) error {
	relPath, err := filepath.Rel(tf.basePath, path)
	if err != nil {
		return fmt.Errorf("failed to get relative path: %w", err)
	}

	if _, exists := testFileMap[relPath]; exists {
		return nil
	}

	labels, err := tf.dirLabelsCache.ForFile(relPath)
	if err != nil {
		return fmt.Errorf("unable to get labels for path %q: %w", relPath, err)
	}

	testFileMap[relPath] = TestFile{
		Name:   tf.getTestName(relPath),
		Path:   filepath.Join(tf.basePath, relPath),
		Labels: labels,
	}

	return nil
}

// walkTestDirectory processes all test files in a directory
func (tf *TestFinder) walkTestDirectory(testsDir string, testFileMap map[string]TestFile) error {
	absTestsDir := filepath.Join(tf.basePath, testsDir)

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
		return tf.processTestFile(path, testFileMap)
	})

	if err != nil {
		return fmt.Errorf("failed to walk tests directory %q: %w", absTestsDir, err)
	}
	return nil
}

func (tf *TestFinder) getTestName(path string) string {
	if tf.repo != nil {
		// if there is a git repo, define the test name as git repo + git path
		return filepath.Join(tf.repo.Repo, path)
	}

	// otherwise use the hostname + absolute path
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	return hostname + "@" + filepath.Join(tf.basePath, path)
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

func isTestFile(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".star"
}
