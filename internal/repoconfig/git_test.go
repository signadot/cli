package repoconfig

import "testing"

func TestGitRepoNormalization(t *testing.T) {
	u1, err := normalizeGitRepo("git@github.com:signadot/cli.git")
	if err != nil {
		t.Fatalf("TestGitRepoNormalization failed: %v", err)
	}
	u2, err := normalizeGitRepo("https://github.com/signadot/cli")
	if err != nil {
		t.Fatalf("TestGitRepoNormalization failed: %v", err)
	}
	if u1 != u2 {
		t.Fatalf("TestGitRepoNormalization failed: got different repos %q, %q", u1, u2)
	}
}
