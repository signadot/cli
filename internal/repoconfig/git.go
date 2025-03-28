package repoconfig

import (
	"fmt"
	"path/filepath"

	"github.com/go-git/go-git/v5"
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
	}

	return &GitRepo{
		Path:      worktree.Filesystem.Root(),
		Repo:      remoteURL,
		Branch:    head.Name().Short(),
		CommitSHA: head.Hash().String(),
	}, nil
}

// GetRelativePathFromGitRoot returns the relative path of a directory within
// the git root directory. For example, if git root is "/aa/bb/cc/" and the
// directory is "/aa/bb/cc/dd/ee", it will return "dd/ee".
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
