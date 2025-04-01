package repoconfig

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	giturls "github.com/whilp/git-urls"
)

type GitRepo struct {
	Path      string
	Repo      string
	Branch    string
	CommitSHA string
}

func FindGitRepo(startPath string) (*GitRepo, error) {
	// Open the repository with dot git detection
	repo, err := git.PlainOpenWithOptions(startPath, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return nil, fmt.Errorf("not a git repository (or any parent up to mount point %s): %w",
			startPath, err)
	}

	// Get the worktree to find the root path
	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	// Get the current branch
	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Get the remote URL
	remotes, err := repo.Remotes()
	if err != nil {
		return nil, fmt.Errorf("failed to get remotes: %w", err)
	}

	var remoteURL string
	if len(remotes) > 0 {
		// Use the first remote's URL
		remoteURL = remotes[0].Config().URLs[0]

		// Normalize the URL
		// E.g.:
		// git@github.com:signadot/cli.git -> github.com/signadot/cli
		// https://github.com/signadot/cli -> github.com/signadot/cli
		remoteURL, err = normalizeGitRepo(remoteURL)
		if err != nil {
			return nil, fmt.Errorf("could not normalize git remote URL: %w", err)
		}
	}

	return &GitRepo{
		Path:      worktree.Filesystem.Root(),
		Repo:      remoteURL,
		Branch:    head.Name().Short(),
		CommitSHA: head.Hash().String(),
	}, nil
}

// GetRelativePathFromGitRoot returns the relative path of a directory within the git root directory.
// For example, if git root is "/aa/bb/cc/" and the directory is "aa/bb/cc/dd/ee",
// it will return "dd/ee".
func GetRelativePathFromGitRoot(gitRoot, dirPath string) (string, error) {
	// Clean both paths to handle any trailing slashes or ".." components
	gitRoot = filepath.Clean(gitRoot)
	dirPath = filepath.Clean(dirPath)

	// Get the relative path
	relPath, err := filepath.Rel(gitRoot, dirPath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	return relPath, nil
}

func normalizeGitRepo(url string) (string, error) {
	// In case of URLs like git@github.com:signadot/cli.git, this will convert
	// it to a standard SSH URL, e.g.:
	// ssh://git@github.com/signadot/cli.git
	u, err := giturls.Parse(url)
	if err != nil {
		return "", err
	}
	// Get the host + the path (triming the .git suffix if exists)
	return filepath.Join(u.Host, strings.TrimSuffix(u.Path, ".git")), nil
}
