package repoconfig

import (
	"fmt"
	"path/filepath"
	"strings"

	giturls "github.com/chainguard-dev/git-urls"
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
		// needed to resolve HEAD and refs in linked worktrees
		// (git worktree add), whose .git is a file pointing at the
		// common dir
		EnableDotGitCommonDir: true,
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
	if u := pickRemoteURL(remotes); u != "" {
		// Normalize the URL
		// E.g.:
		// git@github.com:signadot/cli.git -> github.com/signadot/cli
		// https://github.com/signadot/cli -> github.com/signadot/cli
		remoteURL, err = normalizeGitRepo(u)
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

// pickRemoteURL deterministically picks the URL identifying the repo:
// "origin" if it exists, otherwise the first remote by name. Remotes without
// a URL are ignored. go-git returns remotes in map order, so remotes[0] is
// not stable.
func pickRemoteURL(remotes []*git.Remote) string {
	var (
		name string
		url  string
	)
	for _, r := range remotes {
		cfg := r.Config()
		if len(cfg.URLs) == 0 {
			continue
		}
		if cfg.Name == git.DefaultRemoteName {
			return cfg.URLs[0]
		}
		if url == "" || cfg.Name < name {
			name, url = cfg.Name, cfg.URLs[0]
		}
	}
	return url
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
