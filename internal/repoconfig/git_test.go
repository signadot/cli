package repoconfig

import (
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/storage/memory"
)

func TestPickRemoteURL(t *testing.T) {
	remote := func(name string, urls ...string) *git.Remote {
		return git.NewRemote(memory.NewStorage(), &config.RemoteConfig{Name: name, URLs: urls})
	}
	cases := []struct {
		name    string
		remotes []*git.Remote
		want    string
	}{
		{"none", nil, ""},
		{"origin preferred", []*git.Remote{remote("upstream", "u"), remote("origin", "o")}, "o"},
		{"first by name", []*git.Remote{remote("zed", "z"), remote("fork", "f")}, "f"},
		{"skip no url", []*git.Remote{remote("origin"), remote("fork", "f")}, "f"},
		{"only no url", []*git.Remote{remote("origin")}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := pickRemoteURL(c.remotes); got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}
